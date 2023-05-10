package logging

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/sirupsen/logrus" //nolint:depguard
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/France-ioi/AlgoreaBackend/app/loggingtest"
)

func TestNewRawDBLogger_TextLog(t *testing.T) {
	nulllogger, hook := loggingtest.NewNullLogger()
	config := viper.New()
	config.Set("Format", "text")
	config.Set("LogRawSQLQueries", true)
	ourLogger := &Logger{nulllogger, config}
	ourLogger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	ourLogger.Configure(ourLogger.config)
	rawLogMode := ourLogger.GetRawSQLLogMode()

	rawLogger := NewRawDBLogger(nulllogger, rawLogMode)
	rawLogger.Log(context.TODO(), "some message", "err", nil)
	assert.Contains(t, hook.GetAllStructuredLogs(), "msg=\"some message\" err=\"<nil>\" type=rawsql")
}

func TestNewRawDBLogger_HonoursLogMode(t *testing.T) {
	nulllogger, hook := loggingtest.NewNullLogger()
	config := viper.New()
	config.Set("Format", "text")
	config.Set("LogRawSQLQueries", false)
	logger := &Logger{nulllogger, config}
	logger.Configure(logger.config)
	rawLogMode := logger.GetRawSQLLogMode()
	rawLogger := NewRawDBLogger(nulllogger, rawLogMode)
	rawLogger.Log(context.TODO(), "some message", "err", nil)
	assert.Empty(t, hook.GetAllStructuredLogs())
}

func TestNewRawDBLogger_JSONLog(t *testing.T) {
	nulllogger, hook := loggingtest.NewNullLogger()
	config := viper.New()
	config.Set("Format", "json")
	config.Set("LogRawSQLQueries", true)
	logger := &Logger{nulllogger, config}
	logger.SetFormatter(&logrus.JSONFormatter{})
	rawLogMode := logger.GetRawSQLLogMode()
	rawLogger := NewRawDBLogger(nulllogger, rawLogMode)
	rawLogger.Log(context.TODO(), "some message", "err", nil)
	assert.Contains(t, hook.GetAllStructuredLogs(), `"msg":"some message"`)
	assert.Contains(t, hook.GetAllStructuredLogs(), `"err":null`)
}

func TestNewRawDBLogger_TextLog_WithQuery(t *testing.T) {
	nulllogger, hook := loggingtest.NewNullLogger()
	config := viper.New()
	config.Set("Format", "text")
	config.Set("LogRawSQLQueries", true)
	ourLogger := &Logger{nulllogger, config}
	ourLogger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	ourLogger.Configure(ourLogger.config)
	rawLogMode := ourLogger.GetRawSQLLogMode()

	rawLogger := NewRawDBLogger(nulllogger, rawLogMode)
	rawLogger.Log(context.TODO(), "some message", "err", nil, "query", "SELECT 1")
	assert.Contains(t, hook.GetAllStructuredLogs(), "msg=\"some message\\nSELECT 1\\n\" err=\"<nil>\" type=rawsql")
}

func TestRawDBLogger_ShouldSkipSkippedActions(t *testing.T) {
	nulllogger, hook := loggingtest.NewNullLogger()
	config := viper.New()
	config.Set("Format", "json")
	config.Set("LogRawSQLQueries", true)
	logger := &Logger{nulllogger, config}
	rawLogMode := logger.GetRawSQLLogMode()
	rawLogger := NewRawDBLogger(nulllogger, rawLogMode)
	rawLogger.Log(context.TODO(), "sql-stmt-exec", "err", driver.ErrSkip)
	assert.Empty(t, hook.GetAllStructuredLogs())
}

func Test_prepareRawDBLoggerValuesMap(t *testing.T) {
	tests := []struct {
		name    string
		keyvals []interface{}
		want    map[string]interface{}
	}{
		{
			name: "simple",
			keyvals: []interface{}{
				"query", "SELECT * FROM users WHERE users.id=? and users.name=?",
				"args", "{[int64 1], [string \"Joe\"]}",
			},
			want: map[string]interface{}{
				"query": `SELECT * FROM users WHERE users.id=1 and users.name="Joe"`,
			},
		},
		{
			name: "with time",
			keyvals: []interface{}{
				"query", "UPDATE `users_items` SET `children_validated` = ?, `tasks_solved` = ?, `tasks_tried` = ?, " +
					"`tasks_with_help` = ?, `latest_activity_at` = ? WHERE (id=13)",
				"args", "{[int64 8], [int64 7], [int64 5], [int64 6], [time.Time 2019-03-18 16:24:01 +0000 UTC]}",
			},
			want: map[string]interface{}{
				"query": "UPDATE `users_items` SET `children_validated` = 8, `tasks_solved` = 7, `tasks_tried` = 5, " +
					"`tasks_with_help` = 6, `latest_activity_at` = \"2019-03-18 16:24:01 +0000 UTC\" WHERE (id=13)",
			},
		},
		{
			name: "with time and nils",
			keyvals: []interface{}{
				"query", "UPDATE some_table SET column1 = ?, column2 = ?, column3 = ?, column4 = ?",
				"args", "{[<nil> <nil>], [time.Time 2019-03-18 16:24:01 +0000 UTC], [<nil> <nil>], [time.Time 2018-03-18 16:24:01 +0000 UTC]}",
			},
			want: map[string]interface{}{
				"query": "UPDATE some_table SET column1 = NULL, column2 = \"2019-03-18 16:24:01 +0000 UTC\", column3 = NULL, " +
					"column4 = \"2018-03-18 16:24:01 +0000 UTC\"",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := prepareRawDBLoggerValuesMap(tt.keyvals)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_convertRawSQLArgValue(t *testing.T) {
	tests := []struct {
		value   string
		typeStr string
		want    interface{}
	}{
		{value: `"some string \"' "`, typeStr: "string", want: `some string "' `},
		{value: `1234`, typeStr: "int", want: int64(1234)},
		{value: `1234`, typeStr: "int8", want: int64(1234)},
		{value: `1234`, typeStr: "int16", want: int64(1234)},
		{value: `1234`, typeStr: "int32", want: int64(1234)},
		{value: `1234`, typeStr: "int64", want: int64(1234)},
		{value: `1234`, typeStr: "int64", want: int64(1234)},
		{value: `12.34`, typeStr: "float64", want: float64(12.34)},
		{value: `12.34`, typeStr: "float32", want: float64(12.34)},
		{value: `some_value`, typeStr: "<nil>", want: nil},
		{value: `some_value`, typeStr: "unknown type", want: `some_value`},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.typeStr, func(t *testing.T) {
			got := convertRawSQLArgValue(tt.value, tt.typeStr)
			assert.Equal(t, tt.want, got)
		})
	}
}
