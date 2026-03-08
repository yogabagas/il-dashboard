# Pricing API Documentation

Base URL: `/api/v1/instant-link/pricing`

**Authorization Required:** All endpoints require authentication

**Important Notes:**
- **ADMIN** and **OWNER** users can view pricing configurations
- Only **ADMIN** users can create, update, or delete pricing entries
- Pricing is configurable per service type (whatsapp, sms, email) and category
- Supports default pricing (client_id = NULL) and client-specific pricing
- Changes to pricing affect future message cost calculations immediately

**Access Control (Multi-Tenancy):**
- **PROVIDER (Super Admin)**: Can manage all pricing entries (default and client-specific)
- **Non-PROVIDER users (Regular Clients)**:
  - Can only create/view/update/delete pricing for their own `clientId`
  - Cannot access default pricing (`clientId = NULL`) or other clients' pricing
  - `clientId` is automatically enforced based on authenticated user
  - Attempts to access other clients' pricing will return 403 Forbidden

---

## Overview

The Pricing API allows administrators to manage message pricing configurations dynamically without requiring code deployment. This enables flexible pricing strategies and client-specific pricing models.

**Pricing Model:**
- Each pricing entry defines the cost for a specific service type and category
- WhatsApp pricing supports categories: `marketing`, `utility`, `authentication`
- SMS and Email pricing use empty category (single price point)
- Client-specific pricing overrides default pricing when available

---

## 1. Search Pricing

**Endpoint:** `GET /api/v1/instant-link/pricing`

**Authorization:** ADMIN, OWNER

**Description:** Search and filter pricing configurations with pagination. Supports filtering by service type, category, status, and general text search.

**Headers:**
```
Authorization: Bearer <your-token>
```

**Query Parameters:**
- `filter` (optional): General text filter (searches in service_type and category)
- `serviceType` (optional): Filter by service type (`whatsapp`, `sms`, `email`)
- `category` (optional): Filter by category (`marketing`, `utility`, `authentication`)
- `status` (optional): Filter by status (`active`, `inactive`)
- `clientId` (optional): Filter by client ID (only for PROVIDER users; ignored for regular clients)
- `pageNo` (optional): Page number (default: 1)
- `rowPerPage` (optional): Items per page (default: 20)

**Access Control:**
- PROVIDER users can search all pricing entries
- Non-PROVIDER users automatically see only their own pricing (clientId filter is enforced)

