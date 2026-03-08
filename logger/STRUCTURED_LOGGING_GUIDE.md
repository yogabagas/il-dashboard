# Structured Logging Guide

## Overview

The JSON logger now supports **structured logging** with separate fields for messages and data. This makes logs more readable and queryable.

## Two Approaches

### 1. **Nested Data** - Data in separate `data` object
### 2. **Flat Fields** - Data at root level alongside message

## Approach 1: Nested Data (Recommended)

Data is stored in a separate `data` field, keeping the message clean.

### Functions Available

- `InfoWithData(message, data)`
- `ErrorWithData(message, data)`
- `WarnWithData(message, data)`
- `DebugWithData(message, data)`
- `FatalWithData(message, data)`

### Usage Examples

```go
import customLogger "gitlab.com/bot3342545/il-dashboard/logger"

// Example 1: Log user action
customLogger.InfoWithData("User message received", map[string]interface{}{
    "user_phone": "6281234567890",
    "message_id": "wamid.123",
    "message_type": "text",
    "content_length": 45,
})

// Example 2: Log payment processing
customLogger.InfoWithData("Payment initiated", map[string]interface{}{
    "bill_id": "BILL123456",
    "amount": 150000,
    "bank_code": "BCA",
    "user_phone": "6281234567890",
    "session_id": "01HN7G9XYZ",
})

// Example 3: Log error with context
customLogger.ErrorWithData("Database connection failed", map[string]interface{}{
    "error": err.Error(),
    "host": "192.168.192.75",
    "port": 5433,
    "database": "instant_link",
    "retry_count": 3,
})

// Example 4: Log webhook processing
customLogger.InfoWithData("WhatsApp webhook processed", map[string]interface{}{
    "worker_id": 1,
    "message_count": 42,
    "processing_time_ms": 234,
    "from_number": "6281234567890",
    "client_id": "CLIENT001",
})
```

### JSON Output (Nested Data)

```json
{
  "timestamp": "2024-01-24T12:34:56+07:00",
  "level": "info",
  "message": "User message received",
  "caller": {
    "file": "updateStatus.go",
    "line": 172,
    "function": "processIncomingMessage"
  },
  "data": {
    "user_phone": "6281234567890",
    "message_id": "wamid.123",
    "message_type": "text",
    "content_length": 45
  }
}
```

## Approach 2: Flat Fields

Data fields are merged at the root level, alongside the message.

### Functions Available

- `InfoWithFields(message, fields)`
- `ErrorWithFields(message, fields)`
- `WarnWithFields(message, fields)`
- `DebugWithFields(message, fields)`

### Usage Examples

```go
import customLogger "gitlab.com/bot3342545/il-dashboard/logger"

// Example 1: Log with flat fields
customLogger.InfoWithFields("User logged in", map[string]interface{}{
    "user_id": "12345",
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0",
})

// Example 2: Performance logging
customLogger.InfoWithFields("Request completed", map[string]interface{}{
    "method": "POST",
    "path": "/api/messages/send",
    "status_code": 200,
    "duration_ms": 145,
    "request_id": "req-abc-123",
})
```

### JSON Output (Flat Fields)

```json
{
  "timestamp": "2024-01-24T12:34:56+07:00",
  "level": "info",
  "message": "User logged in",
  "caller": {
    "file": "auth.go",
    "line": 89,
    "function": "Login"
  },
  "user_id": "12345",
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0"
}
```

## Real-World Examples

### Example 1: WhatsApp Message Processing

```go
import customLogger "gitlab.com/bot3342545/il-dashboard/logger"

func (o *WhatsappWebHookImpl) processIncomingMessage(message dtos.WhatsAppMessage, contactName string, metadata dtos.WhatsAppMetadata) {
    // Log with structured data
    customLogger.InfoWithData("Processing incoming message", map[string]interface{}{
        "from_number": message.From,
        "message_id": message.ID,
        "message_type": message.Type,
        "contact_name": contactName,
        "business_phone": metadata.DisplayPhoneNumber,
        "timestamp": message.Timestamp,
    })
    
    // ... rest of the processing logic
}
```

**Output:**
```json
{
  "timestamp": "2024-01-24T12:34:57+07:00",
  "level": "info",
  "message": "Processing incoming message",
  "caller": {
    "file": "updateStatus.go",
    "line": 172,
    "function": "processIncomingMessage"
  },
  "data": {
    "from_number": "6281234567890",
    "message_id": "wamid.HBgNNjI4MTMxODIx",
    "message_type": "text",
    "contact_name": "John Doe",
    "business_phone": "6281987654321",
    "timestamp": "1706074497"
  }
}
```

### Example 2: Payment Processing

