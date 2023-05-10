package database

import (
	"errors"

	"gorm.io/gorm"
)

// GroupJoiningByCodeInfo represents info related to ability to join a team by code.
type GroupJoiningByCodeInfo struct {
	GroupID             int64
	Type                string
	CodeExpiresAtIsNull bool
	CodeLifetimeIsNull  bool
	FrozenMembership    bool
}

// GetGroupJoiningByCodeInfoByCode returns GroupJoiningByCodeInfo for a given code
// (or null if there is no public team with this code or the code has expired).
func (s *DataStore) GetGroupJoiningByCodeInfoByCode(code string, withLock bool) (*GroupJoiningByCodeInfo, error) {
	var info GroupJoiningByCodeInfo
	query := s.Groups().
		Where("type <> 'User'").
		Where("code = ?", code).
		Where("code_expires_at IS NULL OR NOW() < code_expires_at").
		Select(`
			id AS group_id, type, code_expires_at IS NULL AS code_expires_at_is_null,
			code_lifetime IS NULL AS code_lifetime_is_null, frozen_membership`)
	if withLock {
		query = query.WithWriteLock()
	}
	err := query.Take(&info).Error()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &info, err
}
