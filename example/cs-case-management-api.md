# Customer Service Case Management API Documentation

Base URL: `/api/v1/cs`

**Authorization Required:** All endpoints require authentication

**Roles & Permissions:**
- **ADMIN** users can access and manage all cases across all clients
- **OWNER** users can access and manage cases for their own client (based on `authInfo.ClientId`)
- **STAFF** users (CS Agents) can access and manage cases for their client (claim, send messages, resolve, close, etc.)

---

## 1. Get Cases List

**Endpoint:** `GET /api/v1/cs/cases`

**Headers:**
```
Authorization: Bearer <your-token>
```

**Query Parameters:**
- `pageNo` (optional): Page number (default: 1)
- `rowPerPage` (optional): Rows per page (default: 20)
- `status` (optional): Filter by status (new, assigned, resolved, escalated, closed)
- `severity` (optional): Filter by severity (low, normal, high, urgent)
- `category` (optional): Filter by category (initial_question, gangguan, inquiry_payment, campaign, other)
- `assigned_to` (optional): Filter by assigned CS agent user_id
- `sla_breached` (optional): Filter by SLA breach status (true/false)
- `sort` (optional): Sort field (created_at, severity, sla_deadline)
- `sort_order` (optional): Sort order (asc, desc)

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/cs/cases?pageNo=1&rowPerPage=20&status=new&sort=created_at&sort_order=desc" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": [
    {
      "id": "01JMQV8N9PQRST4UVWXYZ",
      "case_number": "CS-20251204-001",
      "client_id": "01JPETM6DBJ1TPSKZNKNSE5HB6",
      "user_phone": "628123456789",
      "user_name": "John Doe",
      "category": "gangguan",
      "subject": "Air mati di area saya",
      "description": "Sudah 3 hari air tidak mengalir",
      "status": "new",
      "severity": "high",
      "priority": 2,
      "assigned_to": null,
      "assigned_to_name": null,
      "assigned_at": null,
      "sla_deadline": "2025-12-04T14:30:00Z",
      "sla_breached": false,
      "first_response_at": null,
      "first_response_time": null,
      "resolved_at": null,
      "closed_at": null,
      "total_messages": 0,
      "cs_messages": 0,
      "user_messages": 0,
      "last_message_at": null,
      "last_message_from": null,
      "wait_time_seconds": 3600,
      "wait_time_formatted": "1h 0m",
      "tags": [],
      "metadata": {},
      "billing_cost": 0,
      "created_at": "2025-12-04T10:30:00Z",
      "updated_at": "2025-12-04T10:30:00Z"
    }
  ],
  "pageNo": 1,
  "haveNext": false,
  "total": 1
}
```

---

## 2. Get Case Details

**Endpoint:** `GET /api/v1/cs/cases/:id`

**Headers:**
```
Authorization: Bearer <your-token>
```

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/cs/cases/01JMQV8N9PQRST4UVWXYZ" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "case": {
      "id": "01JMQV8N9PQRST4UVWXYZ",
      "case_number": "CS-20251204-001",
      "client_id": "01JPETM6DBJ1TPSKZNKNSE5HB6",
      "user_phone": "628123456789",
      "user_name": "John Doe",
      "category": "gangguan",
      "subject": "Air mati di area saya",
      "description": "Sudah 3 hari air tidak mengalir",
      "status": "assigned",
      "severity": "high",
      "priority": 2,
      "assigned_to": "01JMQV4M5XYZ123ABC",
      "assigned_to_name": "Budi Santoso",
      "assigned_at": "2025-12-04T11:00:00Z",
      "sla_deadline": "2025-12-04T14:30:00Z",
      "sla_breached": false,
      "created_at": "2025-12-04T10:30:00Z",
      "updated_at": "2025-12-04T11:00:00Z"
    },
    "conversation_history": [
      {
        "id": "msg-001",
        "case_id": "01JMQV8N9PQRST4UVWXYZ",
        "sender_type": "user",
        "sender_id": null,
        "sender_name": "John Doe",
        "message_type": "text",
        "message_content": "Air sudah 3 hari tidak mengalir",
        "whatsapp_message_id": "wamid.xxx",
        "status": "delivered",
        "sent_at": "2025-12-04T10:30:00Z",
        "delivered_at": "2025-12-04T10:30:01Z",
        "read_at": null
      },
      {
        "id": "msg-002",
        "case_id": "01JMQV8N9PQRST4UVWXYZ",
        "sender_type": "cs",
        "sender_id": "01JMQV4M5XYZ123ABC",
        "sender_name": "Budi Santoso",
        "message_type": "text",
        "message_content": "Baik, kami akan segera cek area Anda",
        "whatsapp_message_id": "wamid.yyy",
        "status": "read",
        "sent_at": "2025-12-04T11:05:00Z",
        "delivered_at": "2025-12-04T11:05:01Z",
        "read_at": "2025-12-04T11:06:00Z"
      }
    ],
    "notes": [
      {
        "id": "note-001",
        "case_id": "01JMQV8N9PQRST4UVWXYZ",
        "note_type": "general",
        "note_content": "Sudah koordinasi dengan tim lapangan",
        "created_by": "01JMQV4M5XYZ123ABC",
        "created_by_name": "Budi Santoso",
        "is_internal": true,
        "created_at": "2025-12-04T11:10:00Z",
        "updated_at": "2025-12-04T11:10:00Z"
      }
    ],
    "assignments": [
      {
        "id": "assign-001",
        "case_id": "01JMQV8N9PQRST4UVWXYZ",
        "action": "claimed",
        "from_user_id": null,
        "from_user_name": null,
        "to_user_id": "01JMQV4M5XYZ123ABC",
        "to_user_name": "Budi Santoso",
        "severity_before": null,
        "severity_after": "high",
        "notes": "Case claimed",
        "performed_by": "01JMQV4M5XYZ123ABC",
        "performed_by_name": "Budi Santoso",
        "performed_at": "2025-12-04T11:00:00Z"
      }
    ]
  }
}
```

