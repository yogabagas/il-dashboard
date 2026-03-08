package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// LogInterceptor intercepts and parses gocom log output
type LogInterceptor struct {
	jsonLogger *logrus.Logger
	logFile    *os.File
}

var (
	globalInterceptor *LogInterceptor
	// Regex patterns to parse log lines
	logLinePattern = regexp.MustCompile(`\[([^\]]+)\]\s+(.+)`)
	levelPattern   = regexp.MustCompile(`(INFO|DEBUG|WARN|ERROR|FATAL|PANIC)`)
)

// NewLogInterceptor creates a new log interceptor
func NewLogInterceptor(logDir string) (*LogInterceptor, error) {
	interceptor := &LogInterceptor{
		jsonLogger: logrus.New(),
	}

	// Set JSON formatter
	interceptor.jsonLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
		PrettyPrint: false,
	})

	interceptor.jsonLogger.SetLevel(logrus.DebugLevel)

	// Create log file
	if logDir != "" {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %v", err)
		}

		logFileName := fmt.Sprintf("%s/app-%s.json", logDir, time.Now().Format("2006-01-02"))
		logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %v", err)
		}

		interceptor.logFile = logFile
		interceptor.jsonLogger.SetOutput(logFile)
	} else {
		interceptor.jsonLogger.SetOutput(os.Stdout)
	}

	globalInterceptor = interceptor
	return interceptor, nil
}

// ParseAndLog parses a log line and outputs it as JSON
func (i *LogInterceptor) ParseAndLog(logLine string) {
	if i == nil || i.jsonLogger == nil {
		return
	}

	// Parse log level
	level := logrus.InfoLevel
	if levelMatch := levelPattern.FindString(logLine); levelMatch != "" {
		switch strings.ToUpper(levelMatch) {
		case "DEBUG":
			level = logrus.DebugLevel
		case "INFO":
			level = logrus.InfoLevel
		case "WARN":
			level = logrus.WarnLevel
		case "ERROR":
			level = logrus.ErrorLevel
		case "FATAL":
			level = logrus.FatalLevel
		case "PANIC":
			level = logrus.PanicLevel
		}
	}

	// Parse message
	message := logLine
	if matches := logLinePattern.FindStringSubmatch(logLine); len(matches) > 2 {
		message = matches[2]
	}

	// Extract structured fields if present (e.g., key=value pairs)
	fields := logrus.Fields{}
	
	// Try to detect JSON-like structures in the message
	if strings.Contains(message, "{") && strings.Contains(message, "}") {
		// Extract potential JSON
		start := strings.Index(message, "{")
		end := strings.LastIndex(message, "}") + 1
		if start < end {
			jsonStr := message[start:end]
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(jsonStr), &jsonData); err == nil {
				for k, v := range jsonData {
					fields[k] = v
				}
				message = message[:start] + message[end:]
			}
		}
	}

	// Extract [tag] patterns
	tagPattern := regexp.MustCompile(`\[([^\]]+)\]`)
	tags := tagPattern.FindAllString(message, -1)
	if len(tags) > 0 {
		fields["tags"] = tags
	}

	// Log with appropriate level
	entry := i.jsonLogger.WithFields(fields)
	switch level {
	case logrus.DebugLevel:
		entry.Debug(strings.TrimSpace(message))
	case logrus.InfoLevel:
		entry.Info(strings.TrimSpace(message))
	case logrus.WarnLevel:
		entry.Warn(strings.TrimSpace(message))
	case logrus.ErrorLevel:
		entry.Error(strings.TrimSpace(message))
	case logrus.FatalLevel:
		entry.Fatal(strings.TrimSpace(message))
	case logrus.PanicLevel:
		entry.Panic(strings.TrimSpace(message))
	}
}

// Write implements io.Writer interface
func (i *LogInterceptor) Write(p []byte) (n int, err error) {
	// Split into lines and process each
	lines := bytes.Split(p, []byte("\n"))
	for _, line := range lines {
		if len(line) > 0 {
			i.ParseAndLog(string(line))
		}
	}
	return len(p), nil
}

// Close closes the log file
func (i *LogInterceptor) Close() error {
	if i.logFile != nil {
		return i.logFile.Close()
	}
	return nil
}

// GetGlobalInterceptor returns the global interceptor instance
func GetGlobalInterceptor() *LogInterceptor {
	return globalInterceptor
}
