package service

import (
	"net/http"

	"github.com/France-ioi/AlgoreaBackend/app/database"
)

// QueryLimiter applies the limit parameter from an HTTP request to a given DB query (see QueryLimiter.Apply()).
type QueryLimiter struct {
	defaultLimit    int64
	maxAllowedLimit int64
}

// NewQueryLimiter creates a QueryLimiter with default settings:
//   - default limit is 500,
//   - maximum allowed limit is 1000.
func NewQueryLimiter() *QueryLimiter {
	return &QueryLimiter{
		defaultLimit:    500,
		maxAllowedLimit: 1000,
	}
}

// SetDefaultLimit sets the default limit used when the 'limit' request parameter is missing.
func (ql *QueryLimiter) SetDefaultLimit(defaultLimit int64) *QueryLimiter {
	ql.defaultLimit = defaultLimit
	return ql
}

// SetMaxAllowedLimit sets the maximum allowed limit.
func (ql *QueryLimiter) SetMaxAllowedLimit(maxAllowedLimit int64) *QueryLimiter {
	ql.maxAllowedLimit = maxAllowedLimit
	return ql
}

// Apply limits the number of records of the given DB query according to the `limit` request parameter.
func (ql *QueryLimiter) Apply(r *http.Request, db *database.DB) *database.DB {
	limit, err := ResolveURLQueryGetInt64Field(r, "limit")
	if err != nil || limit < 0 {
		limit = ql.defaultLimit
	}
	if limit > ql.maxAllowedLimit {
		limit = ql.maxAllowedLimit
	}
	return db.Limit(int(limit))
}
