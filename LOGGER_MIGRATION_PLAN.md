# Logger Migration Plan - Old Logger Files

## Summary

**Total Go files using old logger:** 42 files  
**Total logger calls to migrate:** ~1,500+ calls

## Migration Status

### ✅ Already Migrated (2 files)
- ✅ `main.go` - 11 logger calls → **DONE**
- ✅ `pubsub/updateStatus.go` - 70 logger calls → **DONE**

### ⏳ Pending Migration (42 files)

Files are organized by **priority** based on number of logger calls (most calls first).

---

## HIGH PRIORITY - Services Layer (Top 10 files, ~800 calls)

| File | Logger Calls | Priority | Notes |
|------|--------------|----------|-------|
| `services/wa_send.go` | 125 | 🔴 Critical | WhatsApp send service |
| `services/ppobService.go` | 108 | 🔴 Critical | PPOB service |
| `services/wa_template.go` | 105 | 🔴 Critical | Template management |
| `services/groqService.go` | 96 | 🔴 Critical | AI/Groq integration |
| `services/campaigns.go` | 91 | 🔴 Critical | Campaign management |
| `services/contacts.go` | 52 | 🟡 High | Contact management |
| `services/contact_groups.go` | 41 | 🟡 High | Contact groups |
| `services/billing.go` | 39 | 🟡 High | Billing service |
| `services/aiKnowledgeCacheService.go` | 30 | 🟡 High | AI cache |
| `services/senders.go` | 30 | 🟡 High | Sender management |

**Subtotal:** ~717 logger calls

---

## MEDIUM PRIORITY - Customer Service (6 files, ~187 calls)

| File | Logger Calls | Priority | Notes |
|------|--------------|----------|-------|
| `customer_service/services/customer_service.go` | 49 | 🟡 High | CS main service |
| `customer_service/pubsub/handlers.go` | 40 | 🟡 High | CS pubsub handlers |
| `customer_service/services/agent_service.go` | 33 | 🟡 High | Agent service |
| `customer_service/repositories/csAgent/csAgent.go` | 25 | 🟠 Medium | CS repository |
| `customer_service/csControllers/customer_service_controller.go` | 20 | 🟠 Medium | CS controller |
| `customer_service/csControllers/agent_controller.go` | 19 | 🟠 Medium | Agent controller |

**Subtotal:** ~186 logger calls

---

## MEDIUM PRIORITY - Workers (3 files, ~84 calls)

| File | Logger Calls | Priority | Notes |
|------|--------------|----------|-------|
| `workers/template_status_checker.go` | 34 | 🟡 High | Template checker worker |
| `workers/campaign_executor.go` | 31 | 🟡 High | Campaign worker |
| `workers/cs_timeout_checker.go` | 19 | 🟠 Medium | CS timeout worker |

**Subtotal:** ~84 logger calls

---

## MEDIUM PRIORITY - PubSub Handlers (6 files, ~109 calls)

| File | Logger Calls | Priority | Notes |
|------|--------------|----------|-------|
| `pubsub/payment/handler.go` | 23 | 🟠 Medium | Payment handler |
| `pubsub/message_router.go` | 22 | 🟠 Medium | Message router |
| `pubsub/message_subscribers.go` | 20 | 🟠 Medium | Message subscribers |
| `pubsub/cs_escalation.go` | 18 | 🟠 Medium | CS escalation |
| `pubsub/image_processor.go` | 14 | 🟠 Medium | Image processor |
| `pubsub/ai/handler.go` | 14 | 🟠 Medium | AI handler |

**Subtotal:** ~111 logger calls

---

## LOW PRIORITY - Other Services (8 files, ~115 calls)

| File | Logger Calls | Priority | Notes |
|------|--------------|----------|-------|
| `services/pricing.go` | 23 | 🟠 Medium | Pricing service |
| `services/message_logs.go` | 22 | 🟠 Medium | Message logs |
| `services/aiKnowledgeBaseService.go` | 22 | 🟠 Medium | AI knowledge base |
| `services/messageConversationService.go` | 13 | 🟢 Low | Conversation service |
| `services/dashboard.go` | 10 | 🟢 Low | Dashboard service |
| `services/auth.go` | 9 | 🟢 Low | Auth service |
| `services/users.go` | 8 | 🟢 Low | User service |
| `services/roles.go` | 8 | 🟢 Low | Role service |

**Subtotal:** ~115 logger calls

---

## LOW PRIORITY - Controllers & Utils (4 files, ~30 calls)

| File | Logger Calls | Priority | Notes |
|------|--------------|----------|-------|
| `utils/whatsapp.go` | 11 | 🟢 Low | WhatsApp utilities |
| `controllers/wa_send.go` | 8 | 🟢 Low | WA send controller |
| `controllers/aiController.go` | 6 | 🟢 Low | AI controller |
| `controllers/index.go` | 3 | 🟢 Low | Index controller |
| `controllers/role.go` | 1 | 🟢 Low | Role controller |
| `controllers/billing.go` | 1 | 🟢 Low | Billing controller |

