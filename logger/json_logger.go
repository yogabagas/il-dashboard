package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	jsonLogger *logrus.Logger
	useJSON    = false
)

// InitJSONLogger initializes the JSON logger with file output
func InitJSONLogger(logDir string, enableJSON bool) error {
	useJSON = enableJSON

	if !useJSON {
		return nil
	}

	jsonLogger = logrus.New()

	// Set JSON formatter with pretty print for better readability in log viewers (Portainer, etc)
	jsonLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
			logrus.FieldKeyFunc:  "caller",
		},
		PrettyPrint: false, // Enable pretty printing for Portainer/log viewers
	})

	// Set log level
	jsonLogger.SetLevel(logrus.DebugLevel)

	// Create log directory if it doesn't exist
	if logDir != "" {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %v", err)
		}

		// Create log file with date
		logFileName := filepath.Join(logDir, fmt.Sprintf("app-%s.json", time.Now().Format("2006-01-02")))
		logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to open log file: %v", err)
		}

		// Log to both file and stdout
		// jsonLogger.SetOutput(logFile)

		// Also output to stdout for development
		jsonLogger.SetOutput(io.MultiWriter(os.Stdout, logFile))
	} else {
		jsonLogger.SetOutput(os.Stdout)
	}

	return nil
}

// getCaller returns the file and line number of the caller
func getCaller(skip int) map[string]interface{} {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return nil
	}

	// Get just the filename without full path
	fileName := filepath.Base(file)

	// Get function name
	pc, _, _, ok := runtime.Caller(skip)
	funcName := "unknown"
	if ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			funcName = fn.Name()
			// Get just the function name without package path
			parts := strings.Split(funcName, ".")
			if len(parts) > 0 {
				funcName = parts[len(parts)-1]
			}
		}
	}

	return map[string]interface{}{
		"file":     fileName,
		"line":     line,
		"function": funcName,
	}
}

// logJSON logs a message with JSON format
func logJSON(level logrus.Level, format string, args ...interface{}) {
	if !useJSON || jsonLogger == nil {
		return
	}

	message := fmt.Sprintf(format, args...)
	caller := getCaller(3) // Skip 3 levels to get the actual caller

	fields := logrus.Fields{}
	if caller != nil {
		fields["caller"] = caller
	}

	entry := jsonLogger.WithFields(fields)

	switch level {
	case logrus.DebugLevel:
		entry.Debug(message)
	case logrus.InfoLevel:
		entry.Info(message)
	case logrus.WarnLevel:
		entry.Warn(message)
	case logrus.ErrorLevel:
		entry.Error(message)
	case logrus.FatalLevel:
		entry.Fatal(message)
	case logrus.PanicLevel:
		entry.Panic(message)
	}
}

// logJSONWithData logs a message with separate data fields
func logJSONWithData(level logrus.Level, message string, data map[string]interface{}) {
	if !useJSON || jsonLogger == nil {
		return
	}

	caller := getCaller(3) // Skip 3 levels to get the actual caller

	fields := logrus.Fields{}
	if caller != nil {
		fields["caller"] = caller
	}

	// Add data fields separately
	if len(data) > 0 {
		fields["data"] = data
	}

	entry := jsonLogger.WithFields(fields)

	switch level {
	case logrus.DebugLevel:
		entry.Debug(message)
	case logrus.InfoLevel:
		entry.Info(message)
	case logrus.WarnLevel:
		entry.Warn(message)
	case logrus.ErrorLevel:
		entry.Error(message)
	case logrus.FatalLevel:
		entry.Fatal(message)
	case logrus.PanicLevel:
		entry.Panic(message)
	}
}

// Debug logs a debug message in JSON format
func Debug(args ...interface{}) {
	if useJSON {
		logJSON(logrus.DebugLevel, "%s", fmt.Sprint(args...))
	}
}

// Debugf logs a formatted debug message in JSON format
func Debugf(format string, args ...interface{}) {
	if useJSON {
		logJSON(logrus.DebugLevel, format, args...)
	}
}

// Info logs an info message in JSON format
func Info(args ...interface{}) {
	if useJSON {
		logJSON(logrus.InfoLevel, "%s", fmt.Sprint(args...))
	}
}

// Infof logs a formatted info message in JSON format
func Infof(format string, args ...interface{}) {
	if useJSON {
		logJSON(logrus.InfoLevel, format, args...)
	}
}

// Warn logs a warning message in JSON format
func Warn(args ...interface{}) {
	if useJSON {
		logJSON(logrus.WarnLevel, "%s", fmt.Sprint(args...))
	}
}

// Warnf logs a formatted warning message in JSON format
func Warnf(format string, args ...interface{}) {
	if useJSON {
		logJSON(logrus.WarnLevel, format, args...)
	}
}

