-- # Customer Service Hub - Database Migration

-- ## Migration Script

```sql
-- ============================================================================
-- Customer Service Hub - Database Schema
-- Version: 1.0
-- Date: 2025-12-03
-- Description: Tables for AI-to-Human handover customer service system
-- ============================================================================

-- Table 1: Customer Service Cases
-- ============================================================================
CREATE TABLE IF NOT EXISTS customer_service_cases (
    id VARCHAR(26) PRIMARY KEY COMMENT 'ULID',

    -- Client & User Info
    client_id VARCHAR(26) NOT NULL COMMENT 'Client who owns this conversation',
    user_phone VARCHAR(20) NOT NULL COMMENT 'User WhatsApp number',
    user_name VARCHAR(255) DEFAULT NULL COMMENT 'User contact name from WhatsApp',

    -- Case Details
    case_number VARCHAR(50) NOT NULL COMMENT 'Human-readable case number (CS-20251203-0001)',
    category VARCHAR(50) DEFAULT 'pdam_complaint' COMMENT 'pdam_complaint, pdam_inquiry, technical_issue',
    subject TEXT COMMENT 'First user message that triggered escalation',
    description TEXT COMMENT 'Full context of the issue',

    -- Status & Assignment
    status VARCHAR(50) DEFAULT 'new' COMMENT 'new, ai_answered, pending_escalation, assigned, resolved, escalated, closed',
    severity VARCHAR(20) DEFAULT 'normal' COMMENT 'low, normal, high, urgent (set by CS when claiming)',
    priority INT DEFAULT 3 COMMENT '1=Urgent, 2=High, 3=Normal, 4=Low',

    -- CS Assignment
    assigned_to VARCHAR(26) DEFAULT NULL COMMENT 'CS user_id who claimed this case',
    assigned_at TIMESTAMP NULL COMMENT 'When CS claimed the case',

    -- Escalation
    escalated_to VARCHAR(26) DEFAULT NULL COMMENT 'Senior CS/Manager user_id',
    escalated_at TIMESTAMP NULL COMMENT 'When escalated',
    escalation_reason TEXT COMMENT 'Why escalated',

    -- Resolution
    resolved_at TIMESTAMP NULL COMMENT 'When CS marked as resolved',
    resolution_notes TEXT COMMENT 'CS notes about resolution',
    closed_at TIMESTAMP NULL COMMENT 'When case fully closed',

    -- SLA Tracking
    sla_deadline TIMESTAMP NOT NULL COMMENT 'Must respond within 2 hours from created_at',
    sla_breached BOOLEAN DEFAULT FALSE COMMENT 'TRUE if exceeded 2 hours without response',
    first_response_at TIMESTAMP NULL COMMENT 'When CS sent first message',
    first_response_time INT DEFAULT NULL COMMENT 'Seconds from created_at to first_response_at',

    -- Metrics
    total_messages INT DEFAULT 0 COMMENT 'Total messages in this case',
    cs_messages INT DEFAULT 0 COMMENT 'Messages sent by CS',
    user_messages INT DEFAULT 0 COMMENT 'Messages sent by user',
    session_duration INT DEFAULT NULL COMMENT 'Seconds from assigned to closed',

    -- Conversation Tracking
    conversation_session_id VARCHAR(26) COMMENT 'Links to message_conversations',
    last_message_at TIMESTAMP NULL COMMENT 'Last activity timestamp',
    last_message_from VARCHAR(20) COMMENT 'user or cs',

    -- Billing
    billing_cost DECIMAL(10,2) DEFAULT 0.00 COMMENT 'Total cost for this case conversation',
    billing_reference VARCHAR(26) COMMENT 'Link to billing_transactions.id',

    -- Metadata
    tags JSON COMMENT 'Array of tags: ["billing_issue", "water_quality", "meter_problem"]',
    metadata JSON COMMENT 'Additional data: {ai_confidence: 0.45, knowledge_gap: "meter_installation"}',

    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    -- Indexes
    INDEX idx_client_status (client_id, status),
    INDEX idx_assigned_to (assigned_to, status),
    INDEX idx_user_phone (user_phone, client_id),
    INDEX idx_status_created (status, created_at),
    INDEX idx_sla_breach (sla_breached, sla_deadline),
    INDEX idx_case_number (case_number),
    INDEX idx_created_at (created_at),

    -- Foreign Keys
    FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE,
    FOREIGN KEY (assigned_to) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (escalated_to) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Customer service cases for AI-to-human handover';

-- Table 2: Customer Service Messages
-- ============================================================================
CREATE TABLE IF NOT EXISTS customer_service_messages (
    id VARCHAR(26) PRIMARY KEY COMMENT 'ULID',

    -- Case Reference
    case_id VARCHAR(26) NOT NULL COMMENT 'Links to customer_service_cases.id',

    -- Message Info
    sender_type VARCHAR(10) NOT NULL COMMENT 'user or cs',
    sender_id VARCHAR(26) COMMENT 'CS user_id if sender_type=cs, NULL if user',
    sender_name VARCHAR(255) COMMENT 'CS agent name or user name',

    -- Content
    message_type VARCHAR(20) DEFAULT 'text' COMMENT 'text, image, document (future)',
    message_content TEXT NOT NULL COMMENT 'Message text',

    -- WhatsApp Integration
    whatsapp_message_id VARCHAR(100) COMMENT 'WhatsApp wamid if sent via API',
    message_log_id VARCHAR(26) COMMENT 'Links to message_logs.id',

    -- Status
    status VARCHAR(20) DEFAULT 'sent' COMMENT 'pending, sent, delivered, read, failed',
    error_message TEXT COMMENT 'Error details if failed',

    -- Timestamps
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    delivered_at TIMESTAMP NULL,
    read_at TIMESTAMP NULL,

    -- Indexes
    INDEX idx_case_id (case_id, sent_at),
    INDEX idx_whatsapp_msg (whatsapp_message_id),
    INDEX idx_message_log (message_log_id),
    INDEX idx_sent_at (sent_at),

    -- Foreign Keys
    FOREIGN KEY (case_id) REFERENCES customer_service_cases(id) ON DELETE CASCADE,
    FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (message_log_id) REFERENCES message_logs(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Messages exchanged between CS and users';

-- Table 3: Customer Service Assignments (Audit Trail)
-- ============================================================================
CREATE TABLE IF NOT EXISTS customer_service_assignments (
    id VARCHAR(26) PRIMARY KEY COMMENT 'ULID',

    -- Case Reference
    case_id VARCHAR(26) NOT NULL COMMENT 'Links to customer_service_cases.id',

    -- Assignment Details
    action VARCHAR(50) NOT NULL COMMENT 'claimed, released, escalated, reassigned',
    from_user_id VARCHAR(26) COMMENT 'Previous CS (if reassigned)',
    to_user_id VARCHAR(26) COMMENT 'New CS assigned',

    -- Context
    severity_before VARCHAR(20) COMMENT 'Severity before this action',
    severity_after VARCHAR(20) COMMENT 'Severity after this action',
    notes TEXT COMMENT 'Reason for action',

    -- Metadata
    performed_by VARCHAR(26) NOT NULL COMMENT 'User who performed this action',
    performed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Indexes
    INDEX idx_case_id (case_id, performed_at),
    INDEX idx_to_user (to_user_id),
    INDEX idx_performed_at (performed_at),

    -- Foreign Keys
    FOREIGN KEY (case_id) REFERENCES customer_service_cases(id) ON DELETE CASCADE,
    FOREIGN KEY (from_user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (to_user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (performed_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Audit trail for case assignments and transfers';

-- Table 4: Customer Service Notes (Internal CS Notes)
-- ============================================================================
CREATE TABLE IF NOT EXISTS customer_service_notes (
    id VARCHAR(26) PRIMARY KEY COMMENT 'ULID',

    -- Case Reference
    case_id VARCHAR(26) NOT NULL COMMENT 'Links to customer_service_cases.id',

    -- Note Details
    note_type VARCHAR(50) DEFAULT 'general' COMMENT 'general, resolution, escalation, internal',
    note_content TEXT NOT NULL COMMENT 'CS internal note (NOT sent to user)',

    -- Author
    created_by VARCHAR(26) NOT NULL COMMENT 'CS user_id',
    created_by_name VARCHAR(255) COMMENT 'CS agent name',

    -- Visibility
    is_internal BOOLEAN DEFAULT TRUE COMMENT 'TRUE = internal only, FALSE = visible to other CS',

    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    -- Indexes
    INDEX idx_case_id (case_id, created_at),
    INDEX idx_created_by (created_by),
    INDEX idx_created_at (created_at),

    -- Foreign Keys
    FOREIGN KEY (case_id) REFERENCES customer_service_cases(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Internal notes by CS agents (not visible to users)';

-- ============================================================================
-- Initial Data Seeds
-- ============================================================================

-- Add AI Session Pricing (AI handles complaint until closed)
-- Charged ONCE per session when user confirms issue resolved or conversation ends
INSERT INTO pricing (id, client_id, service_type, category, cost, currency, is_active, created_at, updated_at)
VALUES (
    UPPER(CONCAT(
        SUBSTRING(MD5(RAND()) FROM 1 FOR 8),
        SUBSTRING(MD5(RAND()) FROM 1 FOR 4),
        '4', SUBSTRING(MD5(RAND()) FROM 1 FOR 3),
        SUBSTRING(MD5(RAND()) FROM 1 FOR 4),
        SUBSTRING(MD5(RAND()) FROM 1 FOR 12)
    )),
    NULL,                          -- Global pricing
    'whatsapp',                    -- service_type
    'ai_session',                  -- category (NEW!) - For AI-handled complaints
    380.00,                        -- cost in IDR (same as utility, charged once per session)
    'IDR',                         -- currency
    TRUE,                          -- is_active
    NOW(),                         -- created_at
    NOW()                          -- updated_at
)
ON DUPLICATE KEY UPDATE
    cost = 380.00,
    updated_at = NOW();

-- Add CS Conversation Pricing (Human CS handles case until closed)
-- Charged ONCE per case when CS closes the case
INSERT INTO pricing (id, client_id, service_type, category, cost, currency, is_active, created_at, updated_at)
VALUES (
    UPPER(CONCAT(
        SUBSTRING(MD5(RAND()) FROM 1 FOR 8),
        SUBSTRING(MD5(RAND()) FROM 1 FOR 4),
        '4', SUBSTRING(MD5(RAND()) FROM 1 FOR 3),
        SUBSTRING(MD5(RAND()) FROM 1 FOR 4),
        SUBSTRING(MD5(RAND()) FROM 1 FOR 12)
    )),
    NULL,                          -- Global pricing
    'whatsapp',                    -- service_type
    'cs_conversation',             -- category (NEW!) - For CS-handled cases
    500.00,                        -- cost in IDR (higher than AI because human involvement)
    'IDR',                         -- currency
    TRUE,                          -- is_active
    NOW(),                         -- created_at
    NOW()                          -- updated_at
)
ON DUPLICATE KEY UPDATE
    cost = 500.00,
    updated_at = NOW();

-- ============================================================================
-- Verification Queries
-- ============================================================================

-- Check tables created
SHOW TABLES LIKE 'customer_service%';

-- Check pricing entry
SELECT * FROM pricing WHERE category IN ('ai_session', 'cs_conversation');

-- Check indexes
SHOW INDEX FROM customer_service_cases;
SHOW INDEX FROM customer_service_messages;
SHOW INDEX FROM customer_service_assignments;
SHOW INDEX FROM customer_service_notes;

-- ============================================================================
-- Rollback Script (if needed)
-- ============================================================================

-- DROP TABLE IF EXISTS customer_service_notes;
-- DROP TABLE IF EXISTS customer_service_assignments;
-- DROP TABLE IF EXISTS customer_service_messages;
-- DROP TABLE IF EXISTS customer_service_cases;
-- DELETE FROM pricing WHERE category = 'cs_conversation';
```

