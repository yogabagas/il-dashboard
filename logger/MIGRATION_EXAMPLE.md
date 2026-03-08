# Migration Example - updateStatus.go

## How to Migrate Existing Logs to Structured Format

This guide shows practical examples of migrating logs in `updateStatus.go` to use structured logging.

## Example 1: Worker Message Processing

### Before

```go
logger.Infof("[handleActionWaWebHook] Worker #%d consumed message #%d", workerNo, consumed)
```

### After

```go
import customLogger "gitlab.com/bot3342545/il-dashboard/logger"

customLogger.InfoWithData("Worker consumed message", map[string]interface{}{
    "component": "handleActionWaWebHook",
    "worker_id": workerNo,
    "message_count": consumed,
})
```

### JSON Output

```json
{
  "timestamp": "2024-01-24T12:34:56+07:00",
  "level": "info",
  "message": "Worker consumed message",
  "caller": {
    "file": "updateStatus.go",
    "line": 79,
    "function": "processWebhookMessage"
  },
  "data": {
    "component": "handleActionWaWebHook",
    "worker_id": 1,
    "message_count": 42
  }
}
```

## Example 2: Message Reception

### Before

```go
logger.Infof("[handleActionWaWebHook] Received message from: %s, ID: %s, Type: %s",
    message.From, message.ID, message.Type)
```

### After

```go
customLogger.InfoWithData("Received WhatsApp message", map[string]interface{}{
    "component": "handleActionWaWebHook",
    "from_number": message.From,
    "message_id": message.ID,
    "message_type": message.Type,
})
```

## Example 3: Message Content Logging

### Before

```go
logger.Infof("[handleActionWaWebHook] Message content from %s: %s", contactName, messageContent)
```

### After

```go
customLogger.InfoWithData("Message content received", map[string]interface{}{
    "component": "handleActionWaWebHook",
    "contact_name": contactName,
    "content": messageContent,
    "content_length": len(messageContent),
})
```

## Example 4: Button Click Event

### Before

```go
logger.Infof("[handleActionWaWebHook] Button clicked - ID: %s, Title: %s", buttonID, message.Interactive.ButtonReply.Title)
```

### After

```go
customLogger.InfoWithData("Interactive button clicked", map[string]interface{}{
    "component": "handleActionWaWebHook",
    "button_id": buttonID,
    "button_title": message.Interactive.ButtonReply.Title,
    "from_number": message.From,
    "message_id": message.ID,
})
```

## Example 5: Routing Decision

### Before

```go
logger.Infof("[handleActionWaWebHook] Routing message - Greeting: %v, Active: %v, PDAM: %v, From: %s",
    isGreeting, isConversationActive, isPDAMRelated, message.From)
```

### After

```go
customLogger.InfoWithData("Message routing decision", map[string]interface{}{
    "component": "handleActionWaWebHook",
    "from_number": message.From,
    "is_greeting": isGreeting,
    "is_conversation_active": isConversationActive,
    "is_pdam_related": isPDAMRelated,
    "will_route": isGreeting || isConversationActive,
})
```

## Example 6: Error Logging

### Before

```go
logger.Errorf("[handleActionWaWebHook] Failed to unmarshal request: %v", err)
```

### After

```go
customLogger.ErrorWithData("Failed to unmarshal webhook request", map[string]interface{}{
    "component": "handleActionWaWebHook",
    "error": err.Error(),
    "raw_message_length": len(msg),
})
```

## Example 7: Status Update

### Before

```go
logger.Infof("[handleActionWaWebHook] Message status update - ID: %s, Status: %s, Recipient: %s",
    status.ID, status.Status, status.RecipientID)
```

### After

```go
customLogger.InfoWithData("Message status updated", map[string]interface{}{
    "component": "handleActionWaWebHook",
    "message_id": status.ID,
    "new_status": status.Status,
    "recipient_id": status.RecipientID,
    "has_errors": len(status.Errors) > 0,
})
```

## Example 8: Image Message Processing

### Before

```go
logger.Infof("[handleActionWaWebHook] Image message received from %s - ImageID: %s", message.From, message.Image.ID)
```

### After

```go
customLogger.InfoWithData("Image message received", map[string]interface{}{
    "component": "handleActionWaWebHook",
    "from_number": message.From,
    "image_id": message.Image.ID,
    "mime_type": message.Image.MimeType,
    "has_caption": message.Image.Caption != "",
})
```

## Example 9: Conversation Window Tracking

### Before

```go
logger.Infof("[handleActionWaWebHook] User-initiated conversation window created for %s - 24h free reply window", message.From)
```

### After

