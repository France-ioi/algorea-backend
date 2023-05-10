//go:build !prod

package testhelpers

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"regexp"

	"bou.ke/monkey"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app/database"
)

var (
	nowRegexp      = regexp.MustCompile(`(?i)\bNOW\s*\(\s*(?:\d+\s*)?\)`)
	patchedMethods []*monkey.PatchGuard
)

// MockDBTime replaces the DB NOW() function call with a given constant value in all the queries.
func MockDBTime(timeStrRaw string) {
	timeStr := fmt.Sprintf("%q", timeStrRaw)

	patchDatabaseDBMethods(timeStr)
	database.MockNow(timeStrRaw)
	patchGormMethods(timeStr)
	patchDBMethods(timeStr)
}

func patchDBMethods(timeStr string) {
	var prepareContextGuard, queryContextGuard *monkey.PatchGuard
	prepareContextGuard = monkey.PatchInstanceMethod(
		reflect.TypeOf(&sql.DB{}), "PrepareContext",
		func(db *sql.DB, c context.Context, query string) (*sql.Stmt, error) {
			prepareContextGuard.Unpatch()
			defer prepareContextGuard.Restore()
			query = nowRegexp.ReplaceAllString(query, timeStr)
			return db.PrepareContext(c, query)
		})
	patchedMethods = append(patchedMethods, prepareContextGuard)
	queryContextGuard = monkey.PatchInstanceMethod(
		reflect.TypeOf(&sql.DB{}), "QueryContext",
		func(db *sql.DB, c context.Context, query string, args ...interface{}) (*sql.Rows, error) {
			queryContextGuard.Unpatch()
			defer queryContextGuard.Restore()
			query = nowRegexp.ReplaceAllString(query, timeStr)
			return db.QueryContext(c, query, args...)
		})
	patchedMethods = append(patchedMethods, queryContextGuard)
}

func patchGormMethods(timeStr string) {
	var execGuard, rawGuard *monkey.PatchGuard
	execGuard = monkey.PatchInstanceMethod(
		reflect.TypeOf(&gorm.DB{}), "Exec",
		func(db *gorm.DB, query string, args ...interface{}) *gorm.DB {
			execGuard.Unpatch()
			defer execGuard.Restore()
			query = nowRegexp.ReplaceAllString(query, timeStr)
			return db.Exec(query, args...)
		})
	patchedMethods = append(patchedMethods, execGuard)
	rawGuard = monkey.PatchInstanceMethod(
		reflect.TypeOf(&gorm.DB{}), "Raw",
		func(db *gorm.DB, query string, args ...interface{}) *gorm.DB {
			rawGuard.Unpatch()
			defer rawGuard.Restore()
			query = nowRegexp.ReplaceAllString(query, timeStr)
			return db.Raw(query, args...)
		})
	patchedMethods = append(patchedMethods, rawGuard)
}

func patchDatabaseDBMethods(timeStr string) {
	patchDatabaseDBMethodsWithIntQueryAndArgs(timeStr)
	patchDatabaseDBMethodsWithStringQueryAndArgs(timeStr)
	patchDatabaseDBMethodsWithStringQuery(timeStr)
	patchDatabaseDBMethodsWithIntQuery(timeStr)
	var orderGuard *monkey.PatchGuard
	orderGuard = monkey.PatchInstanceMethod(
		reflect.TypeOf(&database.DB{}), "Order",
		func(db *database.DB, value interface{}) *database.DB {
			orderGuard.Unpatch()
			defer orderGuard.Restore()
			if valueStr, ok := value.(string); ok {
				value = nowRegexp.ReplaceAllString(valueStr, timeStr)
			}
			return db.Order(value)
		})
	patchedMethods = append(patchedMethods, orderGuard)

	var pluckGuard *monkey.PatchGuard
	pluckGuard = monkey.PatchInstanceMethod(
		reflect.TypeOf(&database.DB{}), "Pluck",
		func(db *database.DB, column string, values interface{}) *database.DB {
			pluckGuard.Unpatch()
			defer pluckGuard.Restore()
			column = nowRegexp.ReplaceAllString(column, timeStr)
			return db.Pluck(column, values)
		})
	patchedMethods = append(patchedMethods, pluckGuard)

	var takeGuard *monkey.PatchGuard
	takeGuard = monkey.PatchInstanceMethod(
		reflect.TypeOf(&database.DB{}), "Take",
		func(db *database.DB, out interface{}, where ...interface{}) *database.DB {
			takeGuard.Unpatch()
			defer takeGuard.Restore()
			if len(where) > 0 {
				if whereStr, ok := where[0].(string); ok {
					where[0] = nowRegexp.ReplaceAllString(whereStr, timeStr)
				}
			}
			return db.Take(out, where...)
		})
	patchedMethods = append(patchedMethods, takeGuard)

	var deleteGuard *monkey.PatchGuard
	deleteGuard = monkey.PatchInstanceMethod(
		reflect.TypeOf(&database.DB{}), "Delete",
		func(db *database.DB, where ...interface{}) *database.DB {
			deleteGuard.Unpatch()
			defer deleteGuard.Restore()
			if len(where) > 0 {
				if whereStr, ok := where[0].(string); ok {
					where[0] = nowRegexp.ReplaceAllString(whereStr, timeStr)
				}
			}
			return db.Delete(where...)
		})
	patchedMethods = append(patchedMethods, deleteGuard)
}

