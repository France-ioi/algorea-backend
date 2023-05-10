package currentuser

import (
	"errors"
	"io/ioutil"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
	"github.com/France-ioi/AlgoreaBackend/app/servicetest"
)

func TestService_getDump_ReturnsErrorRightInsideTheResponseBody(t *testing.T) {
	response, mock, _, err := servicetest.GetResponseForRouteWithMockedDBAndUser(
		"GET", "/current-user/full-dump", ``,
		&database.User{GroupID: 11},
		func(sqlmock sqlmock.Sqlmock) {
			sqlmock.ExpectQuery("^" + regexp.QuoteMeta(
				"SELECT CONCAT('`', TABLE_NAME, '`.`', COLUMN_NAME, '`') FROM `INFORMATION_SCHEMA`.`COLUMNS`  "+
					"WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?",
			) + "$").WillReturnRows(sqlmock.NewRows([]string{"names"}).AddRow("users.group_id").AddRow("users.name"))
			sqlmock.ExpectQuery("^" + regexp.QuoteMeta(
				"SELECT users.group_id, users.name FROM `users`  WHERE users.group_id = ?") + "$").
				WillReturnError(errors.New("some error"))
		},
		func(router *chi.Mux, baseService *service.Base) {
			srv := &Service{Base: baseService}
			router.Get("/current-user/full-dump", service.AppHandler(srv.getFullDump).ServeHTTP)
		})
	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "attachment; filename=user_data.json", response.Header.Get("Content-Disposition"))
	assert.Equal(t, "application/json; charset=utf-8", response.Header.Get("Content-Type"))
	body, _ := ioutil.ReadAll(response.Body)
	_ = response.Body.Close()
	assert.Equal(t, `{"current_user":{"success":false,"message":"Internal Server Error","error_text":"Some error"}`+"\n",
		string(body)) // Note that the response is a malformed JSON in case of error
	assert.NoError(t, mock.ExpectationsWereMet())
}