```go
customLogger.InfoWithData("User conversation window created", map[string]interface{}{
    "component": "handleActionWaWebHook",
    "user_phone": message.From,
    "client_id": client.ClientId,
    "window_duration_hours": 24,
    "initiated_at": time.Now().Format(time.RFC3339),
})
```

## Example 10: Template Status Update

### Before

```go
logger.Infof("[handleActionWaWebHook] Template status - Name: %s, Event: %s, Category: %s, Language: %s, ID: %d",
    templateStatus.MessageTemplateName,
    templateStatus.Event,
    templateStatus.MessageTemplateCategory,
    templateStatus.MessageTemplateLanguage,
    templateStatus.MessageTemplateID)
```

### After

```go
customLogger.InfoWithData("Template status updated", map[string]interface{}{
    "component": "handleActionWaWebHook",
    "template_id": templateStatus.MessageTemplateID,
    "template_name": templateStatus.MessageTemplateName,
    "event": templateStatus.Event,
    "category": templateStatus.MessageTemplateCategory,
    "language": templateStatus.MessageTemplateLanguage,
    "has_reason": templateStatus.Reason != "",
})
```

## Complete Function Example

Here's a complete example showing how to update a function:

### Before

```go
func (o *WhatsappWebHookImpl) processIncomingMessage(message dtos.WhatsAppMessage, contactName string, metadata dtos.WhatsAppMetadata) {
    logger.Infof("[handleActionWaWebHook] Received message from: %s, ID: %s, Type: %s",
        message.From, message.ID, message.Type)

    messageContent := o.extractMessageContent(message)
    if messageContent == "" {
        logger.Warnf("[handleActionWaWebHook] Empty message content for type: %s", message.Type)
        return
    }

    logger.Infof("[handleActionWaWebHook] Message content from %s: %s", contactName, messageContent)
    
    // ... rest of function
}
```

### After

```go
import customLogger "gitlab.com/bot3342545/il-dashboard/logger"

func (o *WhatsappWebHookImpl) processIncomingMessage(message dtos.WhatsAppMessage, contactName string, metadata dtos.WhatsAppMetadata) {
    customLogger.InfoWithData("Received message", map[string]interface{}{
        "component": "handleActionWaWebHook",
        "from_number": message.From,
        "message_id": message.ID,
        "message_type": message.Type,
        "business_phone": metadata.DisplayPhoneNumber,
    })

    messageContent := o.extractMessageContent(message)
    if messageContent == "" {
        customLogger.WarnWithData("Empty message content", map[string]interface{}{
            "component": "handleActionWaWebHook",
            "message_type": message.Type,
            "message_id": message.ID,
        })
        return
    }

    customLogger.InfoWithData("Message content extracted", map[string]interface{}{
        "component": "handleActionWaWebHook",
        "contact_name": contactName,
        "content_length": len(messageContent),
        "has_media": message.Type != "text",
    })
    
    // ... rest of function
}
```

## Import Statement

Add this to the top of your file:

```go
import (
    // ... existing imports
    customLogger "gitlab.com/bot3342545/il-dashboard/logger"
)
```

## Query Examples

After migration, you can query your logs like this:

### Find all messages from specific user

```bash
cat logs/il-dashboard/app-*.json | jq 'select(.data.from_number=="6281234567890")'
```

### Find all button clicks

```bash
cat logs/il-dashboard/app-*.json | jq 'select(.message=="Interactive button clicked")'
```

### Find routing decisions that resulted in ignoring message

```bash
cat logs/il-dashboard/app-*.json | jq 'select(.message=="Message routing decision" and .data.will_route==false)'
```

### Count messages by type

```bash
cat logs/il-dashboard/app-*.json | jq 'select(.data.message_type) | .data.message_type' | sort | uniq -c
```

### Find errors with component

```bash
cat logs/il-dashboard/app-*.json | jq 'select(.level=="error" and .data.component=="handleActionWaWebHook")'
```

## Migration Checklist

- [ ] Add import for custom logger
- [ ] Identify logs with multiple data points
- [ ] Replace string formatting with `*WithData()` functions
- [ ] Extract data from message into separate fields
- [ ] Use consistent field names
- [ ] Add component/tag for filtering
- [ ] Test log output
- [ ] Verify JSON structure
- [ ] Test queries with jq

## Benefits After Migration

✅ **Clean Messages** - No more long concatenated strings  
✅ **Queryable Data** - Filter by any field  
✅ **Type Safety** - Structured data types  
✅ **Consistent Format** - Same structure across all logs  
✅ **Easy Analysis** - Aggregate and analyze with simple queries  
✅ **Better Debugging** - Find issues faster with precise queries  

## Next Steps

1. Choose a few critical log statements to migrate first
2. Test the new format
3. Gradually migrate more logs
4. Update dashboards/alerts to use new field names
5. Document field naming conventions for your team

Happy logging! 🎉
