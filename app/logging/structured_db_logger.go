package logging

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus" //nolint:depguard
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

// NewStructuredDBLogger creates a database structured logger.
func NewStructuredDBLogger(logrusLogger *logrus.Logger, config logger.Config) logger.Interface {
	return &StructuredDBLogger{
		logrusLogger: logrusLogger,
		Config:       config,
	}
}

// StructuredDBLogger is a database structured logger.
type StructuredDBLogger struct {
	logrusLogger *logrus.Logger
	logger.Config
}

// LogMode log mode.
func (l *StructuredDBLogger) LogMode(level logger.LogLevel) logger.Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

// Info print info.
func (l *StructuredDBLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		l.logrusLogger.
			WithFields(map[string]interface{}{"file_line": utils.FileWithLineNum(), "data": data}).
			Info(msg)
	}
}

// Warn print warn messages.
func (l *StructuredDBLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		l.logrusLogger.
			WithFields(map[string]interface{}{"file_line": utils.FileWithLineNum(), "data": data}).
			Warn(msg)
	}
}

// Error print error messages.
func (l *StructuredDBLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		l.logrusLogger.
			WithFields(map[string]interface{}{"file_line": utils.FileWithLineNum(), "data": data}).
			Error(msg)
	}
}

// Trace print sql message.
func (l *StructuredDBLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	data := map[string]interface{}{
		"file_line": utils.FileWithLineNum(),
		"duration":  float64(elapsed.Nanoseconds()) / 1e6,
		"sql":       sql,
		"type":      "db",
	}
	if rows != -1 {
		data["rows"] = rows
	}
	l.trace(err, data, elapsed)
}

func (l *StructuredDBLogger) trace(err error, data map[string]interface{}, elapsed time.Duration) {
	switch {
	case err != nil && l.LogLevel >= logger.Error && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		l.logrusLogger.WithFields(data).Error(err)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= logger.Warn:
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		data["slow_log"] = slowLog
		l.logrusLogger.WithFields(data).Warn("")
	case l.LogLevel == logger.Info:
		l.logrusLogger.WithFields(data).Info("")
	}
}
