package groups

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

func TestService_checkThatUserCanManageTheGroup_HandlesError(t *testing.T) {
	db, mock := database.NewDBMock()
	defer func() { _ = db.Close() }()

	expectedError := errors.New("some error")
	mock.ExpectQuery("^"+regexp.QuoteMeta("SELECT count(*) FROM `groups_ancestors`")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(expectedError)

	user := &database.User{GroupID: 1}
	apiErr := checkThatUserCanManageTheGroup(database.NewDataStore(db), user, 123)

	assert.Equal(t, service.ErrUnexpected(expectedError), apiErr)
	assert.NoError(t, mock.ExpectationsWereMet())
}
