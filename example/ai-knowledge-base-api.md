# AI Knowledge Base API Documentation

Base URL: `/api/v1/instant-link/ai/knowledge-base`

**Authorization Required:** All endpoints require authentication

**Roles & Permissions:**
- **ADMIN** users can manage knowledge base for any client
- **OWNER** users can only manage knowledge base for their own client (based on `authInfo.ClientId`)
- **STAFF** users can read (GET) knowledge base for their client

---

## 1. Create Knowledge Base Entry

**Endpoint:** `POST /api/v1/instant-link/ai/knowledge-base/create`

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "clientId": "client-123",
  "category": "Product Information",
  "question": "What are your business hours?",
  "answer": "We are open Monday to Friday, 9 AM to 6 PM. We are closed on weekends and public holidays.",
  "keywords": "hours, opening, schedule, time, business hours",
  "priority": 1
}
```

**Field Descriptions:**
- `clientId` (required): Client identifier
- `category` (optional): Category/topic of the knowledge
- `question` (required): Question or topic
- `answer` (required): Answer or content
- `keywords` (optional): Comma-separated keywords for better search
- `priority` (optional): Priority level for ordering (default: 0)

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/ai/knowledge-base/create" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "clientId": "client-123",
    "category": "Product Information",
    "question": "What are your business hours?",
    "answer": "We are open Monday to Friday, 9 AM to 6 PM. We are closed on weekends and public holidays.",
    "keywords": "hours, opening, schedule, time, business hours",
    "priority": 1
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "kb-uuid-12345",
    "clientId": "client-123",
    "category": "Product Information",
    "question": "What are your business hours?",
    "answer": "We are open Monday to Friday, 9 AM to 6 PM. We are closed on weekends and public holidays.",
    "keywords": "hours, opening, schedule, time, business hours",
    "isActive": true,
    "priority": 1,
    "createdBy": "user-456",
    "updatedBy": "user-456",
    "createdAt": "2025-11-26T10:30:00Z",
    "updatedAt": "2025-11-26T10:30:00Z"
  }
}
```

**Error Responses:**

400 Bad Request - Missing required fields:
```json
{
  "code": 400,
  "message": "ClientId is required"
}
```

403 Forbidden - OWNER trying to create for different client:
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

---

## 2. Update Knowledge Base Entry

**Endpoint:** `POST /api/v1/instant-link/ai/knowledge-base/update`

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "id": "kb-uuid-12345",
  "category": "Product Information - Updated",
  "question": "What are your business hours?",
  "answer": "We are open Monday to Friday, 9 AM to 8 PM (extended hours). We are closed on weekends and public holidays.",
  "keywords": "hours, opening, schedule, time, business hours, extended",
  "isActive": true,
  "priority": 2
}
```

**Field Descriptions:**
- `id` (required): Knowledge base entry ID
- All other fields are optional - only provided fields will be updated

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/ai/knowledge-base/update" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "kb-uuid-12345",
    "category": "Product Information - Updated",
    "answer": "We are open Monday to Friday, 9 AM to 8 PM (extended hours). We are closed on weekends and public holidays.",
    "isActive": true,
    "priority": 2
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "kb-uuid-12345",
    "clientId": "client-123",
    "category": "Product Information - Updated",
    "question": "What are your business hours?",
    "answer": "We are open Monday to Friday, 9 AM to 8 PM (extended hours). We are closed on weekends and public holidays.",
    "keywords": "hours, opening, schedule, time, business hours, extended",
    "isActive": true,
    "priority": 2,
    "createdBy": "user-456",
    "updatedBy": "user-789",
    "createdAt": "2025-11-26T10:30:00Z",
    "updatedAt": "2025-11-26T14:20:00Z"
  }
}
```

**Error Responses:**

404 Not Found - Knowledge base entry not found:
```json
{
  "code": 404,
  "message": "Not found"
}
```

403 Forbidden - OWNER trying to update entry from different client:
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

---

## 3. Get Knowledge Base Entry by ID

**Endpoint:** `GET /api/v1/instant-link/ai/knowledge-base/:id`

**Headers:**
```
Authorization: Bearer <your-token>
```