**cURL Example - All Pricing (First Page):**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/pricing?pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Filter by Service Type:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/pricing?serviceType=whatsapp&pageNo=1&rowPerPage=10" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Filter by Category:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/pricing?serviceType=whatsapp&category=marketing&pageNo=1" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Filter by Status:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/pricing?status=active&pageNo=1&rowPerPage=50" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - General Text Search:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/pricing?filter=marketing&pageNo=1" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": [
    {
      "id": "01HZQK1A2B3C4D5E6F7G8H9J0K",
      "clientId": null,
      "serviceType": "whatsapp",
      "category": "marketing",
      "cost": 600.00,
      "currency": "IDR",
      "isActive": true,
      "createdAt": "2025-11-28T10:00:00Z",
      "updatedAt": "2025-11-28T10:00:00Z"
    },
    {
      "id": "01HZQK2B3C4D5E6F7G8H9J0K1L",
      "clientId": null,
      "serviceType": "whatsapp",
      "category": "utility",
      "cost": 380.00,
      "currency": "IDR",
      "isActive": true,
      "createdAt": "2025-11-28T10:00:00Z",
      "updatedAt": "2025-11-28T10:00:00Z"
    },
    {
      "id": "01HZQK3C4D5E6F7G8H9J0K1L2M",
      "clientId": null,
      "serviceType": "whatsapp",
      "category": "authentication",
      "cost": 200.00,
      "currency": "IDR",
      "isActive": true,
      "createdAt": "2025-11-28T10:00:00Z",
      "updatedAt": "2025-11-28T10:00:00Z"
    },
    {
      "id": "01HZQK4D5E6F7G8H9J0K1L2M3N",
      "clientId": null,
      "serviceType": "sms",
      "category": "",
      "cost": 350.00,
      "currency": "IDR",
      "isActive": true,
      "createdAt": "2025-11-28T10:00:00Z",
      "updatedAt": "2025-11-28T10:00:00Z"
    },
    {
      "id": "01HZQK5E6F7G8H9J0K1L2M3N4O",
      "clientId": null,
      "serviceType": "email",
      "category": "",
      "cost": 50.00,
      "currency": "IDR",
      "isActive": true,
      "createdAt": "2025-11-28T10:00:00Z",
      "updatedAt": "2025-11-28T10:00:00Z"
    }
  ],
  "currPage": 1,
  "haveNext": false,
  "totalPage": 5
}
```

**Response Fields:**
- `code`: Status code (0 = success)
- `messages`: Status message
- `data`: Array of pricing entries
  - `id`: Unique pricing identifier (ULID)
  - `clientId`: Client ID for client-specific pricing, `null` for default pricing
  - `serviceType`: Service type (`whatsapp`, `sms`, `email`)
  - `category`: Message category (for WhatsApp: `marketing`, `utility`, `authentication`; empty for SMS/Email)
  - `cost`: Cost per message in specified currency
  - `currency`: Currency code (default: `IDR`)
  - `isActive`: Whether this pricing is currently active
  - `createdAt`: ISO8601 timestamp when pricing was created
  - `updatedAt`: ISO8601 timestamp when pricing was last updated
- `currPage`: Current page number
- `haveNext`: Boolean indicating if there are more pages
- `totalPage`: Total number of records matching the search

---

## 2. Get Pricing by ID

**Endpoint:** `GET /api/v1/instant-link/pricing/:id`

**Authorization:** ADMIN, OWNER

**Description:** Retrieve a specific pricing configuration by its ID.

**Headers:**
```
Authorization: Bearer <your-token>
```

**URL Parameters:**
- `id` (required): Pricing ID

**Access Control:**
- PROVIDER users can view any pricing entry
- Non-PROVIDER users can only view their own pricing entries
- Returns 403 Forbidden if attempting to access another client's pricing

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/pricing/01HZQK1A2B3C4D5E6F7G8H9J0K" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01HZQK1A2B3C4D5E6F7G8H9J0K",
    "clientId": null,
    "serviceType": "whatsapp",
    "category": "marketing",
    "cost": 600.00,
    "currency": "IDR",
    "isActive": true,
    "createdAt": "2025-11-28T10:00:00Z",
    "updatedAt": "2025-11-28T10:00:00Z"
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

---

## 3. Create Pricing

**Endpoint:** `POST /api/v1/instant-link/pricing`

**Authorization:** ADMIN only

**Description:** Create a new pricing configuration. Use this to add new pricing rules or client-specific pricing.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "serviceType": "whatsapp",
  "category": "marketing",
  "cost": 650.00,
  "currency": "IDR",
  "isActive": true,
  "clientId": null
}
```

**Request Fields:**
- `serviceType` (required): Service type - `whatsapp`, `sms`, `email`
- `category` (optional): Category for WhatsApp (`marketing`, `utility`, `authentication`), empty for SMS/Email
- `cost` (required): Cost per message (must be non-negative)
- `currency` (optional): Currency code (default: `IDR`)
- `isActive` (optional): Active status (default: `true`)
- `clientId` (optional): Client ID for client-specific pricing, `null` for default pricing (PROVIDER only)

**Access Control:**
- PROVIDER users can create pricing for any client or default pricing
- Non-PROVIDER users can only create pricing for their own clientId (automatically enforced)
- Non-PROVIDER users cannot create default pricing (clientId = NULL)

**cURL Example - Default WhatsApp Marketing Pricing:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/pricing" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "serviceType": "whatsapp",
    "category": "marketing",
    "cost": 650.00,
    "currency": "IDR",
    "isActive": true
  }'
```

**cURL Example - Client-Specific Pricing:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/pricing" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "serviceType": "whatsapp",
    "category": "utility",
    "cost": 300.00,
    "currency": "IDR",
    "isActive": true,
    "clientId": "01HZQK5X8V9Y2N1P3R4T6W8Z0A"
  }'
```