**Subtotal:** ~30 logger calls

---

## LOW PRIORITY - Repositories (2 files, ~15 calls)

| File | Logger Calls | Priority | Notes |
|------|--------------|----------|-------|
| `repositories/messageConversation/messageConversation.go` | 8 | 🟢 Low | Message repo |
| `repositories/customerServiceCases/customerServiceCases.go` | 7 | 🟢 Low | CS cases repo |

**Subtotal:** ~15 logger calls

---

## LOW PRIORITY - PubSub Specialized (2 files, ~8 calls)

| File | Logger Calls | Priority | Notes |
|------|--------------|----------|-------|
| `pubsub/waimage/handler.go` | 5 | 🟢 Low | WA image handler |
| `pubsub/cs/handler.go` | 3 | 🟢 Low | CS handler |

**Subtotal:** ~8 logger calls

---

## LOW PRIORITY - Logger Setup Files (2 files, ~5 calls)

| File | Logger Calls | Priority | Notes |
|------|--------------|----------|-------|
| `logger/setup.go` | 3 | 🟢 Low | Logger setup (keep old) |
| `logger/hook.go` | 2 | 🟢 Low | Logger hook (keep old) |

**Note:** These files are part of the logger infrastructure and should keep using the old logger for backward compatibility.

---

## 🎯 Recommended Migration Strategy

### Phase 1: Critical Services (Week 1)
Focus on the **top 10 services** that have the most logger calls:
1. `services/wa_send.go` (125 calls)
2. `services/ppobService.go` (108 calls)
3. `services/wa_template.go` (105 calls)
4. `services/groqService.go` (96 calls)
5. `services/campaigns.go` (91 calls)
6. `services/contacts.go` (52 calls)
7. `services/contact_groups.go` (41 calls)
8. `services/billing.go` (39 calls)
9. `services/aiKnowledgeCacheService.go` (30 calls)
10. `services/senders.go` (30 calls)

**Impact:** ~717 logger calls migrated (~48% of total)

### Phase 2: Customer Service & Workers (Week 2)
11. All `customer_service/` files (186 calls)
12. All `workers/` files (84 calls)

**Impact:** +270 logger calls migrated (~66% of total)

### Phase 3: PubSub & Handlers (Week 3)
13. All `pubsub/` files (111 calls)

**Impact:** +111 logger calls migrated (~72% of total)

### Phase 4: Remaining Services (Week 4)
14. All remaining `services/` files (115 calls)
15. All `controllers/` files (30 calls)
16. All `repositories/` files (15 calls)
17. All `utils/` files (11 calls)

**Impact:** +171 logger calls migrated (~90%+ of total)

---

## 📊 Current State

```
Migration Progress: 2/44 files (4.5%)
Logger Calls Migrated: 81/~1,500 (5.4%)

✅ Completed:     2 files  (81 calls)
⏳ Pending:      42 files  (~1,419 calls)
```

---

## 🔍 Quick Start Commands

### Find all old logger imports
```bash
grep -r "github.com/ariandi/gocom/logger" --include="*.go" --exclude-dir=vendor .
```

### Count logger calls in a specific file
```bash
grep -c "logger\." services/wa_send.go
```

### View logger calls in a specific file
```bash
grep -n "logger\." services/wa_send.go | head -20
```

### Find all Info logs
```bash
grep -rn "logger\.Info" services/ --include="*.go"
```

### Find all Error logs
```bash
grep -rn "logger\.Error" services/ --include="*.go"
```

---

## 💡 Migration Tips

### 1. Start with High-Impact Files
Migrate files with the most logger calls first to maximize the impact of structured logging.

### 2. Follow the Pattern from updateStatus.go
Use the same structured logging approach:
- Extract data from message strings
- Use `map[string]interface{}` for structured data
- Keep messages clean and descriptive

### 3. Batch Similar Files
Migrate similar files together (e.g., all services, all workers) to maintain consistency.

### 4. Test After Each File
Run the application after migrating each file to ensure no errors were introduced.

### 5. Use Linter
Run `go vet` and linter checks after each migration to catch type errors early.

---

## 🎯 Next Steps

**Option A: Start with High Priority**
Migrate `services/wa_send.go` (125 calls) - Biggest impact

**Option B: Start with Current Issue**
Migrate `pubsub/message_subscribers.go` (20 calls) - The file causing your current log output

**Option C: Batch Migration**
Migrate all `pubsub/` files together (111 calls) - Complete one module

**Option D: Gradual Approach**
Migrate 1-2 files per day, complete in ~1 month

---

## 🚀 Ready to Start?

Choose which file(s) you want to migrate next, and I'll help you convert them to structured JSON logging!

**Most Common Choice:** Start with `pubsub/message_subscribers.go` since it's causing your current mixed log output.

---

**Last Updated:** January 24, 2026  
**Migration Status:** Phase 0 Complete (2/44 files)
