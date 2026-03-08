# Structured Logging - Quick Reference

## Import

```go
import customLogger "gitlab.com/bot3342545/il-dashboard/logger"
```

## Nested Data Format (Recommended)

### Info
```go
customLogger.InfoWithData("User action", map[string]interface{}{
    "user_id": "123",
    "action": "login",
})
```

### Error
```go
customLogger.ErrorWithData("Operation failed", map[string]interface{}{
    "error": err.Error(),
    "user_id": "123",
})
```

### Warn
```go
customLogger.WarnWithData("Slow operation", map[string]interface{}{
    "duration_ms": 1500,
    "threshold_ms": 1000,
})
```

### Debug
```go
customLogger.DebugWithData("Debug info", map[string]interface{}{
    "var1": value1,
    "var2": value2,
})
```

## Flat Fields Format

### Info
```go
customLogger.InfoWithFields("Request completed", map[string]interface{}{
    "method": "POST",
    "status": 200,
    "duration_ms": 145,
})
```

### Error
```go
customLogger.ErrorWithFields("Request failed", map[string]interface{}{
    "method": "POST",
    "status": 500,
    "error": err.Error(),
})
```

## Output Examples

### Nested Data Output
```json
{
  "timestamp": "2024-01-24T12:34:56+07:00",
  "level": "info",
  "message": "User action",
  "caller": {...},
  "data": {
    "user_id": "123",
    "action": "login"
  }
}
```

### Flat Fields Output
```json
{
  "timestamp": "2024-01-24T12:34:56+07:00",
  "level": "info",
  "message": "Request completed",
  "method": "POST",
  "status": 200,
  "duration_ms": 145,
  "caller": {...}
}
```

## Common Queries

### Find by Field
```bash
# Nested data
jq 'select(.data.user_id=="123")'

# Flat fields
jq 'select(.user_id=="123")'
```

### Find Errors
```bash
jq 'select(.level=="error")'
```

### Find Slow Operations
```bash
jq 'select(.data.duration_ms > 1000)'
```

### Count by Type
```bash
jq '.data.message_type' | sort | uniq -c
```

## Common Patterns

### User Action
```go
customLogger.InfoWithData("User action", map[string]interface{}{
    "component": "AuthHandler",
    "user_id": userID,
    "action": "login",
    "ip": ipAddress,
})
```

### Payment Event
```go
customLogger.InfoWithData("Payment processed", map[string]interface{}{
    "component": "PaymentHandler",
    "bill_id": billID,
    "amount": amount,
    "user_phone": userPhone,
    "status": "success",
})
```

### Error with Context
```go
customLogger.ErrorWithData("Operation failed", map[string]interface{}{
    "component": "DatabaseService",
    "error": err.Error(),
    "operation": "insert",
    "table": "messages",
    "retry_count": retryCount,
})
```

### Performance Tracking
```go
startTime := time.Now()
// ... do work ...
duration := time.Since(startTime).Milliseconds()

customLogger.InfoWithData("Operation completed", map[string]interface{}{
    "component": "MessageRouter",
    "operation": "route_message",
    "duration_ms": duration,
    "items_processed": count,
})
```

## Field Naming Conventions

Use consistent field names across your codebase:

| Field | Name |
|-------|------|
| User phone | `user_phone` |
| Message ID | `message_id` |
| Session ID | `session_id` |
| Client ID | `client_id` |
| Bill ID | `bill_id` |
| Duration | `duration_ms` |
| Error | `error` |
| Component | `component` |
| Status | `status` |
| Count | `*_count` |

## Best Practices

✅ **DO:**
- Use descriptive messages
- Add component tags
- Include relevant context
- Use consistent field names
- Add IDs for tracing

❌ **DON'T:**
- Put data in message text
- Use inconsistent field names
- Log sensitive data (passwords, tokens)
- Mix nested and flat formats in same context

## Full Documentation

- **`STRUCTURED_LOGGING_GUIDE.md`** - Complete guide
- **`MIGRATION_EXAMPLE.md`** - Migration examples
- **`STRUCTURED_LOGGING_SUMMARY.md`** - Summary and benefits
- **`README.md`** - Full JSON logging docs
