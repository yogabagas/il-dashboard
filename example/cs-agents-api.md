# CS Agent Management API Documentation

Base URL: `/api/v1/cs/agents`

**Authorization Required:** All endpoints require authentication

**Roles & Permissions:**
- **ADMIN** users can manage CS agents for any client
- **OWNER** users can manage CS agents for their own client (based on `authInfo.ClientId`)
- **STAFF** (CS Agents) can update their own availability and view available agents

---

## 1. Create CS Agent

**Endpoint:** `POST /api/v1/cs/agents/create`

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "user_id": "01JMQV4M5XYZ123ABC",
  "client_id": "01JPETM6DBJ1TPSKZNKNSE5HB6",
  "name": "Budi Santoso",
  "email": "budi@example.com",
  "phone": "628123456789",
  "max_concurrent_cases": 10,
  "skills": ["billing", "technical", "complaint"],
  "shift_schedule": {
    "monday": {
      "start": "09:00",
      "end": "17:00"
    },
    "tuesday": {
      "start": "09:00",
      "end": "17:00"
    },
    "wednesday": {
      "start": "09:00",
      "end": "17:00"
    },
    "thursday": {
      "start": "09:00",
      "end": "17:00"
    },
    "friday": {
      "start": "09:00",
      "end": "17:00"
    }
  },
  "timezone": "Asia/Jakarta"
}
```

**Field Descriptions:**
- `user_id` (required): User ID from users table
- `client_id` (required): Client identifier
- `name` (required): Agent's full name
- `email` (optional): Agent's email
- `phone` (optional): Agent's phone number
- `max_concurrent_cases` (optional): Maximum cases agent can handle simultaneously (default: 5)
- `skills` (optional): Array of skills (default: ["general"])
- `shift_schedule` (optional): JSON object with shift schedule per day
- `timezone` (optional): Timezone (default: "Asia/Jakarta")

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/cs/agents/create" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "01JMQV4M5XYZ123ABC",
    "client_id": "01JPETM6DBJ1TPSKZNKNSE5HB6",
    "name": "Budi Santoso",
    "email": "budi@example.com",
    "max_concurrent_cases": 10,
    "skills": ["billing", "technical", "complaint"],
    "timezone": "Asia/Jakarta"
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01JMQV8N9PQRST4UVWXYZ",
    "user_id": "01JMQV4M5XYZ123ABC",
    "client_id": "01JPETM6DBJ1TPSKZNKNSE5HB6",
    "name": "Budi Santoso",
    "email": "budi@example.com",
    "phone": "628123456789",
    "status": "active",
    "availability_status": "available",
    "current_cases_count": 0,
    "max_concurrent_cases": 10,
    "skills": ["billing", "technical", "complaint"],
    "shift_schedule": {...},
    "timezone": "Asia/Jakarta",
    "last_active_at": null,
    "last_case_assigned_at": null,
    "total_cases_handled": 0,
    "total_cases_resolved": 0,
    "avg_response_time_min": 0,
    "avg_resolution_time_min": 0,
    "satisfaction_score": 0,
    "created_at": "2025-12-04T10:30:00Z",
    "updated_at": "2025-12-04T10:30:00Z"
  }
}
```

**Error Responses:**

400 Bad Request - Missing required fields:
```json
{
  "code": 400,
  "message": "Invalid request: user_id is required"
}
```

400 Bad Request - CS Agent already exists:
```json
{
  "code": 400,
  "message": "CS Agent already exists for this user"
}
```

---

## 2. Get CS Agents List

**Endpoint:** `GET /api/v1/cs/agents/list`

**Headers:**
```
Authorization: Bearer <your-token>
```

**Query Parameters:**
- `client_id` (optional): Filter by client ID
- `status` (optional): Filter by status (active, inactive, offline, busy, away)
- `availability_status` (optional): Filter by availability (available, busy, offline, break)

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/cs/agents/list?client_id=01JPETM6DBJ1TPSKZNKNSE5HB6&status=active" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "agents": [
      {
        "id": "01JMQV8N9PQRST4UVWXYZ",
        "user_id": "01JMQV4M5XYZ123ABC",
        "client_id": "01JPETM6DBJ1TPSKZNKNSE5HB6",
        "name": "Budi Santoso",
        "email": "budi@example.com",
        "status": "active",
        "availability_status": "available",
        "current_cases_count": 3,
        "max_concurrent_cases": 10,
        "skills": ["billing", "technical", "complaint"],
        "total_cases_handled": 45,
        "total_cases_resolved": 42,
        "avg_response_time_min": 2.5,
        "avg_resolution_time_min": 15.3,
        "satisfaction_score": 4.7,
        "created_at": "2025-12-04T10:30:00Z",
        "updated_at": "2025-12-04T15:45:00Z"
      }
    ],
    "total": 1
  }
}
```

---

## 3. Get CS Agent Details

**Endpoint:** `GET /api/v1/cs/agents/:id`

**Headers:**
```
Authorization: Bearer <your-token>
```

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/cs/agents/01JMQV8N9PQRST4UVWXYZ" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01JMQV8N9PQRST4UVWXYZ",
    "user_id": "01JMQV4M5XYZ123ABC",
    "client_id": "01JPETM6DBJ1TPSKZNKNSE5HB6",
    "name": "Budi Santoso",
    "email": "budi@example.com",
    "phone": "628123456789",
    "status": "active",
    "availability_status": "available",
    "current_cases_count": 3,
    "max_concurrent_cases": 10,
    "skills": ["billing", "technical", "complaint"],
    "shift_schedule": {
      "monday": {"start": "09:00", "end": "17:00"},
      "tuesday": {"start": "09:00", "end": "17:00"}
    },
    "timezone": "Asia/Jakarta",
    "last_active_at": "2025-12-04T15:45:00Z",
    "last_case_assigned_at": "2025-12-04T14:30:00Z",
    "total_cases_handled": 45,
    "total_cases_resolved": 42,
    "avg_response_time_min": 2.5,
    "avg_resolution_time_min": 15.3,
    "satisfaction_score": 4.7,
    "created_at": "2025-12-04T10:30:00Z",
    "updated_at": "2025-12-04T15:45:00Z"
  }
}
```

