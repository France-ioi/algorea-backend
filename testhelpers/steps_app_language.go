//go:build !prod

package testhelpers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/messages-go/v10"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/rand"
	"github.com/France-ioi/AlgoreaBackend/app/utils"
)

const ReferencePrefix = '@'

// ctx.getParameterMap parses parameters in format key1=val1,key2=val2,... into a map.
func (ctx *TestContext) getParameterMap(parameters string) map[string]string {
	parameterMap := make(map[string]string)
	arrayParameters := strings.Split(parameters, ",")
	for _, paramKeyValue := range arrayParameters {
		keyVal := strings.Split(paramKeyValue, "=")

		parameterMap[keyVal[0]] = keyVal[1]
	}

	return parameterMap
}

// getParameterString converts parameters into a string with format key1=val1,key2=val2,...
func getParameterString(parameters map[string]string) string {
	var str string
	for key, value := range parameters {
		if str != "" {
			str += ","
		}
		str += key + "=" + value
	}

	return str
}

// referenceToName returns the name of a reference.
func referenceToName(reference string) string {
	if reference[0] == ReferencePrefix {
		return reference[1:]
	}

	return reference
}

// getRowMap convert a PickleTable's row into a map where the keys are the column headers.
func (ctx *TestContext) getRowMap(rowIndex int, table *messages.PickleStepArgument_PickleTable) map[string]string {
	rowHeader := table.Rows[0]
	sourceRow := table.Rows[rowIndex]

	rowMap := map[string]string{}
	for i := 0; i < len(rowHeader.Cells); i++ {
		value := sourceRow.Cells[i].Value
		if value == "" {
			continue
		}

		rowMap[rowHeader.Cells[i].Value] = value
	}

	return rowMap
}

