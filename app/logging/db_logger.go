package logging

import (
	"gorm.io/gorm/logger"
)

// NewDBLogger returns a logger for the database and the `logMode` as well as the 'rawLogMode', according to the config.
func (l *Logger) NewDBLogger() (logger.Interface, bool) {
	if l.config == nil {
		// if cannot parse config, log on error to stdout
		return logger.New(l, logger.Config{
			Colorful: false,
			LogLevel: logger.Info,
		}), false
	}

	logMode := l.config.GetBool("LogSQLQueries")
	gormLogLevel := logger.Info
	if !logMode {
		gormLogLevel = logger.Warn
	}
	switch l.config.GetString("format") {
	case formatText:
		return logger.New(l, logger.Config{
			Colorful: true,
			LogLevel: gormLogLevel,
		}), logMode
	case formatJSON:
		return NewStructuredDBLogger(l.Logger, logger.Config{
			LogLevel: gormLogLevel,
		}), logMode
	default:
		panic("Logging format must be either 'text' or 'json'. Got: " + l.config.GetString("format"))
	}
}

// GetRawSQLLogMode returns whether the raw sql logging is enabled.
func (l *Logger) GetRawSQLLogMode() bool {
	if l.config == nil {
		return false
	}
	return l.config.GetBool("LogRawSQLQueries")
}
