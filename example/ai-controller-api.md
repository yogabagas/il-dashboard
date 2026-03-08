# AI Controller API Documentation

Base URL: `/api/v1/instant-link/ai`

**Authorization Required:** All endpoints require authentication with specific roles (see each endpoint)

---

## 1. Simple AI Chat

**Endpoint:** `POST /api/v1/instant-link/ai/chat`

**Authorization:** ADMIN, OWNER, STAFF

**Description:** Send a simple message to AI and get a response. This endpoint does not maintain conversation context.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "message": "What is the capital of Indonesia?",
  "from": "user-123"
}
```

**Field Descriptions:**
- `message` (required): The message/question to send to AI
- `from` (optional): Identifier of the sender

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/ai/chat" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "What is the capital of Indonesia?",
    "from": "user-123"
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "response": "The capital of Indonesia is Jakarta.",
    "from": "user-123"
  }
}
```

**Error Responses:**

400 Bad Request:
```json
{
  "code": 400,
  "message": "Invalid request"
}
```

401 Unauthorized:
```json
{
  "code": 401,
  "message": "Unauthorized"
}
```

---

## 2. AI Chat with Context

**Endpoint:** `POST /api/v1/instant-link/ai/chat-context`

**Authorization:** ADMIN, OWNER, STAFF

**Description:** Send messages to AI with conversation context. This endpoint maintains conversation history for better contextual responses.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "messages": [
    {
      "role": "user",
      "content": "What is the capital of Indonesia?"
    },
    {
      "role": "assistant",
      "content": "The capital of Indonesia is Jakarta."
    },
    {
      "role": "user",
      "content": "What is the population of that city?"
    }
  ],
  "from": "user-123"
}
```

**Field Descriptions:**
- `messages` (required): Array of conversation messages
  - `role`: Either "user" or "assistant"
  - `content`: The message content
- `from` (optional): Identifier of the sender

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/ai/chat-context" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {
        "role": "user",
        "content": "What is the capital of Indonesia?"
      },
      {
        "role": "assistant",
        "content": "The capital of Indonesia is Jakarta."
      },
      {
        "role": "user",
        "content": "What is the population of that city?"
      }
    ],
    "from": "user-123"
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "response": "As of 2023, Jakarta has a population of approximately 10.6 million people in the city proper, and around 30 million in the greater metropolitan area.",
    "from": "user-123"
  }
}
```

**Error Responses:**

400 Bad Request:
```json
{
  "code": 400,
  "message": "Invalid request"
}
```

---

## 3. AI Conversation with WhatsApp Auto-Reply

**Endpoint:** `POST /api/v1/instant-link/ai/conversation`

**Authorization:** ADMIN, OWNER, STAFF

**Description:** Process AI conversation and automatically send response via WhatsApp. The client ID is automatically extracted from the authenticated user's context.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "sessionId": "62812345678",
  "from": "62812345678",
  "message": "Halo, saya mau tanya tentang tagihan PDAM"
}
```

**Field Descriptions:**
- `sessionId` (required): Session identifier (usually phone number)
- `from` (required): Sender's phone number
- `message` (required): Incoming message from user

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/ai/conversation" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "62812345678",
    "from": "62812345678",
    "message": "Halo, saya mau tanya tentang tagihan PDAM"
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "messageId": "msg-uuid-12345",
    "to": "62812345678",
    "status": "sent",
    "message": "AI response sent successfully",
    "aiResponse": "Halo! Saya siap membantu Anda dengan pertanyaan seputar tagihan PDAM. Anda bisa cek tagihan dengan cara:\n\n1. Via WhatsApp ini dengan mengetik: CEK TAGIHAN [Nomor Pelanggan]\n2. Via website: www.pdam-jakarta.go.id/cek-tagihan\n3. Via mobile app PDAM Jakarta\n\nAda yang bisa saya bantu lebih lanjut?"
  }
}
```

**Response Fields:**
- `messageId`: Unique identifier for the sent message
- `to`: Recipient phone number
- `status`: Delivery status (e.g., "sent")
- `message`: Status message
- `aiResponse`: The actual AI-generated response that was sent

**Notes:**
- The AI uses knowledge base from the authenticated user's client
- Response is automatically sent via WhatsApp
- Conversation context is maintained per session

---

## 4. Refresh Knowledge Cache (Single Client)

**Endpoint:** `POST /api/v1/instant-link/ai/knowledge/refresh`

**Authorization:** ADMIN, OWNER

**Description:** Refresh AI knowledge base cache from database to Redis for a specific client.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "client_id": "client-123"
}
```

**Field Descriptions:**
- `client_id` (required): Client identifier whose cache needs to be refreshed

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/ai/knowledge/refresh" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-123"
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "status": "success",
    "message": "Knowledge cache refreshed successfully",
    "client_id": "client-123"
  }
}
```

**Error Responses:**

400 Bad Request - Missing client_id:
```json
{
  "code": 400,
  "message": "Invalid request - client_id required"
}
```

500 Internal Server Error - Refresh failed:
```json
{
  "code": 500,
  "message": "Failed to refresh cache"
}
```

**Use Cases:**
- After bulk insert/update knowledge base entries
- When knowledge base data is updated via database directly
- Scheduled maintenance via cron jobs

**Script Example:**
```bash
# Using the provided script
./scripts/refresh_ai_knowledge.sh https://your-domain.com YOUR_ADMIN_TOKEN
```

---

## 5. Refresh All Knowledge Caches

**Endpoint:** `POST /api/v1/instant-link/ai/knowledge/refresh-all`

**Authorization:** ADMIN only

