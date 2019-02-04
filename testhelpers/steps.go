package testhelpers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"time"

	"bou.ke/monkey"
	"github.com/DATA-DOG/godog/gherkin"
	_ "github.com/go-sql-driver/mysql" // use to force database/sql to use mysql
	"github.com/pmezard/go-difflib/difflib"
	"github.com/spf13/viper"

	"github.com/France-ioi/AlgoreaBackend/app"
	"github.com/France-ioi/AlgoreaBackend/app/service"
)

type dbquery struct {
	sql    string
	values []interface{}
}

// TestContext implements context for tests
type TestContext struct {
	// nolint
	application                      *app.Application // do NOT call it directly, use `app()`
	userID                           int64            // userID that will be used for the next requests
	featureQueries                   []dbquery
	lastResponse                     *http.Response
	lastResponseBody                 string
	inScenario                       bool
}

const (
	noID int64 = -1
)

func (ctx *TestContext) SetupTestContext(interface{}) { // nolint
	ctx.application = nil
	ctx.userID = 999 // the default for the moment
	ctx.lastResponse = nil
	ctx.lastResponseBody = ""
	ctx.inScenario = true
}

func (ctx *TestContext) ScenarioTeardown(interface{}, error) { // nolint
}

func (ctx *TestContext) app() *app.Application {

	if ctx.application == nil {
		var err error
		ctx.application, err = app.New()
		if err != nil {
			fmt.Println("Unable to load app")
			panic(err)
		}
		// reset the seed to get predictable results on PRNG for tests
		rand.Seed(1)

		err = ctx.initDB()
		if err != nil {
			fmt.Println("Unable to empty db")
			panic(err)
		}
	}
	return ctx.application
}

func testRequest(ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string, error) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		return nil, "", err
	}

	// set a dummy auth cookie
	req.AddCookie(&http.Cookie{Name: "PHPSESSID", Value: "dummy"})

	// execute the query
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	defer func() { /* #nosec */ _ = resp.Body.Close()}()

	return resp, string(respBody), nil
}

func (ctx *TestContext) setupAuthProxyServer() *httptest.Server {
	// set the auth proxy server up
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		dataJSON := fmt.Sprintf(`{"userID": %d, "error":""}`, ctx.userID)
		_, _ = w.Write([]byte(dataJSON)) // nolint
	}))

	// put the backend URL into the config
	backendURL, _ := url.Parse(backend.URL) // nolint
	ctx.app().Config.Auth.ProxyURL = backendURL.String()

	return backend
}

func (ctx *TestContext) db() *sql.DB {
	conf := ctx.app().Config
	conn, err := sql.Open("mysql", conf.Database.Connection.FormatDSN())
	if err != nil {
		fmt.Println("Unable to connect to the database: ", err)
		os.Exit(1)
	}
	return conn
}

