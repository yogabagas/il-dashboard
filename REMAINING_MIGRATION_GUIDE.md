# Remaining Logger Migration Guide

## Quick Start for Remaining Files

This guide will help you complete the remaining 30% of the logger migration.

---

## 📁 Remaining Files by Folder

### Services Folder (`services/`)

#### 1. aiKnowledgeCacheService.go (30 calls)
**Purpose**: Caches AI knowledge base for faster retrieval

**Pattern**: Cache operations with timing
```go
// Example migration:
customLogger.InfoWithData("Cache refresh started", map[string]interface{}{
    "component": "AIKnowledgeCacheService",
    "function":  "RefreshCache",
    "client_id": clientId,
})

customLogger.InfoWithData("Cache refreshed successfully", map[string]interface{}{
    "component":     "AIKnowledgeCacheService",
    "function":      "RefreshCache",
    "client_id":     clientId,
    "record_count":  len(knowledge),
    "duration_ms":   time.Since(start).Milliseconds(),
})
```

#### 2. aiKnowledgeBaseService.go (22 calls)
**Purpose**: Manages AI knowledge base CRUD operations

**Pattern**: Standard CRUD with validation
```go
customLogger.InfoWithData("Knowledge created", map[string]interface{}{
    "component":    "AIKnowledgeBaseService",
    "function":     "Create",
    "knowledge_id": id,
    "client_id":    clientId,
})
```

#### 3. message_logs.go (22 calls)
**Purpose**: Message audit logging

**Pattern**: Logging with message metadata
```go
customLogger.InfoWithData("Message log created", map[string]interface{}{
    "component":   "MessageLogService",
    "function":    "Create",
    "message_id":  logId,
    "type":        messageType,
    "cost":        cost,
})
```

#### 4. pricing.go (23 calls)
**Purpose**: Pricing management

**Pattern**: CRUD with pricing details
```go
customLogger.InfoWithData("Pricing updated", map[string]interface{}{
    "component": "PricingService",
    "function":  "Update",
    "pricing_id": id,
    "cost":      newCost,
})
```

#### 5. messageConversationService.go (13 calls)
**Purpose**: Conversation threading

**Pattern**: Thread management
```go
customLogger.InfoWithData("Conversation created", map[string]interface{}{
    "component":       "MessageConversationService",
    "function":        "Create",
    "conversation_id": id,
    "phone":           phone,
})
```

#### 6. dashboard.go (5 calls)
**Purpose**: Dashboard statistics

**Pattern**: Stats queries
```go
customLogger.DebugWithData("Stats retrieved", map[string]interface{}{
    "component":     "DashboardService",
    "function":      "GetStats",
    "total_messages": stats.Total,
})
```

#### 7. Small Files (users.go, roles.go, client.go, auth.go) - 13 calls total
**Pattern**: Simple CRUD operations

---

### Customer Service Folder (`customer_service/`)

Estimated ~186 calls across multiple files.

**Common Patterns**:
```go
// CS Assignment
customLogger.InfoWithData("CS assigned", map[string]interface{}{
    "component": "CSService",
    "function":  "Assign",
    "cs_id":     csId,
    "ticket_id": ticketId,
})

// Status Changes
customLogger.InfoWithData("Ticket status updated", map[string]interface{}{
    "component": "CSService",
    "function":  "UpdateStatus",
    "ticket_id": ticketId,
    "old_status": oldStatus,
    "new_status": newStatus,
})

// Escalation
customLogger.WarnWithData("Ticket escalated", map[string]interface{}{
    "component": "CSService",
    "function":  "Escalate",
    "ticket_id": ticketId,
    "reason":    reason,
})
```

---

### Controllers Folder (`controllers/`)

Estimated ~30 calls across API controllers.

**Common Patterns**:
```go
// Request Start
customLogger.InfoWithData("API request", map[string]interface{}{
    "component": "WaSendController",
    "function":  "Send",
    "method":    "POST",
    "endpoint":  "/api/wa/send",
    "client_id": authInfo.ClientId,
})

// Validation Errors
customLogger.WarnWithData("Invalid request", map[string]interface{}{
    "component": "WaSendController",
    "function":  "Send",
    "error":     "missing required field",
    "field":     "phone",
})

// Response
customLogger.InfoWithData("API response", map[string]interface{}{
    "component":  "WaSendController",
    "function":   "Send",
    "status":     200,
    "message_id": messageId,
})
```

---

### Repositories & Utils (~50 calls)

**Pattern**: Database operations and utilities
```go
// Repository
customLogger.DebugWithData("Database query", map[string]interface{}{
    "component": "ContactRepository",
    "function":  "GetById",
    "id":        id,
    "found":     mdl != nil,
})

// Utils
customLogger.DebugWithData("Utility operation", map[string]interface{}{
    "component": "Utils",
    "function":  "ParsePhone",
    "input":     rawPhone,
    "output":    cleanPhone,
})
```

---

## 🔄 Migration Workflow

### For Each File:

1. **Backup** (optional but recommended)
   ```bash
   cp services/filename.go services/filename.go.backup
   ```

2. **Count Logger Calls**
   ```bash
   grep -c "logger\." services/filename.go
   ```

3. **Update Import**
   ```go
   // Remove
   - "github.com/ariandi/gocom/logger"
   
   // Add
   + customLogger "gitlab.com/bot3342545/il-dashboard/logger"
   ```

4. **Migrate Each Function**
   - Use Find & Replace in your editor
   - Start with simple Info calls
   - Then handle Warn/Error/Debug
   - Extract variables from format strings