```go
import customLogger "gitlab.com/bot3342545/il-dashboard/logger"

func (h *Handler) HandlePaymentButtonClick(sessionID, fromNumber, buttonID, clientId string) {
    // Extract bill ID from button
    billID := strings.TrimPrefix(buttonID, "pay_")
    
    customLogger.InfoWithData("Payment button clicked", map[string]interface{}{
        "session_id": sessionID,
        "user_phone": fromNumber,
        "button_id": buttonID,
        "bill_id": billID,
        "client_id": clientId,
    })
    
    // Process payment...
    
    customLogger.InfoWithData("Payment processing complete", map[string]interface{}{
        "bill_id": billID,
        "amount": 150000,
        "status": "pending",
        "va_number": "8808012345678901",
        "bank_code": "BCA",
    })
}
```

### Example 3: Error Logging with Context

```go
import customLogger "gitlab.com/bot3342545/il-dashboard/logger"

func (o *WhatsappWebHookImpl) saveIncomingMessage(message dtos.WhatsAppMessage, content string, contactName string, sessionID string, metadata dtos.WhatsAppMetadata) string {
    // ... save logic
    
    if err != nil {
        customLogger.ErrorWithData("Failed to save message", map[string]interface{}{
            "error": err.Error(),
            "message_id": message.ID,
            "from_number": message.From,
            "session_id": sessionID,
            "client_id": client.ClientId,
            "retry_attempted": false,
        })
        return ""
    }
    
    customLogger.InfoWithData("Message saved successfully", map[string]interface{}{
        "message_id": message.ID,
        "session_id": sessionID,
        "conversation_id": conv.ID,
        "log_id": messageLog.ID,
    })
    
    return conv.ID
}
```

### Example 4: AI Conversation Logging

```go
import customLogger "gitlab.com/bot3342545/il-dashboard/logger"

func (h *Handler) TriggerAIConversation(sessionID, fromNumber, userMessage, businessPhoneNumber string) {
    client := h.resolveClient(businessPhoneNumber)
    
    customLogger.InfoWithData("AI conversation triggered", map[string]interface{}{
        "session_id": sessionID,
        "user_phone": fromNumber,
        "client_id": client.ClientId,
        "message_length": len(userMessage),
        "has_greeting": h.ShouldTriggerAI(userMessage),
    })
    
    // Call AI service
    response, err := groqService.ProcessMessage(userMessage, context)
    
    if err != nil {
        customLogger.ErrorWithData("AI processing failed", map[string]interface{}{
            "error": err.Error(),
            "session_id": sessionID,
            "user_phone": fromNumber,
            "prompt_length": len(userMessage),
        })
        return
    }
    
    customLogger.InfoWithData("AI response generated", map[string]interface{}{
        "session_id": sessionID,
        "response_length": len(response),
        "processing_time_ms": 1234,
        "tokens_used": 150,
    })
}
```

### Example 5: Performance Monitoring

```go
import (
    customLogger "gitlab.com/bot3342545/il-dashboard/logger"
    "time"
)

func (o *WhatsappWebHookImpl) routeTextMessage(message dtos.WhatsAppMessage, contactName string, metadata dtos.WhatsAppMetadata) {
    startTime := time.Now()
    
    // Check conditions
    isGreeting := o.aiHandler.ShouldTriggerAI(message.Text.Body)
    isConversationActive := messageConversation.GetRepo().IsConversationActive(message.From)
    isPDAMRelated := o.aiHandler.IsPDAMRelatedMessage(message.Text.Body)
    
    duration := time.Since(startTime).Milliseconds()
    
    customLogger.InfoWithData("Message routing decision", map[string]interface{}{
        "user_phone": message.From,
        "is_greeting": isGreeting,
        "is_active": isConversationActive,
        "is_pdam_related": isPDAMRelated,
        "decision": isGreeting || isConversationActive,
        "check_duration_ms": duration,
    })
    
    // Route message...
}
```

## Querying Structured Logs

### Query by Data Fields (Nested)

```bash
# Find all payments with specific bill ID
cat logs/il-dashboard/app-*.json | jq 'select(.data.bill_id=="BILL123456")'

# Find all messages from specific user
cat logs/il-dashboard/app-*.json | jq 'select(.data.user_phone=="6281234567890")'

# Find slow operations (> 1 second)
cat logs/il-dashboard/app-*.json | jq 'select(.data.processing_time_ms > 1000)'

# Find errors with retry count
cat logs/il-dashboard/app-*.json | jq 'select(.level=="error" and .data.retry_count > 0)'
```

### Query by Flat Fields