## Migration Execution Steps

### Step 1: Backup Database
```bash
mysqldump -u username -p database_name > backup_before_cs_migration_$(date +%Y%m%d).sql
```

### Step 2: Execute Migration
```bash
mysql -u username -p database_name < migrations/008_customer_service_hub.sql
```

### Step 3: Verify Migration
```sql
-- Check all tables created
SELECT
    TABLE_NAME,
    TABLE_ROWS,
    CREATE_TIME
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = 'your_database_name'
  AND TABLE_NAME LIKE 'customer_service%';

-- Expected output:
-- customer_service_cases (0 rows)
-- customer_service_messages (0 rows)
-- customer_service_assignments (0 rows)
-- customer_service_notes (0 rows)

-- Check pricing
SELECT * FROM pricing WHERE category = 'cs_conversation';
-- Expected: 1 row with cost = 500.00
```

### Step 4: Test Inserts
```sql
-- Test case insert
INSERT INTO customer_service_cases (
    id, client_id, user_phone, user_name, case_number,
    category, subject, status, severity, sla_deadline, created_at
) VALUES (
    '01HZQ0001TESTCASE00001',
    'CLIENT_TEST',
    '6281234567890',
    'Test User',
    'CS-20251203-0001',
    'pdam_complaint',
    'Test complaint for migration',
    'new',
    'normal',
    DATE_ADD(NOW(), INTERVAL 2 HOUR),
    NOW()
);

-- Verify
SELECT * FROM customer_service_cases WHERE id = '01HZQ0001TESTCASE00001';

-- Cleanup test data
DELETE FROM customer_service_cases WHERE id = '01HZQ0001TESTCASE00001';
```