// Error logs an error message in JSON format
func Error(args ...interface{}) {
	if useJSON {
		logJSON(logrus.ErrorLevel, "%s", fmt.Sprint(args...))
	}
}

// Errorf logs a formatted error message in JSON format
func Errorf(format string, args ...interface{}) {
	if useJSON {
		logJSON(logrus.ErrorLevel, format, args...)
	}
}

// Fatal logs a fatal message in JSON format and exits
func Fatal(args ...interface{}) {
	if useJSON {
		logJSON(logrus.FatalLevel, "%s", fmt.Sprint(args...))
	}
}

// Fatalf logs a formatted fatal message in JSON format and exits
func Fatalf(format string, args ...interface{}) {
	if useJSON {
		logJSON(logrus.FatalLevel, format, args...)
	}
}

// Panic logs a panic message in JSON format and panics
func Panic(args ...interface{}) {
	if useJSON {
		logJSON(logrus.PanicLevel, "%s", fmt.Sprint(args...))
	}
}

// Panicf logs a formatted panic message in JSON format and panics
func Panicf(format string, args ...interface{}) {
	if useJSON {
		logJSON(logrus.PanicLevel, format, args...)
	}
}

// IsJSONEnabled returns whether JSON logging is enabled
func IsJSONEnabled() bool {
	return useJSON
}

// WithFields returns a logger entry with additional fields
func WithFields(fields map[string]interface{}) *logrus.Entry {
	if useJSON && jsonLogger != nil {
		caller := getCaller(2)
		if caller != nil {
			fields["caller"] = caller
		}
		return jsonLogger.WithFields(fields)
	}
	return nil
}

// ====================================================================
// Structured Logging Functions with Separate Data Fields
// ====================================================================

// DebugWithData logs a debug message with structured data
func DebugWithData(message string, data map[string]interface{}) {
	if useJSON {
		logJSONWithData(logrus.DebugLevel, message, data)
	}
}

// InfoWithData logs an info message with structured data
func InfoWithData(message string, data map[string]interface{}) {
	if useJSON {
		logJSONWithData(logrus.InfoLevel, message, data)
	}
}

// WarnWithData logs a warning message with structured data
func WarnWithData(message string, data map[string]interface{}) {
	if useJSON {
		logJSONWithData(logrus.WarnLevel, message, data)
	}
}

// ErrorWithData logs an error message with structured data
func ErrorWithData(message string, data map[string]interface{}) {
	if useJSON {
		logJSONWithData(logrus.ErrorLevel, message, data)
	}
}

// FatalWithData logs a fatal message with structured data and exits
func FatalWithData(message string, data map[string]interface{}) {
	if useJSON {
		logJSONWithData(logrus.FatalLevel, message, data)
	}
}

// ====================================================================
// Advanced Structured Logging - Flat Fields (no nested data object)
// ====================================================================

// logJSONWithFlatFields logs with fields merged at root level
func logJSONWithFlatFields(level logrus.Level, message string, fields map[string]interface{}) {
	if !useJSON || jsonLogger == nil {
		return
	}

	caller := getCaller(3)

	allFields := logrus.Fields{}
	if caller != nil {
		allFields["caller"] = caller
	}

	// Merge data fields at root level
	for k, v := range fields {
		allFields[k] = v
	}

	entry := jsonLogger.WithFields(allFields)

	switch level {
	case logrus.DebugLevel:
		entry.Debug(message)
	case logrus.InfoLevel:
		entry.Info(message)
	case logrus.WarnLevel:
		entry.Warn(message)
	case logrus.ErrorLevel:
		entry.Error(message)
	case logrus.FatalLevel:
		entry.Fatal(message)
	case logrus.PanicLevel:
		entry.Panic(message)
	}
}

// InfoWithFields logs info message with flat fields at root level
func InfoWithFields(message string, fields map[string]interface{}) {
	if useJSON {
		logJSONWithFlatFields(logrus.InfoLevel, message, fields)
	}
}

// ErrorWithFields logs error message with flat fields at root level
func ErrorWithFields(message string, fields map[string]interface{}) {
	if useJSON {
		logJSONWithFlatFields(logrus.ErrorLevel, message, fields)
	}
}

// WarnWithFields logs warning message with flat fields at root level
func WarnWithFields(message string, fields map[string]interface{}) {
	if useJSON {
		logJSONWithFlatFields(logrus.WarnLevel, message, fields)
	}
}

// DebugWithFields logs debug message with flat fields at root level
func DebugWithFields(message string, fields map[string]interface{}) {
	if useJSON {
		logJSONWithFlatFields(logrus.DebugLevel, message, fields)
	}
}

func JsonToString(data any) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}
