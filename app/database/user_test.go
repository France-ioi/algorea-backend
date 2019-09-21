package database

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
)

func TestUser_Clone(t *testing.T) {
	ts := time.Now()
	user := &User{
		ID: 1, Login: "login", DefaultLanguage: "fr", DefaultLanguageID: 12,
		IsTempUser: true, IsAdmin: true, SelfGroupID: ptrInt64(2), OwnedGroupID: ptrInt64(3), AccessGroupID: ptrInt64(4),
		AllowSubgroups: true, NotificationReadDate: (*Time)(&ts)}
	userClone := user.Clone()
	assert.False(t, userClone == user)
	assert.False(t, user.NotificationReadDate == userClone.NotificationReadDate)
	assert.Equal(t, *user.NotificationReadDate, *userClone.NotificationReadDate)
	userClone.NotificationReadDate = nil
	user.NotificationReadDate = nil
	assert.Equal(t, *user, *userClone)
	userClone = user.Clone() // clone again with NotificationReadDate = nil
	assert.Nil(t, userClone.NotificationReadDate)
	assert.Equal(t, *user, *userClone)
}

func (u *User) LoadByID(dataStore *DataStore, id int64) error {
	err := dataStore.Users().ByID(id).
		Select(`
						users.id, users.login, users.is_admin, users.self_group_id, users.owned_group_id, users.access_group_id,
						users.temp_user, users.allow_subgroups, users.notification_read_date,
						users.default_language, l.id as default_language_id`).
		Joins("LEFT JOIN languages l ON users.default_language = l.code").
		Take(&u).Error()
	if gorm.IsRecordNotFoundError(err) {
		u.ID = id
		return nil
	}
	return err
}
