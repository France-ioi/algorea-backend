package logging_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/sirupsen/logrus"            //nolint:depguard
	"github.com/sirupsen/logrus/hooks/test" //nolint:depguard
	"github.com/spf13/viper"
	assertlib "github.com/stretchr/testify/assert"
	"gorm.io/gorm/logger"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/logging"
)

func TestStructuredDBLogger_Print_SQL(t *testing.T) {
	assert := assertlib.New(t)
	var hook *test.Hook
	logging.SharedLogger, hook = logging.NewMockLogger()
	defer logging.ResetShared()
	conf := viper.New()
	conf.Set("Format", "json")
	conf.Set("Output", "stdout")
	conf.Set("LogSQLQueries", true)
	logging.SharedLogger.Configure(conf)
	db, mock := database.NewDBMock()
	defer func() { _ = db.Close() }()

	timeParam := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`^SELECT \?, \?, \?, \?, \?$`).
		WithArgs(1, timeParam, "foo", []byte("bar"), nil).
		WillReturnRows(mock.NewRows([]string{"1"}).AddRow(1))

	var result []map[string]interface{}
	db.Raw("SELECT ?, ?, ?, ?, ?", 1, timeParam, "foo", []byte("bar"), nil).Scan(&result)
	data := hook.LastEntry().Data
	assert.Equal(`SELECT 1, '2009-11-10 23:00:00', 'foo', 'bar', NULL`, data["sql"])
	assert.Equal("db", data["type"])
	assert.True(data["duration"].(float64) < 1.0, "unexpected duration: %v", data["duration"])
	assert.Equal(int64(1), data["rows"].(int64))
	assert.NoError(mock.ExpectationsWereMet())
}

func TestStructuredDBLogger_Print_SQLWithInterrogationMark(t *testing.T) {
	assert := assertlib.New(t)
	var hook *test.Hook
	logging.SharedLogger, hook = logging.NewMockLogger()
	defer logging.ResetShared()
	conf := viper.New()
	conf.Set("Format", "json")
	conf.Set("Output", "stdout")
	conf.Set("LogSQLQueries", true)
	logging.SharedLogger.Configure(conf)
	db, mock := database.NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(`^SELECT \?$`).WithArgs(1).WillReturnRows(mock.NewRows([]string{"1"}).AddRow(1))

	var result []interface{}
	db.Raw("SELECT ?", 1).Scan(&result)
	assert.Equal("SELECT 1", hook.LastEntry().Data["sql"])
	assert.NoError(mock.ExpectationsWereMet())
}

func TestStructuredDBLogger_Print_SQLError(t *testing.T) {
	assert := assertlib.New(t)
	var hook *test.Hook
	logging.SharedLogger, hook = logging.NewMockLogger()
	defer logging.ResetShared()
	conf := viper.New()
	conf.Set("Format", "json")
	conf.Set("Output", "stdout")
	conf.Set("LogSQLQueries", true)
	conf.Set("Level", "debug")
	logging.SharedLogger.Configure(conf)
	db, mock := database.NewDBMock()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT 2").WillReturnError(errors.New("a query error"))

	var result []interface{}
	db.Raw("SELECT 2").Scan(&result)

	assert.Contains(hook.Entries[0].Message, "a query error")
	data := hook.Entries[0].Data
	assert.Equal("SELECT 2", data["sql"])
	assert.Equal("db", data["type"])
	assert.True(data["duration"].(float64) < 1.0, "unexpected duration: %v", data["duration"])
	assert.Nil(data["rows"])
	assert.NoError(mock.ExpectationsWereMet())
}

func TestStructuredDBLogger_LogMode(t *testing.T) {
	tests := []struct {
		name  string
		level logger.LogLevel
	}{
		{name: "error", level: logger.Error},
		{name: "warn", level: logger.Warn},
		{name: "info", level: logger.Info},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			l := &logging.StructuredDBLogger{}
			l1 := l.LogMode(tt.level)
			assertlib.Equal(t, tt.level, l1.(*logging.StructuredDBLogger).LogLevel)
		})
	}
}

func TestStructuredDBLogger_Levels(t *testing.T) {
	for _, tt := range []struct {
		name        string
		gormLevel   logger.LogLevel
		logrusLevel logrus.Level
	}{
		{name: "Info", gormLevel: logger.Info, logrusLevel: logrus.InfoLevel},
		{name: "Warn", gormLevel: logger.Warn, logrusLevel: logrus.WarnLevel},
		{name: "Error", gormLevel: logger.Error, logrusLevel: logrus.ErrorLevel},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assert := assertlib.New(t)
			l, hook := logging.NewMockLogger()
			dbLogger := logging.NewStructuredDBLogger(l.Logger, logger.Config{LogLevel: tt.gormLevel})
			reflect.ValueOf(dbLogger).MethodByName(tt.name).Call([]reflect.Value{
				reflect.ValueOf(context.Background()), reflect.ValueOf("message"), reflect.ValueOf("val1"), reflect.ValueOf("val2"),
			})

			lastEntry := hook.LastEntry()
			assert.Equal(tt.logrusLevel, lastEntry.Level)
			assert.Equal("message", lastEntry.Message)
			assert.Equal([]interface{}{"val1", "val2"}, lastEntry.Data["data"])
		})
	}
}

func TestStructuredDBLogger_SlowQuery(t *testing.T) {
	assert := assertlib.New(t)
	l, hook := logging.NewMockLogger()
	dbLogger := logging.NewStructuredDBLogger(l.Logger, logger.Config{LogLevel: logger.Warn, SlowThreshold: 1})
	dbLogger.Trace(context.Background(), time.Date(2019, 5, 30, 11, 0, 0, 0, time.UTC),
		func() (string, int64) { return "SELECT 1", 1 }, nil)

	lastEntry := hook.LastEntry()
	assert.Equal(logrus.WarnLevel, lastEntry.Level)
	assert.Equal("SLOW SQL >= 1ns", lastEntry.Data["slow_log"])
	assert.Equal("SELECT 1", lastEntry.Data["sql"])
	assert.Equal(int64(1), lastEntry.Data["rows"])
}
