package domain

import (
	"context"
)

type ctxKey int

const (
	ctxDomainConfig ctxKey = iota
)

// CtxConfig contains domain-specific settings related to a request context
type CtxConfig struct {
	AllUsersGroupID  int64
	TempUsersGroupID int64
}

// ConfigItem is one item of the configuration list
type ConfigItem struct {
	Domains        []string
	AllUsersGroup  int64
	TempUsersGroup int64
}

// ConfigFromContext retrieves the current domain configuration from a context set by the middleware
func ConfigFromContext(ctx context.Context) *CtxConfig {
	conf := ctx.Value(ctxDomainConfig).(*CtxConfig)
	confCopy := *conf
	return &confCopy
}
