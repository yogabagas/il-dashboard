package logger

import (
	"io"
	"os"
	"sync"

	gocomLogger "github.com/ariandi/gocom/logger"
)

var (
	loggerHook     io.Writer
	hookMutex      sync.Mutex
	originalOutput io.Writer
)

// LogHook is a custom writer that intercepts log output
type LogHook struct {
	output io.Writer
}

// Write implements io.Writer interface and forwards to JSON logger
func (h *LogHook) Write(p []byte) (n int, err error) {
	// Forward to original output (gocom logger)
	n, err = h.output.Write(p)
	
	// Also log to JSON logger if enabled
	// Parse the log message and extract components if needed
	// For now, we'll rely on the direct logger calls
	
	return n, err
}

// InstallHook installs a hook to intercept gocom logger output
func InstallHook() {
	hookMutex.Lock()
	defer hookMutex.Unlock()
	
	// Create hook that wraps stdout
	originalOutput = os.Stdout
	loggerHook = &LogHook{output: originalOutput}
	
	// Note: gocom logger might not expose a way to change output
	// In this case, we rely on direct logger call interception
}

// RemoveHook removes the logger hook
func RemoveHook() {
	hookMutex.Lock()
	defer hookMutex.Unlock()
	
	loggerHook = nil
}

// InterceptedLogger is a wrapper that intercepts logger calls
type InterceptedLogger struct{}

// Info intercepts Info calls
func (l *InterceptedLogger) Info(args ...interface{}) {
	gocomLogger.Info(args...)
	Info(args...)
}

// Infof intercepts Infof calls
func (l *InterceptedLogger) Infof(format string, args ...interface{}) {
	gocomLogger.Infof(format, args...)
	Infof(format, args...)
}

// Debug intercepts Debug calls
func (l *InterceptedLogger) Debug(args ...interface{}) {
	gocomLogger.Debug(args...)
	Debug(args...)
}

// Debugf intercepts Debugf calls
func (l *InterceptedLogger) Debugf(format string, args ...interface{}) {
	gocomLogger.Debugf(format, args...)
	Debugf(format, args...)
}

// Warn intercepts Warn calls
func (l *InterceptedLogger) Warn(args ...interface{}) {
	gocomLogger.Warn(args...)
	Warn(args...)
}

// Warnf intercepts Warnf calls
func (l *InterceptedLogger) Warnf(format string, args ...interface{}) {
	gocomLogger.Warnf(format, args...)
	Warnf(format, args...)
}

// Error intercepts Error calls
func (l *InterceptedLogger) Error(args ...interface{}) {
	gocomLogger.Error(args...)
	Error(args...)
}

// Errorf intercepts Errorf calls
func (l *InterceptedLogger) Errorf(format string, args ...interface{}) {
	gocomLogger.Errorf(format, args...)
	Errorf(format, args...)
}

// Fatal intercepts Fatal calls
func (l *InterceptedLogger) Fatal(args ...interface{}) {
	gocomLogger.Fatal(args...)
	Fatal(args...)
}

// Fatalf intercepts Fatalf calls
func (l *InterceptedLogger) Fatalf(format string, args ...interface{}) {
	gocomLogger.Fatalf(format, args...)
	Fatalf(format, args...)
}

// NewInterceptedLogger creates a new intercepted logger instance
func NewInterceptedLogger() *InterceptedLogger {
	return &InterceptedLogger{}
}
