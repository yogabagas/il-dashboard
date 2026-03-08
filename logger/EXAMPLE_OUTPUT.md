# JSON Logger - Example Output

## Before (Traditional Format)

```
2024-01-24 12:34:56 [INFO] [STARTUP] Initializing message routing subscribers...
2024-01-24 12:34:56 [INFO] [STARTUP] Message routing subscribers initialized successfully
2024-01-24 12:34:56 [INFO] [handleActionWaWebHook] Worker #1 started subscribing to topic: wa_webhook_worker
2024-01-24 12:34:57 [INFO] [handleActionWaWebHook] Worker #1 consumed message #1
2024-01-24 12:34:57 [INFO] [handleActionWaWebHook] Received message from: 6281234567890, ID: wamid.123, Type: text
2024-01-24 12:34:57 [INFO] [handleActionWaWebHook] Message content from John Doe: Halo
2024-01-24 12:34:57 [INFO] [handleActionWaWebHook] Routing message - Greeting: true, Active: false, PDAM: false, From: 6281234567890
2024-01-24 12:34:58 [ERROR] [PaymentHandler] Payment validation failed: insufficient balance
```

## After (JSON Format)

```json
{"timestamp":"2024-01-24T12:34:56+07:00","level":"info","message":"[STARTUP] Initializing message routing subscribers...","caller":{"file":"main.go","line":31,"function":"main"},"tags":["STARTUP"]}
{"timestamp":"2024-01-24T12:34:56+07:00","level":"info","message":"[STARTUP] Message routing subscribers initialized successfully","caller":{"file":"main.go","line":33,"function":"main"},"tags":["STARTUP"]}
{"timestamp":"2024-01-24T12:34:56+07:00","level":"info","message":"[handleActionWaWebHook] Worker #1 started subscribing to topic: wa_webhook_worker","caller":{"file":"updateStatus.go","line":70,"function":"handleActionWaWebHook"},"tags":["handleActionWaWebHook"]}
{"timestamp":"2024-01-24T12:34:57+07:00","level":"info","message":"[handleActionWaWebHook] Worker #1 consumed message #1","caller":{"file":"updateStatus.go","line":79,"function":"processWebhookMessage"},"tags":["handleActionWaWebHook"]}
{"timestamp":"2024-01-24T12:34:57+07:00","level":"info","message":"[handleActionWaWebHook] Received message from: 6281234567890, ID: wamid.123, Type: text","caller":{"file":"updateStatus.go","line":172,"function":"processIncomingMessage"},"tags":["handleActionWaWebHook"]}
{"timestamp":"2024-01-24T12:34:57+07:00","level":"info","message":"[handleActionWaWebHook] Message content from John Doe: Halo","caller":{"file":"updateStatus.go","line":177,"function":"processIncomingMessage"},"tags":["handleActionWaWebHook"]}
{"timestamp":"2024-01-24T12:34:57+07:00","level":"info","message":"[handleActionWaWebHook] Routing message - Greeting: true, Active: false, PDAM: false, From: 6281234567890","caller":{"file":"updateStatus.go","line":348,"function":"routeTextMessage"},"tags":["handleActionWaWebHook"]}
{"timestamp":"2024-01-24T12:34:58+07:00","level":"error","message":"[PaymentHandler] Payment validation failed: insufficient balance","caller":{"file":"handler.go","line":145,"function":"HandlePayment"},"tags":["PaymentHandler"]}
```

## Formatted (Pretty Print with jq)

```json
{
  "timestamp": "2024-01-24T12:34:56+07:00",
  "level": "info",
  "message": "[STARTUP] Initializing message routing subscribers...",
  "caller": {
    "file": "main.go",
    "line": 31,
    "function": "main"
  },
  "tags": [
    "STARTUP"
  ]
}

{
  "timestamp": "2024-01-24T12:34:57+07:00",
  "level": "info",
  "message": "[handleActionWaWebHook] Worker #1 consumed message #1",
  "caller": {
    "file": "updateStatus.go",
    "line": 79,
    "function": "processWebhookMessage"
  },
  "tags": [
    "handleActionWaWebHook"
  ]
}

{
  "timestamp": "2024-01-24T12:34:57+07:00",
  "level": "info",
  "message": "[handleActionWaWebHook] Received message from: 6281234567890, ID: wamid.123, Type: text",
  "caller": {
    "file": "updateStatus.go",
    "line": 172,
    "function": "processIncomingMessage"
  },
  "tags": [
    "handleActionWaWebHook"
  ]
}

{
  "timestamp": "2024-01-24T12:34:58+07:00",
  "level": "error",
  "message": "[PaymentHandler] Payment validation failed: insufficient balance",
  "caller": {
    "file": "handler.go",
    "line": 145,
    "function": "HandlePayment"
  },
  "tags": [
    "PaymentHandler"
  ]
}
```

## With Custom Fields