// populateDatabase populate the database with all the initialized data.
func (ctx *TestContext) populateDatabase() error {
	db, err := database.Open(ctx.db())
	if err != nil {
		return err
	}

	// add all the defined table rows in the database.
	err = database.NewDataStore(db).InTransaction(func(store *database.DataStore) error {
		store.Exec("SET FOREIGN_KEY_CHECKS=0")
		defer store.Exec("SET FOREIGN_KEY_CHECKS=1")

		for tableName, tableRows := range ctx.dbTables {
			for _, tableRow := range tableRows {
				err = database.NewDataStoreWithTable(store.DB, tableName).InsertOrUpdateMap(tableRow, nil)
				if err != nil {
					return fmt.Errorf("populateDatabase %s %+v: %v", tableName, tableRow, err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return ctx.DBGroupsAncestorsAreComputed()
}

func (ctx *TestContext) isInDatabase(tableName, key string) bool {
	if ctx.dbTables[tableName] == nil {
		return false
	}

	_, ok := ctx.dbTables[tableName][key]
	return ok
}

func mergeFields(row1, row2 map[string]interface{}) map[string]interface{} {
	merged := row1
	for key, value := range row2 {
		merged[key] = value
	}

	return merged
}

func (ctx *TestContext) addInDatabase(tableName, key string, row map[string]interface{}) {
	if ctx.dbTables[tableName] == nil {
		ctx.dbTables[tableName] = make(map[string]map[string]interface{})
	}

	if oldRow, ok := ctx.dbTables[tableName][key]; ok {
		row = mergeFields(oldRow, row)
	}

	ctx.dbTables[tableName][key] = row
}

func (ctx *TestContext) getUserKey(fields map[string]string) string {
	if _, ok := fields["group_id"]; !ok {
		panic(fmt.Errorf("getUserKey: %v must have a group_id", fields))
	}

	return fields["group_id"]
}

// addUser adds a user in database.
func (ctx *TestContext) addUser(fields map[string]string) {
	dbFields := make(map[string]interface{})
	for key, value := range fields {
		if key == "user" {
			key = "login"
		}

		switch {
		case strings.HasSuffix(key, "_id"):
			dbFields[key] = ctx.getReference(value)
		case value[0] == ReferencePrefix:
			dbFields[key] = value[1:]
		default:
			dbFields[key] = value
		}
	}

	ctx.addInDatabase("users", ctx.getUserKey(fields), dbFields)
}

// addGroup adds a group in database.
func (ctx *TestContext) addGroup(group, name, groupType string) {
	groupID := ctx.getReference(group)

	ctx.addInDatabase("groups", strconv.FormatInt(groupID, 10), map[string]interface{}{
		"id":   groupID,
		"name": referenceToName(name),
		"type": groupType,
	})
}

// addPermissionGenerated adds a permission generated in the database.
func (ctx *TestContext) addPersonalInfoViewApprovedFor(childGroup, parentGroup string) {
	parentGroupID := ctx.getReference(parentGroup)
	childGroupID := ctx.getReference(childGroup)

	groupGroupTable := "groups_groups"
	key := ctx.getGroupGroupKey(parentGroupID, childGroupID)
	if !ctx.isInDatabase(groupGroupTable, key) {
		ctx.addGroupGroup(parentGroup, childGroup)
	}

	ctx.dbTables[groupGroupTable][key]["personal_info_view_approved_at"] = time.Now()
}

// getGroupGroupKey gets a group group unique key for the groupgroup's map.
func (ctx *TestContext) getGroupGroupKey(parentGroupID, childGroupID int64) string {
	return strconv.FormatInt(parentGroupID, 10) + "," + strconv.FormatInt(childGroupID, 10)
}

// addGroupGroup adds a group-group in the database.
func (ctx *TestContext) addGroupGroup(parentGroup, childGroup string) {
	parentGroupID := ctx.getReference(parentGroup)
	childGroupID := ctx.getReference(childGroup)

	ctx.addInDatabase("groups_groups", ctx.getGroupGroupKey(parentGroupID, childGroupID), map[string]interface{}{
		"parent_group_id": parentGroupID,
		"child_group_id":  childGroupID,
	})
}

// addGroupManager adds a group manager in the database.
func (ctx *TestContext) addGroupManager(manager, group, canWatchMembers string) {
	managerID := ctx.getReference(manager)
	groupID := ctx.getReference(group)

	ctx.addInDatabase(
		"group_managers",
		strconv.FormatInt(managerID, 10)+","+strconv.FormatInt(groupID, 10),
		map[string]interface{}{
			"manager_id":        managerID,
			"group_id":          groupID,
			"can_watch_members": canWatchMembers,
		},
	)
}

// addPermissionGenerated adds a permission generated in the database.
func (ctx *TestContext) addPermissionGenerated(group, item, watchType, watchValue string) {
	groupID := ctx.getReference(group)
	itemID := ctx.getReference(item)

	permissionsGeneratedTable := "permissions_generated"
	key := strconv.FormatInt(groupID, 10) + "," + strconv.FormatInt(itemID, 10)
	if !ctx.isInDatabase(permissionsGeneratedTable, key) {
		ctx.addInDatabase(permissionsGeneratedTable, key, map[string]interface{}{
			"group_id": groupID,
			"item_id":  itemID,
		})
	}

	ctx.dbTables[permissionsGeneratedTable][key]["can_"+watchType+"_generated"] = watchValue
}

// addPermissionsGranted adds a permission granted in the database.
func (ctx *TestContext) addPermissionGranted(group, sourceGroup, item, canRequestHelpTo string) {
	groupID := ctx.getReference(group)
	sourceGroupID := ctx.getReference(sourceGroup)
	itemID := ctx.getReference(item)

	ctx.addInDatabase(
		"permissions_granted",
		strconv.FormatInt(groupID, 10)+","+strconv.FormatInt(itemID, 10),
		map[string]interface{}{
			"group_id":            groupID,
			"source_group_id":     sourceGroupID,
			"item_id":             itemID,
			"can_request_help_to": canRequestHelpTo,
		},
	)
}

// addAttempt adds an attempt in database.
func (ctx *TestContext) addAttempt(item, participant string) {
	itemID := ctx.getReference(item)
	participantID := ctx.getReference(participant)

	ctx.addInDatabase(
		`attempts`,
		strconv.FormatInt(itemID, 10)+","+strconv.FormatInt(participantID, 10),
		map[string]interface{}{
			"id":             itemID,
			"participant_id": participantID,
		},
	)
}

// addResult adds a result in database.
func (ctx *TestContext) addResult(attemptID, participant, item string, validatedAt time.Time) {
	participantID := ctx.getReference(participant)
	itemID := ctx.getReference(item)

	ctx.addInDatabase(
		"results",
		attemptID+","+strconv.FormatInt(participantID, 10)+","+strconv.FormatInt(itemID, 10),
		map[string]interface{}{
			"attempt_id":     attemptID,
			"participant_id": participantID,
			"item_id":        itemID,
			"validated_at":   validatedAt,
		},
	)
}

// addItem adds an item in the database.
func (ctx *TestContext) addItem(item string) {
	itemID := ctx.getReference(item)

	ctx.addInDatabase("items", strconv.FormatInt(itemID, 10), map[string]interface{}{
		"id":                   itemID,
		"default_language_tag": "en",
		"type":                 "Task",
	})
}

// getThreadKey gets a thread unique key for the thread's map.
func (ctx *TestContext) getThreadKey(itemID, participantID int64) string {
	return strconv.FormatInt(itemID, 10) + "," + strconv.FormatInt(participantID, 10)
}

// addThread adds a thread in database.
func (ctx *TestContext) addThread(item, participant, helperGroup, status, messageCount, latestUpdateAt string) {
	itemID := ctx.getReference(item)
	participantID := ctx.getReference(participant)
	helperGroupID := ctx.getReference(helperGroup)

	latestUpdateAtDate, err := time.Parse(utils.DateTimeFormat, latestUpdateAt)
	if err != nil {
		panic(err)
	}

	ctx.addInDatabase("threads", ctx.getThreadKey(itemID, participantID), map[string]interface{}{
		"item_id":          itemID,
		"participant_id":   participantID,
		"helper_group_id":  helperGroupID,
		"status":           status,
		"message_count":    messageCount,
		"latest_update_at": latestUpdateAtDate,
	})
}

// IAm Sets the current user.
func (ctx *TestContext) IAm(name string) error {
	err := ctx.ThereIsAUser(name)
	if err != nil {
		return err
	}

	err = ctx.IAmUserWithID(ctx.getReference(name))
	if err != nil {
		return err
	}

	ctx.user = name

	return nil
}

// ThereIsAUser create a user.
func (ctx *TestContext) ThereIsAUser(name string) error {
	return ctx.ThereIsAUserWith(getParameterString(map[string]string{
		"group_id": name,
		"user":     name,
	}))
}

// ThereIsAUserWith creates a new user.
func (ctx *TestContext) ThereIsAUserWith(parameters string) error {
	user := ctx.getParameterMap(parameters)

	if _, ok := user["group_id"]; !ok {
		user["group_id"] = user["user"]
	}

	ctx.addUser(user)

	return ctx.ThereIsAGroupWith(getParameterString(map[string]string{
		"id":   user["group_id"],
		"name": user["user"],
		"type": "User",
	}))
}

// ThereAreTheFollowingGroups defines groups.
func (ctx *TestContext) ThereAreTheFollowingGroups(groups *messages.PickleStepArgument_PickleTable) error {
	for i := 1; i < len(groups.Rows); i++ {
		group := ctx.getRowMap(i, groups)

		err := ctx.ThereIsAGroup(group["group"])
		if err != nil {
			return err
		}

		if _, ok := group["parent"]; ok {
			err = ctx.GroupIsAChildOfTheGroup(group["group"], group["parent"])
			if err != nil {
				return err
			}
		}

		if _, ok := group["members"]; ok {
			members := strings.Split(group["members"], ",")

			for _, member := range members {
				err = ctx.ThereIsAUser(member)
				if err != nil {
					return err
				}

				err = ctx.ThereIsAGroup(member)
				if err != nil {
					return err
				}

				err = ctx.GroupIsAChildOfTheGroup(member, group["group"])
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ThereIsAGroupWith creates a new group.
func (ctx *TestContext) ThereIsAGroupWith(parameters string) error {
	group := ctx.getParameterMap(parameters)

	if _, ok := group["name"]; !ok {
		group["name"] = "Group " + group["id"]
	}
	if _, ok := group["type"]; !ok {
		group["type"] = "Class"
	}

	ctx.addGroup(group["id"], group["name"], group["type"])

	return nil
}

// ThereAreTheFollowingUsers defines users.
func (ctx *TestContext) ThereAreTheFollowingUsers(users *messages.PickleStepArgument_PickleTable) error {
	for i := 1; i < len(users.Rows); i++ {
		user := ctx.getRowMap(i, users)

		err := ctx.ThereIsAUserWith(getParameterString(user))
		if err != nil {
			return err
		}

		err = ctx.ThereIsAGroup(user["user"])
		if err != nil {
			return err
		}
	}

	return nil
}

// ThereIsAGroup creates a new group.
func (ctx *TestContext) ThereIsAGroup(group string) error {
	return ctx.ThereIsAGroupWith(getParameterString(map[string]string{
		"id":   group,
		"name": group,
	}))
}

// UserIsAManagerOfTheGroupWith sets the current user as the manager of a group.
func (ctx *TestContext) UserIsAManagerOfTheGroupWith(parameters string) error {
	err := ctx.ThereIsAGroupWith(parameters)
	if err != nil {
		return err
	}

	// We create a parent group of which the user is the manager.
	group := ctx.getParameterMap(parameters)

	canWatchMembers := "0"
	watchedGroupName := group["user_id"] + " manages " + group["name"]

	if group["can_watch_members"] == "true" {
		canWatchMembers = "1"
		watchedGroupName += " with can_watch_members"
	}

	err = ctx.ThereIsAGroupWith(getParameterString(map[string]string{
		"id":   watchedGroupName,
		"name": watchedGroupName,
	}))
	if err != nil {
		return err
	}

	ctx.IsAMemberOfTheGroup(group["id"], watchedGroupName)

	ctx.addGroupManager(group["user_id"], watchedGroupName, canWatchMembers)

	return nil
}

// IAmAManagerOfTheGroupWithID sets the user as a manager of a group with an id.
func (ctx *TestContext) IAmAManagerOfTheGroupWithID(group string) error {
	return ctx.UserIsAManagerOfTheGroupWith(getParameterString(map[string]string{
		"id":                group,
		"user_id":           ctx.user,
		"can_watch_members": "false",
	}))
}

// IAmAManagerOfTheGroup sets the user as a manager of a group with an id.
func (ctx *TestContext) IAmAManagerOfTheGroup(group string) error {
	return ctx.UserIsAManagerOfTheGroupWith(getParameterString(map[string]string{
		"id":                group,
		"user_id":           ctx.user,
		"name":              group,
		"can_watch_members": "false",
	}))
}

// IAmAManagerOfTheGroupAndCanWatchItsMembers sets the user as a manager of a group with can_watch permission.
func (ctx *TestContext) IAmAManagerOfTheGroupAndCanWatchItsMembers(group string) error {
	return ctx.UserIsAManagerOfTheGroupWith(getParameterString(map[string]string{
		"id":                group,
		"user_id":           ctx.user,
		"name":              group,
		"can_watch_members": "true",
	}))
}

// UserIsAManagerOfTheGroupAndCanWatchItsMembers sets the user as a manager of a group with can_watch permission.
func (ctx *TestContext) UserIsAManagerOfTheGroupAndCanWatchItsMembers(user, group string) error {
	return ctx.UserIsAManagerOfTheGroupWith(getParameterString(map[string]string{
		"id":                group,
		"user_id":           user,
		"name":              group,
		"can_watch_members": "true",
	}))
}

// theGroupIsADescendantOfTheGroup sets a group as a descendant of another.
func (ctx *TestContext) theGroupIsADescendantOfTheGroup(descendant, parent string) error {
	// we add another group in between to increase the robustness of the tests.
	middle := parent + " -> X -> " + descendant

	groups := []string{descendant, middle, parent}
	for _, group := range groups {
		err := ctx.ThereIsAGroupWith(getParameterString(map[string]string{
			"id": group,
		}))
		if err != nil {
			return err
		}
	}

	ctx.IsAMemberOfTheGroup(middle, parent)
	ctx.IsAMemberOfTheGroup(descendant, middle)

	return nil
}

// ICanWatchGroupWithID adds the permission for the user to watch a group.
func (ctx *TestContext) ICanWatchGroupWithID(group string) error {
	return ctx.UserIsAManagerOfTheGroupWith(getParameterString(map[string]string{
		"id":                group,
		"user_id":           ctx.user,
		"can_watch_members": "true",
	}))
}

// ThereAreTheFollowingTasks defines item tasks.
func (ctx *TestContext) ThereAreTheFollowingTasks(tasks *messages.PickleStepArgument_PickleTable) error {
	for i := 1; i < len(tasks.Rows); i++ {
		task := ctx.getRowMap(i, tasks)

		ctx.addItem(task["item"])
	}

	return nil
}

// ThereAreTheFollowingItemPermissions defines item permissions.
func (ctx *TestContext) ThereAreTheFollowingItemPermissions(itemPermissions *messages.PickleStepArgument_PickleTable) error {
	for i := 1; i < len(itemPermissions.Rows); i++ {
		itemPermission := ctx.getRowMap(i, itemPermissions)

		if itemPermission["can_view"] != "" {
			err := ctx.UserCanViewOnItemWithID(itemPermission["can_view"], itemPermission["group"], itemPermission["item"])
			if err != nil {
				return err
			}
		}

		if itemPermission["can_watch"] != "" {
			err := ctx.UserCanWatchOnItemWithID(itemPermission["can_watch"], itemPermission["group"], itemPermission["item"])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ICanWatchGroup adds the permission for the user to watch a group.
func (ctx *TestContext) ICanWatchGroup(groupName string) error {
	return ctx.UserIsAManagerOfTheGroupWith(getParameterString(map[string]string{
		"id":                groupName,
		"user_id":           ctx.user,
		"name":              groupName,
		"can_watch_members": "true",
	}))
}

// IsAMemberOfTheGroup Puts a group in a group.
func (ctx *TestContext) IsAMemberOfTheGroup(childGroupName, parentGroupName string) {
	ctx.addGroupGroup(parentGroupName, childGroupName)
}

// IAmAMemberOfTheGroupWithID creates a group and add the user in it.
func (ctx *TestContext) IAmAMemberOfTheGroupWithID(group string) error {
	err := ctx.ThereIsAGroupWith("id=" + group)
	if err != nil {
		return err
	}

	ctx.IsAMemberOfTheGroup(
		ctx.user,
		group,
	)

	return nil
}

// GroupIsAChildOfTheGroup puts a group as a child of another group.
func (ctx *TestContext) GroupIsAChildOfTheGroup(childGroup, parentGroup string) error {
	err := ctx.ThereIsAGroupWith(getParameterString(map[string]string{
		"id":   childGroup,
		"name": childGroup,
	}))
	if err != nil {
		return err
	}

	err = ctx.ThereIsAGroupWith(getParameterString(map[string]string{
		"id":   parentGroup,
		"name": parentGroup,
	}))
	if err != nil {
		return err
	}

	ctx.IsAMemberOfTheGroup(childGroup, parentGroup)

	return nil
}

// UserIsAMemberOfTheGroup puts a user in a group.
func (ctx *TestContext) UserIsAMemberOfTheGroup(user, group string) error {
	err := ctx.ThereIsAUserWith(getParameterString(map[string]string{
		"group_id": user,
		"user":     user,
	}))
	if err != nil {
		return err
	}

	return ctx.GroupIsAChildOfTheGroup(user, group)
}

// ThereAreTheFollowingGroupMembers defines group memberships.
func (ctx *TestContext) ThereAreTheFollowingGroupMembers(groupMembers *messages.PickleStepArgument_PickleTable) error {
	for i := 1; i < len(groupMembers.Rows); i++ {
		groupMember := ctx.getRowMap(i, groupMembers)

		err := ctx.GroupIsAChildOfTheGroup(groupMember["member"], groupMember["group"])
		if err != nil {
			return err
		}
	}

	return nil
}

// UserIsAMemberOfTheGroupWhoHasApprovedAccessToHisPersonalInfo puts a user in a group with approved access to his personnel info.
func (ctx *TestContext) UserIsAMemberOfTheGroupWhoHasApprovedAccessToHisPersonalInfo(user, group string) error {
	err := ctx.UserIsAMemberOfTheGroup(user, group)
	if err != nil {
		return err
	}

	ctx.addPersonalInfoViewApprovedFor(user, group)

	return nil
}

// IAmAMemberOfTheGroup puts a user in a group.
func (ctx *TestContext) IAmAMemberOfTheGroup(name string) error {
	return ctx.IAmAMemberOfTheGroupWithID(name)
}

// UserCanOnItemWithID gives a user a permission on an item.
func (ctx *TestContext) UserCanOnItemWithID(watchType, watchValue, user, item string) error {
	ctx.addPermissionGenerated(user, item, watchType, watchValue)

	return nil
}

// ICanOnItemWithID gives the user a permission on an item.
func (ctx *TestContext) ICanOnItemWithID(watchType, watchValue, item string) error {
	return ctx.UserCanOnItemWithID(watchType, watchValue, ctx.user, item)
}

func (ctx *TestContext) UserCanViewOnItemWithID(watchValue, user, item string) error {
	return ctx.UserCanOnItemWithID("view", watchValue, user, item)
}

// ICanViewOnItemWithID gives the user a "view" permission on an item.
func (ctx *TestContext) ICanViewOnItemWithID(watchValue, item string) error {
	return ctx.UserCanOnItemWithID("view", watchValue, ctx.user, item)
}

// UserCanWatchOnItemWithID gives a user a "watch" permission on an item.
func (ctx *TestContext) UserCanWatchOnItemWithID(watchValue, user, item string) error {
	return ctx.UserCanOnItemWithID("watch", watchValue, user, item)
}

// ICanWatchOnItemWithID gives the user a "watch" permission on an item.
func (ctx *TestContext) ICanWatchOnItemWithID(watchValue, item string) error {
	return ctx.UserCanOnItemWithID("watch", watchValue, ctx.user, item)
}

func (ctx *TestContext) UserHaveValidatedItemWithID(user, item string) error {
	attemptID := rand.Int63()

	ctx.addAttempt(item, user)
	ctx.addResult(
		strconv.FormatInt(attemptID, 10),
		user,
		item,
		time.Now(),
	)

	return nil
}

func (ctx *TestContext) ThereAreTheFollowingResults(results *messages.PickleStepArgument_PickleTable) error {
	for i := 1; i < len(results.Rows); i++ {
		result := ctx.getRowMap(i, results)

		ctx.addItem(result["item"])

		err := ctx.UserHaveValidatedItemWithID(result["participant"], result["item"])
		if err != nil {
			return err
		}
	}

	return nil
}

// IHaveValidatedItemWithID states that user has validated an item.
func (ctx *TestContext) IHaveValidatedItemWithID(item string) error {
	return ctx.UserHaveValidatedItemWithID(ctx.user, item)
}

// ThereAreTheFollowingThreads create threads.
func (ctx *TestContext) ThereAreTheFollowingThreads(threads *messages.PickleStepArgument_PickleTable) error {
	if len(threads.Rows) > 1 {
		for i := 1; i < len(threads.Rows); i++ {
			thread := ctx.getRowMap(i, threads)
			threadParameters := make(map[string]string)

			threadParameters["participant_id"] = thread["participant"]

			if thread["item"] != "" {
				threadParameters["item_id"] = thread["item"]
			}

			if thread["helper_group"] != "" {
				threadParameters["helper_group_id"] = thread["helper_group"]
			}

			if thread["status"] != "" {
				threadParameters["status"] = thread["status"]
			}

			if thread["latest_update_at"] != "" {
				threadParameters["latest_update_at"] = thread["latest_update_at"]
			}

			if thread["message_count"] != "" {
				threadParameters["message_count"] = thread["message_count"]
			}

			err := ctx.ThereIsAThreadWith(getParameterString(threadParameters))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ThereIsAThreadWith creates a thread.
func (ctx *TestContext) ThereIsAThreadWith(parameters string) error {
	thread := ctx.getParameterMap(parameters)

	// add item
	if _, ok := thread["item_id"]; !ok {
		thread["item_id"] = strconv.FormatInt(rand.Int63(), 10)
	}
	ctx.addItem(thread["item_id"])

	// add helper_group_id
	if _, ok := thread["helper_group_id"]; !ok {
		helperGroupID := rand.Int63()

		err := ctx.ThereIsAGroupWith(getParameterString(map[string]string{
			"id":   strconv.FormatInt(helperGroupID, 10),
			"name": "helper_group_for_" + thread["item_id"] + "-" + thread["participant_id"],
		}))
		if err != nil {
			return err
		}

		thread["helper_group_id"] = strconv.FormatInt(helperGroupID, 10)
	}

	// add status
	if _, ok := thread["status"]; !ok {
		thread["status"] = "waiting_for_trainer"
	}

	// add message count
	if _, ok := thread["message_count"]; !ok {
		thread["message_count"] = "0"
	}

	// add latest update at
	if _, ok := thread["latest_update_at"]; !ok {
		thread["latest_update_at"] = time.Now().Format(utils.DateTimeFormat)
	}

	ctx.currentThreadKey = ctx.getThreadKey(
		ctx.getReference(thread["item_id"]),
		ctx.getReference(thread["participant_id"]),
	)

	ctx.addThread(
		thread["item_id"],
		thread["participant_id"],
		thread["helper_group_id"],
		thread["status"],
		thread["message_count"],
		thread["latest_update_at"],
	)

	return nil
}

// ThereIsNoThreadWith states that a given thread doesn't exist.
func (ctx *TestContext) ThereIsNoThreadWith(parameters string) error {
	thread := ctx.getParameterMap(parameters)

	ctx.addItem(thread["item_id"])

	return nil
}

// IAmPartOfTheHelperGroupOfTheThread states that user is a member of the helper group, of a given thread.
func (ctx *TestContext) IAmPartOfTheHelperGroupOfTheThread() error {
	threadHelperGroupID := ctx.dbTables["threads"][ctx.currentThreadKey]["helper_group_id"].(int64)

	ctx.IsAMemberOfTheGroup(
		ctx.user,
		strconv.FormatInt(threadHelperGroupID, 10),
	)

	return nil
}

// ICanRequestHelpToTheGroupWithIDOnTheItemWithID gives the user the permission to request help from a given group
// to a given item.
func (ctx *TestContext) ICanRequestHelpToTheGroupWithIDOnTheItemWithID(group, item string) error {
	ctx.addPermissionGranted(
		ctx.user,
		ctx.user,
		item,
		group,
	)

	return nil
}
