package groups

import (
	"net/http"
	"strings"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation GET /groups/{group_id}/path-from-root group-memberships groupPathFromRootFind
//
//		---
//		summary: Find a group path
//		description: >
//	  Finds a path from any of root groups to a given group.
//
//
//	  A path is an array of group ids from a visible group root
//	  (a visible non-"base" group without non-"base" parent) to the input group.
//	  Each group must be visible, so either
//	  1) ancestors of groups he joined,
//	  2) ancestors of non-user groups he manages,
//	  3) descendants of groups he manages,
//	  4) groups with is_public=1.
//	  Of all possible paths the service chooses any of shortest ones.
//
//
//	  At least one path should exist, otherwise the 'forbidden' error is returned.
//		parameters:
//			- name: group_id
//				in: path
//				type: integer
//				format: int64
//				required: true
//		responses:
//			"200":
//				description: OK. Success response with the found group path
//				schema:
//						type: object
//						properties:
//							path:
//								type: array
//								items:
//									type: string
//									format: int64
//								example: ["1", "2", "3"]
//						required:
//							- path
//			"400":
//				"$ref": "#/responses/badRequestResponse"
//			"401":
//				"$ref": "#/responses/unauthorizedResponse"
//			"403":
//				"$ref": "#/responses/forbiddenResponse"
//			"500":
//				"$ref": "#/responses/internalErrorResponse"
func (srv *Service) getPathFromRoot(w http.ResponseWriter, r *http.Request) service.APIError {
	user := srv.GetUser(r)
	groupID, err := service.ResolveURLQueryPathInt64Field(r, "group_id")
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	ids := findGroupPath(srv.GetStore(r), groupID, user)
	if ids == nil {
		return service.InsufficientAccessRightsError
	}
	render.Respond(w, r, map[string]interface{}{"path": ids})
	return service.NoError
}

func findGroupPath(store *database.DataStore, groupID int64, user *database.User) []string {
	visibleAncestors := store.Groups().PickVisibleGroups(
		store.ActiveGroupAncestors().Where("child_group_id = ?", groupID).
			Joins("JOIN `groups` ON groups.id = ancestor_group_id"), user).
		Select("ancestor_group_id AS id").
		Where("groups.type != 'Base'")

	var pathStrings []string
	service.MustNotBeError(store.Raw(`
			WITH RECURSIVE paths (path, length, last_group_id) AS (
				WITH visible_ancestors AS (?)
				(SELECT CAST(id AS CHAR(1024)), 1, id FROM visible_ancestors
				WHERE NOT EXISTS(
					SELECT 1 FROM groups_groups_active
					JOIN visible_ancestors AS parent ON parent.id = groups_groups_active.parent_group_id
					WHERE groups_groups_active.child_group_id = visible_ancestors.id
				)
				UNION
				SELECT CONCAT(paths.path, '/', visible_ancestors.id), length+1, visible_ancestors.id
				FROM paths
				JOIN groups_groups_active ON groups_groups_active.parent_group_id = paths.last_group_id
				JOIN visible_ancestors ON visible_ancestors.id = groups_groups_active.child_group_id)
			)
			SELECT path FROM paths WHERE last_group_id = ? ORDER BY length ASC, path LIMIT 1`,
		visibleAncestors.SubQuery(), groupID).
		ScanIntoSlices(&pathStrings).Error())

	if len(pathStrings) == 0 {
		return nil
	}
	pathString := pathStrings[0]
	idStrings := strings.Split(pathString, "/")
	return idStrings
}
