package logging

import (
	"context"
	"database/sql/driver"
	"regexp"
	"strconv"
	"strings"

	"github.com/luna-duclos/instrumentedsql"
	"github.com/sirupsen/logrus" //nolint:depguard
	"gorm.io/gorm/logger"
)

var (
	rawArgsRegexp = regexp.MustCompile(`^\[(<nil>|[\w.]+) (.+?)\](?:(?:, \[(?:<nil>|[\w.]+) )|$)`)
	spaceRegexp   = regexp.MustCompile(`\s+`)
)

// NewRawDBLogger returns a logger for raw database actions using an existing dblogger and rawLogMode setting.
func NewRawDBLogger(logrusLogger *logrus.Logger, rawLogMode bool) instrumentedsql.Logger {
	return instrumentedsql.LoggerFunc(func(ctx context.Context, msg string, keyvals ...interface{}) {
		if !rawLogMode {
			return
		}

		valuesMap := prepareRawDBLoggerValuesMap(keyvals)
		if valuesMap["err"] != nil && valuesMap["err"] == driver.ErrSkip { // duplicated message
			return
		}
		valuesMap["type"] = "rawsql"
		if query, ok := valuesMap["query"]; ok && msg != "sql-prepare" {
			msg += "\n" + query.(string) + "\n"
			delete(valuesMap, "query")
		}

		logrusLogger.WithFields(valuesMap).Println(msg)
	})
}

func prepareRawDBLoggerValuesMap(keyvals []interface{}) map[string]interface{} {
	valuesMap := make(map[string]interface{}, len(keyvals)/2)
	for index := 0; index < len(keyvals); index += 2 {
		valuesMap[keyvals[index].(string)] = keyvals[index+1]
	}
	if valuesMap["query"] != nil {
		if valuesMap["args"] != nil {
			argsString := valuesMap["args"].(string)
			argsString = argsString[1 : len(argsString)-1]
			var argsValues []interface{}
			for argsString != "" {
				indices := rawArgsRegexp.FindStringSubmatchIndex(argsString)
				typeStr := argsString[indices[2]:indices[3]]
				valueCopy := make([]byte, indices[5]-indices[4])
				copy(valueCopy, argsString[indices[4]:indices[5]])
				value := string(valueCopy)
				convertedValue := convertRawSQLArgValue(value, typeStr)
				argsValues = append(argsValues, convertedValue)
				if indices[5]+3 >= len(argsString) {
					break
				}
				argsString = argsString[indices[5]+3:]
			}
			valuesMap["query"] = logger.ExplainSQL(valuesMap["query"].(string), nil, `"`, argsValues...)
		}
		valuesMap["query"] = strings.TrimSpace(spaceRegexp.ReplaceAllString(valuesMap["query"].(string), " "))
		delete(valuesMap, "args")
	}
	return valuesMap
}

func convertRawSQLArgValue(value, typeStr string) interface{} {
	var convertedValue interface{} = value
	switch typeStr {
	case "string":
		if unquoted, err := strconv.Unquote(value); err == nil {
			convertedValue = unquoted
		}
	case "int64", "int32", "int16", "int8", "int":
		if converted, err := strconv.ParseInt(value, 10, 64); err == nil {
			convertedValue = converted
		}
	case "float64", "float32":
		if converted, err := strconv.ParseFloat(value, 64); err == nil {
			convertedValue = converted
		}
	case "<nil>":
		convertedValue = nil
	}
	return convertedValue
}