// nolint: gosec
func (ctx *TestContext) emptyDB() error {

	db := ctx.db()
	defer func() {_ = db.Close()}()

	dbName := ctx.app().Config.Database.Connection.DBName
	rows, err := db.Query(`SELECT CONCAT(table_schema, '.', table_name)
                         FROM   information_schema.tables
                         WHERE  table_type   = 'BASE TABLE'
                           AND  table_schema = '` + dbName + `'
                           AND  table_name  != 'gorp_migrations'`)
	if err != nil {
		return err
	}
	defer func() {_ = rows.Close()}()

	for rows.Next() {
		var tableName string
		if err = rows.Scan(&tableName); err != nil {
			return err
		}
		_, err = db.Exec("TRUNCATE TABLE " + tableName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ctx *TestContext) initDB() error {
	err := ctx.emptyDB()
	if err != nil {
		return err
	}
	db := ctx.db()
	defer func() { /* #nosec */ _ = db.Close()}()

	for _, query := range ctx.featureQueries {
		_, err := db.Exec(query.sql, query.values)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ctx *TestContext) iSendrequestGeneric(method string, path string, reqBody string) error {
	// app server
	testServer := httptest.NewServer(ctx.app().HTTPHandler)
	defer testServer.Close()

	// auth proxy server
	authProxyServer := ctx.setupAuthProxyServer()
	defer authProxyServer.Close()

	// do request
	response, body, err := testRequest(testServer, method, path, strings.NewReader(reqBody))
	if err != nil {
		return err
	}
	ctx.lastResponse = response
	ctx.lastResponseBody = body

	return nil
}

// dbDataTableValue converts a string value that we can find the db seeding table to a valid type for the db
// e.g., the string "null" means the SQL `NULL`
func dbDataTableValue(input string) interface{} {
	switch input {
	case "false":
		return false
	case "true":
		return true
	case "null":
		return nil
	default:
		return input
	}
}

/** Steps **/

func (ctx *TestContext) DBHasTable(tableName string, data *gherkin.DataTable) error { // nolint

	db := ctx.db()
	defer func() {/* #nosec */ _ = db.Close()}()

	var fields []string
	var marks []string
	head := data.Rows[0].Cells
	for _, cell := range head {
		fields = append(fields, cell.Value)
		marks = append(marks, "?")
	}
	query := "INSERT INTO " + tableName + " (" + strings.Join(fields, ", ") + ") VALUES(" + strings.Join(marks, ", ") + ")" // nolint: gosec
	for i := 1; i < len(data.Rows); i++ {
		var vals []interface{}
		for _, cell := range data.Rows[i].Cells {
			vals = append(vals, dbDataTableValue(cell.Value))
		}
		if ctx.inScenario {
			_, err := db.Exec(query, vals...)
			if err != nil {
				return err
			}
		} else {
			ctx.featureQueries = append(ctx.featureQueries, dbquery{query, vals})
		}
	}
	return nil
}

func (ctx *TestContext) RunFallbackServer() error { // nolint
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Got-Query", r.URL.Path)
	}))
	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		return err
	}
	viper.Set("ReverseProxy.Server", backendURL.String())
	return nil
}

func (ctx *TestContext) IAmUserWithID(id int64) error { // nolint
	ctx.userID = id
	return nil
}

func (ctx *TestContext) TimeNow(timeStr string) error { // nolint
	testTime, err := time.Parse(time.RFC3339Nano, timeStr)
	if err == nil {
		monkey.Patch(time.Now, func() time.Time { return testTime })
	}
	return err
}

func (ctx *TestContext) ISendrequestToWithBody(method string, path string, body *gherkin.DocString) error { // nolint
	return ctx.iSendrequestGeneric(method, path, body.Content)
}

func (ctx *TestContext) ISendrequestTo(method string, path string) error { // nolint
	return ctx.iSendrequestGeneric(method, path, "")
}

func (ctx *TestContext) ItShouldBeAJSONArrayWithEntries(count int) error { // nolint
	var objmap []map[string]*json.RawMessage

	if err := json.Unmarshal([]byte(ctx.lastResponseBody), &objmap); err != nil {
		return fmt.Errorf("unable to decode the response as JSON: %s\nData:%v", err, ctx.lastResponseBody)
	}

	if count != len(objmap) {
		return fmt.Errorf("the result does not have the expected length. Expected: %d, received: %d", count, len(objmap))
	}

	return nil
}

func (ctx *TestContext) TheResponseCodeShouldBe(code int) error { // nolint
	if code != ctx.lastResponse.StatusCode {
		return fmt.Errorf("expected http response code: %d, actual is: %d. \n Data: %s", code, ctx.lastResponse.StatusCode, ctx.lastResponseBody)
	}
	return nil
}

func (ctx *TestContext) TheResponseBodyShouldBeJSON(body *gherkin.DocString) (err error) { // nolint
	var expected, actual []byte
	var exp, act interface{}

	// re-encode expected response
	if err = json.Unmarshal([]byte(body.Content), &exp); err != nil {
		return
	}
	if expected, err = json.MarshalIndent(exp, "", "\t"); err != nil {
		return
	}

	// re-encode actual response too
	if err = json.Unmarshal([]byte(ctx.lastResponseBody), &act); err != nil {
		return fmt.Errorf("unable to decode the response as JSON: %s -- Data: %v", err, ctx.lastResponseBody)
	}
	if actual, err = json.MarshalIndent(act, "", "\t"); err != nil {
		return
	}

	sExpected := string(expected)
	sActual := string(actual)

	if sExpected != sActual {
		diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        difflib.SplitLines(string(expected)),
			B:        difflib.SplitLines(string(actual)),
			FromFile: "Expected",
			FromDate: "",
			ToFile:   "Actual",
			ToDate:   "",
			Context:  1,
		})

		return fmt.Errorf(
			"expected JSON does not match actual.\n     Diff:\n%s",
			diff,
		)
	}
	return
}