**cURL Example - SMS Pricing:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/pricing" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "serviceType": "sms",
    "category": "",
    "cost": 400.00,
    "currency": "IDR",
    "isActive": true
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01HZQK6F7G8H9J0K1L2M3N4O5P",
    "clientId": null,
    "serviceType": "whatsapp",
    "category": "marketing",
    "cost": 650.00,
    "currency": "IDR",
    "isActive": true,
    "createdAt": "2025-11-28T14:00:00Z",
    "updatedAt": "2025-11-28T14:00:00Z"
  }
}
```

**Error Responses:**

400 Bad Request - Missing service_type:
```json
{
  "code": 400,
  "message": "service_type is required"
}
```

400 Bad Request - Invalid cost:
```json
{
  "code": 400,
  "message": "cost must be non-negative"
}
```

400 Bad Request - Invalid request body:
```json
{
  "code": 400,
  "message": "Invalid request body"
}
```

---

## 4. Update Pricing

**Endpoint:** `PUT /api/v1/instant-link/pricing/:id`

**Authorization:** ADMIN only

**Description:** Update an existing pricing configuration. All fields except `id` can be updated.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**URL Parameters:**
- `id` (required): Pricing ID to update

**Request Body:**
```json
{
  "serviceType": "whatsapp",
  "category": "marketing",
  "cost": 700.00,
  "currency": "IDR",
  "isActive": true
}
```

**Request Fields:**
All fields are optional, only provided fields will be updated:
- `serviceType`: Service type
- `category`: Message category
- `cost`: Cost per message
- `currency`: Currency code
- `isActive`: Active status
- `clientId`: Client ID for client-specific pricing (PROVIDER only)

**Access Control:**
- PROVIDER users can update any pricing entry
- Non-PROVIDER users can only update their own pricing entries
- Non-PROVIDER users cannot change clientId
- Returns 403 Forbidden if attempting to update another client's pricing

**cURL Example - Update Cost:**
```bash
curl -X PUT "http://localhost:8080/api/v1/instant-link/pricing/01HZQK1A2B3C4D5E6F7G8H9J0K" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "cost": 700.00
  }'
```

**cURL Example - Deactivate Pricing:**
```bash
curl -X PUT "http://localhost:8080/api/v1/instant-link/pricing/01HZQK1A2B3C4D5E6F7G8H9J0K" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "isActive": false
  }'
```

**cURL Example - Update Multiple Fields:**
```bash
curl -X PUT "http://localhost:8080/api/v1/instant-link/pricing/01HZQK1A2B3C4D5E6F7G8H9J0K" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "cost": 750.00,
    "isActive": true,
    "category": "marketing"
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01HZQK1A2B3C4D5E6F7G8H9J0K",
    "clientId": null,
    "serviceType": "whatsapp",
    "category": "marketing",
    "cost": 700.00,
    "currency": "IDR",
    "isActive": true,
    "createdAt": "2025-11-28T10:00:00Z",
    "updatedAt": "2025-11-28T14:30:00Z"
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

400 Bad Request - Invalid request body:
```json
{
  "code": 400,
  "message": "Invalid request body"
}
```

---

## 5. Delete Pricing

**Endpoint:** `DELETE /api/v1/instant-link/pricing/:id`

**Authorization:** ADMIN only

**Description:** Delete a pricing configuration. Use with caution as this will affect future pricing calculations.

**Headers:**
```
Authorization: Bearer <your-token>
```

**URL Parameters:**
- `id` (required): Pricing ID to delete

**Access Control:**
- PROVIDER users can delete any pricing entry
- Non-PROVIDER users can only delete their own pricing entries
- Returns 403 Forbidden if attempting to delete another client's pricing

**cURL Example:**
```bash
curl -X DELETE "http://localhost:8080/api/v1/instant-link/pricing/01HZQK1A2B3C4D5E6F7G8H9J0K" \
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

404 Not Found:
```json
{
  "code": 404,
  "message": "Not found"
}
```

403 Forbidden - Accessing another client's pricing:
```json
{
  "code": 403,
  "message": "Not allowed"
}
```

---

## Access Control & Multi-Tenancy

The Pricing API implements strict access control to support multi-tenancy:

### User Roles

**1. PROVIDER (Super Admin)**
- Can create, view, update, and delete all pricing entries
- Can manage default pricing (clientId = NULL)
- Can manage client-specific pricing for any client
- Has full visibility across all pricing configurations

**2. Non-PROVIDER (Regular Client Users)**
- Can only manage pricing for their own clientId
- Cannot view or modify default pricing
- Cannot view or modify other clients' pricing
- clientId is automatically enforced in all operations

### Access Control Implementation

**Search/List Operations:**
- PROVIDER: Returns all pricing entries
- Non-PROVIDER: Automatically filters to show only their clientId pricing
- clientId query parameter is ignored for non-PROVIDER users

**Create Operations:**
- PROVIDER: Can set any clientId or NULL for default pricing
- Non-PROVIDER: clientId is automatically set to their own, cannot create default pricing

**Read/Update/Delete Operations:**
- PROVIDER: Can access any pricing entry by ID
- Non-PROVIDER: Must own the pricing entry (clientId matches)
- Returns 403 Forbidden if attempting to access another client's pricing

### Example Scenarios

**Scenario 1: PROVIDER User**
```bash
# Can create default pricing
curl -X POST ".../pricing" -d '{"serviceType":"whatsapp","category":"marketing","cost":600}'

