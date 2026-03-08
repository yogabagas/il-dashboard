# Message Logs API Documentation

Base URL: `/api/v1/instant-link/message-logs`

**Authorization Required:** All endpoints require authentication with role `ADMIN`, `OWNER`, or `STAFF`

**Important Notes:**
- **ADMIN** users can view message logs for all clients
- **OWNER** users can only view message logs for their own client (based on `authInfo.ClientId`)
- **STAFF** users can view message logs for their client

---

## 1. Get Message Log by ID

**Endpoint:** `GET /api/v1/instant-link/message-logs/:id`

**Authorization:** ADMIN, OWNER, STAFF

**Description:** Retrieve a single message log entry by its ID.

**Headers:**
```
Authorization: Bearer <your-token>
```

**URL Parameters:**
- `id` (required): Message log ID

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs/01HZQK5X8V9Y2N1P3R4T6W8Z0A" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
    "clientId": "client-123",
    "senderId": "sender-456",
    "senderName": "Customer Service Bot",
    "campaignId": "campaign-789",
    "campaignName": "Welcome Campaign",
    "type": "whatsapp",
    "direction": "outbound",
    "recipientType": "phone",
    "recipientValue": "6281234567890",
    "templateId": "template-abc",
    "templateName": "welcome_message",
    "messageId": "wamid.HBgLNjI4MTIzNDU2Nzg5MA==",
    "sessionId": "01HZQK5X8V9Y2N1P3R4T6W8Z0B",
    "messageContent": "Template: welcome_message",
    "status": "delivered",
    "cost": 0.05,
    "sentAt": "2025-11-26T10:30:00Z",
    "deliveredAt": "2025-11-26T10:30:05Z",
    "readAt": "2025-11-26T10:31:00Z",
    "createdAt": "2025-11-26T10:30:00Z",
    "updatedAt": "2025-11-26T10:31:00Z"
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

403 Forbidden - OWNER trying to access log from different client:
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

---

## 2. Search Message Logs

**Endpoint:** `GET /api/v1/instant-link/message-logs`

**Authorization:** ADMIN, OWNER, STAFF

**Description:** Search and filter message logs with pagination. Supports filtering by client, campaign, type, direction, status, and date range.

**Headers:**
```
Authorization: Bearer <your-token>
```

**Query Parameters:**
- `filter` (optional): Search keyword (searches in recipientValue, messageId, messageContent, sessionId)
- `clientId` (optional): Filter by client ID
- `campaignId` (optional): Filter by campaign ID
- `type` (optional): Message type (whatsapp, sms, email)
- `direction` (optional): Message direction (inbound, outbound)
- `status` (optional): Message status (sent, delivered, read, failed, received)
- `recipientVal` (optional): Filter by specific recipient (exact match)
- `dateFrom` (optional): Filter by created date from (format: YYYY-MM-DD)
- `dateTo` (optional): Filter by created date to (format: YYYY-MM-DD)
- `sortBy` (optional): Field name to sort by (e.g., createdAt, sentAt, deliveredAt, status)
- `sortOrder` (optional): Sort order - Options: `asc` (ascending), `desc` (descending)
- `pageNo` (optional): Page number (default: 1)
- `rowPerPage` (optional): Items per page (default: 20)

