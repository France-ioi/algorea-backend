package database

import (
	"gorm.io/gorm"
)

// SessionStore implements database operations on `sessions`.
type SessionStore struct {
	*DataStore
}

// InsertNewOAuth inserts a new OAuth token for the given user into the DB.
func (s *SessionStore) InsertNewOAuth(userID int64, token string, secondsUntilExpiry int32, issuer string) error {
	return s.InsertMap(map[string]interface{}{
		"access_token": token,
		"expires_at":   gorm.Expr("?  + INTERVAL ? SECOND", Now(), secondsUntilExpiry),
		"user_id":      userID,
		"issuer":       issuer,
		"issued_at":    Now(),
	})
}
