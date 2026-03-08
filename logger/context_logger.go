package logger

import (
	"context"

	"github.com/oklog/ulid/v2"
)

// Context key for request ID
type contextKey string

const requestIDKey contextKey = "request_id"

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// logWithRequestID adds request_id to log data from context, or generates one if not present
func logWithRequestID(ctx context.Context, data map[string]interface{}) map[string]interface{} {
	requestID := GetRequestID(ctx)
	if requestID == "" {
		// Generate a new request_id if not present in context
		requestID = ulid.Make().String()
	}
	data["request_id"] = requestID
	return data
}

// InfoWithContext logs info message with automatic request_id from context
func InfoWithContext(ctx context.Context, message string, data map[string]interface{}) {
	InfoWithData(message, logWithRequestID(ctx, data))
}

// DebugWithContext logs debug message with automatic request_id from context
func DebugWithContext(ctx context.Context, message string, data map[string]interface{}) {
	DebugWithData(message, logWithRequestID(ctx, data))
}

// WarnWithContext logs warning message with automatic request_id from context
func WarnWithContext(ctx context.Context, message string, data map[string]interface{}) {
	WarnWithData(message, logWithRequestID(ctx, data))
}

// ErrorWithContext logs error message with automatic request_id from context
func ErrorWithContext(ctx context.Context, message string, data map[string]interface{}) {
	ErrorWithData(message, logWithRequestID(ctx, data))
}
