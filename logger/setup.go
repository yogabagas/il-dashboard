package logger

import (
	"github.com/ariandi/gocom/config"
	gocomLogger "github.com/ariandi/gocom/logger"
)

// Setup initializes both traditional and JSON logging
func Setup(logDir string) error {
	// Setup traditional gocom logger
	gocomLogger.SetupLogs(logDir)
	
	// Check if JSON logging is enabled via config
	enableJSON := config.GetBool("app.log.json.enabled", true) // Default to true for JSON logging
	
	// Initialize JSON logger
	if err := InitJSONLogger(logDir, enableJSON); err != nil {
		gocomLogger.Errorf("Failed to initialize JSON logger: %v", err)
		return err
	}
	
	if enableJSON {
		gocomLogger.Info("[LOGGER] JSON logging enabled")
	} else {
		gocomLogger.Info("[LOGGER] JSON logging disabled, using traditional format")
	}
	
	return nil
}

// Wrapper functions that call both loggers
// These maintain compatibility with existing code while adding JSON logging

// Debug logs a debug message to both loggers
func DebugBoth(args ...interface{}) {
	gocomLogger.Debug(args...)
	Debug(args...)
}

// Debugf logs a formatted debug message to both loggers
func DebugfBoth(format string, args ...interface{}) {
	gocomLogger.Debugf(format, args...)
	Debugf(format, args...)
}

// Info logs an info message to both loggers
func InfoBoth(args ...interface{}) {
	gocomLogger.Info(args...)
	Info(args...)
}

// Infof logs a formatted info message to both loggers
func InfofBoth(format string, args ...interface{}) {
	gocomLogger.Infof(format, args...)
	Infof(format, args...)
}

// Warn logs a warning message to both loggers
func WarnBoth(args ...interface{}) {
	gocomLogger.Warn(args...)
	Warn(args...)
}

// Warnf logs a formatted warning message to both loggers
func WarnfBoth(format string, args ...interface{}) {
	gocomLogger.Warnf(format, args...)
	Warnf(format, args...)
}

// Error logs an error message to both loggers
func ErrorBoth(args ...interface{}) {
	gocomLogger.Error(args...)
	Error(args...)
}

// Errorf logs a formatted error message to both loggers
func ErrorfBoth(format string, args ...interface{}) {
	gocomLogger.Errorf(format, args...)
	Errorf(format, args...)
}

// Fatal logs a fatal message to both loggers and exits
func FatalBoth(args ...interface{}) {
	gocomLogger.Fatal(args...)
	Fatal(args...)
}

// Fatalf logs a formatted fatal message to both loggers and exits
func FatalfBoth(format string, args ...interface{}) {
	gocomLogger.Fatalf(format, args...)
	Fatalf(format, args...)
}
