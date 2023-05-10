package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestNow(t *testing.T) {
	expectedExpr := gorm.Expr("123")
	nowExpr = expectedExpr
	defer func() { nowExpr = gorm.Expr("NOW()") }()
	assert.Equal(t, expectedExpr, Now())
}

func TestMockNow(t *testing.T) {
	expectedExpr := gorm.Expr("?", "2017-01-02T12:34:56Z")
	MockNow("2017-01-02T12:34:56Z")
	defer func() { nowExpr = gorm.Expr("NOW()") }()
	assert.Equal(t, expectedExpr, Now())
}

func TestRestoreNow(t *testing.T) {
	expectedExpr := gorm.Expr("NOW()")
	nowExpr = gorm.Expr("123")
	defer func() { nowExpr = expectedExpr }()
	RestoreNow()
	assert.Equal(t, expectedExpr, nowExpr)
}