## Index Performance Testing

```sql
-- Test index on client_id + status
EXPLAIN SELECT * FROM customer_service_cases
WHERE client_id = 'CLIENT_TEST' AND status = 'new';
-- Should use: idx_client_status

-- Test index on assigned_to
EXPLAIN SELECT * FROM customer_service_cases
WHERE assigned_to = 'USER_CS_001' AND status = 'assigned';
-- Should use: idx_assigned_to

-- Test index on sla_breached
EXPLAIN SELECT * FROM customer_service_cases
WHERE sla_breached = TRUE AND sla_deadline < NOW();
-- Should use: idx_sla_breach
```

## Table Relationships

```
┌─────────────────────────────────────────────────────────────┐
│                    customer_service_cases                    │
│  - id (PK)                                                   │
│  - client_id (FK → clients.id)                              │
│  - assigned_to (FK → users.id)                              │
│  - escalated_to (FK → users.id)                             │
│  - conversation_session_id (FK → message_conversations.id)  │
└──────────────┬────────────────────────────────────┬─────────┘
               │                                     │
               │ 1:N                                 │ 1:N
               ↓                                     ↓
┌──────────────────────────┐          ┌──────────────────────────┐
│ customer_service_messages│          │customer_service_assignments│
│  - id (PK)               │          │  - id (PK)               │
│  - case_id (FK)          │          │  - case_id (FK)          │
│  - sender_id (FK)        │          │  - from_user_id (FK)     │
│  - message_log_id (FK)   │          │  - to_user_id (FK)       │
└──────────────────────────┘          │  - performed_by (FK)     │
                                      └──────────────────────────┘
               │
               │ 1:N
               ↓
┌──────────────────────────┐
│ customer_service_notes   │
│  - id (PK)               │
│  - case_id (FK)          │
│  - created_by (FK)       │
└──────────────────────────┘
```