```go
// Code
entry := customLogger.WithFields(map[string]interface{}{
    "user_phone": "6281234567890",
    "session_id": "01HN7G9XYZ",
    "client_id": "CLIENT001",
    "request_duration_ms": 234,
})
if entry != nil {
    entry.Info("User message processed successfully")
}
```

```json
{
  "timestamp": "2024-01-24T12:34:58+07:00",
  "level": "info",
  "message": "User message processed successfully",
  "user_phone": "6281234567890",
  "session_id": "01HN7G9XYZ",
  "client_id": "CLIENT001",
  "request_duration_ms": 234,
  "caller": {
    "file": "message_router.go",
    "line": 89,
    "function": "RouteMessage"
  }
}
```

## Benefits

### 1. Easy Filtering

**Find all errors:**
```bash
cat logs/il-dashboard/app-*.json | jq 'select(.level=="error")'
```

**Find specific component logs:**
```bash
cat logs/il-dashboard/app-*.json | jq 'select(.tags[] | contains("PaymentHandler"))'
```

**Find logs from specific file:**
```bash
cat logs/il-dashboard/app-*.json | jq 'select(.caller.file=="updateStatus.go")'
```

### 2. Structured Analysis

**Count logs by level:**
```bash
cat logs/il-dashboard/app-*.json | jq '.level' | sort | uniq -c
```

Output:
```
  1234 "debug"
  5678 "info"
   123 "warn"
    45 "error"
```

**Find slow operations:**
```bash
cat logs/il-dashboard/app-*.json | jq 'select(.request_duration_ms > 1000)'
```

### 3. Time-based Queries

**Logs in specific time range:**
```bash
cat logs/il-dashboard/app-*.json | jq 'select(.timestamp >= "2024-01-24T12:00:00" and .timestamp <= "2024-01-24T13:00:00")'
```

### 4. Integration with Tools

**Import to Elasticsearch:**
```bash
cat logs/il-dashboard/app-*.json | curl -X POST "localhost:9200/logs/_bulk" -H 'Content-Type: application/x-ndjson' --data-binary @-
```

**Send to Datadog:**
```bash
cat logs/il-dashboard/app-*.json | datadog-agent logs send --service il-dashboard
```

## Real-World Examples

### Example 1: WhatsApp Webhook Processing

```json
{
  "timestamp": "2024-01-24T12:34:57+07:00",
  "level": "info",
  "message": "[handleActionWaWebHook] Received message from: 6281234567890, ID: wamid.HBgNNjI4MTMxODIxODA2OBUCABEYIDNGQTdBOEU3MkY1QzUxMjA5NjhEODhDMUM2MkMxRTI3AA==, Type: text",
  "caller": {
    "file": "updateStatus.go",
    "line": 172,
    "function": "processIncomingMessage"
  },
  "tags": ["handleActionWaWebHook"]
}
```

### Example 2: Payment Processing

```json
{
  "timestamp": "2024-01-24T12:35:12+07:00",
  "level": "info",
  "message": "[PaymentHandler] Processing payment for bill ID: BILL123456",
  "caller": {
    "file": "handler.go",
    "line": 98,
    "function": "HandleBankSelection"
  },
  "tags": ["PaymentHandler"]
}
```

### Example 3: AI Conversation

```json
{
  "timestamp": "2024-01-24T12:35:45+07:00",
  "level": "info",
  "message": "[TriggerAIConversation] Starting AI conversation for user 6281234567890",
  "caller": {
    "file": "handler.go",
    "line": 67,
    "function": "TriggerAIConversation"
  },
  "tags": ["TriggerAIConversation"]
}
```

### Example 4: Error with Context

```json
{
  "timestamp": "2024-01-24T12:36:23+07:00",
  "level": "error",
  "message": "[Database] Failed to save message: connection timeout",
  "caller": {
    "file": "updateStatus.go",
    "line": 442,
    "function": "saveIncomingMessage"
  },
  "tags": ["Database"],
  "error_details": "dial tcp 192.168.192.75:5433: i/o timeout"
}
```

## Comparison: Traditional vs JSON

| Aspect | Traditional | JSON |
|--------|-------------|------|
| **Readability** | Human-friendly | Machine & Human |
| **Parsing** | Regex needed | Native JSON |
| **Filtering** | grep/awk | jq/native |
| **Structure** | Flat text | Nested objects |
| **Metadata** | Limited | Rich (caller, tags, custom fields) |
| **Tool Integration** | Manual | Native support |
| **Search** | Text-based | Field-based |
| **Analytics** | Complex | Simple |

## Conclusion

JSON logging provides:
- ✅ **Better searchability** - Find logs by any field
- ✅ **Richer context** - Caller info, tags, custom fields
- ✅ **Tool compatibility** - Works with modern log platforms
- ✅ **Easy analysis** - Query and aggregate with jq
- ✅ **No code changes** - Works with existing logger calls

The structured format makes debugging and monitoring significantly easier, especially in production environments!