**Error Response (404 Not Found):**
```json
{
  "code": 404,
  "message": "CS Agent not found"
}
```

---

## 4. Update CS Agent

**Endpoint:** `POST /api/v1/cs/agents/update?agent_id=xxx`

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Budi Santoso Updated",
  "email": "budi.new@example.com",
  "phone": "628987654321",
  "status": "active",
  "max_concurrent_cases": 15,
  "skills": ["billing", "technical", "complaint", "general"],
  "timezone": "Asia/Jakarta"
}
```

**Field Descriptions:** (All fields are optional)
- `name`: Update agent's name
- `email`: Update agent's email
- `phone`: Update agent's phone
- `status`: Update status (active, inactive)
- `max_concurrent_cases`: Update max cases limit
- `skills`: Update skills array
- `shift_schedule`: Update shift schedule
- `timezone`: Update timezone

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/cs/agents/update?agent_id=01JMQV8N9PQRST4UVWXYZ" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Budi Santoso Updated",
    "max_concurrent_cases": 15
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01JMQV8N9PQRST4UVWXYZ",
    "name": "Budi Santoso Updated",
    "max_concurrent_cases": 15,
    ...
  }
}
```

---

## 5. Delete CS Agent

**Endpoint:** `POST /api/v1/cs/agents/delete?agent_id=xxx`

**Headers:**
```
Authorization: Bearer <your-token>
```

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/cs/agents/delete?agent_id=01JMQV8N9PQRST4UVWXYZ" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": "CS Agent deleted successfully"
}
```

**Error Response (400 Bad Request) - Agent has active cases:**
```json
{
  "code": 400,
  "message": "Cannot delete agent with 3 active cases"
}
```

---

## 6. Update Agent Availability

**Endpoint:** `POST /api/v1/cs/agents/availability/update?agent_id=xxx`

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "availability_status": "available"
}
```

**Valid Availability Statuses:**
- `available` - Agent is online and can receive cases
- `busy` - Agent is busy, cannot receive new cases
- `offline` - Agent is offline
- `break` - Agent is on break

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/cs/agents/availability/update?agent_id=01JMQV8N9PQRST4UVWXYZ" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "availability_status": "available"
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": "Availability updated successfully"
}
```

**Error Response (400 Bad Request):**
```json
{
  "code": 400,
  "message": "Invalid availability status"
}
```

---

## 7. Get Available Agents

**Endpoint:** `GET /api/v1/cs/agents/available?client_id=xxx&skill=billing`

**Headers:**
```
Authorization: Bearer <your-token>
```

**Query Parameters:**
- `client_id` (required): Client ID
- `skill` (optional): Filter by specific skill

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/cs/agents/available?client_id=01JPETM6DBJ1TPSKZNKNSE5HB6&skill=billing" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "agents": [
      {
        "agent_id": "01JMQV8N9PQRST4UVWXYZ",
        "agent_name": "Budi Santoso",
        "status": "active",
        "availability_status": "available",
        "current_cases_count": 3,
        "max_concurrent_cases": 10,
        "can_accept_new_case": true,
        "last_active_at": "2025-12-04T15:45:00Z"
      }
    ],
    "total": 1
  }
}
```

**Error Response (503 Service Unavailable) - No agents available:**
```json
{
  "code": 503,
  "message": "No available CS agents at the moment"
}
```

---

## Use Cases

### 1. Setup New CS Agent
```bash
# Step 1: Create CS agent
curl -X POST "http://localhost:8080/api/v1/cs/agents/create" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_123",
    "client_id": "client_456",
    "name": "Agent Name",
    "skills": ["billing", "technical"],
    "max_concurrent_cases": 10
  }'

# Step 2: Set agent to available
curl -X POST "http://localhost:8080/api/v1/cs/agents/availability/update?agent_id=AGENT_ID" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"availability_status": "available"}'
```

### 2. Agent Login/Logout Flow
```bash
# Agent Login - Set to available
POST /api/v1/cs/agents/availability/update?agent_id=xxx
{"availability_status": "available"}

# Agent Logout - Set to offline
POST /api/v1/cs/agents/availability/update?agent_id=xxx
{"availability_status": "offline"}

# Agent on Break
POST /api/v1/cs/agents/availability/update?agent_id=xxx
{"availability_status": "break"}
```

### 3. Load Balancing - Get Least Loaded Agent
```bash
# Get available agents sorted by load
GET /api/v1/cs/agents/available?client_id=xxx

# Response will be sorted by current_cases_count ASC
# First agent in the list has the least load
```

---

## Integration with AI Escalation

When AI escalates to CS, the system automatically:
1. Calls `GetAvailableAgents` with skill requirement
2. Selects least loaded agent
3. Assigns case to agent
4. Increments agent's `current_cases_count`
5. Notifies agent via webhook/websocket

When case is resolved:
- Decrements agent's `current_cases_count`
- Updates performance stats
- Agent becomes available for new cases

---

## Performance Metrics

Performance metrics are automatically tracked:
- `total_cases_handled` - Total cases assigned to agent
- `total_cases_resolved` - Total cases successfully resolved
- `avg_response_time_min` - Average first response time in minutes
- `avg_resolution_time_min` - Average case resolution time in minutes
- `satisfaction_score` - Customer satisfaction score (0-5)

These metrics are updated periodically and when cases are resolved.