## Storage Estimates

Assumptions:
- 100 cases per day
- Average 10 messages per case
- Average 2 notes per case
- 1 assignment record per case

**Daily Storage:**
- `customer_service_cases`: 100 rows × ~1 KB = 100 KB
- `customer_service_messages`: 1,000 rows × ~500 bytes = 500 KB
- `customer_service_assignments`: 100 rows × ~300 bytes = 30 KB
- `customer_service_notes`: 200 rows × ~400 bytes = 80 KB
- **Total per day**: ~710 KB

**Monthly Storage (30 days):** ~21 MB
**Yearly Storage:** ~252 MB

With indexes: ~400-500 MB per year (reasonable)

## Maintenance

### Archive Old Closed Cases
```sql
-- Archive cases older than 90 days
CREATE TABLE customer_service_cases_archive LIKE customer_service_cases;

INSERT INTO customer_service_cases_archive
SELECT * FROM customer_service_cases
WHERE status = 'closed' AND closed_at < DATE_SUB(NOW(), INTERVAL 90 DAY);

-- Then delete from main table
DELETE FROM customer_service_cases
WHERE status = 'closed' AND closed_at < DATE_SUB(NOW(), INTERVAL 90 DAY);
```

### Optimize Tables
```sql
-- Run monthly
OPTIMIZE TABLE customer_service_cases;
OPTIMIZE TABLE customer_service_messages;
OPTIMIZE TABLE customer_service_assignments;
OPTIMIZE TABLE customer_service_notes;
```

## Migration Checklist

- [ ] Backup database
- [ ] Review migration script
- [ ] Execute migration in dev environment
- [ ] Test all tables created
- [ ] Verify indexes
- [ ] Test insert/update/delete operations
- [ ] Check foreign key constraints
- [ ] Verify pricing entry created
- [ ] Test rollback script (in dev)
- [ ] Execute in staging environment
- [ ] Perform load testing
- [ ] Execute in production (off-peak hours)
- [ ] Monitor database performance
- [ ] Update application configuration
- [ ] Deploy backend code
- [ ] Verify end-to-end functionality

## Notes

- All tables use `utf8mb4` charset for emoji support (WhatsApp messages)
- ULIDs used for all primary keys (sortable, globally unique)
- Soft deletes not implemented (use archive tables instead)
- JSON columns for flexible metadata (MySQL 5.7+)
- Indexes optimized for common queries (dashboard, case lists)
- Foreign keys with `ON DELETE CASCADE` for automatic cleanup
- Timestamps for full audit trail

---

**Migration Version:** 008
**Dependencies:** Tables `clients`, `users`, `message_logs`, `message_conversations`, `pricing` must exist
**Estimated Execution Time:** 2-5 seconds
**Database Size Impact:** Minimal (~1 MB initial, grows with usage)

