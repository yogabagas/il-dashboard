# Campaign API Documentation

Base URL: `/api/v1/instant-link/campaigns`

**Authorization Required:** All endpoints require authentication with specific roles (see each endpoint)

---

## Table of Contents

1. [Campaign Management](#campaign-management)
   - [Create Campaign](#1-create-campaign)
   - [Update Campaign](#2-update-campaign)
   - [Get Campaign by ID](#3-get-campaign-by-id)
   - [Search Campaigns](#4-search-campaigns)
   - [Delete Campaign](#5-delete-campaign)

2. [Campaign Recipients](#campaign-recipients)
   - [Get Campaign Recipients](#6-get-campaign-recipients)
   - [Generate Campaign Recipients](#7-generate-campaign-recipients)

3. [Campaign Statistics](#campaign-statistics)
   - [Get Campaign Stats](#8-get-campaign-stats)

4. [Campaign Execution](#campaign-execution)
   - [Start Campaign](#9-start-campaign)
   - [Pause Campaign](#10-pause-campaign)
   - [Resume Campaign](#11-resume-campaign)

---

# Campaign Management

## 1. Create Campaign

**Endpoint:** `POST /api/v1/instant-link/campaigns`

**Authorization:** ADMIN, OWNER

**Description:** Create a new campaign for sending bulk messages via WhatsApp, SMS, or Email. The campaign can be scheduled or executed immediately.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Promo Ramadan 2024",
  "type": "whatsapp",
  "senderId": "sender-123",
  "templateId": "template-456",
  "scheduledAt": "2024-03-15T08:00:00Z",
  "batchSize": 100,
  "delaySeconds": 5,
  "parameters": {
    "discount": "20%",
    "validUntil": "31 March 2024"
  },
  "targets": [
    {
      "targetType": "group",
      "targetId": "group-789"
    },
    {
      "targetType": "contact",
      "targetId": "contact-012"
    }
  ]
}
```

**Field Descriptions:**
- `name` (required): Campaign name for identification
- `type` (required): Campaign type - Options: `whatsapp`, `sms`, `email`
- `senderId` (required): Sender ID to use for sending messages
- `templateId` (required): Message template ID to use
- `scheduledAt` (optional): Schedule campaign execution at specific time (ISO 8601 format)
- `batchSize` (required): Number of messages to send per batch
- `delaySeconds` (required): Delay in seconds between batches
- `parameters` (optional): Template variable values (key-value pairs)
  - **For text-only templates:** Use `param1`, `param2`, etc. or `1`, `2`, etc. for body variables
  - **For templates with IMAGE header:** Use `image` or `header_image` for the image URL
  - **For templates with VIDEO header:** Use `video` or `header_video` for the video URL
  - **For templates with DOCUMENT header:** Use `document` or `header_document` for the document URL
  - **Media ID:** Use `image_id`, `video_id`, or `document_id` if media is already uploaded to WhatsApp
- `targets` (required): Array of target recipients
  - `targetType`: Type of target - Options: `contact`, `group`, `all`
  - `targetId`: ID of the target (contact ID, group ID, or empty for "all")

### 🚨 **Important: Parameters for Templates with Header Media**

**Error:** `"Number of parameters does not match the expected number of params"`

This error occurs when:
1. Template has **IMAGE/VIDEO/DOCUMENT header** but **no body variables** ({{1}}, {{2}})
2. You send text parameters thinking they're for the body
3. WhatsApp expects **only header media parameter**, not text parameters

**Example - WRONG ❌:**
```json
{
  "templateId": "template-with-image-header",
  "parameters": {
    "image": "https://example.com/promo.jpg",
    "param1": "John Doe"  // ❌ WRONG! Template body has no {{1}}
  }
}
```

**Example - CORRECT ✅:**
```json
{
  "templateId": "template-with-image-header",
  "parameters": {
    "image": "https://example.com/promo.jpg"  // ✅ Only image, no text params
  }
}
```

**If template has BOTH header image AND body variables:**
```json
{
  "templateId": "template-with-image-and-text",
  "parameters": {
    "image": "https://example.com/promo.jpg",  // For header
    "param1": "John Doe",                       // For {{1}} in body
    "param2": "Special Discount"                // For {{2}} in body
  }
}
```

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/campaigns" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Promo Ramadan 2024",
    "type": "whatsapp",
    "senderId": "sender-123",
    "templateId": "template-456",
    "scheduledAt": "2024-03-15T08:00:00Z",
    "batchSize": 100,
    "delaySeconds": 5,
    "parameters": {
      "discount": "20%",
      "validUntil": "31 March 2024"
    },
    "targets": [
      {
        "targetType": "group",
        "targetId": "group-789"
      }
    ]
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "camp-12345",
    "clientId": "client-123",
    "senderId": "sender-123",
    "senderName": "My Business",
    "name": "Promo Ramadan 2024",
    "type": "whatsapp",
    "templateId": "template-456",
    "templateName": "promo_ramadan",
    "status": "draft",
    "scheduledAt": "2024-03-15T08:00:00Z",
    "batchSize": 100,
    "delaySeconds": 5,
    "totalRecipients": 0,
    "totalSent": 0,
    "totalDelivered": 0,
    "totalFailed": 0,
    "totalRead": 0,
    "parameters": {
      "discount": "20%",
      "validUntil": "31 March 2024"
    },
    "createdBy": "user-123",
    "createdAt": "2024-03-10T10:30:00Z",
    "updatedAt": "2024-03-10T10:30:00Z"
  }
}
```

**Error Responses:**

400 Bad Request:
```json
{
  "code": 400,
  "message": "Invalid request - missing required fields"
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

## 2. Update Campaign

**Endpoint:** `PUT /api/v1/instant-link/campaigns/:id`

**Authorization:** ADMIN, OWNER

**Description:** Update an existing campaign. Only campaigns in 'draft' or 'scheduled' status can be updated.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Promo Ramadan 2024 - Updated",
  "status": "scheduled",
  "scheduledAt": "2024-03-16T09:00:00Z",
  "batchSize": 150,
  "delaySeconds": 3,
  "parameters": {
    "discount": "25%",
    "validUntil": "31 March 2024"
  }
}
```

**Field Descriptions:**
- `name` (optional): New campaign name
- `status` (optional): Campaign status - Options: `draft`, `scheduled`, `running`, `paused`, `completed`, `cancelled`
- `scheduledAt` (optional): New scheduled time
- `batchSize` (optional): New batch size
- `delaySeconds` (optional): New delay between batches
- `parameters` (optional): Updated template parameters

**cURL Example:**
```bash
curl -X PUT "http://localhost:8080/api/v1/instant-link/campaigns/camp-12345" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Promo Ramadan 2024 - Updated",
    "status": "scheduled",
    "batchSize": 150
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "camp-12345",
    "clientId": "client-123",
    "name": "Promo Ramadan 2024 - Updated",
    "type": "whatsapp",
    "status": "scheduled",
    "batchSize": 150,
    "delaySeconds": 3,
    "updatedBy": "user-123",
    "updatedAt": "2024-03-10T11:00:00Z"
  }
}
```

**Error Responses:**

400 Bad Request - Campaign cannot be updated:
```json
{
  "code": 400,
  "message": "Campaign is already running and cannot be updated"
}
```

404 Not Found:
```json
{
  "code": 404,
  "message": "Campaign not found"
}
```

---

## 3. Get Campaign by ID

**Endpoint:** `GET /api/v1/instant-link/campaigns/:id`

**Authorization:** ADMIN, OWNER, STAFF

**Description:** Retrieve detailed information about a specific campaign.

**Headers:**
```
Authorization: Bearer <your-token>
```

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/campaigns/camp-12345" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "camp-12345",
    "clientId": "client-123",
    "senderId": "sender-123",
    "senderName": "My Business",
    "name": "Promo Ramadan 2024",
    "type": "whatsapp",
    "templateId": "template-456",
    "templateName": "promo_ramadan",
    "status": "running",
    "scheduledAt": "2024-03-15T08:00:00Z",
    "startedAt": "2024-03-15T08:00:05Z",
    "batchSize": 100,
    "delaySeconds": 5,
    "totalRecipients": 1000,
    "totalSent": 450,
    "totalDelivered": 420,
    "totalFailed": 10,
    "totalRead": 280,
    "parameters": {
      "discount": "20%",
      "validUntil": "31 March 2024"
    },
    "createdBy": "user-123",
    "updatedBy": "user-123",
    "createdAt": "2024-03-10T10:30:00Z",
    "updatedAt": "2024-03-15T08:00:05Z"
  }
}
```

**Error Responses:**

404 Not Found:
```json
{
  "code": 404,
  "message": "Campaign not found"
}
```

---

## 4. Search Campaigns

**Endpoint:** `GET /api/v1/instant-link/campaigns`

**Authorization:** ADMIN, OWNER, STAFF

**Description:** Search and filter campaigns with pagination support. Supports searching by campaign name and filtering by multiple criteria.

**Headers:**
```
Authorization: Bearer <your-token>
```

**Query Parameters:**
- `pageNo` (optional): Page number (default: 1)
- `rowPerPage` (optional): Number of items per page (default: 10)
- `filter` (optional): Search term to filter campaign names (uses SQL LIKE '%filter%')
- `clientId` (optional): Filter by specific client ID (ADMIN only)
- `senderId` (optional): Filter by specific sender ID
- `type` (optional): Filter by campaign type (`whatsapp`, `sms`, `email`)
- `status` (optional): Filter by status (`draft`, `scheduled`, `running`, `paused`, `completed`, `cancelled`)
- `dateFrom` (optional): Filter by created date from (format: YYYY-MM-DD)
- `dateTo` (optional): Filter by created date to (format: YYYY-MM-DD)

**Parameter Details:**

**`filter` - Campaign Name Search:**
- Searches campaign names using pattern matching
- Uses SQL LIKE with wildcards: `name LIKE '%filter%'`
- Case-insensitive search
- Example: `filter=ramadan` will match "Promo Ramadan 2024", "Info Ramadan", etc.

**`clientId` - Client Filtering:**
- **ADMIN users**: Can specify any clientId or leave empty to see all clients
- **OWNER users**: Parameter is ignored, always shows only their own campaigns
- **STAFF users**: Parameter is ignored, always shows only their client's campaigns

**`senderId` - Sender Filtering:**
- Filter campaigns by specific sender
- Useful for tracking which sender is used in campaigns
- All roles can use this filter

**`dateFrom` / `dateTo` - Date Range Filtering:**
- Filter campaigns by creation date range
- Format: `YYYY-MM-DD` (e.g., "2024-03-01")
- `dateFrom`: Include campaigns created on or after this date
- `dateTo`: Include campaigns created up to and including this date (adds 24 hours internally)
- Can use either parameter independently or both together
- All roles can use these filters

**cURL Example - Basic Search:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/campaigns?pageNo=1&rowPerPage=10" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Search by Campaign Name:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/campaigns?filter=ramadan&pageNo=1&rowPerPage=10" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Filter by Type and Status:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/campaigns?type=whatsapp&status=running&pageNo=1&rowPerPage=10" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Filter by Sender (All Roles):**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/campaigns?senderId=sender-123&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Filter by Date Range:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/campaigns?dateFrom=2024-03-01&dateTo=2024-03-31&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - ADMIN Filter by Client:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/campaigns?clientId=client-456&pageNo=1&rowPerPage=10" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

**cURL Example - Combined Filters:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/campaigns?filter=promo&type=whatsapp&status=completed&senderId=sender-123&dateFrom=2024-03-01&dateTo=2024-03-31&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": [
    {
      "id": "camp-12345",
      "clientId": "client-123",
      "senderId": "sender-123",
      "senderName": "My Business WhatsApp",
      "name": "Promo Ramadan 2024",
      "type": "whatsapp",
      "status": "running",
      "totalRecipients": 1000,
      "totalSent": 450,
      "totalDelivered": 420,
      "totalFailed": 10,
      "createdAt": "2024-03-10T10:30:00Z"
    },
    {
      "id": "camp-12346",
      "clientId": "client-123",
      "senderId": "sender-124",
      "senderName": "Customer Service",
      "name": "Info Pemeliharaan Jaringan",
      "type": "whatsapp",
      "status": "running",
      "totalRecipients": 500,
      "totalSent": 500,
      "totalDelivered": 485,
      "totalFailed": 5,
      "createdAt": "2024-03-12T14:00:00Z"
    }
  ],
  "currPage": 1,
  "haveNext": true,
  "totalPage": 25
}
```

**Search Behavior:**

**Filter Parameter Logic:**
- Empty or not provided: Returns all campaigns (with other filters applied)
- Provided: Searches campaign names containing the filter text
- Pattern: `name LIKE '%<filter>%'`
- Example searches:
  - `filter=2024` → Matches "Promo Ramadan 2024", "Campaign 2024"
  - `filter=promo` → Matches "Promo Ramadan", "Promo Akhir Tahun"
  - `filter=info` → Matches "Info Pemeliharaan", "Info Tagihan"

**Client Filtering Rules:**
- **ADMIN Role:**
  - `clientId` not provided → Shows campaigns from ALL clients
  - `clientId` provided → Shows campaigns from that specific client only
- **OWNER Role:**
  - `clientId` parameter is **ignored**
  - Always shows only campaigns from their own client
- **STAFF Role:**
  - `clientId` parameter is **ignored**
  - Always shows only campaigns from their client

**Pagination:**
- Results are paginated based on `pageNo` and `rowPerPage`
- `currPage`: Current page number
- `haveNext`: Boolean indicating if there are more pages
- `totalPage`: Total number of records matching search criteria

---

## 5. Delete Campaign

**Endpoint:** `DELETE /api/v1/instant-link/campaigns/:id`

**Authorization:** ADMIN, OWNER

**Description:** Delete a campaign. Only campaigns in 'draft' status or completed campaigns can be deleted.

**Headers:**
```
Authorization: Bearer <your-token>
```

**cURL Example:**
```bash
curl -X DELETE "http://localhost:8080/api/v1/instant-link/campaigns/camp-12345" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": true
}
```

**Error Responses:**

400 Bad Request:
```json
{
  "code": 400,
  "message": "Cannot delete running campaign"
}
```

404 Not Found:
```json
{
  "code": 404,
  "message": "Campaign not found"
}
```

---

# Campaign Recipients

## 6. Get Campaign Recipients

**Endpoint:** `GET /api/v1/instant-link/campaigns/:id/recipients`

**Authorization:** ADMIN, OWNER, STAFF

**Description:** Get list of recipients for a specific campaign with their delivery status.

**Headers:**
```
Authorization: Bearer <your-token>
```

**Query Parameters:**
- `pageNo` (optional): Page number (default: 1)
- `rowPerPage` (optional): Number of items per page (default: 10)
- `status` (optional): Filter by status (`pending`, `sent`, `delivered`, `read`, `failed`)

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/campaigns/camp-12345/recipients?pageNo=1&rowPerPage=20&status=sent" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": [
    {
      "id": "recipient-001",
      "campaignId": "camp-12345",
      "contactId": "contact-789",
      "contactName": "Budi Santoso",
      "recipientType": "whatsapp",
      "recipientValue": "628123456789",
      "status": "delivered",
      "messageId": "msg-abc123",
      "parameters": {
        "name": "Budi",
        "discount": "20%"
      },
      "sentAt": "2024-03-15T08:05:00Z",
      "deliveredAt": "2024-03-15T08:05:10Z",
      "retryCount": 0,
      "createdAt": "2024-03-15T07:00:00Z",
      "updatedAt": "2024-03-15T08:05:10Z"
    },
    {
      "id": "recipient-002",
      "campaignId": "camp-12345",
      "contactId": "contact-790",
      "contactName": "Siti Nurhaliza",
      "recipientType": "whatsapp",
      "recipientValue": "628123456790",
      "status": "read",
      "messageId": "msg-abc124",
      "sentAt": "2024-03-15T08:05:05Z",
      "deliveredAt": "2024-03-15T08:05:15Z",
      "readAt": "2024-03-15T08:10:00Z",
      "retryCount": 0,
      "createdAt": "2024-03-15T07:00:00Z",
      "updatedAt": "2024-03-15T08:10:00Z"
    }
  ],
  "paging": {
    "pageNo": 1,
    "haveNext": true,
    "count": 1000
  }
}
```

**Response Fields:**
- `id`: Unique recipient record ID
- `campaignId`: Campaign ID this recipient belongs to
- `contactId`: Contact ID from contacts database
- `contactName`: Contact name
- `recipientType`: Type of recipient (whatsapp, sms, email)
- `recipientValue`: Phone number or email address
- `status`: Delivery status (`pending`, `sent`, `delivered`, `read`, `failed`)
- `messageId`: Message ID from messaging service
- `parameters`: Personalized template parameters for this recipient
- `sentAt`: Timestamp when message was sent
- `deliveredAt`: Timestamp when message was delivered
- `readAt`: Timestamp when message was read
- `failedAt`: Timestamp when message failed
- `errorMessage`: Error description if failed
- `retryCount`: Number of retry attempts

---

## 7. Generate Campaign Recipients

**Endpoint:** `POST /api/v1/instant-link/campaigns/:id/generate-recipients`

**Authorization:** ADMIN, OWNER

**Description:** Generate recipient list based on campaign targets. This processes the targets defined during campaign creation and creates individual recipient records.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/campaigns/camp-12345/generate-recipients" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "success": true,
    "totalRecipients": 1000
  }
}
```

**Error Responses:**

400 Bad Request:
```json
{
  "code": 400,
  "message": "Recipients already generated for this campaign"
}
```

404 Not Found:
```json
{
  "code": 404,
  "message": "Campaign not found"
}
```

**Notes:**
- This endpoint must be called before executing a campaign
- It processes all targets defined in the campaign
- For `targetType: "group"`, it expands all contacts in the group
- For `targetType: "contact"`, it adds individual contacts
- For `targetType: "all"`, it includes all contacts in the client's database
- Recipients are created with `pending` status
- Can only be called once per campaign (unless recipients are cleared first)

---

# Campaign Statistics

## 8. Get Campaign Stats

**Endpoint:** `GET /api/v1/instant-link/campaigns/:id/stats`

**Authorization:** ADMIN, OWNER, STAFF

**Description:** Get detailed statistics and metrics for a campaign.

**Headers:**
```
Authorization: Bearer <your-token>
```

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/campaigns/camp-12345/stats" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "campaignId": "camp-12345",
    "totalRecipients": 1000,
    "totalSent": 950,
    "totalDelivered": 920,
    "totalFailed": 30,
    "totalRead": 680,
    "sentRate": 95.0,
    "deliveredRate": 96.84,
    "readRate": 73.91,
    "failedRate": 3.16
  }
}
```

**Response Fields:**
- `campaignId`: Campaign ID
- `totalRecipients`: Total number of recipients
- `totalSent`: Number of successfully sent messages
- `totalDelivered`: Number of delivered messages
- `totalFailed`: Number of failed messages
- `totalRead`: Number of read messages (for WhatsApp)
- `sentRate`: Percentage of sent messages (sent/recipients * 100)
- `deliveredRate`: Percentage of delivered messages (delivered/sent * 100)
- `readRate`: Percentage of read messages (read/delivered * 100)
- `failedRate`: Percentage of failed messages (failed/sent * 100)

**Error Responses:**

404 Not Found:
```json
{
  "code": 404,
  "message": "Campaign not found"
}
```

---

# Campaign Execution

## 9. Start Campaign

**Endpoint:** `POST /api/v1/instant-link/campaigns/:id/start`

**Authorization:** ADMIN, OWNER

**Description:** Start campaign execution. This will begin sending messages to all pending recipients. Campaign must be in 'draft' or 'scheduled' status and have recipients generated.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Prerequisites:**
- Campaign status must be `draft` or `scheduled`
- Recipients must be generated (totalRecipients > 0)
- Client must have sufficient balance

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/campaigns/camp-12345/start" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "camp-12345",
    "clientId": "client-123",
    "name": "Promo Ramadan 2024",
    "type": "whatsapp",
    "status": "running",
    "startedAt": "2024-03-15T08:00:05Z",
    "totalRecipients": 1000,
    "totalSent": 0,
    "batchSize": 100,
    "delaySeconds": 5,
    "updatedAt": "2024-03-15T08:00:05Z"
  }
}
```

**Error Responses:**

400 Bad Request - Invalid status:
```json
{
  "code": 400,
  "message": "Cannot start campaign with status: running. Only draft or scheduled campaigns can be started."
}
```

400 Bad Request - No recipients:
```json
{
  "code": 400,
  "message": "Campaign has no recipients. Please generate recipients first."
}
```

400 Bad Request - Insufficient balance:
```json
{
  "code": 400,
  "message": "Insufficient balance. Estimated cost: Rp 500000.00"
}
```

404 Not Found:
```json
{
  "code": 404,
  "message": "Campaign not found"
}
```

**Validation Checks:**
1. Campaign exists and belongs to authenticated client
2. Campaign status is `draft` or `scheduled`
3. Campaign has at least one recipient
4. Client has sufficient balance for all pending messages
5. Sender is active and configured

**Notes:**
- Once started, campaign status changes to `running`
- **Campaign Executor Worker** automatically processes the campaign in background
- Messages are sent in batches according to `batchSize` setting
- Delay of `delaySeconds` is applied between batches
- Balance is deducted as messages are sent
- Campaign can be paused during execution
- `startedAt` timestamp is recorded
- Worker checks for running campaigns every 10 seconds
- See `CAMPAIGN_EXECUTOR_WORKER.md` for worker details

---

## 10. Pause Campaign

**Endpoint:** `POST /api/v1/instant-link/campaigns/:id/pause`

**Authorization:** ADMIN, OWNER

**Description:** Pause a running campaign. This stops sending new messages but preserves the current state. Campaign can be resumed later.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Prerequisites:**
- Campaign status must be `running`

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/campaigns/camp-12345/pause" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "camp-12345",
    "clientId": "client-123",
    "name": "Promo Ramadan 2024",
    "type": "whatsapp",
    "status": "paused",
    "startedAt": "2024-03-15T08:00:05Z",
    "totalRecipients": 1000,
    "totalSent": 450,
    "totalDelivered": 420,
    "totalFailed": 10,
    "batchSize": 100,
    "delaySeconds": 5,
    "updatedAt": "2024-03-15T08:30:00Z"
  }
}
```

**Error Responses:**

400 Bad Request - Invalid status:
```json
{
  "code": 400,
  "message": "Cannot pause campaign with status: draft. Only running campaigns can be paused."
}
```

404 Not Found:
```json
{
  "code": 404,
  "message": "Campaign not found"
}
```

**Validation Checks:**
1. Campaign exists and belongs to authenticated client
2. Campaign status is `running`

**Notes:**
- Campaign status changes to `paused`
- Messages already in flight will complete
- Pending recipients remain in `pending` status
- Campaign progress is preserved
- Can be resumed using Resume Campaign endpoint
- Useful for:
  - Temporary suspension due to issues
  - Updating message content or parameters
  - Resolving balance issues
  - Handling rate limit problems

**Use Cases:**
- Emergency stop when errors detected
- Rate limit management
- Budget control
- Time-based suspension (e.g., pause overnight)

---

## 11. Resume Campaign

**Endpoint:** `POST /api/v1/instant-link/campaigns/:id/resume`

**Authorization:** ADMIN, OWNER

**Description:** Resume a paused campaign. Continues sending messages from where it was paused.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Prerequisites:**
- Campaign status must be `paused`
- Client must have sufficient balance for remaining messages

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/campaigns/camp-12345/resume" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "camp-12345",
    "clientId": "client-123",
    "name": "Promo Ramadan 2024",
    "type": "whatsapp",
    "status": "running",
    "startedAt": "2024-03-15T08:00:05Z",
    "totalRecipients": 1000,
    "totalSent": 450,
    "totalDelivered": 420,
    "totalFailed": 10,
    "batchSize": 100,
    "delaySeconds": 5,
    "updatedAt": "2024-03-15T09:00:00Z"
  }
}
```

**Error Responses:**

400 Bad Request - Invalid status:
```json
{
  "code": 400,
  "message": "Cannot resume campaign with status: running. Only paused campaigns can be resumed."
}
```

400 Bad Request - Insufficient balance:
```json
{
  "code": 400,
  "message": "Insufficient balance. Estimated cost: Rp 275000.00"
}
```

404 Not Found:
```json
{
  "code": 404,
  "message": "Campaign not found"
}
```

**Validation Checks:**
1. Campaign exists and belongs to authenticated client
2. Campaign status is `paused`
3. Client has sufficient balance for remaining pending messages

**Notes:**
- Campaign status changes back to `running`
- Resumes from exact point where it was paused
- Only sends to recipients still in `pending` status
- Balance is checked again before resuming
- Same batch size and delay settings apply
- `startedAt` timestamp remains unchanged

**Balance Calculation:**
- Only counts `pending` recipients
- Excludes already `sent`, `delivered`, or `failed` messages
- Cost = (Pending Count) × (Message Cost per Type)

---

## Authorization & Roles

### Role-Based Access Control

| Endpoint | ADMIN | OWNER | STAFF |
|----------|-------|-------|-------|
| `POST /campaigns` | ✅ | ✅ | ❌ |
| `PUT /campaigns/:id` | ✅ | ✅ | ❌ |
| `GET /campaigns/:id` | ✅ | ✅ | ✅ |
| `GET /campaigns` | ✅ | ✅ | ✅ |
| `DELETE /campaigns/:id` | ✅ | ✅ | ❌ |
| `GET /campaigns/:id/recipients` | ✅ | ✅ | ✅ |
| `POST /campaigns/:id/generate-recipients` | ✅ | ✅ | ❌ |
| `GET /campaigns/:id/stats` | ✅ | ✅ | ✅ |
| `POST /campaigns/:id/start` | ✅ | ✅ | ❌ |
| `POST /campaigns/:id/pause` | ✅ | ✅ | ❌ |
| `POST /campaigns/:id/resume` | ✅ | ✅ | ❌ |

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

## Campaign Statuses

| Status | Description |
|--------|-------------|
| `draft` | Campaign created but not scheduled |
| `scheduled` | Campaign scheduled for future execution |
| `running` | Campaign is currently being executed |
| `paused` | Campaign execution paused |
| `completed` | Campaign execution completed |
| `cancelled` | Campaign cancelled by user |

---

## Campaign Types

| Type | Description |
|------|-------------|
| `whatsapp` | WhatsApp Business messages |
| `sms` | SMS messages |
| `email` | Email messages |

---

## Common Error Codes

| Status Code | Description |
|-------------|-------------|
| 400 | Bad Request - Invalid input or missing required fields |
| 401 | Unauthorized - Invalid or missing authentication token |
| 403 | Forbidden - User doesn't have permission for this operation |
| 404 | Not Found - Campaign not found |
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

**Paginated Response:**
```json
{
  "code": 0,
  "messages": "Success",
  "data": [ ... ],
  "paging": {
    "pageNo": 1,
    "haveNext": true,
    "count": 100
  }
}
```

---

## Integration Examples

### Example 1: Create and Execute WhatsApp Campaign

```bash
#!/bin/bash

TOKEN="YOUR_TOKEN_HERE"
BASE_URL="http://localhost:8080/api/v1/instant-link"

# Step 1: Create campaign
CAMPAIGN_ID=$(curl -X POST "$BASE_URL/campaigns" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Promo Akhir Tahun",
    "type": "whatsapp",
    "senderId": "sender-123",
    "templateId": "template-456",
    "batchSize": 50,
    "delaySeconds": 2,
    "parameters": {
      "promo": "Diskon 30%",
      "periode": "1-31 Desember"
    },
    "targets": [
      {
        "targetType": "group",
        "targetId": "group-vip-customers"
      }
    ]
  }' | jq -r '.data.id')

echo "Campaign created: $CAMPAIGN_ID"

# Step 2: Generate recipients
curl -X POST "$BASE_URL/campaigns/$CAMPAIGN_ID/generate-recipients" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"

# Step 3: Check stats
curl -X GET "$BASE_URL/campaigns/$CAMPAIGN_ID/stats" \
  -H "Authorization: Bearer $TOKEN"

# Step 4: Start campaign
curl -X POST "$BASE_URL/campaigns/$CAMPAIGN_ID/start" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"

echo "Campaign started successfully!"
```

### Example 2: Pause and Resume Campaign

```bash
#!/bin/bash

TOKEN="YOUR_TOKEN_HERE"
CAMPAIGN_ID="camp-12345"
BASE_URL="http://localhost:8080/api/v1/instant-link"

# Pause the running campaign
echo "Pausing campaign..."
curl -X POST "$BASE_URL/campaigns/$CAMPAIGN_ID/pause" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"

echo "Campaign paused. Checking status..."

# Check current status
curl -X GET "$BASE_URL/campaigns/$CAMPAIGN_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.data | {status, totalSent, totalRecipients}'

# Wait for some time or perform other operations
echo "Waiting 30 seconds before resuming..."
sleep 30

# Resume the campaign
echo "Resuming campaign..."
curl -X POST "$BASE_URL/campaigns/$CAMPAIGN_ID/resume" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"

echo "Campaign resumed successfully!"
```

### Example 3: Monitor Campaign Progress

```bash
#!/bin/bash

TOKEN="YOUR_TOKEN_HERE"
CAMPAIGN_ID="camp-12345"
BASE_URL="http://localhost:8080/api/v1/instant-link"

# Check campaign status every 30 seconds
while true; do
  echo "=== Campaign Status $(date) ==="

  # Get campaign details
  STATUS=$(curl -s -X GET "$BASE_URL/campaigns/$CAMPAIGN_ID" \
    -H "Authorization: Bearer $TOKEN" | jq -r '.data.status')

  echo "Status: $STATUS"

  # Get statistics
  curl -s -X GET "$BASE_URL/campaigns/$CAMPAIGN_ID/stats" \
    -H "Authorization: Bearer $TOKEN" | jq '.data'

  # Break if completed
  if [ "$STATUS" == "completed" ]; then
    echo "Campaign completed!"
    break
  fi

  sleep 30
done
```

### Example 4: Get Failed Recipients

```bash
#!/bin/bash

TOKEN="YOUR_TOKEN_HERE"
CAMPAIGN_ID="camp-12345"
BASE_URL="http://localhost:8080/api/v1/instant-link"

# Get all failed recipients
curl -X GET "$BASE_URL/campaigns/$CAMPAIGN_ID/recipients?status=failed&rowPerPage=100" \
  -H "Authorization: Bearer $TOKEN" | jq '.data[] | {
    name: .contactName,
    phone: .recipientValue,
    error: .errorMessage,
    retries: .retryCount
  }'
```

### Example 5: Scheduled Campaign with Node.js

```javascript
const axios = require('axios');

const API_URL = 'http://localhost:8080/api/v1/instant-link';
const TOKEN = 'YOUR_TOKEN_HERE';

async function createScheduledCampaign() {
  try {
    // Schedule campaign for tomorrow 9 AM
    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    tomorrow.setHours(9, 0, 0, 0);

    const response = await axios.post(
      `${API_URL}/campaigns`,
      {
        name: 'Morning Greeting Campaign',
        type: 'whatsapp',
        senderId: 'sender-123',
        templateId: 'template-morning-greeting',
        scheduledAt: tomorrow.toISOString(),
        batchSize: 100,
        delaySeconds: 5,
        parameters: {
          greeting: 'Selamat Pagi',
          company: 'PT. Example'
        },
        targets: [
          {
            targetType: 'all',
            targetId: ''
          }
        ]
      },
      {
        headers: {
          'Authorization': `Bearer ${TOKEN}`,
          'Content-Type': 'application/json'
        }
      }
    );

    const campaignId = response.data.data.id;
    console.log('Campaign created:', campaignId);

    // Generate recipients
    const recipientsResp = await axios.post(
      `${API_URL}/campaigns/${campaignId}/generate-recipients`,
      {},
      {
        headers: {
          'Authorization': `Bearer ${TOKEN}`,
          'Content-Type': 'application/json'
        }
      }
    );

    console.log('Recipients generated:', recipientsResp.data.data.totalRecipients);

    // Set campaign to scheduled status
    await axios.put(
      `${API_URL}/campaigns/${campaignId}`,
      { status: 'scheduled' },
      {
        headers: {
          'Authorization': `Bearer ${TOKEN}`,
          'Content-Type': 'application/json'
        }
      }
    );

    console.log('Campaign scheduled successfully');

  } catch (error) {
    console.error('Error:', error.response?.data || error.message);
  }
}

createScheduledCampaign();
```

### Example 6: Campaign with Image Header Template

**Scenario:** Send campaign with template that has IMAGE header but NO body variables.

**Template Structure:**
- Header: IMAGE
- Body: "Check out our latest promotion! Visit us now."
- Footer: "Valid until end of month"
- No variables ({{1}}, {{2}}) in body

**⚠️ Common Mistake:**
```bash
# ❌ WRONG - This will cause error: "Number of parameters does not match"
curl -X POST "$BASE_URL/campaigns" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Promo Campaign",
    "type": "whatsapp",
    "senderId": "sender-123",
    "templateId": "template-with-image-only",
    "batchSize": 100,
    "delaySeconds": 5,
    "parameters": {
      "image": "https://example.com/promo.jpg",
      "param1": "John Doe"  // ❌ ERROR! Template has no {{1}} in body
    },
    "targets": [{"targetType": "all", "targetId": ""}]
  }'
```

**✅ Correct Way:**
```bash
#!/bin/bash

TOKEN="YOUR_TOKEN_HERE"
BASE_URL="http://localhost:8080/api/v1/instant-link"

# Step 1: Upload image (optional - if you want to use internal storage)
IMAGE_URL=$(curl -X POST "$BASE_URL/wa-send/upload-image" \
  -H "Authorization: Bearer $TOKEN" \
  -F "image=@promo-banner.jpg" | jq -r '.data')

echo "Image uploaded: $IMAGE_URL"

# Step 2: Create campaign with ONLY image parameter (no text parameters)
CAMPAIGN_ID=$(curl -X POST "$BASE_URL/campaigns" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Promo Image Campaign",
    "type": "whatsapp",
    "senderId": "sender-123",
    "templateId": "template-with-image-only",
    "batchSize": 100,
    "delaySeconds": 5,
    "parameters": {
      "image": "'$IMAGE_URL'"
    },
    "targets": [
      {
        "targetType": "group",
        "targetId": "group-customers"
      }
    ]
  }' | jq -r '.data.id')

echo "Campaign created: $CAMPAIGN_ID"

# Step 3: Generate recipients
curl -X POST "$BASE_URL/campaigns/$CAMPAIGN_ID/generate-recipients" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"

# Step 4: Start campaign
curl -X POST "$BASE_URL/campaigns/$CAMPAIGN_ID/start" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"

echo "Campaign with image header started!"
```

### Example 7: Campaign with Image Header AND Body Variables

**Template Structure:**
- Header: IMAGE
- Body: "Hi {{1}}, special discount {{2}} for you!"
- Footer: "Valid until {{3}}"

**✅ Correct Parameters:**
```bash
#!/bin/bash

TOKEN="YOUR_TOKEN_HERE"
BASE_URL="http://localhost:8080/api/v1/instant-link"

# Create campaign with image + text parameters
curl -X POST "$BASE_URL/campaigns" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Personalized Promo",
    "type": "whatsapp",
    "senderId": "sender-123",
    "templateId": "template-with-image-and-text",
    "batchSize": 50,
    "delaySeconds": 3,
    "parameters": {
      "image": "https://example.com/promo.jpg",
      "param1": "Customer Name",
      "param2": "30%",
      "param3": "31 Dec 2025"
    },
    "targets": [
      {
        "targetType": "group",
        "targetId": "group-vip"
      }
    ]
  }'
```

**📌 Key Points:**
1. **Image parameter** goes to header
2. **param1, param2, param3** go to body variables {{1}}, {{2}}, {{3}}
3. Order matters: param1 → {{1}}, param2 → {{2}}, etc.

### Example 8: Upload to WhatsApp Media API (Most Reliable)

```bash
#!/bin/bash

TOKEN="YOUR_TOKEN_HERE"
WA_ACCESS_TOKEN="YOUR_WHATSAPP_ACCESS_TOKEN"
PHONE_NUMBER_ID="YOUR_PHONE_NUMBER_ID"
BASE_URL="http://localhost:8080/api/v1/instant-link"

# Step 1: Upload media to WhatsApp and get Media ID
MEDIA_ID=$(curl -X POST "https://graph.facebook.com/v24.0/$PHONE_NUMBER_ID/media" \
  -H "Authorization: Bearer $WA_ACCESS_TOKEN" \
  -F "file=@promo-banner.jpg" \
  -F "type=image/jpeg" \
  -F "messaging_product=whatsapp" | jq -r '.id')

echo "Media uploaded to WhatsApp, ID: $MEDIA_ID"

# Step 2: Create campaign using WhatsApp Media ID
curl -X POST "$BASE_URL/campaigns" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Promo with WhatsApp Media",
    "type": "whatsapp",
    "senderId": "sender-123",
    "templateId": "template-with-image",
    "batchSize": 100,
    "delaySeconds": 5,
    "parameters": {
      "image_id": "'$MEDIA_ID'"
    },
    "targets": [
      {
        "targetType": "all",
        "targetId": ""
      }
    ]
  }'

echo "Campaign created with WhatsApp Media ID"
```

**Advantages of using Media ID:**
- ✅ Most reliable (no URL accessibility issues)
- ✅ Faster delivery
- ✅ No bandwidth cost from your server
- ✅ Can reuse Media ID for 30 days

---

## Best Practices

1. **Campaign Planning:**
   - Test templates with a small group first
   - Use appropriate batch sizes to avoid rate limiting
   - Set reasonable delays between batches
   - Schedule campaigns during optimal hours

2. **Recipient Management:**
   - Always generate recipients before starting campaign
   - Verify recipient count matches expectations
   - Clean contact lists regularly
   - Use groups for better organization

3. **Monitoring:**
   - Monitor campaign stats during execution
   - Check failed recipients and retry if needed
   - Track read rates to measure engagement
   - Keep logs of campaign performance

4. **Performance:**
   - Use batch sizes between 50-200 for optimal performance
   - Set delays of 2-5 seconds between batches
   - Avoid running multiple large campaigns simultaneously
   - Schedule heavy campaigns during off-peak hours

5. **Error Handling:**
   - Always check response codes
   - Implement retry logic for failed messages
   - Log errors for analysis
   - Monitor failed rate threshold

---

## Troubleshooting

### Issue: "Recipients already generated" Error

**Solution:**
- Delete and recreate the campaign if you need to change targets
- Or update the database directly to clear recipients (advanced)

### Issue: High Failed Rate

**Solution:**
- Verify sender is active and approved
- Check template is approved by WhatsApp
- Validate phone numbers format
- Check rate limits with WhatsApp provider
- Review error messages in failed recipients

### Issue: Campaign Stuck in "Running" Status

**Solution:**
- Check application logs for errors
- Verify message queue is processing
- Check database connections
- Monitor system resources (CPU, memory)
- Contact system administrator

### Issue: Recipients Not Generated

**Solution:**
- Verify targets exist (groups, contacts)
- Check client has contacts in database
- Review application logs
- Ensure user has permission for the targets

---

## Support

For additional support or questions:
- Check API logs for detailed error messages
- Review campaign and recipient records in database
- Test with small campaigns first
- Verify templates are approved before large campaigns