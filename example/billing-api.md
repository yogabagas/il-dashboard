# Billing API Documentation

Base URL: `/api/v1/instant-link/billing`

**Authorization Required:** All endpoints require authentication with role `ADMIN`, `OWNER`, or `STAFF`

**Important Notes:**
- **ADMIN** users can view billing information for all clients
- **OWNER** users can only view billing information for their own client (based on `authInfo.ClientId`)
- **STAFF** users can view billing information for their client
- Only **ADMIN** and **OWNER** can create transactions and topup balance

---

## 1. Get Client Billing Info

**Endpoint:** `GET /api/v1/instant-link/billing/info`

**Authorization:** ADMIN, OWNER, STAFF

**Description:** Retrieve current billing information for the authenticated client, including balance, credit limit, total usage, and account status.

**Headers:**
```
Authorization: Bearer <your-token>
```

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/billing/info" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK) - Prepaid Account:**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "clientId": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
    "clientName": "PT ABC Indonesia",
    "billingType": "prepaid",
    "balance": 500000.00,
    "creditLimit": 0,
    "remainingBalance": 500000.00,
    "totalUsage": 125000.00,
    "canSend": true,
    "status": "active"
  }
}
```

**Success Response (200 OK) - Postpaid Account:**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "clientId": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
    "clientName": "PT XYZ Indonesia",
    "billingType": "postpaid",
    "balance": -75000.00,
    "creditLimit": 1000000.00,
    "remainingBalance": 925000.00,
    "totalUsage": 75000.00,
    "canSend": true,
    "status": "active"
  }
}
```

**Response Fields:**
- `clientId`: Client unique identifier
- `clientName`: Client business name
- `billingType`: Account type (`prepaid` or `postpaid`)
- `balance`: Current balance (positive for prepaid, negative for postpaid)
- `creditLimit`: Credit limit (for postpaid accounts)
- `remainingBalance`: Available balance to use
- `totalUsage`: Total accumulated usage amount
- `canSend`: Boolean indicating if client can send messages
- `status`: Account status (`active`, `insufficient`, `overlimit`)

**Account Status:**
- `active`: Account is in good standing and can send messages
- `insufficient`: Prepaid account with balance ≤ 0
- `overlimit`: Postpaid account exceeding credit limit

**Error Responses:**

404 Not Found - Client not found:
```json
{
  "code": 404,
  "message": "Not found"
}
```

---

## 2. Create Billing Transaction

**Endpoint:** `POST /api/v1/instant-link/billing/transactions`

**Authorization:** ADMIN, OWNER

**Description:** Create a manual billing transaction (topup, usage, payment, refund, or adjustment). This is typically used for manual adjustments or corrections.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "type": "topup",
  "amount": 100000.00,
  "description": "Initial balance topup"
}
```

**Request Fields:**
- `type` (required): Transaction type - `topup`, `usage`, `payment`, `refund`, `adjustment`
- `amount` (required): Transaction amount (must be positive)
- `description` (optional): Transaction description or notes
- `referenceId` (optional): External reference ID
- `referenceType` (optional): Reference type (e.g., "invoice", "order")

**Transaction Types:**
- `topup`: Add balance to account
- `usage`: Deduct balance for service usage
- `payment`: Payment received from client
- `refund`: Refund to client
- `adjustment`: Manual balance adjustment

**cURL Example - Topup:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/billing/transactions" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "topup",
    "amount": 500000.00,
    "description": "Balance topup - Invoice INV-2025-001"
  }'
```

**cURL Example - Refund:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/billing/transactions" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "refund",
    "amount": 10000.00,
    "description": "Refund for failed campaign",
    "referenceId": "campaign-789",
    "referenceType": "campaign"
  }'
```

**cURL Example - Adjustment:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/billing/transactions" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "adjustment",
    "amount": 5000.00,
    "description": "Balance correction - accounting adjustment"
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01HZQK8A1Y2B5Q4S6U7W9Z1C3E",
    "clientId": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
    "type": "topup",
    "amount": 500000.00,
    "balanceBefore": 0.00,
    "balanceAfter": 500000.00,
    "referenceId": "",
    "referenceType": "",
    "description": "Balance topup - Invoice INV-2025-001",
    "createdBy": "user-123",
    "createdAt": "2025-11-28T10:00:00Z"
  }
}
```

**Error Responses:**

400 Bad Request - Invalid type:
```json
{
  "code": 400,
  "message": "Invalid transaction type"
}
```

400 Bad Request - Invalid amount:
```json
{
  "code": 400,
  "message": "Amount must be positive"
}
```

404 Not Found - Client not found:
```json
{
  "code": 404,
  "message": "Not found"
}
```

---

## 3. Get Transaction by ID

**Endpoint:** `GET /api/v1/instant-link/billing/transactions/:id`

**Authorization:** ADMIN, OWNER, STAFF