func (ctx *TestContext) TheResponseHeaderShouldBe(headerName string, headerValue string) (err error) { // nolint
	if ctx.lastResponse.Header.Get(headerName) != headerValue {
		return fmt.Errorf("headers %s different from expected. Expected: %s, got: %s", headerName, headerValue, ctx.lastResponse.Header.Get(headerName))
	}
	return nil
}

func (ctx *TestContext) TheResponseErrorMessageShouldContain(s string) (err error) { // nolint

	errorResp := service.ErrorResponse{}
	// decode response
	if err = json.Unmarshal([]byte(ctx.lastResponseBody), &errorResp); err != nil {
		return fmt.Errorf("unable to decode the response as JSON: %s -- Data: %v", err, ctx.lastResponseBody)
	}
	if !strings.Contains(errorResp.ErrorText, s) {
		return fmt.Errorf("cannot find expected `%s` in error text: `%s`", s, errorResp.ErrorText)
	}

	return nil
}

func (ctx *TestContext) TableShouldBe(tableName string, data *gherkin.DataTable) error { // nolint
	return ctx.TableAtIDShouldBe(tableName, noID, data)
}

func (ctx *TestContext) TableAtIDShouldBe(tableName string, id int64, data *gherkin.DataTable) error { // nolint
	// For that, we build a SQL request with only the attribute we are interested about (those
	// for the test data table) and we convert them to string (in SQL) to compare to table value.
	// Expect 'null' string in the table to check for nullness

	db := ctx.db()
	defer func() { /* #nosec */ _ = db.Close()}()

	var selects []string
	head := data.Rows[0].Cells
	for _, cell := range head {
		selects = append(selects, fmt.Sprintf("CAST(IFNULL(%s,'NULL') as CHAR(50)) AS %s", cell.Value, cell.Value))
	}

	// define 'where' condition if needed
	where := ""
	if id != noID {
		where = fmt.Sprintf(" WHERE ID = '%d' ", id)
	}

	// exec sql
	query := fmt.Sprintf("SELECT %s FROM `%s` %s", strings.Join(selects, ", "), tableName, where) // nolint: gosec
	sqlRows, err := db.Query(query)
	if err != nil {
		return err
	}
	dataCols := data.Rows[0].Cells
	iDataRow := 1
	sqlCols, _ := sqlRows.Columns() // nolint: gosec
	for sqlRows.Next() {
		if iDataRow >= len(data.Rows) {
			return fmt.Errorf("there are more rows in the SQL results than expected. expected: %d", len(data.Rows)-1)
		}
		// Create a slice of string to represent each attribute value,
		// and a second slice to contain pointers to each item.
		rowValues := make([]string, len(sqlCols))
		rowValPtr := make([]interface{}, len(sqlCols))
		for i := range rowValues {
			rowValPtr[i] = &rowValues[i]
		}
		// Scan the result into the column pointers...
		if err := sqlRows.Scan(rowValPtr...); err != nil {
			return err
		}
		// checking that all columns of the test data table match the SQL row
		for iCol, dataCell := range data.Rows[iDataRow].Cells {
			colName := dataCols[iCol].Value
			dataValue := dataCell.Value
			sqlValue := rowValPtr[iCol].(*string)
			if dataValue != *sqlValue {
				return fmt.Errorf("not matching expected value at row %d, col %s, expected '%s', got: '%v'", iDataRow-1, colName, dataValue, *sqlValue)
			}
		}

		iDataRow++
	}

	// check that no row in the test data table has not been uncheck (if less rows in SQL result)
	if iDataRow < len(data.Rows) {
		return fmt.Errorf("there are less rows in the SQL results than expected. SQL: %d, expected: %d", iDataRow-1, len(data.Rows)-1)
	}
	return nil
}
