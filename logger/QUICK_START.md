# Quick Start - JSON Logging

## Installation Complete ✅

The JSON logging system has been installed and configured. Here's how to use it:

## 1. Build the Application

```bash
cd /Users/yogabagas/Project/gitlab/kilat/il-dashboard
go build
```

## 2. Run the Application

```bash
./il-dashboard
```

## 3. Check JSON Logs

The application will create JSON log files in:

```
logs/il-dashboard/app-2024-01-24.json
```

## 4. View Logs (Pretty Format)

### Using jq (Recommended)

```bash
# Real-time log viewing
tail -f logs/il-dashboard/app-*.json | jq '.'

# View specific log file
cat logs/il-dashboard/app-2024-01-24.json | jq '.'

# Filter by log level
cat logs/il-dashboard/app-2024-01-24.json | jq 'select(.level=="error")'

# Filter by message content
cat logs/il-dashboard/app-2024-01-24.json | jq 'select(.message | contains("Worker"))'

# Show only specific fields
cat logs/il-dashboard/app-2024-01-24.json | jq '{timestamp, level, message}'
```

### Without jq

```bash
# Python pretty print
cat logs/il-dashboard/app-2024-01-24.json | python3 -m json.tool

# Simple tail
tail -f logs/il-dashboard/app-*.json
```

## 5. Example JSON Log Output

```json
{
  "caller": {
    "file": "updateStatus.go",
    "function": "processWebhookMessage",
    "line": 79
  },
  "level": "info",
  "message": "[handleActionWaWebHook] Worker #1 consumed message #42",
  "tags": ["handleActionWaWebHook"],
  "timestamp": "2024-01-24T12:34:56+07:00"
}
```

## 6. Configuration

### Enable/Disable JSON Logging

Edit `config-dev.properties` or `config-local.properties`:

```properties
# Enable JSON logging
app.log.json.enabled=true

# Disable JSON logging (use traditional format only)
app.log.json.enabled=false
```

## 7. Testing

### Test if JSON logging is working:

```bash
# Start the application
./il-dashboard

# In another terminal, check if JSON file is created
ls -lh logs/il-dashboard/

# View the latest log entry
tail -1 logs/il-dashboard/app-*.json | jq '.'
```

Expected output:
```json
{
  "level": "info",
  "message": "[STARTUP] Initializing message routing subscribers...",
  "timestamp": "2024-01-24T12:34:56+07:00",
  ...
}
```

## 8. Integration with Log Tools

### Elasticsearch/Kibana

```bash
# Using Filebeat
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /path/to/logs/il-dashboard/*.json
  json.keys_under_root: true
```

### Splunk

```bash
# Add as JSON source
[monitor:///path/to/logs/il-dashboard/*.json]
sourcetype = _json
source = il-dashboard
```

### Datadog

```yaml
logs:
  - type: file
    path: /path/to/logs/il-dashboard/*.json
    service: il-dashboard
    source: golang
```

## 9. Troubleshooting

### Problem: JSON log file not created

**Solution:**
```bash
# Create log directory manually
mkdir -p logs/il-dashboard
chmod 755 logs/il-dashboard

# Check if JSON logging is enabled in config
grep "app.log.json.enabled" config-*.properties
```

### Problem: Empty JSON file

**Solution:**
```bash
# Check if application is running
ps aux | grep il-dashboard

# Check application stdout for errors
./il-dashboard 2>&1 | tee startup.log
```

### Problem: Malformed JSON

**Solution:**
```bash
# Validate JSON syntax
cat logs/il-dashboard/app-*.json | jq empty

# Check last few entries
tail -10 logs/il-dashboard/app-*.json | jq '.'
```

## 10. Performance

JSON logging is designed to have minimal performance impact:

- ✅ Asynchronous writing
- ✅ Buffered I/O
- ✅ Efficient JSON serialization
- ✅ No blocking on main thread

## 11. Advanced Usage

### Custom Fields in Code

```go
import customLogger "gitlab.com/bot3342545/il-dashboard/logger"

// Add custom structured fields
entry := customLogger.WithFields(map[string]interface{}{
    "user_id": "12345",
    "request_id": "req-abc-123",
    "ip_address": "192.168.1.1",
})

if entry != nil {
    entry.Info("User login successful")
}
```

This will produce:
```json
{
  "timestamp": "2024-01-24T12:34:56+07:00",
  "level": "info",
  "message": "User login successful",
  "user_id": "12345",
  "request_id": "req-abc-123",
  "ip_address": "192.168.1.1",
  "caller": {...}
}
```

## 12. Log Rotation

JSON log files are automatically created daily:

```
logs/il-dashboard/app-2024-01-23.json
logs/il-dashboard/app-2024-01-24.json
logs/il-dashboard/app-2024-01-25.json
```

### Manual Cleanup

```bash
# Keep last 7 days
find logs/il-dashboard/ -name "app-*.json" -mtime +7 -delete

# Compress old logs
find logs/il-dashboard/ -name "app-*.json" -mtime +1 -exec gzip {} \;
```

## 13. Monitoring Queries

### Count errors in last hour

```bash
# Using jq
cat logs/il-dashboard/app-*.json | jq 'select(.level=="error")' | wc -l
```

### Find slow operations

```bash
# If duration is logged
cat logs/il-dashboard/app-*.json | jq 'select(.duration_ms > 1000)'
```

### Group by tags

```bash
cat logs/il-dashboard/app-*.json | jq '.tags[]' | sort | uniq -c
```

## Need Help?

1. Check `logger/README.md` for detailed documentation
2. Review example logs in `logs/il-dashboard/`
3. Verify configuration in `config-*.properties`
4. Check application startup logs for JSON logger initialization

## What's Next?

- ✅ JSON logging is enabled and running
- ✅ All existing log statements work without changes
- ✅ Logs are structured and searchable
- ✅ Ready for log aggregation tools

Happy logging! 🎉
