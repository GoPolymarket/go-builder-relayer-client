// Package logger provides a unified logging interface for the SDK.
// It supports multiple log levels and allows users to inject custom loggers.
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

// Level represents the severity level of a log message.
type Level int

const (
	// LevelDebug is for detailed debugging information.
	LevelDebug Level = iota
	// LevelInfo is for general informational messages.
	LevelInfo
	// LevelWarn is for warning messages.
	LevelWarn
	// LevelError is for error messages.
	LevelError
	// LevelNone disables all logging.
	LevelNone
)

// String returns the string representation of the log level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelNone:
		return "NONE"
	default:
		return "UNKNOWN"
	}
}

// Logger is the interface for logging in the SDK.
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	SetLevel(level Level)
	GetLevel() Level
}

// StandardLogger is the default logger implementation using Go's standard log package.
type StandardLogger struct {
	mu     sync.RWMutex
	level  Level
	logger *log.Logger
}

// NewStandardLogger creates a new standard logger with the specified level and output.
func NewStandardLogger(level Level, out io.Writer) *StandardLogger {
	if out == nil {
		out = os.Stderr
	}
	return &StandardLogger{
		level:  level,
		logger: log.New(out, "", log.LstdFlags),
	}
}

// Debug logs a debug message.
func (l *StandardLogger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an informational message.
func (l *StandardLogger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message.
func (l *StandardLogger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message.
func (l *StandardLogger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// SetLevel sets the minimum log level.
func (l *StandardLogger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current log level.
func (l *StandardLogger) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// log is the internal logging method.
func (l *StandardLogger) log(level Level, format string, args ...interface{}) {
	l.mu.RLock()
	currentLevel := l.level
	l.mu.RUnlock()

	if level < currentLevel {
		return
	}

	prefix := fmt.Sprintf("[%s] ", level.String())
	message := fmt.Sprintf(format, args...)
	l.logger.Print(prefix + message)
}

// NoOpLogger is a logger that does nothing.
type NoOpLogger struct{}

// NewNoOpLogger creates a new no-op logger.
func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{}
}

// Debug does nothing.
func (l *NoOpLogger) Debug(format string, args ...interface{}) {}

// Info does nothing.
func (l *NoOpLogger) Info(format string, args ...interface{}) {}

// Warn does nothing.
func (l *NoOpLogger) Warn(format string, args ...interface{}) {}

// Error does nothing.
func (l *NoOpLogger) Error(format string, args ...interface{}) {}

// SetLevel does nothing.
func (l *NoOpLogger) SetLevel(level Level) {}

// GetLevel returns LevelNone.
func (l *NoOpLogger) GetLevel() Level {
	return LevelNone
}

var (
	mu            sync.RWMutex
	defaultLogger Logger
)

func init() {
	// Default to Info level, can be overridden by environment variable
	level := LevelInfo
	if envLevel := os.Getenv("POLYMARKET_LOG_LEVEL"); envLevel != "" {
		switch envLevel {
		case "DEBUG":
			level = LevelDebug
		case "INFO":
			level = LevelInfo
		case "WARN":
			level = LevelWarn
		case "ERROR":
			level = LevelError
		case "NONE":
			level = LevelNone
		}
	}
	defaultLogger = NewStandardLogger(level, os.Stderr)
}

// SetDefault sets the default logger for the SDK.
func SetDefault(logger Logger) {
	mu.Lock()
	defer mu.Unlock()
	defaultLogger = logger
}

// GetDefault returns the default logger.
func GetDefault() Logger {
	mu.RLock()
	defer mu.RUnlock()
	return defaultLogger
}

// Debug logs a debug message using the default logger.
func Debug(format string, args ...interface{}) {
	GetDefault().Debug(format, args...)
}

// Info logs an informational message using the default logger.
func Info(format string, args ...interface{}) {
	GetDefault().Info(format, args...)
}

// Warn logs a warning message using the default logger.
func Warn(format string, args ...interface{}) {
	GetDefault().Warn(format, args...)
}

// Error logs an error message using the default logger.
func Error(format string, args ...interface{}) {
	GetDefault().Error(format, args...)
}

// SetLevel sets the log level for the default logger.
func SetLevel(level Level) {
	GetDefault().SetLevel(level)
}

// GetLevel returns the current log level of the default logger.
func GetLevel() Level {
	return GetDefault().GetLevel()
}
