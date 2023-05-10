package database

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestUser_Clone(t *testing.T) {
	ts := time.Now()
	user := &User{
		Login: "login", LoginID: ptrInt64(5), DefaultLanguage: "fr",
		IsTempUser: true, IsAdmin: true, GroupID: 2, AccessGroupID: ptrInt64(4), NotificationsReadAt: (*Time)(&ts),
	}
	userClone := user.Clone()
	assert.False(t, userClone == user)
	assert.False(t, user.NotificationsReadAt == userClone.NotificationsReadAt)
	assert.Equal(t, *user.NotificationsReadAt, *userClone.NotificationsReadAt)
	assert.False(t, user.LoginID == userClone.LoginID)
	assert.Equal(t, *user.LoginID, *userClone.LoginID)
	assert.False(t, user.AccessGroupID == userClone.AccessGroupID)
	assert.Equal(t, *user.AccessGroupID, *userClone.AccessGroupID)
	userClone.NotificationsReadAt = nil
	userClone.LoginID = nil
	userClone.AccessGroupID = nil
	user.NotificationsReadAt = nil
	user.LoginID = nil
	user.AccessGroupID = nil
	assert.Equal(t, *user, *userClone)
	userClone = user.Clone() // clone again when pointer-type fields are nils
	assert.Nil(t, userClone.NotificationsReadAt)
	assert.Nil(t, userClone.LoginID)
	assert.Nil(t, userClone.AccessGroupID)
	assert.Equal(t, *user, *userClone)
}

func (u *User) LoadByID(dataStore *DataStore, userID int64) error {
	err := dataStore.Users().ByID(userID).
		Select(`
						users.login, users.login_id, users.is_admin, users.group_id, users.access_group_id,
						users.temp_user, users.notifications_read_at, users.default_language`).
		Take(&u).Error()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		u.GroupID = userID
		return nil
	}
	return err
}
