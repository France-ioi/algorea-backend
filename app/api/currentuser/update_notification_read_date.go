package currentuser

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:operation PUT /current-user/notification-read-date users userNotificationReadDateUpdate
// ---
// summary: Update user's notification read date
// description: Set users.notification_read_date to NOW() for the current user
// responses:
//   "200":
//     "$ref": "#/responses/updatedResponse"
//   "401":
//     "$ref": "#/responses/unauthorizedResponse"
//   "500":
//     "$ref": "#/responses/internalErrorResponse"
func (srv *Service) updateNotificationReadDate(w http.ResponseWriter, r *http.Request) service.APIError {
	user := srv.GetUser(r)
	// the user middleware has already checked that the user exists so we just ignore the case where nothing is updated
	service.MustNotBeError(srv.Store.Users().ByID(user.ID).
		UpdateColumn("notification_read_date", database.Now()).Error())

	response := service.Response{Success: true, Message: "updated"}
	render.Respond(w, r, &response)

	return service.NoError
}
