package database

import (
	"errors"
	"time"

	"github.com/jinzhu/gorm"
)

// GroupGroupStore implements database operations on `groups_groups`
// (which stores parent-child relationships between groups).
type GroupGroupStore struct {
	*DataStore
}

// WhereUserIsMember returns a composable query of direct ancestors (parents) of user's self group,
// i.e. groups of which he is a direct member
func (s *GroupGroupStore) WhereUserIsMember(user *User) *DB {
	result := s.Where(QuoteName(s.tableName)+".child_group_id = ?", user.GroupID)
	if s.tableName != "groups_groups_active" {
		result = result.WhereGroupRelationIsActual()
	}
	return result
}

func (s *GroupGroupStore) createNewAncestors() {
	s.DataStore.createNewAncestors("groups", "group")
}

// ErrRelationCycle is returned by CreateRelation() if the relation is impossible because it would
// create a cycle in the groups_groups graph.
var ErrRelationCycle = errors.New("a group cannot become an ancestor of itself")

const groupsRelationsLockTimeout = 3 * time.Second

// ParentChild represents a (ParentID, ChildID) pair with a role.
// If the role is empty, the default value ("member") is used.
type ParentChild struct {
	ParentID int64
	ChildID  int64
	Role     string
}

// CreateRelation creates a direct relation between two groups
func (s *GroupGroupStore) CreateRelation(parentGroupID, childGroupID int64) (err error) {
	s.mustBeInTransaction()
	defer recoverPanics(&err)

	mustNotBeError(s.WithNamedLock(s.tableName, groupsRelationsLockTimeout, func(store *DataStore) (err error) {
		mustNotBeError(store.GroupGroups().Delete("child_group_id = ? AND parent_group_id = ?", childGroupID, parentGroupID).Error())

		var rows []interface{}
		mustNotBeError(store.GroupAncestors().
			WithWriteLock().
			Select("id").
			// do not allow cycles even via expired relations
			Where("child_group_id = ? AND ancestor_group_id = ?", parentGroupID, childGroupID).
			Limit(1).
			Scan(&rows).Error())
		if len(rows) > 0 {
			return ErrRelationCycle
		}

		groupGroupStore := store.GroupGroups()
		groupGroupStore.createRelation(parentGroupID, childGroupID, "")
		groupGroupStore.createNewAncestors()
		return nil
	}))
	return err
}

func (s *GroupGroupStore) createRelation(parentGroupID, childGroupID int64, role string) {
	s.mustBeInTransaction()
	mustNotBeError(s.db.Exec(
		"SET @maxIChildOrder = IFNULL((SELECT MAX(child_order) FROM `groups_groups` WHERE `parent_group_id` = ? FOR UPDATE), 0)",
		parentGroupID).Error)

	mustNotBeError(s.retryOnDuplicatePrimaryKeyError(func(db *DB) error {
		store := NewDataStore(db).GroupGroups()
		newID := store.NewID()
		relationMap := map[string]interface{}{
			"id":              newID,
			"parent_group_id": parentGroupID,
			"child_group_id":  childGroupID,
			"child_order":     gorm.Expr("@maxIChildOrder+1"),
		}
		if role != "" {
			relationMap["role"] = role
		}
		return store.GroupGroups().InsertMap(relationMap)
	}))
}

// CreateRelationsWithoutChecking creates multiple direct relations between group pairs at once
// without checking for possible cycles in the graph and without deletion of old relations.
// This method is only suitable to create relations with new groups.
func (s *GroupGroupStore) CreateRelationsWithoutChecking(pairs []ParentChild) (err error) {
	s.mustBeInTransaction()
	defer recoverPanics(&err)

	mustNotBeError(s.WithNamedLock(s.tableName, groupsRelationsLockTimeout, func(store *DataStore) (err error) {
		groupGroupStore := store.GroupGroups()
		for _, pair := range pairs {
			groupGroupStore.createRelation(pair.ParentID, pair.ChildID, pair.Role)
		}
		groupGroupStore.createNewAncestors()
		return nil
	}))
	return err
}

// ErrGroupBecomesOrphan is to be returned if a group is going to become an orphan
// after the relation is deleted. DeleteRelation() returns this error to inform
// the caller that a confirmation is needed (shouldDeleteOrphans should be true).
var ErrGroupBecomesOrphan = errors.New("a group cannot become an orphan")