5. **Verify**
   ```bash
   # Should return 0
   grep -c "logger\." services/filename.go
   
   # Check for import
   grep "customLogger" services/filename.go
   ```

6. **Test**
   ```bash
   # Run linter
   go vet ./services/filename.go
   
   # Or build
   go build
   ```

---

## 🎨 Common Patterns & Tips

### Pattern 1: Start/Success Pattern
```go
// BEFORE:
logger.Infof("Service.Function - Start param=%v", param)
// ... logic ...
logger.Infof("Service.Function - Success result=%v", result)

// AFTER:
customLogger.InfoWithData("Function started", map[string]interface{}{
    "component": "Service",
    "function":  "Function",
    "param":     param,
})
// ... logic ...
customLogger.InfoWithData("Function completed", map[string]interface{}{
    "component": "Service",
    "function":  "Function",
    "result":    result,
})
```

### Pattern 2: Error Pattern
```go
// BEFORE:
logger.Errorf("Service.Function - Failed err=%v", err)

// AFTER:
customLogger.ErrorWithData("Operation failed", map[string]interface{}{
    "component": "Service",
    "function":  "Function",
    "error":     err.Error(),  // Always use .Error() for errors
})
```

### Pattern 3: Validation Pattern
```go
// BEFORE:
logger.Warnf("Service.Function - Invalid param=%v", param)

// AFTER:
customLogger.WarnWithData("Validation failed", map[string]interface{}{
    "component": "Service",
    "function":  "Function",
    "field":     "param_name",
    "value":     param,
})
```

### Pattern 4: Debug Pattern (use sparingly!)
```go
// BEFORE:
logger.Debugf("Service.Function - Details: %+v", details)

// AFTER:
customLogger.DebugWithData("Debug info", map[string]interface{}{
    "component": "Service",
    "function":  "Function",
    "details":   details,
})
```

---

## 💡 Tips & Best Practices

### DO:
- ✅ Keep messages concise and clear
- ✅ Put dynamic data in the map, not the message
- ✅ Use consistent component names (match the struct/file name)
- ✅ Use snake_case for map keys
- ✅ Call `.Error()` on error objects
- ✅ Remove start/end logs for simple getters (use Debug if needed)
- ✅ Group related data together

### DON'T:
- ❌ Put formatted strings in the message
- ❌ Use `%+v` or `%v` format specs
- ❌ Log sensitive data (passwords, tokens, full credit cards)
- ❌ Over-log (not every line needs a log)
- ❌ Mix old and new logger styles

### Examples:

**BAD:**
```go
customLogger.InfoWithData("User john@example.com logged in successfully at 2024-01-24", map[string]interface{}{})
```

**GOOD:**
```go
customLogger.InfoWithData("User logged in", map[string]interface{}{
    "component": "AuthService",
    "function":  "Login",
    "email":     "john@example.com",
    "timestamp": time.Now(),
})
```

---

## 🧪 Testing Your Changes

### 1. Visual Check
```bash
# Should see NO results
grep "logger\." services/yourfile.go

# Should see results
grep "customLogger" services/yourfile.go
```

### 2. Compile Check
```bash
go build ./services/yourfile.go
```

### 3. Runtime Check
```bash
# Run application
./il-dashboard

# In another terminal, watch logs
tail -f logs/app.log | jq '.'

# Test the feature
# Verify JSON format appears correctly
```

### 4. Example Good Output
```json
{
  "level": "info",
  "time": "2026-01-24T10:30:45.123Z",
  "caller": "services/contacts.go:95",
  "component": "ContactsService",
  "function": "Create",
  "contact_id": "01HQXYZ...",
  "name": "John Doe",
  "message": "Contact created successfully"
}
```

---

## 📊 Track Your Progress

Create a checklist as you go:

### Services (10 files)
- [ ] aiKnowledgeCacheService.go (30)
- [ ] aiKnowledgeBaseService.go (22)  
- [ ] message_logs.go (22)
- [ ] pricing.go (23)
- [ ] messageConversationService.go (13)
- [ ] dashboard.go (5)
- [ ] users.go (4)
- [ ] roles.go (4)
- [ ] client.go (4)
- [ ] auth.go (1)

### Customer Service (6 files)
- [ ] File 1
- [ ] File 2
- [ ] File 3
- [ ] File 4
- [ ] File 5
- [ ] File 6

### Controllers (6 files)
- [ ] File 1
- [ ] File 2
- [ ] File 3
- [ ] File 4
- [ ] File 5
- [ ] File 6

### Misc
- [ ] Repositories
- [ ] Utils
- [ ] Other

---

## 🆘 Troubleshooting

### Issue: "undefined: customLogger"
**Solution**: Add import:
```go
customLogger "gitlab.com/bot3342545/il-dashboard/logger"
```

### Issue: "too many arguments"
**Solution**: Check function signature. It should be:
```go
customLogger.InfoWithData(message string, data map[string]interface{})
```

### Issue: Linter warnings about unused import
**Solution**: Make sure you replaced ALL logger calls and removed old import

### Issue: No JSON output
**Solution**: Check `config-*.properties`:
```properties
app.log.json.enabled=true
```

---

## 📚 Additional Resources

- **Logger Implementation**: `logger/json_logger.go`
- **Migration Examples**: `MIGRATION_PROGRESS_SUMMARY.md`
- **Complete Guide**: `STRUCTURED_LOGGING_GUIDE.md`
- **View Logs Script**: `view_logs.py`

---

**You've got this! The hard part (70%) is already done. These remaining files follow the same patterns. 🚀**