**Description:** Retrieve a specific billing transaction by its ID.

**Headers:**
```
Authorization: Bearer <your-token>
```

**URL Parameters:**
- `id` (required): Transaction ID

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/billing/transactions/01HZQK8A1Y2B5Q4S6U7W9Z1C3E" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01HZQK8A1Y2B5Q4S6U7W9Z1C3E",
    "clientId": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
    "type": "usage",
    "amount": 350.00,
    "balanceBefore": 500000.00,
    "balanceAfter": 499650.00,
    "referenceId": "msg-log-abc123",
    "referenceType": "message_log",
    "description": "WhatsApp message - UTILITY category",
    "createdBy": "system",
    "createdAt": "2025-11-28T10:30:00Z"
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

403 Forbidden - OWNER trying to access transaction from different client:
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

---

## 4. Search Billing Transactions

**Endpoint:** `GET /api/v1/instant-link/billing/transactions`

**Authorization:** ADMIN, OWNER, STAFF

**Description:** Search and filter billing transactions with pagination. Supports filtering by transaction type and date range.

**Headers:**
```
Authorization: Bearer <your-token>
```

**Query Parameters:**
- `type` (optional): Filter by transaction type (`topup`, `usage`, `payment`, `refund`, `adjustment`)
- `dateFrom` (optional): Filter by created date from (format: YYYY-MM-DD)
- `dateTo` (optional): Filter by created date to (format: YYYY-MM-DD)
- `pageNo` (optional): Page number (default: 1)
- `rowPerPage` (optional): Items per page (default: 20)

**cURL Example - All Transactions:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/billing/transactions?pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Filter by Type:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/billing/transactions?type=usage&pageNo=1&rowPerPage=50" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Filter by Date Range:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/billing/transactions?dateFrom=2025-11-01&dateTo=2025-11-30&pageNo=1&rowPerPage=100" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Topup Transactions:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/billing/transactions?type=topup&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": [
    {
      "id": "01HZQK8A1Y2B5Q4S6U7W9Z1C3E",
      "clientId": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
      "type": "topup",
      "amount": 500000.00,
      "balanceBefore": 0.00,
      "balanceAfter": 500000.00,
      "description": "Balance topup - Invoice INV-2025-001",
      "createdBy": "admin-user-123",
      "createdAt": "2025-11-28T09:00:00Z"
    },
    {
      "id": "01HZQK9B2Z3C6R5T7V8X0A2D4F",
      "clientId": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
      "type": "usage",
      "amount": 600.00,
      "balanceBefore": 500000.00,
      "balanceAfter": 499400.00,
      "referenceId": "msg-log-xyz789",
      "referenceType": "message_log",
      "description": "WhatsApp message - MARKETING category",
      "createdBy": "system",
      "createdAt": "2025-11-28T10:15:00Z"
    },
    {
      "id": "01HZQKAC3A4D7S6U8W9Y1B3E5G",
      "clientId": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
      "type": "usage",
      "amount": 350.00,
      "balanceBefore": 499400.00,
      "balanceAfter": 499050.00,
      "referenceId": "msg-log-abc456",
      "referenceType": "message_log",
      "description": "WhatsApp message - UTILITY category",
      "createdBy": "system",
      "createdAt": "2025-11-28T10:30:00Z"
    }
  ],
  "currPage": 1,
  "haveNext": true,
  "totalPage": 125
}
```

**Response Fields:**
- `code`: Status code (0 = success)
- `messages`: Status message
- `data`: Array of transaction entries
- `currPage`: Current page number
- `haveNext`: Boolean indicating if there are more pages
- `totalPage`: Total number of records matching the search

---

## 5. Topup Balance

**Endpoint:** `POST /api/v1/instant-link/billing/topup`

**Authorization:** ADMIN, OWNER

**Description:** Quick topup endpoint to add balance to the authenticated client's account. This creates a `topup` transaction automatically.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "amount": 1000000.00,
  "description": "Monthly balance topup"
}
```

