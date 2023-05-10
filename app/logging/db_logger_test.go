package logging

import (
	"fmt"
	"testing"

	"github.com/spf13/viper"
	assertlib "github.com/stretchr/testify/assert"
	"gorm.io/gorm/logger"
)

func TestNewDBLogger_ErrorFallback(t *testing.T) {
	assert := assertlib.New(t)
	ourLogger := createLogger() // no config
	dbLogger, logMode := ourLogger.NewDBLogger()
	assert.IsType(Logger{}, dbLogger)
	assert.False(logMode)
}

func TestLogger_GetRawSQLLogMode_ErrorFallback(t *testing.T) {
	assert := assertlib.New(t)
	ourLogger := createLogger() // no config
	rawLogMode := ourLogger.GetRawSQLLogMode()
	assert.False(rawLogMode)
}

func TestLoggerFromConfig_TextLog(t *testing.T) {
	assert := assertlib.New(t)
	ourLogger := createLogger()
	config := viper.New()
	config.Set("Format", "text")
	config.Set("Output", "file")
	ourLogger.Configure(config)
	dbLogger, _ := ourLogger.NewDBLogger()
	assert.Implements((*logger.Interface)(nil), dbLogger)
}

func TestLoggerFromConfig_JSONLog(t *testing.T) {
	assert := assertlib.New(t)
	ourLogger := createLogger()
	config := viper.New()
	config.Set("Format", "json")
	config.Set("Output", "file")
	ourLogger.Configure(config)
	dbLogger, _ := ourLogger.NewDBLogger()
	assert.IsType(&StructuredDBLogger{}, dbLogger)
}

func TestLoggerFromConfig_WrongFormat(t *testing.T) {
	assert := assertlib.New(t)
	ourLogger := createLogger()
	config := viper.New()
	config.Set("Format", "yml")
	config.Set("Output", "file")
	ourLogger.config = config
	assert.Panics(func() { ourLogger.NewDBLogger() })
}

func TestNewDBLogger_LogMode(t *testing.T) {
	tests := []struct {
		name          string
		format        string
		logSQLQueries bool
	}{
		{name: "text: without SQL", format: "text", logSQLQueries: false},
		{name: "text: with SQL", format: "text", logSQLQueries: true},
		{name: "text: only raw SQL", format: "text", logSQLQueries: false},
		{name: "text: full SQL logging", format: "text", logSQLQueries: true},
		{name: "json: without SQL", format: "json", logSQLQueries: false},
		{name: "json: with SQL", format: "json", logSQLQueries: true},
		{name: "json: only raw SQL", format: "json", logSQLQueries: false},
		{name: "json: full SQL logging", format: "json", logSQLQueries: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			assert := assertlib.New(t)
			ourLogger := createLogger()
			config := viper.New()
			config.Set("LogSQLQueries", test.logSQLQueries)
			config.Set("Format", test.format)
			config.Set("Output", "file")
			ourLogger.Configure(config)
			_, logMode := ourLogger.NewDBLogger()
			assert.Equal(test.logSQLQueries, logMode)
		})
	}
}

func TestLogger_GetRawSqlLog(t *testing.T) {
	tests := []bool{false, true}
	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("%v", test), func(t *testing.T) {
			assert := assertlib.New(t)
			ourLogger := createLogger()
			config := viper.New()
			config.Set("LogRawSQLQueries", test)
			ourLogger.Configure(config)
			rawLogMode := ourLogger.GetRawSQLLogMode()
			assert.Equal(test, rawLogMode)
		})
	}
}