**cURL Example - Search All Messages:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Filter by Client and Direction:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?clientId=client-123&direction=outbound&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Filter by Status and Date Range:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?status=delivered&dateFrom=2025-11-01&dateTo=2025-11-30&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Search Inbound Messages:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?direction=inbound&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Search by Keyword:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?filter=halo&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Filter by Campaign:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?campaignId=campaign-789&status=sent&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Sort by Created Date (Descending):**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?sortBy=createdAt&sortOrder=desc&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Sort by Status (Ascending):**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?status=delivered&sortBy=sentAt&sortOrder=asc&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": [
    {
      "id": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
      "clientId": "client-123",
      "senderId": "sender-456",
      "senderName": "Customer Service Bot",
      "type": "whatsapp",
      "direction": "outbound",
      "recipientType": "phone",
      "recipientValue": "6281234567890",
      "templateId": "template-abc",
      "templateName": "welcome_message",
      "messageId": "wamid.HBgLNjI4MTIzNDU2Nzg5MA==",
      "sessionId": "01HZQK5X8V9Y2N1P3R4T6W8Z0B",
      "messageContent": "Template: welcome_message",
      "status": "delivered",
      "cost": 0.05,
      "sentAt": "2025-11-26T10:30:00Z",
      "deliveredAt": "2025-11-26T10:30:05Z",
      "createdAt": "2025-11-26T10:30:00Z",
      "updatedAt": "2025-11-26T10:30:05Z"
    },
    {
      "id": "01HZQK6Y9W0Z3O2Q4S5U7X9A1B",
      "clientId": "client-123",
      "type": "whatsapp",
      "direction": "inbound",
      "recipientType": "phone",
      "recipientValue": "628119876543",
      "messageId": "wamid.HBgLNjI4MTE5ODc2NTQzMA==",
      "sessionId": "01HZQK5X8V9Y2N1P3R4T6W8Z0B",
      "messageContent": "Halo, saya mau tanya tentang tagihan",
      "status": "received",
      "cost": 0,
      "sentAt": "2025-11-26T10:35:00Z",
      "createdAt": "2025-11-26T10:35:00Z",
      "updatedAt": "2025-11-26T10:35:00Z"
    },
    {
      "id": "01HZQK7Z0X1A4P3R5T6V8Y0B2C",
      "clientId": "client-123",
      "type": "whatsapp",
      "direction": "outbound",
      "recipientType": "phone",
      "recipientValue": "6281234567890",
      "messageId": "wamid.HBgLNjI4MTIzNDU2Nzg5MQ==",
      "sessionId": "01HZQK5X8V9Y2N1P3R4T6W8Z0B",
      "messageContent": "Halo! Saya siap membantu Anda dengan pertanyaan seputar tagihan PDAM.",
      "status": "sent",
      "cost": 0.02,
      "sentAt": "2025-11-26T10:35:02Z",
      "createdAt": "2025-11-26T10:35:02Z",
      "updatedAt": "2025-11-26T10:35:02Z"
    }
  ],
  "currPage": 1,
  "haveNext": false,
  "totalPage": 3
}
```

**Response Fields:**
- `code`: Status code (0 = success)
- `messages`: Status message
- `data`: Array of message log entries
- `currPage`: Current page number
- `haveNext`: Boolean indicating if there are more pages
- `totalPage`: Total number of records matching the search

---

## Message Log Fields Description

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique message log identifier (ULID) |
| `clientId` | string | Client identifier who owns this message |
| `senderId` | string | Sender identifier (for outbound template messages) |
| `senderName` | string | Sender name (joined from senders table) |
| `campaignId` | string | Campaign identifier (if part of campaign) |
| `campaignName` | string | Campaign name (joined from campaigns table) |
| `type` | string | Message type: `whatsapp`, `sms`, `email` |
| `direction` | string | Message direction: `inbound` (received), `outbound` (sent) |
| `recipientType` | string | Recipient type: `phone`, `email` |
| `recipientValue` | string | Recipient value (phone number or email address) |
| `templateId` | string | Template ID (for outbound template messages) |
| `templateName` | string | Template name (joined from templates table) |
| `messageId` | string | WhatsApp/SMS message ID from provider |
| `sessionId` | string | Conversation session ID (links to conversation) |
| `messageContent` | string | Actual message content text |
| `status` | string | Message status: `sent`, `delivered`, `read`, `failed`, `received` |
| `cost` | float | Message cost in currency units |
| `sentAt` | string | ISO8601 timestamp when message was sent |
| `deliveredAt` | string | ISO8601 timestamp when message was delivered |
| `readAt` | string | ISO8601 timestamp when message was read |
| `failedAt` | string | ISO8601 timestamp when message failed |
| `errorMessage` | string | Error message if status is failed |
| `createdAt` | string | ISO8601 timestamp when record was created |
| `updatedAt` | string | ISO8601 timestamp when record was last updated |

---

## Message Directions Explained

### Inbound Messages (`direction: "inbound"`)

**Source:** Messages received from customers/users via WhatsApp webhook

**Characteristics:**
- `recipientValue`: Business phone number that received the message
- `status`: Usually "received"
- `cost`: Always 0 (no cost for receiving messages)
- `messageContent`: Customer's actual message text

**Example Use Cases:**
- Customer inquiries via WhatsApp
- User responses to AI chatbot
- Incoming support requests

**cURL to View Only Inbound:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?direction=inbound&pageNo=1&rowPerPage=50" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### Outbound Messages (`direction: "outbound"`)

**Source:** Messages sent to customers via:
- Template messages (campaigns)
- AI chatbot responses
- Manual sends

**Characteristics:**
- `recipientValue`: Customer phone number receiving the message
- `status`: "sent", "delivered", "read", or "failed"
- `cost`: May have cost depending on message type
- `messageContent`: Message text or template name

**Example Use Cases:**
- Campaign broadcasts
- AI chatbot automated replies
- Transactional notifications
- Marketing messages

**cURL to View Only Outbound:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?direction=outbound&pageNo=1&rowPerPage=50" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

## Common Usage Patterns

### 1. Monitor Recent Inbound Messages

Track all incoming customer messages:

```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?direction=inbound&pageNo=1&rowPerPage=50" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### 2. Check Campaign Performance