**Request Fields:**
- `amount` (required): Topup amount (must be positive)
- `description` (optional): Topup description or notes

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/billing/topup" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 1000000.00,
    "description": "Monthly balance topup - November 2025"
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01HZQKBD4B5E8T7V9X0Z2C4F6H",
    "clientId": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
    "type": "topup",
    "amount": 1000000.00,
    "balanceBefore": 499050.00,
    "balanceAfter": 1499050.00,
    "description": "Monthly balance topup - November 2025",
    "createdBy": "user-123",
    "createdAt": "2025-11-28T11:00:00Z"
  }
}
```

**Error Responses:**

400 Bad Request - Invalid amount:
```json
{
  "code": 400,
  "message": "Amount must be positive"
}
```

404 Not Found - Client not found:
```json
{
  "code": 404,
  "message": "Not found"
}
```

---

## 6. Get Usage Summary

**Endpoint:** `GET /api/v1/instant-link/billing/usage-summary`

**Authorization:** ADMIN, OWNER, STAFF

**Description:** Get usage summary and statistics for a specific period (month). Provides breakdown of message costs by type and category.

**Headers:**
```
Authorization: Bearer <your-token>
```

**Query Parameters:**
- `period` (required): Period in format YYYY-MM (e.g., "2025-11")

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/billing/usage-summary?period=2025-11" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "period": "2025-11",
    "clientId": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
    "clientName": "PT ABC Indonesia",
    "totalUsage": 125750.00,
    "totalTopup": 1500000.00,
    "balanceStart": 0.00,
    "balanceEnd": 1374250.00,
    "messageCount": {
      "whatsapp": 350,
      "sms": 25,
      "email": 100
    },
    "messageCost": {
      "whatsapp": {
        "marketing": 72000.00,
        "utility": 52500.00,
        "authentication": 1750.00
      },
      "sms": 8750.00,
      "email": 5000.00
    },
    "breakdown": [
      {
        "date": "2025-11-01",
        "totalCost": 4500.00,
        "messageCount": 10
      },
      {
        "date": "2025-11-02",
        "totalCost": 5250.00,
        "messageCount": 12
      }
    ]
  }
}
```

**Response Fields:**
- `period`: The requested period (YYYY-MM)
- `clientId`: Client identifier
- `clientName`: Client business name
- `totalUsage`: Total usage cost for the period
- `totalTopup`: Total topup amount for the period
- `balanceStart`: Balance at the start of period
- `balanceEnd`: Balance at the end of period
- `messageCount`: Message count breakdown by type
- `messageCost`: Cost breakdown by type and category
- `breakdown`: Daily breakdown of costs

**Error Responses:**

400 Bad Request - Missing period:
```json
{
  "code": 400,
  "message": "Period is required (format: YYYY-MM)"
}
```

---

## Billing Transaction Fields Description

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique transaction identifier (ULID) |
| `clientId` | string | Client identifier who owns this transaction |
| `type` | string | Transaction type: `topup`, `usage`, `payment`, `refund`, `adjustment` |
| `amount` | float | Transaction amount (always positive) |
| `balanceBefore` | float | Account balance before this transaction |
| `balanceAfter` | float | Account balance after this transaction |
| `referenceId` | string | External reference ID (e.g., message log ID, invoice ID) |
| `referenceType` | string | Type of reference (e.g., `message_log`, `invoice`, `campaign`) |
| `description` | string | Human-readable transaction description |
| `createdBy` | string | User ID who created this transaction |
| `createdAt` | string | ISO8601 timestamp when transaction was created |

---

## Message Pricing (WhatsApp)

WhatsApp messages are priced based on template category:

| Category | Price (IDR) | Description |
|----------|-------------|-------------|
| **MARKETING** | 600 | Promotional messages, product updates, offers |
| **UTILITY** | 350 | Transactional messages, order confirmations, notifications |
| **AUTHENTICATION** | 175 | OTP codes, verification messages |

**Other Services:**
- **SMS**: Rp 350 per message
- **Email**: Rp 50 per message

**Note:** Prices are subject to change. Use the billing info endpoint to get current pricing.

---

## Common Use Cases

### Check Current Balance
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/billing/info" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

### Add Balance
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/billing/topup" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"amount": 500000, "description": "Balance topup"}'
```

### View Monthly Usage
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/billing/usage-summary?period=2025-11" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

### View All Usage Transactions
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/billing/transactions?type=usage&dateFrom=2025-11-01&dateTo=2025-11-30" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

### View Transaction History
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/billing/transactions?pageNo=1&rowPerPage=50" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

## Error Codes Summary

| Code | Message | Description |
|------|---------|-------------|
| 0 | Success | Request successful |
| 400 | Bad Request | Invalid request parameters |
| 401 | Unauthorized | Missing or invalid authentication token |
| 403 | Forbidden | User doesn't have permission to access resource |
| 404 | Not found | Resource not found |
| 500 | Internal Server Error | Server error occurred |

---

## Best Practices

1. **Monitor Balance Regularly**
   - Check balance before sending bulk campaigns
   - Set up alerts for low balance

2. **Use Usage Summary**
   - Review monthly usage to understand costs
   - Identify cost optimization opportunities

3. **Track Transactions**
   - Keep referenceId for all transactions
   - Use descriptive descriptions for easier tracking

4. **Prepaid vs Postpaid**
   - Prepaid: Must maintain positive balance
   - Postpaid: Monitor usage against credit limit

5. **Category Selection**
   - Choose appropriate template category for better pricing
   - UTILITY is default and most cost-effective for transactional messages
   - Use MARKETING only for promotional content
   - Use AUTHENTICATION for OTP/verification codes

---

**Last Updated:** November 28, 2025  
**API Version:** v1

