package currentuser

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/France-ioi/AlgoreaBackend/app/formdata"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

// swagger:model
type userDataUpdateRequest struct {
	DefaultLanguage string `json:"default_language"`
}

// swagger:operation PUT /current-user users userDataUpdate
//
//	---
//	summary: Update user's data
//	description: Allows changing the user's default language
//	parameters:
//		- name: data
//			in: body
//			required: true
//			schema:
//				"$ref": "#/definitions/userDataUpdateRequest"
//	responses:
//		"200":
//			"$ref": "#/responses/updatedResponse"
//		"401":
//			"$ref": "#/responses/unauthorizedResponse"
//		"500":
//			"$ref": "#/responses/internalErrorResponse"
func (srv *Service) update(w http.ResponseWriter, r *http.Request) service.APIError {
	user := srv.GetUser(r)

	var requestData userDataUpdateRequest
	formData := formdata.NewFormData(&requestData)
	err := formData.ParseJSONRequestData(r)
	if err != nil {
		return service.ErrInvalidRequest(err)
	}

	// the user middleware has already checked that the user exists so we just ignore the case where nothing is updated
	service.MustNotBeError(srv.GetStore(r).Users().ByID(user.GroupID).UpdateColumns(requestData).Error())

	response := service.Response{Success: true, Message: "updated"}
	render.Respond(w, r, &response)

	return service.NoError
}
