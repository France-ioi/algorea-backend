package currentuser

import (
	"net/http"

	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation POST /current-user/group-invitations/{group_id}/reject groups users groupInvitationReject
// ---
// summary: Reject a group invitation
// description:
//   Let a user reject an invitation to join a group.
//   On success the service sets `groups_groups.sType` to `invitationRefused` and `sStatusDate` to current UTC time.
//
//   * There should be a row in `groups_groups` with the `group_id` as a parent
//   and the authenticated user’s selfGroup’s `ID` as a child with `sType`=`invitationSent`/`invitationRefused`.
//   Otherwise the unprocessable entity error is returned.
//
//   * If `groups_groups.sType` is `invitationRefused` already, the "not changed" (200) response is returned.
// parameters:
// - name: group_id
//   in: path
//   type: integer
//   required: true
// responses:
//   "200":
//     "$ref": "#/responses/updatedOrNotChangedResponse"
//   "400":
//     "$ref": "#/responses/badRequestResponse"
//   "401":
//     "$ref": "#/responses/unauthorizedResponse"
//   "403":
//     "$ref": "#/responses/forbiddenResponse"
//   "422":
//     "$ref": "#/responses/unprocessableEntityResponse"
//   "500":
//     "$ref": "#/responses/internalErrorResponse"
func (srv *Service) rejectGroupInvitation(w http.ResponseWriter, r *http.Request) service.APIError {
	return srv.performGroupRelationAction(w, r, rejectInvitationAction)
}