// DeleteRelation deletes a relation between two groups. It can also delete orphaned groups.
func (s *GroupGroupStore) DeleteRelation(parentGroupID, childGroupID int64, shouldDeleteOrphans bool) (err error) {
	s.mustBeInTransaction()
	defer recoverPanics(&err)

	mustNotBeError(s.WithNamedLock(s.tableName, groupsRelationsLockTimeout, func(store *DataStore) error {
		// check if parent_group_id is the only parent of child_group_id
		shouldDeleteChildGroup := false
		var result []interface{}
		mustNotBeError(s.GroupGroups().WithWriteLock().
			Select("1").
			Where("child_group_id = ?", childGroupID).
			Where("parent_group_id != ?", parentGroupID).
			Limit(1).Scan(&result).Error())
		if len(result) == 0 {
			shouldDeleteChildGroup = true
			if !shouldDeleteOrphans {
				return ErrGroupBecomesOrphan
			}
		}

		var candidatesForDeletion []int64
		if shouldDeleteChildGroup {
			// Candidates for deletion are all groups that are descendants of childGroupID filtered by type
			mustNotBeError(s.Groups().WithWriteLock().
				Joins(`
					JOIN groups_ancestors AS ancestors ON
						ancestors.child_group_id = groups.id AND
						ancestors.is_self = 0 AND
						ancestors.ancestor_group_id = ?`, childGroupID).
				Where("groups.type NOT IN('Base', 'UserAdmin', 'UserSelf')").
				Pluck("groups.id", &candidatesForDeletion).Error())
		}

		// triggers/cascading delete from groups_ancestors (all except self-links) and groups_propagate,
		// but the `before_delete_groups_groups` trigger inserts into groups_propagate again :(
		const deleteGroupsQuery = `
			DELETE ` + "`groups`" + `, group_children, group_parents, groups_attempts,
						 groups_login_prefixes, filters
			FROM ` + "`groups`" + `
			LEFT JOIN groups_groups AS group_children
				ON group_children.parent_group_id = groups.id
			LEFT JOIN groups_groups AS group_parents
				ON group_parents.child_group_id = groups.id
			LEFT JOIN groups_attempts
				ON groups_attempts.group_id = groups.id
			LEFT JOIN groups_login_prefixes
				ON groups_login_prefixes.group_id = groups.id
			LEFT JOIN filters
				ON filters.group_id = groups.id
			WHERE groups.id IN(?)`

		// delete the relation we are asked to delete (triggers will delete a lot from groups_ancestors and mark relations for propagation)
		mustNotBeError(s.GroupGroups().Delete("parent_group_id = ? AND child_group_id = ?", parentGroupID, childGroupID).Error())

		if shouldDeleteChildGroup {
			// we delete the orphan here in order to recalculate new ancestors correctly
			mustNotBeError(s.db.Exec(deleteGroupsQuery, []int64{childGroupID}).Error)
		}
		// recalculate relations
		s.GroupGroups().createNewAncestors()

		if shouldDeleteChildGroup {
			var idsToDelete []int64
			// besides the group with id = childGroupID, we also want to delete its descendants
			// whose ancestors list consists only of childGroupID descendants
			// (since they would become orphans)
			if len(candidatesForDeletion) > 0 {
				mustNotBeError(s.Groups().WithWriteLock().
					Joins(`
						LEFT JOIN(
							SELECT groups_ancestors.child_group_id
							FROM groups_ancestors
							WHERE
								groups_ancestors.ancestor_group_id NOT IN(?) AND
								groups_ancestors.child_group_id IN(?) AND
								groups_ancestors.is_self = 0
							GROUP BY groups_ancestors.child_group_id
							FOR UPDATE
						) AS ancestors
						ON ancestors.child_group_id = groups.id`, candidatesForDeletion, candidatesForDeletion).
					Where("groups.id IN (?)", candidatesForDeletion).
					Where("ancestors.child_group_id IS NULL").
					Pluck("groups.id", &idsToDelete).Error())

				if len(idsToDelete) > 0 {
					deleteResult := s.db.Exec(deleteGroupsQuery, idsToDelete)
					mustNotBeError(deleteResult.Error)
					if deleteResult.RowsAffected > 0 {
						s.GroupGroups().createNewAncestors()
					}
				}
			}

			idsToDelete = append(idsToDelete, childGroupID)
			// delete self relations of the removed groups
			mustNotBeError(s.GroupAncestors().Delete("ancestor_group_id IN (?)", idsToDelete).Error())
			// delete removed groups from groups_propagate
			mustNotBeError(s.Table("groups_propagate").Delete("id IN (?)", idsToDelete).Error())
		}
		return nil
	}))
	return nil
}

// After is a "listener" that calls GroupGroupStore::createNewAncestors()
func (s *GroupGroupStore) After() (err error) {
	s.mustBeInTransaction()
	defer recoverPanics(&err)

	s.createNewAncestors()
	return nil
}