View all messages from a specific campaign:

```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?campaignId=campaign-789&direction=outbound&pageNo=1&rowPerPage=100" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### 3. Find Failed Messages

Identify messages that failed to send:

```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?status=failed&direction=outbound&pageNo=1&rowPerPage=50" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### 4. Track Delivery Status

Check delivered messages in date range:

```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?status=delivered&dateFrom=2025-11-01&dateTo=2025-11-30&pageNo=1&rowPerPage=100" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### 5. View Conversation History

Get all messages (inbound & outbound) for a specific session:

```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?sessionId=01HZQK5X8V9Y2N1P3R4T6W8Z0B&pageNo=1&rowPerPage=50" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### 6. Search Message Content

Find messages containing specific keywords:

```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?filter=tagihan&pageNo=1&rowPerPage=50" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### 7. Client-Specific Analytics

Get all message logs for a specific client:

```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?clientId=client-123&dateFrom=2025-11-01&dateTo=2025-11-30&pageNo=1&rowPerPage=100" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### 8. Billing Report

Calculate costs for a date range:

```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/message-logs?clientId=client-123&direction=outbound&dateFrom=2025-11-01&dateTo=2025-11-30&pageNo=1&rowPerPage=1000" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

Then sum the `cost` field from all results.

---

## Common Error Codes

| Status Code | Description |
|-------------|-------------|
| 400 | Bad Request - Invalid query parameters |
| 401 | Unauthorized - Invalid or missing authentication token |
| 403 | Forbidden - User doesn't have permission to view this data |
| 404 | Not Found - Message log entry doesn't exist |
| 500 | Internal Server Error - Server error occurred |

---

## Notes

1. **Automatic Logging:**
   - All outbound messages (template, AI responses) are automatically logged
   - All inbound messages from WhatsApp webhook are automatically logged
   - Status updates (delivered, read) are automatically updated via webhook

2. **Search Behavior:**
   - `filter` searches across: `recipientValue`, `messageId`, `messageContent`, and `sessionId`
   - Empty filter returns all entries (with other filters applied)
   - OWNER users automatically have results filtered to their clientId
   - Case-insensitive search using SQL LIKE pattern matching

3. **Sorting:**
   - Use `sortBy` to specify which field to sort by (e.g., `createdAt`, `sentAt`, `deliveredAt`, `status`)
   - Use `sortOrder` to specify ascending (`asc`) or descending (`desc`) order
   - Default sort order is by creation date descending (newest first)
   - Common sort fields: `createdAt`, `sentAt`, `deliveredAt`, `readAt`, `status`, `cost`

4. **Pagination:**
   - Default page size is 20 items
   - Maximum recommended page size is 100 for performance
   - Use `haveNext` to check if there are more pages

5. **Date Filtering:**
   - Dates are inclusive (includes both `dateFrom` and `dateTo`)
   - `dateTo` includes the entire day (23:59:59)
   - Format: YYYY-MM-DD

6. **Direction Field:**
   - **inbound**: Customer → Business (received messages)
   - **outbound**: Business → Customer (sent messages)

7. **Status Tracking:**
   - **sent**: Message sent to WhatsApp API successfully
   - **delivered**: WhatsApp confirmed delivery to recipient device
   - **read**: Recipient opened/read the message
   - **failed**: Message failed to send
   - **received**: Inbound message received from customer

8. **Cost Tracking:**
   - Inbound messages always have `cost: 0`
   - Outbound messages may have cost depending on message type
   - Costs are in the configured currency

9. **Session Linking:**
   - Use `sessionId` to track conversation threads
   - Links message_logs with message_conversations table
   - Useful for viewing complete conversation history

---

## Integration Examples

### Python Example - Get Recent Inbound Messages

```python
import requests