**Description:** Refresh AI knowledge base cache from database to Redis for ALL clients. This is an admin-only operation.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{}
```

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/ai/knowledge/refresh-all" \
  -H "Authorization: Bearer ADMIN_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "status": "success",
    "message": "All knowledge caches refreshed successfully"
  }
}
```

**Error Responses:**

403 Forbidden - Non-admin user:
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

500 Internal Server Error:
```json
{
  "code": 500,
  "message": "Failed to refresh all caches"
}
```

**Use Cases:**
- System-wide cache refresh after major updates
- Scheduled daily/weekly refresh via cron
- After system maintenance or database migration

**Script Example:**
```bash
# Using the provided script
./scripts/refresh_ai_knowledge.sh https://your-domain.com ADMIN_TOKEN
```

**⚠️ Warning:**
- This operation refreshes cache for ALL clients
- May take time depending on the number of clients
- Use sparingly, prefer single client refresh when possible

---

## Authorization & Roles

### Role-Based Access Control

| Endpoint | ADMIN | OWNER | STAFF |
|----------|-------|-------|-------|
| `/ai/chat` | ✅ | ✅ | ✅ |
| `/ai/chat-context` | ✅ | ✅ | ✅ |
| `/ai/conversation` | ✅ | ✅ | ✅ |
| `/ai/knowledge/refresh` | ✅ | ✅ | ❌ |
| `/ai/knowledge/refresh-all` | ✅ | ❌ | ❌ |

### Token Requirements

All endpoints require a valid JWT Bearer token in the Authorization header:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Getting a Token:**
- Login via the authentication endpoint
- Token expires after a configured duration
- Refresh token before expiry or re-authenticate

---

## Common Error Codes

| Status Code | Description |
|-------------|-------------|
| 400 | Bad Request - Invalid input or missing required fields |
| 401 | Unauthorized - Invalid or missing authentication token |
| 403 | Forbidden - User doesn't have permission for this operation |
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

**Error Response:**
```json
{
  "code": 400,
  "message": "Error description"
}
```

---

## Integration Examples

### Example 1: Simple Q&A Chatbot

```bash
# Ask a simple question
curl -X POST "http://localhost:8080/api/v1/instant-link/ai/chat" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Jam berapa kantor PDAM buka?",
    "from": "customer-001"
  }'
```

### Example 2: Contextual Conversation

```bash
# Multi-turn conversation with context
curl -X POST "http://localhost:8080/api/v1/instant-link/ai/chat-context" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "Berapa tarif air PDAM per meter kubik?"},
      {"role": "assistant", "content": "Tarif air PDAM Jakarta untuk rumah tangga 0-10 m³ adalah Rp 1.575 per m³"},
      {"role": "user", "content": "Kalau pemakaian saya 25 meter kubik, berapa total tarifnya?"}
    ],
    "from": "customer-001"
  }'
```

### Example 3: WhatsApp Integration

```javascript
// Node.js example for WhatsApp webhook
const axios = require('axios');

async function handleIncomingMessage(from, message) {
  try {
    const response = await axios.post(
      'http://localhost:8080/api/v1/instant-link/ai/conversation',
      {
        sessionId: from,
        from: from,
        message: message
      },
      {
        headers: {
          'Authorization': `Bearer ${process.env.API_TOKEN}`,
          'Content-Type': 'application/json'
        }
      }
    );

    console.log('AI Response:', response.data.data.aiResponse);
    console.log('Message ID:', response.data.data.messageId);
  } catch (error) {
    console.error('Error:', error.response?.data || error.message);
  }
}
```

### Example 4: Scheduled Cache Refresh (Cron)

```bash
# Add to crontab for daily refresh at 2 AM
# crontab -e

# Refresh all clients cache daily at 2 AM (Admin token required)
0 2 * * * /path/to/scripts/refresh_ai_knowledge.sh https://api.example.com $(cat /path/to/admin-token.txt) >> /var/log/ai-refresh.log 2>&1

# Or use environment variable
0 2 * * * AUTH_TOKEN=$ADMIN_TOKEN /path/to/scripts/refresh_ai_knowledge.sh https://api.example.com >> /var/log/ai-refresh.log 2>&1
```

---

## Best Practices

1. **Token Management:**
   - Store tokens securely (environment variables, secret managers)
   - Implement token refresh logic
   - Never commit tokens to version control

2. **Error Handling:**
   - Always check response codes
   - Implement retry logic for transient failures
   - Log errors for debugging

3. **Rate Limiting:**
   - Implement client-side rate limiting
   - Cache responses when appropriate
   - Use conversation context efficiently

4. **Cache Refresh:**
   - Refresh cache after knowledge base updates
   - Schedule periodic refreshes (daily/weekly)
   - Use single-client refresh when possible

5. **Context Management:**
   - Keep conversation context under 10 messages for optimal performance
   - Clear old sessions periodically
   - Track session IDs properly

---

## Troubleshooting

### Issue: "Unauthorized" Error

**Solution:**
- Check if token is valid and not expired
- Verify token format: `Authorization: Bearer <token>`
- Ensure user has required role for the endpoint

### Issue: "Invalid request" Error

**Solution:**
- Validate JSON format
- Check all required fields are present
- Verify field data types match specification

### Issue: Cache Not Updating

**Solution:**
- Call refresh endpoint after knowledge base changes
- Check Redis connection
- Verify client_id is correct
- Check logs for refresh errors

### Issue: AI Response Quality Poor

**Solution:**
- Update knowledge base with better Q&A pairs
- Add more keywords to knowledge entries
- Refresh cache after updates
- Check if client has sufficient knowledge base entries

---

## Support

For additional support or questions:
- Check API logs for detailed error messages
- Review knowledge base entries in database
- Test with simple requests first
- Verify authentication and authorization setup