func patchDatabaseDBMethodsWithIntQuery(timeStr string) {
	interfaceDBMethods := [...]string{
		"Union", "UnionAll",
	}
	interfaceDBGuards := make(map[string]*monkey.PatchGuard, len(interfaceDBMethods))
	for _, methodName := range interfaceDBMethods {
		methodName := methodName
		interfaceDBGuards[methodName] = monkey.PatchInstanceMethod(
			reflect.TypeOf(&database.DB{}), methodName,
			func(db *database.DB, query interface{}) *database.DB {
				interfaceDBGuards[methodName].Unpatch()
				defer interfaceDBGuards[methodName].Restore()
				if queryStr, ok := query.(string); ok {
					query = nowRegexp.ReplaceAllString(queryStr, timeStr)
				}
				reflMethod := reflect.ValueOf(db).MethodByName(methodName)
				reflArgs := []reflect.Value{reflect.ValueOf(query)}

				return reflMethod.Call(reflArgs)[0].Interface().(*database.DB)
			})
		patchedMethods = append(patchedMethods, interfaceDBGuards[methodName])
	}
}

func patchDatabaseDBMethodsWithStringQuery(timeStr string) {
	stringDBMethods := [...]string{
		"Group",
	}
	stringDBGuards := make(map[string]*monkey.PatchGuard, len(stringDBMethods))
	for _, methodName := range stringDBMethods {
		methodName := methodName
		stringDBGuards[methodName] = monkey.PatchInstanceMethod(
			reflect.TypeOf(&database.DB{}), methodName,
			func(db *database.DB, query string) *database.DB {
				stringDBGuards[methodName].Unpatch()
				defer stringDBGuards[methodName].Restore()
				query = nowRegexp.ReplaceAllString(query, timeStr)
				reflMethod := reflect.ValueOf(db).MethodByName(methodName)
				reflArgs := []reflect.Value{reflect.ValueOf(query)}

				return reflMethod.Call(reflArgs)[0].Interface().(*database.DB)
			})
		patchedMethods = append(patchedMethods, stringDBGuards[methodName])
	}
}

func patchDatabaseDBMethodsWithStringQueryAndArgs(timeStr string) {
	stringAndArgsDBMethods := [...]string{
		"Table", "Joins", "Raw", "Exec",
	}
	stringAndArgsDBGuards := make(map[string]*monkey.PatchGuard, len(stringAndArgsDBMethods))
	for _, methodName := range stringAndArgsDBMethods {
		methodName := methodName
		stringAndArgsDBGuards[methodName] = monkey.PatchInstanceMethod(
			reflect.TypeOf(&database.DB{}), methodName,
			func(db *database.DB, query string, args ...interface{}) *database.DB {
				stringAndArgsDBGuards[methodName].Unpatch()
				defer stringAndArgsDBGuards[methodName].Restore()
				query = nowRegexp.ReplaceAllString(query, timeStr)
				reflMethod := reflect.ValueOf(db).MethodByName(methodName)
				reflArgs := make([]reflect.Value, 0, len(args))
				reflArgs = append(reflArgs, reflect.ValueOf(query))
				for _, arg := range args {
					arg := arg
					reflArgs = append(reflArgs, reflect.ValueOf(arg))
				}

				return reflMethod.Call(reflArgs)[0].Interface().(*database.DB)
			})
		patchedMethods = append(patchedMethods, stringAndArgsDBGuards[methodName])
	}
}

func patchDatabaseDBMethodsWithIntQueryAndArgs(timeStr string) {
	standardDBMethods := [...]string{
		"Where", "Select", "Having",
	}
	standardDBGuards := make(map[string]*monkey.PatchGuard, len(standardDBMethods))
	for _, methodName := range standardDBMethods {
		methodName := methodName
		standardDBGuards[methodName] = monkey.PatchInstanceMethod(
			reflect.TypeOf(&database.DB{}), methodName,
			func(db *database.DB, query interface{}, args ...interface{}) *database.DB {
				standardDBGuards[methodName].Unpatch()
				defer standardDBGuards[methodName].Restore()
				if queryStr, ok := query.(string); ok {
					query = nowRegexp.ReplaceAllString(queryStr, timeStr)
				}
				reflMethod := reflect.ValueOf(db).MethodByName(methodName)
				reflArgs := make([]reflect.Value, 0, len(args))
				reflArgs = append(reflArgs, reflect.ValueOf(query))
				for _, arg := range args {
					arg := arg
					reflArgs = append(reflArgs, reflect.ValueOf(arg))
				}

				return reflMethod.Call(reflArgs)[0].Interface().(*database.DB)
			})
		patchedMethods = append(patchedMethods, standardDBGuards[methodName])
	}
}

// RestoreDBTime restores the usual behavior of the NOW() function.
func RestoreDBTime() {
	database.RestoreNow()
	for i := len(patchedMethods) - 1; i >= 0; i-- {
		patchedMethods[i].Unpatch()
	}
	patchedMethods = patchedMethods[:0]
}