**Error Response (404 Not Found):**
```json
{
  "code": 404,
  "message": "Case not found"
}
```

---

## 3. Claim Case

**Endpoint:** `POST /api/v1/cs/cases/:id/claim`

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "severity": "high",
  "priority": 2,
  "notes": "Taking this case"
}
```

**Field Descriptions:**
- `severity` (optional): Set severity (low, normal, high, urgent)
- `priority` (optional): Set priority (1=Urgent, 2=High, 3=Normal, 4=Low)
- `notes` (optional): Claim notes

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/cs/cases/01JMQV8N9PQRST4UVWXYZ/claim" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "severity": "high",
    "priority": 2,
    "notes": "Taking this case"
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "success": true,
    "case_id": "01JMQV8N9PQRST4UVWXYZ"
  }
}
```

**Error Response (400 Bad Request) - Case already assigned:**
```json
{
  "code": 400,
  "message": "Case is already assigned to another agent"
}
```

---

## 4. Send Message to User

**Endpoint:** `POST /api/v1/cs/cases/:id/messages`

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "message_content": "Baik, kami akan segera cek area Anda"
}
```

**Field Descriptions:**
- `message_content` (required): Message text to send to user via WhatsApp

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/cs/cases/01JMQV8N9PQRST4UVWXYZ/messages" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "message_content": "Baik, kami akan segera cek area Anda"
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "success": true,
    "case_id": "01JMQV8N9PQRST4UVWXYZ"
  }
}
```

---

## 5. Add Note

**Endpoint:** `POST /api/v1/cs/cases/:id/notes`

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "note_type": "general",
  "note_content": "Sudah koordinasi dengan tim lapangan",
  "is_internal": true
}
```

**Field Descriptions:**
- `note_type` (optional): Note type (general, resolution, escalation, internal)
- `note_content` (required): Note content
- `is_internal` (optional): Whether note is internal (not visible to user)

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/cs/cases/01JMQV8N9PQRST4UVWXYZ/notes" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "note_type": "general",
    "note_content": "Sudah koordinasi dengan tim lapangan",
    "is_internal": true
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "success": true,
    "case_id": "01JMQV8N9PQRST4UVWXYZ"
  }
}
```

---

## 6. Resolve Case

**Endpoint:** `POST /api/v1/cs/cases/:id/resolve`

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "resolution_notes": "Masalah sudah diselesaikan. Air sudah mengalir kembali."
}
```

**Field Descriptions:**
- `resolution_notes` (required): Resolution notes describing how the issue was resolved

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/cs/cases/01JMQV8N9PQRST4UVWXYZ/resolve" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "resolution_notes": "Masalah sudah diselesaikan. Air sudah mengalir kembali."
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "success": true,
    "case_id": "01JMQV8N9PQRST4UVWXYZ"
  }
}
```

**Error Response (400 Bad Request) - Case not assigned:**
```json
{
  "code": 400,
  "message": "Cannot resolve unassigned case"
}
```

---

## 7. Close Case

**Endpoint:** `POST /api/v1/cs/cases/:id/close`

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "close_notes": "Case closed. User confirmed issue resolved."
}
```

**Field Descriptions:**
- `close_notes` (optional): Notes for closing the case

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/cs/cases/01JMQV8N9PQRST4UVWXYZ/close" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "close_notes": "Case closed. User confirmed issue resolved."
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "success": true,
    "case_id": "01JMQV8N9PQRST4UVWXYZ",
    "message": "Case closed and billing triggered"
  }
}
```

**Note:** Closing a case will trigger billing calculation for CS session time.

---

## 8. Escalate Case

**Endpoint:** `POST /api/v1/cs/cases/:id/escalate`

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "escalate_to": "01JMQV4M5XYZ456DEF",
  "escalation_reason": "Requires senior CS expertise",
  "severity": "urgent",
  "priority": 1
}
```

**Field Descriptions:**
- `escalate_to` (required): User ID of senior CS agent to escalate to
- `escalation_reason` (required): Reason for escalation
- `severity` (optional): Update severity level
- `priority` (optional): Update priority

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/cs/cases/01JMQV8N9PQRST4UVWXYZ/escalate" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "escalate_to": "01JMQV4M5XYZ456DEF",
    "escalation_reason": "Requires senior CS expertise",
    "severity": "urgent",
    "priority": 1
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "success": true,
    "case_id": "01JMQV8N9PQRST4UVWXYZ",
    "escalated_to": "01JMQV4M5XYZ456DEF"
  }
}
```