# Can create client-specific pricing
curl -X POST ".../pricing" -d '{"serviceType":"whatsapp","category":"marketing","cost":500,"clientId":"CLIENT123"}'

# Can view all pricing
curl -X GET ".../pricing?pageNo=1&rowPerPage=20"
```

**Scenario 2: Regular Client User (CLIENT123)**
```bash
# Creates pricing automatically for CLIENT123
curl -X POST ".../pricing" -d '{"serviceType":"whatsapp","category":"marketing","cost":500}'
# Result: clientId is automatically set to "CLIENT123"

# Can only see their own pricing
curl -X GET ".../pricing?pageNo=1&rowPerPage=20"
# Result: Only returns pricing where clientId = "CLIENT123"

# Cannot access another client's pricing
curl -X GET ".../pricing/OTHER_CLIENT_PRICING_ID"
# Result: 403 Forbidden
```

---

## Pricing Configuration Guidelines

### Service Types

| Service Type | Category Support | Description |
|--------------|------------------|-------------|
| `whatsapp` | Yes | WhatsApp Business API messages |
| `sms` | No | SMS messages (single price) |
| `email` | No | Email messages (single price) |

### WhatsApp Categories

| Category | Typical Use Cases | Recommended Pricing |
|----------|-------------------|---------------------|
| `marketing` | Promotional messages, product updates, offers, newsletters | Higher cost (e.g., Rp 600-800) |
| `utility` | Order confirmations, account updates, transactional notifications | Medium cost (e.g., Rp 350-400) |
| `authentication` | OTP codes, 2FA, login verification | Lower cost (e.g., Rp 175-250) |

### Default vs Client-Specific Pricing

**Default Pricing (clientId = null):**
- Applies to all clients by default
- Used as fallback when no client-specific pricing exists
- Easier to manage for standard pricing

**Client-Specific Pricing (clientId = specific ID):**
- Overrides default pricing for specific client
- Useful for:
  - Volume discounts for large clients
  - Premium pricing for special features
  - Testing new pricing models
  - Grandfathered pricing for existing clients

**Priority:** Client-specific pricing > Default pricing

---

## Common Use Cases

### 1. View Current Pricing
```bash
# Get all pricing (first page)
curl -X GET "http://localhost:8080/api/v1/instant-link/pricing?pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"

# Get WhatsApp pricing only
curl -X GET "http://localhost:8080/api/v1/instant-link/pricing?serviceType=whatsapp&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"

# Get active pricing only
curl -X GET "http://localhost:8080/api/v1/instant-link/pricing?status=active&pageNo=1" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

### 2. Update WhatsApp Marketing Price
```bash
curl -X PUT "http://localhost:8080/api/v1/instant-link/pricing/01HZQK1A2B3C4D5E6F7G8H9J0K" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"cost": 650.00}'
```

### 3. Create Client-Specific Pricing
```bash
# Give special pricing to VIP client
curl -X POST "http://localhost:8080/api/v1/instant-link/pricing" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "serviceType": "whatsapp",
    "category": "marketing",
    "cost": 500.00,
    "clientId": "01HZQK5X8V9Y2N1P3R4T6W8Z0A"
  }'
```

### 4. Temporarily Disable Pricing
```bash
# Disable without deleting (can be re-enabled later)
curl -X PUT "http://localhost:8080/api/v1/instant-link/pricing/01HZQK1A2B3C4D5E6F7G8H9J0K" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"isActive": false}'
```