```bash
# Find requests by status code
cat logs/il-dashboard/app-*.json | jq 'select(.status_code==500)'

# Find slow requests
cat logs/il-dashboard/app-*.json | jq 'select(.duration_ms > 500)'

# Find specific user activity
cat logs/il-dashboard/app-*.json | jq 'select(.user_id=="12345")'
```

## Comparison: Before vs After

### Before (String Concatenation)

```go
logger.Infof("[handleActionWaWebHook] Received message from: %s, ID: %s, Type: %s", 
    message.From, message.ID, message.Type)
```

**Output:**
```json
{
  "timestamp": "2024-01-24T12:34:56+07:00",
  "level": "info",
  "message": "[handleActionWaWebHook] Received message from: 6281234567890, ID: wamid.123, Type: text"
}
```

**Problems:**
- Hard to query specific fields
- Need regex to extract data
- Not machine-readable structure

### After (Structured Data)

```go
customLogger.InfoWithData("Received WhatsApp message", map[string]interface{}{
    "component": "handleActionWaWebHook",
    "from_number": message.From,
    "message_id": message.ID,
    "message_type": message.Type,
})
```

**Output:**
```json
{
  "timestamp": "2024-01-24T12:34:56+07:00",
  "level": "info",
  "message": "Received WhatsApp message",
  "caller": {
    "file": "updateStatus.go",
    "line": 172,
    "function": "processIncomingMessage"
  },
  "data": {
    "component": "handleActionWaWebHook",
    "from_number": "6281234567890",
    "message_id": "wamid.123",
    "message_type": "text"
  }
}
```

**Benefits:**
- ✅ Easy field-based queries
- ✅ Clean message text
- ✅ Machine-readable structure
- ✅ Type-safe data

## Best Practices

### 1. Keep Messages Concise

```go
// Good ✅
customLogger.InfoWithData("User authenticated", map[string]interface{}{
    "user_id": userID,
    "method": "jwt",
})

// Bad ❌
customLogger.InfoWithData("User 12345 authenticated using JWT method", map[string]interface{}{
    // Data already in message
})
```

### 2. Use Consistent Field Names

```go
// Good ✅ - Consistent naming
customLogger.InfoWithData("Message received", map[string]interface{}{
    "user_phone": "6281234567890",
})

customLogger.InfoWithData("Message sent", map[string]interface{}{
    "user_phone": "6281234567890",
})

// Bad ❌ - Inconsistent
customLogger.InfoWithData("Message received", map[string]interface{}{
    "user_phone": "6281234567890",
})

customLogger.InfoWithData("Message sent", map[string]interface{}{
    "phone": "6281234567890",  // Different field name
})
```

### 3. Include Contextual Data

```go
// Good ✅
customLogger.ErrorWithData("Payment failed", map[string]interface{}{
    "error": err.Error(),
    "bill_id": billID,
    "user_phone": userPhone,
    "amount": amount,
    "retry_count": retryCount,
    "session_id": sessionID,
})

// Bad ❌ - Missing context
customLogger.ErrorWithData("Payment failed", map[string]interface{}{
    "error": err.Error(),
})
```

### 4. Use Meaningful Messages

```go
// Good ✅
customLogger.InfoWithData("Payment initiated", data)
customLogger.InfoWithData("Bank selection completed", data)
customLogger.InfoWithData("VA number generated", data)

// Bad ❌
customLogger.InfoWithData("Step 1", data)
customLogger.InfoWithData("Process", data)
customLogger.InfoWithData("Done", data)
```

## Migration Guide

### Step 1: Identify Logs to Migrate

Look for logs with multiple pieces of data:

```go
// Old style
logger.Infof("[PaymentHandler] Processing bill %s for user %s, amount: %d", 
    billID, userPhone, amount)
```

### Step 2: Convert to Structured Format

```go
// New style
import customLogger "gitlab.com/bot3342545/il-dashboard/logger"

customLogger.InfoWithData("Processing payment", map[string]interface{}{
    "component": "PaymentHandler",
    "bill_id": billID,
    "user_phone": userPhone,
    "amount": amount,
})
```

### Step 3: Add More Context

```go
// Enhanced with more data
customLogger.InfoWithData("Processing payment", map[string]interface{}{
    "component": "PaymentHandler",
    "bill_id": billID,
    "user_phone": userPhone,
    "amount": amount,
    "bank_code": bankCode,
    "session_id": sessionID,
    "client_id": clientID,
})
```

## Summary

- ✅ **Use `*WithData()` functions** for clean message + structured data
- ✅ **Use `*WithFields()` functions** for flat field structure
- ✅ **Keep messages concise** - let data tell the details
- ✅ **Be consistent** with field names across your codebase
- ✅ **Add context** - include IDs, session info, error details
- ✅ **Easy querying** - field-based search with jq

Your logs are now **production-ready and query-friendly**! 🎉
