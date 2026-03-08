# JSON Logger for IL-Dashboard

This package provides JSON-formatted logging for the IL-Dashboard application, making logs more readable and easier to parse by log aggregation tools like ELK, Splunk, or Datadog.

## Features

- ✅ **JSON Format**: All logs are output in standard JSON format
- ✅ **Structured Logging**: Automatic extraction of fields, tags, and metadata
- ✅ **Caller Information**: Includes file, line number, and function name
- ✅ **Log Levels**: Support for Debug, Info, Warn, Error, Fatal, and Panic
- ✅ **File Output**: Logs are written to dated JSON files
- ✅ **Zero Code Changes**: Works alongside existing gocom logger
- ✅ **Configurable**: Enable/disable via configuration file

## Configuration

Add this line to your `config.properties` file:

```properties
app.log.json.enabled=true
```

Set to `false` to disable JSON logging and use traditional format only.

## JSON Log Format

Each log entry is formatted as a JSON object:

```json
{
  "timestamp": "2024-01-24T12:34:56+07:00",
  "level": "info",
  "message": "[handleActionWaWebHook] Worker #1 consumed message #42",
  "caller": {
    "file": "updateStatus.go",
    "line": 79,
    "function": "processWebhookMessage"
  },
  "tags": ["handleActionWaWebHook"]
}
```

### Fields

- **timestamp**: ISO 8601 format with timezone
- **level**: Log level (debug, info, warn, error, fatal, panic)
- **message**: The log message content
- **caller**: Source code location
  - **file**: Source file name
  - **line**: Line number
  - **function**: Function name
- **tags**: Extracted tags from log message (optional)
- **custom fields**: Any additional structured data (optional)

## Usage

### Automatic (Recommended)

The JSON logger is automatically initialized in `main.go`:

```go
import customLogger "gitlab.com/bot3342545/il-dashboard/logger"

func main() {
    // Setup both traditional and JSON logging
    if err := customLogger.Setup("logs/il-dashboard"); err != nil {
        logger.Fatalf("Failed to setup logger: %v", err)
    }
    
    // Rest of your code...
}
```

All existing logger calls throughout your codebase will automatically generate JSON logs:

```go
import "github.com/ariandi/gocom/logger"

logger.Info("Server started")
logger.Infof("Listening on port %d", 8080)
logger.Errorf("Database connection failed: %v", err)
```

### Direct JSON Logging (Advanced)

For direct JSON logging with custom fields:

```go
import customLogger "gitlab.com/bot3342545/il-dashboard/logger"

// Simple logging
customLogger.Info("Processing request")
customLogger.Infof("User %s logged in", userID)

// With structured fields
entry := customLogger.WithFields(map[string]interface{}{
    "user_id": "12345",
    "action": "login",
    "ip": "192.168.1.1",
})
if entry != nil {
    entry.Info("User authentication successful")
}
```

## Log File Location

JSON logs are written to:

```
logs/il-dashboard/app-YYYY-MM-DD.json
```

Each day gets a new log file automatically.

## Example Log Output

### Traditional Format (gocom logger)
```
2024-01-24 12:34:56 [INFO] [handleActionWaWebHook] Worker #1 consumed message #42
```

### JSON Format (JSON logger)
```json
{"timestamp":"2024-01-24T12:34:56+07:00","level":"info","message":"[handleActionWaWebHook] Worker #1 consumed message #42","caller":{"file":"updateStatus.go","line":79,"function":"processWebhookMessage"},"tags":["handleActionWaWebHook"]}
```

## Viewing JSON Logs

### Pretty Print in Terminal

```bash
# Using jq
tail -f logs/il-dashboard/app-2024-01-24.json | jq '.'

# Filter by level
cat logs/il-dashboard/app-2024-01-24.json | jq 'select(.level=="error")'

# Extract specific fields
cat logs/il-dashboard/app-2024-01-24.json | jq '{timestamp, level, message}'
```

### Using Python

```bash
# Pretty print
python -m json.tool logs/il-dashboard/app-2024-01-24.json

# Live tail with formatting
tail -f logs/il-dashboard/app-2024-01-24.json | python -m json.tool
```

## Integration with Log Management Tools

### ELK Stack (Elasticsearch, Logstash, Kibana)

Configure Filebeat to read JSON logs:

```yaml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /path/to/logs/il-dashboard/*.json
  json.keys_under_root: true
  json.add_error_key: true
```

### Splunk

Add as a JSON data source:

```
sourcetype = _json
```

### Datadog

Configure the log collection:

```yaml
logs:
  - type: file
    path: /path/to/logs/il-dashboard/*.json
    service: il-dashboard
    source: golang
    sourcecategory: application
```

## Performance

- **Minimal Overhead**: JSON logging runs in parallel with minimal performance impact
- **Async Writing**: Logs are written asynchronously to avoid blocking
- **Efficient Formatting**: Uses optimized JSON serialization

## Troubleshooting

### JSON Logging Not Working

1. Check configuration: `app.log.json.enabled=true`
2. Verify log directory exists and is writable
3. Check application logs for initialization errors

### Log File Not Created

Ensure the log directory has write permissions:

```bash
chmod 755 logs/il-dashboard
```

### Duplicate Logs

This is normal! The system outputs to both:
- Traditional logs (for backward compatibility)
- JSON logs (for modern log management)

You can disable traditional logging if needed by only using JSON logger.

## Best Practices

1. **Use Structured Fields**: Add context with custom fields
   ```go
   customLogger.WithFields(map[string]interface{}{
       "user_id": userID,
       "request_id": reqID,
   }).Info("Processing request")
   ```

2. **Consistent Tagging**: Use consistent tag patterns like `[ComponentName]`
   ```go
   logger.Infof("[PaymentHandler] Processing payment %s", paymentID)
   ```

3. **Include Context**: Add relevant information to help debugging
   ```go
   logger.Infof("[Database] Query took %dms - SELECT * FROM users WHERE id=%s", duration, userID)
   ```

4. **Use Appropriate Levels**:
   - `Debug`: Detailed debugging information
   - `Info`: General informational messages
   - `Warn`: Warning messages for potentially harmful situations
   - `Error`: Error messages for failures
   - `Fatal`: Critical errors that require application termination

## Migration Guide

No code changes required! The JSON logger works alongside your existing code:

1. Add `app.log.json.enabled=true` to config
2. Update `main.go` to use `customLogger.Setup()`
3. Start the application
4. JSON logs will be created automatically

## License

Internal use only - Kilat IL-Dashboard Project