### 5. Bulk Price Update Strategy
```bash
# Step 1: Get all current pricing
curl -X GET "http://localhost:8080/api/v1/instant-link/pricing" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" > current_pricing.json

# Step 2: Update each pricing individually
# (Use script to iterate through IDs and update)

# Step 3: Verify changes
curl -X GET "http://localhost:8080/api/v1/instant-link/pricing" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

## Integration with Billing Service

The Pricing API directly integrates with the Billing Service's `CalculateMessageCostWithCategory()` function:

**Flow:**
1. When calculating message cost, system queries pricing table
2. If client-specific pricing exists (clientId matches), use that
3. Otherwise, fall back to default pricing (clientId = null)
4. If no pricing found in database, use hardcoded fallback values

**Example:**
```go
// In billing service
cost := BillingService.CalculateMessageCostWithCategory("whatsapp", "marketing")
// Returns: 600.00 (from pricing table, not hardcoded)
```

**Impact:**
- Changes to pricing table affect future message cost calculations immediately
- No application restart required
- Historical transactions remain unchanged (cost is stored in transaction record)

---

## Best Practices

### 1. Pricing Strategy
- **Start Conservative:** Begin with default pricing before adding client-specific rules
- **Document Changes:** Use descriptive names or keep change log
- **Test Before Production:** Test pricing changes on staging environment first
- **Gradual Rollout:** Use client-specific pricing to test before changing default

### 2. Data Management
- **Backup Before Changes:** Export current pricing before bulk updates
- **Use Deactivation:** Prefer `isActive: false` over deletion for historical tracking
- **Version Control:** Keep record of pricing changes over time
- **Audit Trail:** Monitor who changes pricing and when

### 3. Client Communication
- **Advance Notice:** Inform clients before pricing changes
- **Grandfather Existing:** Use client-specific pricing to maintain old rates for existing clients
- **Clear Documentation:** Provide pricing documentation to clients

### 4. Performance Optimization
- **Keep Active Count Low:** Too many inactive pricing entries can slow queries
- **Index Strategy:** Ensure database indexes on serviceType and category
- **Cache Pricing:** Consider caching pricing in application memory for high-volume scenarios

### 5. Security
- **ADMIN Only:** Restrict create/update/delete to ADMIN users only
- **Audit Logging:** Log all pricing changes for compliance
- **Validation:** Always validate cost values are non-negative
- **Review Regularly:** Periodic review of client-specific pricing

---

## Migration from Hardcoded Pricing

If migrating from hardcoded pricing:

### Step 1: Run Migration SQL
```bash
mysql -u root -p instantlink < migrations/002_create_pricing_table.sql
```

### Step 2: Verify Default Pricing Created
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/pricing" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

### Step 3: Application Restart
```bash
# Restart application to initialize pricing repository
./il-dashboard
```

### Step 4: Test Cost Calculation
- Send test message
- Verify cost is calculated from database
- Check billing transaction for correct cost

### Step 5: Monitor Logs
```bash
tail -f logs/il-dashboard.log | grep "CalculateMessageCostWithCategory"
# Should see: "Found pricing in database: cost=XXX.XX"
```

---

## Troubleshooting

### Pricing Not Found in Database
**Log Message:** `Pricing not found in database for type=X category=Y, using fallback`

**Solutions:**
1. Verify pricing entry exists in database
2. Check serviceType and category match exactly (case-insensitive)
3. Ensure pricing is active (`isActive = true`)
4. Run migration if table is empty

### Wrong Cost Being Charged
**Possible Causes:**
1. Client-specific pricing overriding default
2. Inactive pricing entry
3. Category mismatch (e.g., "MARKETING" vs "marketing")

**Debug Steps:**
1. Check what pricing exists for that client: `GET /pricing?service_type=whatsapp`
2. Verify category in template matches pricing category
3. Check application logs for cost calculation

### Unique Constraint Violation
**Error:** Duplicate entry for service_type and category

**Solution:**
- Each combination of serviceType + category + clientId must be unique
- Update existing entry instead of creating new one
- Or delete old entry before creating new one

---

## Error Codes Summary

| Code | Message | Description |
|------|---------|-------------|
| 0 | Success | Request successful |
| 400 | Bad Request | Invalid request parameters or body |
| 401 | Unauthorized | Missing or invalid authentication token |
| 403 | Forbidden | User doesn't have permission (requires ADMIN role) or accessing another client's pricing |
| 404 | Not found | Pricing entry not found |
| 500 | Internal Server Error | Database or server error occurred |

---

## API Reference Table

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/pricing` | ADMIN, OWNER | List all pricing configurations |
| GET | `/pricing/:id` | ADMIN, OWNER | Get specific pricing by ID |
| POST | `/pricing` | ADMIN | Create new pricing configuration |
| PUT | `/pricing/:id` | ADMIN | Update existing pricing |
| DELETE | `/pricing/:id` | ADMIN | Delete pricing configuration |

---

**Last Updated:** December 2, 2025
**API Version:** v1