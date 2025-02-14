//go:build !prod

package testhelpers

import (
	"context"
	"testing"

	"github.com/cucumber/godog"
	"github.com/zenovich/flowmingo"
)

var ctx = &TestContext{}

// InitializeScenario binds the supported steps to the verifying functions.
func InitializeScenario(s *godog.ScenarioContext) {
	var restoreFunc flowmingo.RestoreFunc
	s.Before(func(contextCtx context.Context, sc *godog.Scenario) (context.Context, error) {
		if !testing.Verbose() { // Do not suppress output in verbose mode
			restoreFunc = flowmingo.CaptureStdoutAndStderr() // Suppress the output of the scenario
		}
		ctx.SetupTestContext(sc)
		return contextCtx, nil
	})

	s.Step(`^the template constant "([^"]+)" is "(.*)"$`, ctx.TheTemplateConstantIsString)
	s.Step(`^the template constant "([^"]+)" is:$`, ctx.TheTemplateConstantIsDocString)

	s.Step(`^the database has the following table "([^"]+)":$`, ctx.DBHasTable)
	s.Step(`^the database table "([^"]+)"(?: also)? has the following rows?:$`, ctx.DBHasTable)
	s.Step(`^the database(?: also)? has the following users?:$`, ctx.DBHasUsers)
	s.Step(`^the groups ancestors are computed$`, ctx.DBGroupsAncestorsAreComputed)

	ctx.registerFeaturesForSessions(s)
	ctx.registerFeaturesForUsers(s)

	s.Step(`^the time now is "([^"]*)"$`, ctx.TimeNow)
	s.Step(`^the server time now is "([^"]*)"$`, ctx.ServerTimeNow)
	s.Step(`^the time is frozen$`, ctx.TimeIsFrozen)
	s.Step(`^the server time is frozen$`, ctx.ServerTimeIsFrozen)
	s.Step(`^the generated group code is "([^"]*)"$`, ctx.TheGeneratedGroupCodeIs)
	s.Step(`^the generated group codes are ("[^"]*"(?:\s*,\s*"[^"]*")*)$`, ctx.TheGeneratedGroupCodesAre)
	s.Step(`^the generated auth key is "([^"]*)"$`, ctx.TheGeneratedAuthKeyIs)
	s.Step(`^the application config is:$`, ctx.TheApplicationConfigIs)
	s.Step(`^the context variable "([^"]*)" is "([^"]*)"$`, ctx.TheContextVariableIs)

	ctx.registerFeaturesForGroups(s)

	s.Step(`^I am a manager of the group with id "([^"]*)"$`, ctx.IAmAManagerOfTheGroupWithID)
	s.Step(`^I am a manager of the group (@\w+)$`, ctx.IAmAManagerOfTheGroup)
	s.Step(`^I am a manager of the group (.+) and can watch for submissions from the group and its descendants$`,
		ctx.IAmAManagerOfTheGroupAndCanWatchItsMembers)
	s.Step(`^the group (@\w+) is a manager of the group (@\w+) and can watch for submissions from the group and its descendants$`,
		ctx.GroupIsAManagerOfTheGroupAndCanWatchItsMembers)
	s.Step(`^the group (@\w+) is a manager of the group (@\w+) and can grant group access`,
		ctx.GroupIsAManagerOfTheGroupAndCanGrantGroupAccess)
	s.Step(
		`^the group (@\w+) is a manager of the group (@\w+) and can manage memberships and the group`,
		ctx.GroupIsAManagerOfTheGroupAndCanManageMembershipsAndGroup,
	)

	s.Step(`^there are the following items:$`, ctx.ThereAreTheFollowingItems)
	s.Step(`^there are the following tasks:$`, ctx.ThereAreTheFollowingTasks)
	s.Step(`^there are the following item permissions:$`, ctx.ThereAreTheFollowingItemPermissions)
	s.Step(`^there are the following item relations:$`, ctx.ThereAreTheFollowingItemRelations)
	s.Step(`^I can view (none|info|content|content_with_descendants|solution) of the item (.+)$`,
		ctx.IHaveViewPermissionOnItem)
	s.Step(`^the group (\@\w+) can view (none|info|content|content_with_descendants|solution) of the item (.+)$`,
		ctx.GroupHasViewPermissionOnItem)
	s.Step(`^I have the watch permission set to "(none|result|answer|answer_with_grant)" on the item (.+)$`, ctx.IHaveWatchPermissionOnItem)
	s.Step(`^the group (.+) has the watch permission set to "(none|result|answer|answer_with_grant)" on the item (.+)$`,
		ctx.GroupHasWatchPermissionOnItem)
	s.Step(`^I can request help to the group with id "([^"]*)" on the item with id "([^"]*)"$`,
		ctx.ICanRequestHelpToTheGroupWithIDOnTheItemWithID)

	ctx.registerFeaturesForGroupMembershipChanges(s)
	ctx.registerFeaturesForGroupPendingRequests(s)

	s.Step(`^there are the following validated results:$`, ctx.ThereAreTheFollowingValidatedResults)
	s.Step(`^I have a validated result on the item (.+)$`, ctx.IHaveValidatedResultOnItem)
	s.Step(`^the group (.+) has a validated result on the item (.+)$`, ctx.GroupHasValidatedResultOnItem)

	s.Step(`^there are the following threads:$`, ctx.ThereAreTheFollowingThreads)
	s.Step(`^there is a thread with "(.*)"$`, ctx.ThereIsAThreadWith)
	s.Step(`^there is no thread with "([^"]*)"$`, ctx.ThereIsNoThreadWith)
	s.Step(`^I am part of the helper group of the thread$`, ctx.IAmPartOfTheHelperGroupOfTheThread)

	s.Step(`^the "([^"]*)" request header is "(.*)"$`, ctx.TheRequestHeaderIs)
	s.Step(`^I send a (GET|POST|PUT|DELETE) request to "([^"]*)"$`, ctx.ISendrequestTo)
	s.Step(`^I send a (GET|POST|PUT|DELETE) request to "([^"]*)" with the following body:$`, ctx.ISendrequestToWithBody)
	s.Step(`^the response code should be (\d+)$`, ctx.TheResponseCodeShouldBe)
	s.Step(`^the response body should be, in JSON:$`, ctx.TheResponseBodyShouldBeJSON)
	s.Step(`^the response body should be:$`, ctx.TheResponseBodyShouldBe)
	s.Step(`^the response body decoded as "([^"]+)" should be, in JSON:$`, ctx.TheResponseDecodedBodyShouldBeJSON)
	s.Step(`^the response header "([^"]*)" should be "([^"]*)"$`, ctx.TheResponseHeaderShouldBe)
	s.Step(`^the response header "([^"]*)" should not be set$`, ctx.TheResponseHeaderShouldNotBeSet)
	s.Step(`^the response headers? "([^"]*)" should be:`, ctx.TheResponseHeadersShouldBe)
	s.Step(`^the response should be "([^"]*)"$`, ctx.TheResponseShouldBe)
	s.Step(`^the response error message should contain "(.*)"$`, ctx.TheResponseErrorMessageShouldContain)

	s.Step(`^the response should be a JSON array with (\d+) entr(?:ies|y)$`, ctx.ItShouldBeAJSONArrayWithEntries)
	s.Step(`^the response at ([^ ]+) should be "([^"]*)"$`, ctx.TheResponseAtShouldBeTheValue)
	s.Step("^the response at ([^ ]+) should be:$", ctx.TheResponseAtShouldBe)
	s.Step("^the response at ([^ ]+) in JSON should be:$", ctx.TheResponseAtInJSONShouldBe)
	s.Step("^the response at ([^ ]+) should be the base64 of an AES-256-GCM encrypted JSON object containing:",
		ctx.TheResponseAtShouldBeTheBase64OfAnAES256GCMEncryptedJSONObjectContaining)

	s.Step(`^the table "([^"]*)" should be:$`, ctx.TableShouldBe)
	s.Step(`^the table "([^"]*)" should be empty$`, ctx.TableShouldBeEmpty)
	s.Step(`^the table "([^"]*)" at (\w+?)s? "([^"]*)" should be:$`, ctx.TableAtColumnValueShouldBe)
	s.Step(`^the table "([^"]*)" at (\w+?)s? "([^"]*)" should be empty$`, ctx.TableAtColumnValueShouldBeEmpty)
	s.Step(`^the table "([^"]*)" should not contain (\w+?)s? "([^"]*)"$`, ctx.TableShouldNotContainColumnValue)
	s.Step(`^the table "([^"]*)" should remain unchanged$`, ctx.TableShouldRemainUnchanged)
	s.Step(`^the table "([^"]*)" should remain unchanged, regardless of the rows? with (\w+?)s? "([^"]*)"$`,
		ctx.TableShouldStayUnchangedRegardlessOfTheRowsWithColumnValue)
	s.Step(`^the table "([^"]*)" should remain unchanged, except that the rows? with ([^"]*) "([^"]*)" should be deleted$`,
		ctx.TableShouldRemainUnchangedExceptThatTheRowsWithColumnValueShouldBeDeleted)
	s.Step(`^the DB time now is "([^"]*)"$`, ctx.DBTimeNow)

	s.Step(`^"([^"]+)" is a token signed by (.+) with the following payload:$`, ctx.SignedTokenIsDistributed)
	s.Step(`^"([^"]+)" is a falsified token signed by (.+) with the following payload:$`, ctx.FalsifiedSignedTokenIsDistributed)
	s.Step(`^logs should contain:$`, ctx.LogsShouldContain)

	s.Step(`^the login module "token" endpoint for code "([^"]*)" returns (\d+) with body:$`, ctx.TheLoginModuleTokenEndpointForCodeReturns)
	s.Step(`^the login module "token" endpoint for code "([^"]*)" and code_verifier "([^"]*)" returns (\d+) with body:$`,
		ctx.TheLoginModuleTokenEndpointForCodeAndCodeVerifierReturns)
	s.Step(
		`^the login module "token" endpoint for code "([^"]*)" and code_verifier "([^"]*)" and redirect_uri "([^"]*)" returns (\d+) with body:$`,
		ctx.TheLoginModuleTokenEndpointForCodeAndCodeVerifierAndRedirectURIReturns)
	s.Step(`^the login module "token" endpoint for refresh token "([^"]*)" returns (\d+) with body:$`,
		ctx.TheLoginModuleTokenEndpointForRefreshTokenReturns)
	s.Step(`^the login module "account" endpoint for token "([^"]*)" returns (\d+) with body:$`,
		ctx.TheLoginModuleAccountEndpointForTokenReturns)
	s.Step(`^the login module "unlink_client" endpoint for user id "([^"]*)" returns (\d+) with encoded body:$`,
		ctx.TheLoginModuleUnlinkClientEndpointForUserIDReturns)
	s.Step(`^the login module "create" endpoint with params "([^"]*)" returns (\d+) with encoded body:$`,
		ctx.TheLoginModuleCreateEndpointWithParamsReturns)
	s.Step(`^the login module "delete" endpoint with params "([^"]*)" returns (\d+) with encoded body:$`,
		ctx.TheLoginModuleDeleteEndpointWithParamsReturns)
	s.Step(
		`^the login module "lti_result/send" endpoint for user id "([^"]*)", `+
			`content id "([^"]*)", score "([^"]*)" returns (\d+) with encoded body:$`,
		ctx.TheLoginModuleLTIResultSendEndpointForUserIDContentIDScoreReturns)

	s.After(func(contextCtx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		tearDownErr := ctx.ScenarioTeardown(sc, err)
		if err == nil {
			err = tearDownErr
		}
		if restoreFunc != nil { // If we captured the output, restore it
			restoreFunc(err != nil) // Pass through the output if the test failed
		}
		if err != nil { // If the test failed, restore the output of the parent test (if it is captured)
			if parentOutputRestorerFunc := contextCtx.Value(outputRestorerFuncKey).(*flowmingo.RestoreFunc); *parentOutputRestorerFunc != nil {
				(*parentOutputRestorerFunc)(true)
				*parentOutputRestorerFunc = nil
			}
		}
		return contextCtx, tearDownErr
	})
}