url = "http://localhost:8080/api/v1/instant-link/message-logs"
headers = {
    "Authorization": "Bearer YOUR_TOKEN_HERE"
}
params = {
    "direction": "inbound",
    "pageNo": 1,
    "rowPerPage": 50
}

response = requests.get(url, headers=headers, params=params)
data = response.json()

if data["code"] == 0:
    messages = data["data"]
    for msg in messages:
        print(f"From: {msg['recipientValue']}")
        print(f"Content: {msg['messageContent']}")
        print(f"Time: {msg['sentAt']}")
        print("---")
else:
    print(f"Error: {data['message']}")
```

---

### JavaScript Example - Search Failed Messages

```javascript
const axios = require('axios');

async function getFailedMessages(clientId, dateFrom, dateTo) {
  const url = 'http://localhost:8080/api/v1/instant-link/message-logs';

  const response = await axios.get(url, {
    headers: {
      'Authorization': `Bearer ${process.env.API_TOKEN}`
    },
    params: {
      clientId: clientId,
      status: 'failed',
      direction: 'outbound',
      dateFrom: dateFrom,
      dateTo: dateTo,
      pageNo: 1,
      rowPerPage: 100
    }
  });

  if (response.data.code === 0) {
    const failed = response.data.data;
    console.log(`Found ${response.data.totalPage} failed messages`);

    failed.forEach(msg => {
      console.log({
        id: msg.id,
        to: msg.recipientValue,
        error: msg.errorMessage,
        failedAt: msg.failedAt
      });
    });
  }
}

// Usage
getFailedMessages('client-123', '2025-11-01', '2025-11-30');
```

---

### Bash Script Example - Daily Report

```bash
#!/bin/bash

TOKEN="YOUR_TOKEN_HERE"
CLIENT_ID="client-123"
DATE_FROM=$(date -d "yesterday" +%Y-%m-%d)
DATE_TO=$(date -d "yesterday" +%Y-%m-%d)

echo "Message Report for $DATE_FROM"
echo "================================"

# Get total sent
SENT=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/instant-link/message-logs?clientId=$CLIENT_ID&direction=outbound&status=sent&dateFrom=$DATE_FROM&dateTo=$DATE_TO&pageNo=1&rowPerPage=1" \
  | jq -r '.totalPage')

echo "Total Sent: $SENT"

# Get total delivered
DELIVERED=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/instant-link/message-logs?clientId=$CLIENT_ID&direction=outbound&status=delivered&dateFrom=$DATE_FROM&dateTo=$DATE_TO&pageNo=1&rowPerPage=1" \
  | jq -r '.totalPage')

echo "Total Delivered: $DELIVERED"

# Get total failed
FAILED=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/instant-link/message-logs?clientId=$CLIENT_ID&status=failed&dateFrom=$DATE_FROM&dateTo=$DATE_TO&pageNo=1&rowPerPage=1" \
  | jq -r '.totalPage')

echo "Total Failed: $FAILED"

# Get total inbound
INBOUND=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/instant-link/message-logs?clientId=$CLIENT_ID&direction=inbound&dateFrom=$DATE_FROM&dateTo=$DATE_TO&pageNo=1&rowPerPage=1" \
  | jq -r '.totalPage')

echo "Total Inbound: $INBOUND"
```

---

## Support

For additional support or questions:
- Review message_logs table structure in database
- Check application logs for detailed error messages
- Verify authentication token and permissions