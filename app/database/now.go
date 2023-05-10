package database

import "gorm.io/gorm"

var nowExpr = gorm.Expr("NOW()")

// Now returns a DB expression that returns current DB time (it is usually gorm.Expr("NOW()")).
func Now() interface{} {
	return nowExpr
}

// MockNow changes the DB expression for getting current DB time so that it will return the given timestamp.
func MockNow(timestamp string) {
	nowExpr = gorm.Expr("?", timestamp)
}

// RestoreNow sets the DB expression for getting current DB time to its default value (gorm.Expr("NOW()")).
func RestoreNow() {
	nowExpr = gorm.Expr("NOW()")
}