---

## 9. Get Statistics

**Endpoint:** `GET /api/v1/cs/statistics`

**Headers:**
```
Authorization: Bearer <your-token>
```

**Query Parameters:**
- `period` (optional): Time period (today, week, month) - default: today

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/cs/statistics?period=today" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "overview": {
      "total_cases": 125,
      "new_cases": 15,
      "assigned_cases": 45,
      "resolved_cases": 50,
      "closed_cases": 10,
      "escalated_cases": 5,
      "sla_breached": 3
    },
    "performance": {
      "average_first_response_time": 180,
      "average_resolution_time": 3600,
      "average_session_duration": 1200,
      "cases_per_cs": 5.2,
      "messages_per_case": 8.5
    },
    "top_cs_agents": [
      {
        "user_id": "01JMQV4M5XYZ123ABC",
        "name": "Budi Santoso",
        "cases_handled": 25,
        "cases_resolved": 23,
        "average_response_time": 120,
        "customer_satisfaction": 4.8
      }
    ],
    "breakdown": {
      "by_severity": {
        "low": 30,
        "normal": 50,
        "high": 35,
        "urgent": 10
      },
      "by_category": {
        "gangguan": 45,
        "inquiry_payment": 30,
        "initial_question": 25,
        "campaign": 15,
        "other": 10
      },
      "by_status": {
        "new": 15,
        "assigned": 45,
        "resolved": 50,
        "escalated": 5,
        "closed": 10
      }
    }
  }
}
```

---

## Case Lifecycle

```
1. NEW (AI creates case)
   ↓
2. ASSIGNED (CS agent claims)
   ↓
3. IN_PROGRESS (CS agent sends messages)
   ↓
4. RESOLVED (CS agent marks as resolved)
   ↓
5. CLOSED (Case closed, billing triggered)

Alternative flows:
- ESCALATED (Escalate to senior CS)
- PENDING_CLOSE (Waiting user confirmation)
```

---

## Use Cases

### 1. CS Agent Claiming New Case
```bash
# Step 1: Get available cases
GET /api/v1/cs/cases?status=new&sort=sla_deadline&sort_order=asc

# Step 2: Claim case
POST /api/v1/cs/cases/:id/claim
{
  "severity": "high",
  "priority": 2
}

# Step 3: Get case details
GET /api/v1/cs/cases/:id
```

### 2. CS Agent Handling Case
```bash
# Step 1: Send message to user
POST /api/v1/cs/cases/:id/messages
{
  "message_content": "Kami akan cek area Anda"
}

# Step 2: Add internal note
POST /api/v1/cs/cases/:id/notes
{
  "note_content": "Koordinasi dengan tim lapangan",
  "is_internal": true
}

# Step 3: Resolve case
POST /api/v1/cs/cases/:id/resolve
{
  "resolution_notes": "Masalah sudah diselesaikan"
}

# Step 4: Close case
POST /api/v1/cs/cases/:id/close
{
  "close_notes": "User confirmed resolved"
}
```

### 3. Dashboard Statistics
```bash
# Get today's statistics
GET /api/v1/cs/statistics?period=today

# Get weekly statistics
GET /api/v1/cs/statistics?period=week

# Get monthly statistics
GET /api/v1/cs/statistics?period=month
```

---

## Integration with AI

### AI → CS Escalation Flow

When AI escalates to CS:
1. AI publishes event to `ai.escalation.confirmed` topic
2. CS Hub creates case with status `new`
3. Auto-assignment finds available CS agent (if enabled)
4. CS agent receives notification
5. CS agent claims case and starts conversation

**Event Structure:**
```json
{
  "session_id": "session_123",
  "client_id": "client_456",
  "user_phone": "628123456789",
  "user_name": "John Doe",
  "user_message": "Air mati sudah 3 hari",
  "confirm_message": "ya",
  "category": "gangguan",
  "metadata": {
    "reason": "Complex issue requires human assistance",
    "auto_escalate": true
  },
  "confirmed_at": "2025-12-04T10:30:00Z"
}
```

---

## SLA Management

**SLA Deadlines:**
- **Urgent**: 2 hours
- **High**: 4 hours
- **Normal**: 8 hours
- **Low**: 24 hours

**SLA Breach Detection:**
```bash
# Get SLA breached cases
GET /api/v1/cs/cases?sla_breached=true&sort=sla_deadline&sort_order=asc
```

**SLA Calculation:**
- Starts when case is created
- Deadline based on severity level
- `sla_breached` flag set automatically when deadline passed
- Tracked in case statistics

---

## Billing

Closing a case triggers billing calculation:
- Session duration = time from case creation to case close
- Billing cost calculated based on client pricing
- `billing_cost` and `billing_reference` updated in case
- Billing event published for accounting system