**URL Parameters:**
- `id` (required): Knowledge base entry ID

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/ai/knowledge-base/kb-uuid-12345" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "kb-uuid-12345",
    "clientId": "client-123",
    "category": "Product Information",
    "question": "What are your business hours?",
    "answer": "We are open Monday to Friday, 9 AM to 6 PM. We are closed on weekends and public holidays.",
    "keywords": "hours, opening, schedule, time, business hours",
    "isActive": true,
    "priority": 1,
    "createdBy": "user-456",
    "updatedBy": "user-456",
    "createdAt": "2025-11-26T10:30:00Z",
    "updatedAt": "2025-11-26T10:30:00Z"
  }
}
```

**Error Responses:**

404 Not Found:
```json
{
  "code": 404,
  "message": "Not found"
}
```

403 Forbidden - OWNER trying to access entry from different client:
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

---

## 4. Search Knowledge Base

**Endpoint:** `GET /api/v1/instant-link/ai/knowledge-base`

**Headers:**
```
Authorization: Bearer <your-token>
```

**Query Parameters (all optional):**
- `filter` - Search keyword (searches in question, answer, keywords)
- `clientId` - Filter by client ID (OWNER/STAFF users: automatically filtered to their client)
- `category` - Filter by category
- `pageNo` - Page number (default: 1)
- `rowPerPage` - Items per page (default: 20)

**cURL Example - Full Search:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/ai/knowledge-base?filter=business%20hours&clientId=client-123&category=Product%20Information&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Simple Search (all entries):**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/ai/knowledge-base?pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Search by filter only:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/ai/knowledge-base?filter=tagihan" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": [
    {
      "id": "kb-uuid-12345",
      "clientId": "client-123",
      "category": "Product Information",
      "question": "What are your business hours?",
      "answer": "We are open Monday to Friday, 9 AM to 6 PM. We are closed on weekends and public holidays.",
      "keywords": "hours, opening, schedule, time, business hours",
      "isActive": true,
      "priority": 1,
      "createdBy": "user-456",
      "updatedBy": "user-456",
      "createdAt": "2025-11-26T10:30:00Z",
      "updatedAt": "2025-11-26T10:30:00Z"
    },
    {
      "id": "kb-uuid-67890",
      "clientId": "client-123",
      "category": "Product Information",
      "question": "Do you offer after-hours support?",
      "answer": "Yes, we offer 24/7 emergency support via our hotline at 1-800-SUPPORT.",
      "keywords": "support, hours, emergency, hotline, 24/7",
      "isActive": true,
      "priority": 2,
      "createdBy": "user-456",
      "updatedBy": "user-456",
      "createdAt": "2025-11-26T11:00:00Z",
      "updatedAt": "2025-11-26T11:00:00Z"
    }
  ],
  "currPage": 1,
  "haveNext": false,
  "totalPage": 2
}
```

**Response Fields:**
- `code`: Status code (0 = success)
- `messages`: Status message
- `data`: Array of knowledge base entries
- `currPage`: Current page number
- `haveNext`: Boolean indicating if there are more pages
- `totalPage`: Total number of records matching the search

---

## 5. Delete Knowledge Base Entry

**Endpoint:** `POST /api/v1/instant-link/ai/knowledge-base/delete`

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "id": "kb-uuid-12345"
}
```

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/ai/knowledge-base/delete" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "kb-uuid-12345"
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "success": true,
    "id": "kb-uuid-12345"
  }
}
```

**Error Responses:**

404 Not Found:
```json
{
  "code": 404,
  "message": "Not found"
}
```

403 Forbidden - OWNER trying to delete entry from different client:
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

---

## Common Error Codes

| Status Code | Description |
|-------------|-------------|
| 400 | Bad Request - Invalid input or missing required fields |
| 401 | Unauthorized - Invalid or missing authentication token |
| 403 | Forbidden - User doesn't have permission to access this resource |
| 404 | Not Found - Requested resource doesn't exist |
| 500 | Internal Server Error - Server error occurred |

---

## Response Format

All API responses follow a standard format:

**Success Response:**
```json
{
  "code": 0,
  "messages": "Success",
  "data": { ... }
}
```

**Paginated Response (for search endpoint):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": [ ... ],
  "currPage": 1,
  "haveNext": false,
  "totalPage": 10
}
```

**Error Response:**
```json
{
  "code": 400,
  "message": "Error description"
}
```

---

## Usage Examples by Role

### ADMIN User Examples

ADMIN users have full access to all clients:

```bash
# Create knowledge base for any client
curl -X POST "http://localhost:8080/api/v1/instant-link/ai/knowledge-base/create" \
  -H "Authorization: Bearer ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "clientId": "any-client-id",
    "question": "Sample question",
    "answer": "Sample answer"
  }'

# Search across all clients
curl -X GET "http://localhost:8080/api/v1/instant-link/ai/knowledge-base?filter=business&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer ADMIN_TOKEN"
```

### OWNER User Examples

OWNER users can only manage their own client's knowledge base:

```bash
# Create knowledge base (must use own clientId)
curl -X POST "http://localhost:8080/api/v1/instant-link/ai/knowledge-base/create" \
  -H "Authorization: Bearer OWNER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "clientId": "owner-client-123",
    "question": "Sample question",
    "answer": "Sample answer"
  }'

# Search automatically filtered to owner's client
curl -X GET "http://localhost:8080/api/v1/instant-link/ai/knowledge-base?filter=business&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer OWNER_TOKEN"
```

### STAFF User Examples

STAFF users can only read (GET) knowledge base:

```bash
# Get knowledge by ID
curl -X GET "http://localhost:8080/api/v1/instant-link/ai/knowledge-base/kb-uuid-12345" \
  -H "Authorization: Bearer STAFF_TOKEN"

# Search knowledge base (automatically filtered to staff's client)
curl -X GET "http://localhost:8080/api/v1/instant-link/ai/knowledge-base?filter=tagihan" \
  -H "Authorization: Bearer STAFF_TOKEN"
```

---

## Notes

1. **Soft Delete**: The delete operation performs a soft delete. Data is marked as deleted but not physically removed from the database.

2. **Cache Management**: After create/update/delete operations, the system automatically refreshes the AI knowledge cache for the affected client.

3. **Pagination**:
   - Default page size is 20 items
   - Pagination is specified via query parameters (`pageNo` and `rowPerPage`)

4. **Search Behavior**:
   - `filter` searches across question, answer, and keywords fields
   - Empty filter returns all entries (with other filters applied)
   - OWNER users automatically have results filtered to their clientId

5. **Priority**: Higher priority entries may be used preferentially by AI systems. Use this to control which answers should be prioritized.

6. **Keywords**: Comma-separated keywords improve search accuracy. Include common variations and synonyms.

7. **Response Wrapping**: All responses are wrapped in a standard format with `code`, `messages`, and `data` fields. Success always returns `code: 0`.