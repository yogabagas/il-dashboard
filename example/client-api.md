# Client API Documentation

Base URL: `/api/v1/instant-link/clients`

**Authorization Required:** All endpoints require authentication

**Important Notes:**
- **ADMIN** users can view and manage all clients
- **OWNER** users can manage their own client and sub-clients
- Client management follows a hierarchical structure (parent-child relationship)
- Each client has an owner with OWNER role automatically created
- Billing information (prepaid/postpaid) can be configured per client

**Access Control (Multi-Tenancy):**
- **PROVIDER (Super Admin)**: Can manage all clients and see all data
- **ADMIN**: Can view and search all clients
- **OWNER**: Can create sub-clients and manage their own client
- **Non-PROVIDER users**: Automatically set as parent when creating clients

---

## Overview

The Client API allows administrators to manage client accounts in a hierarchical structure. Each client represents an organization or business that uses the instant-link platform. Clients can have sub-clients, creating a multi-level hierarchy for resellers or enterprise accounts.

**Client Model:**
- Each client has a unique ID and name
- Clients have parent-child relationships for hierarchical management
- Each client has an owner (user with OWNER role)
- Billing configuration (prepaid/postpaid) can be set per client
- Clients can be active or inactive

---

## 1. Create Client

**Endpoint:** `POST /api/v1/instant-link/clients`

**Authorization:** None (Public endpoint) - **Note:** In production, this should be restricted

**Description:** Create a new client account with an owner user. The system automatically creates an owner user with OWNER role for the new client.

**Headers:**
```
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "PT ABC Indonesia",
  "ownerName": "John Doe",
  "ownerEmail": "john.doe@abc.com",
  "ownerPassword": "SecurePassword123!",
  "parentId": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
  "billingType": "prepaid",
  "creditLimit": 0,
  "balance": 1000000.00,
  "totalUsage": 0
}
```

**Field Descriptions:**
- `name` (required): Client/company name
- `ownerName` (required): Owner's full name
- `ownerEmail` (required): Owner's email address (must be unique)
- `ownerPassword` (required): Owner's password (will be hashed)
- `parentId` (optional): Parent client ID (defaults to authenticated user's clientId, or PROVIDER_ID for non-provider users)
- `billingType` (optional): Billing type - Options: `prepaid`, `postpaid` (default: `prepaid`)
- `creditLimit` (optional): Credit limit for postpaid accounts (default: 0)
- `balance` (optional): Initial balance for prepaid accounts (default: 0)
- `totalUsage` (optional): Initial total usage (default: 0)

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/clients" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "PT ABC Indonesia",
    "ownerName": "John Doe",
    "ownerEmail": "john.doe@abc.com",
    "ownerPassword": "SecurePassword123!",
    "billingType": "prepaid",
    "balance": 1000000.00
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
    "name": "PT ABC Indonesia",
    "parentId": "01HZQK1A2B3C4D5E6F7G8H9J0K",
    "parentName": "Provider Company",
    "status": "active",
    "ownerEmail": "john.doe@abc.com",
    "ownerName": "John Doe",
    "billingType": "prepaid",
    "creditLimit": 0,
    "balance": 1000000.00,
    "totalUsage": 0,
    "children": [],
    "roles": [
      {
        "id": "role-owner-id",
        "name": "OWNER",
        "description": "Owner role with full access"
      }
    ]
  }
}
```

**Response Fields:**
- `id`: Unique client identifier (ULID)
- `name`: Client/company name
- `parentId`: Parent client ID
- `parentName`: Parent client name
- `status`: Client status (`active` or `inactive`)
- `ownerEmail`: Owner's email address
- `ownerName`: Owner's full name
- `billingType`: Billing type (`prepaid` or `postpaid`)
- `creditLimit`: Credit limit for postpaid accounts
- `balance`: Current balance
- `totalUsage`: Total accumulated usage
- `children`: Array of sub-clients
- `roles`: Array of roles assigned to the owner

**Error Responses:**

409 Conflict - Client name or owner email already exists:
```json
{
  "code": 409,
  "message": "Already exist"
}
```

404 Not Found - Parent client not found:
```json
{
  "code": 404,
  "message": "Not found"
}
```

500 Internal Server Error - Unable to create client:
```json
{
  "code": 500,
  "message": "Unable to create"
}
```

---

## 2. Update Client

**Endpoint:** `PUT /api/v1/instant-link/clients/:id`

**Authorization:** ADMIN, OWNER (but only PROVIDER can actually update - others will get 404)

**Description:** Update an existing client's information. Only name, status, billing type, and owner email can be updated. In practice, only PROVIDER users can update clients.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Path Parameters:**
- `id`: Client ID to update

**Request Body:**
```json
{
  "name": "PT ABC Indonesia (Updated)",
  "parentId": "01HZQK1A2B3C4D5E6F7G8H9J0K",
  "ownerEmail": "john.doe@abc.com",
  "status": "active",
  "billingType": "postpaid",
  "creditLimit": 5000000.00,
  "balance": -150000.00,
  "totalUsage": 150000.00
}
```

**Field Descriptions:**
- `name` (required): Updated client/company name
- `parentId` (optional): Updated parent client ID
- `ownerEmail` (required): Owner's email address
- `status` (optional): Client status - Options: `active`, `inactive`
- `billingType` (optional): Billing type - Options: `prepaid`, `postpaid`
- `creditLimit` (optional): Credit limit for postpaid accounts
- `balance` (optional): Updated balance
- `totalUsage` (optional): Updated total usage

**cURL Example:**
```bash
curl -X PUT "http://localhost:8080/api/v1/instant-link/clients/01HZQK5X8V9Y2N1P3R4T6W8Z0A" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "PT ABC Indonesia (Updated)",
    "ownerEmail": "john.doe@abc.com",
    "status": "active",
    "billingType": "postpaid",
    "creditLimit": 5000000.00
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
    "name": "PT ABC Indonesia (Updated)",
    "parentId": "01HZQK1A2B3C4D5E6F7G8H9J0K",
    "parentName": "Provider Company",
    "status": "active",
    "ownerEmail": "john.doe@abc.com",
    "ownerName": "John Doe",
    "billingType": "postpaid",
    "creditLimit": 5000000.00,
    "balance": -150000.00,
    "totalUsage": 150000.00,
    "children": [],
    "roles": []
  }
}
```

**Error Responses:**

403 Forbidden - Insufficient permissions:
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

404 Not Found - Client not found:
```json
{
  "code": 404,
  "message": "Not found"
}
```

500 Internal Server Error - Update failed:
```json
{
  "code": 500,
  "message": "Unable to update"
}
```

---

## 3. Get Client by ID

**Endpoint:** `GET /api/v1/instant-link/clients/:id`

**Authorization:** ADMIN

**Description:** Retrieve detailed information about a specific client by ID, including parent information, children, and owner details.

**Headers:**
```
Authorization: Bearer <your-token>
```

**Path Parameters:**
- `id`: Client ID to retrieve

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/clients/01HZQK5X8V9Y2N1P3R4T6W8Z0A" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
    "name": "PT ABC Indonesia",
    "parentId": "01HZQK1A2B3C4D5E6F7G8H9J0K",
    "parentName": "Provider Company",
    "status": "active",
    "ownerEmail": "john.doe@abc.com",
    "ownerName": "John Doe",
    "billingType": "prepaid",
    "creditLimit": 0,
    "balance": 850000.00,
    "totalUsage": 150000.00,
    "children": [
      {
        "id": "01HZQK6Y9Z0A1B2C3D4E5F6G7H",
        "name": "PT ABC Branch Surabaya",
        "parentId": "01HZQK5X8V9Y2N1P3R4T6W8Z0A",
        "status": "active",
        "ownerEmail": "branch.sby@abc.com",
        "billingType": "prepaid",
        "balance": 200000.00,
        "children": []
      }
    ],
    "roles": [
      {
        "id": "role-owner-id",
        "name": "OWNER",
        "description": "Owner role with full access"
      }
    ]
  }
}
```

**Error Responses:**

403 Forbidden - Insufficient permissions:
```json
{
  "code": 403,
  "message": "Forbidden"
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

## 4. Search Clients

**Endpoint:** `GET /api/v1/instant-link/clients`

**Authorization:** ADMIN with ACCESS("SEARCH_CLIENT")

**Description:** Search and filter clients with pagination. Supports filtering by name and parent ID, with hierarchical access control.

**Headers:**
```
Authorization: Bearer <your-token>
```

**Query Parameters:**
- `filter` (optional): Text filter (searches in client name)
- `parentId` (optional): Filter by parent client ID
- `pageNo` (optional): Page number (default: 1)
- `rowPerPage` (optional): Items per page (default: 20)

**Access Control:**
- PROVIDER users can search all clients
- Non-PROVIDER users automatically see only their clients and sub-clients (parentId is enforced to their clientId)

**cURL Example - All Clients (First Page):**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/clients?pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Filter by Name:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/clients?filter=ABC&pageNo=1" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Filter by Parent:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/clients?parentId=01HZQK1A2B3C4D5E6F7G8H9J0K&pageNo=1" \
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
      "name": "PT ABC Indonesia",
      "parentId": "01HZQK1A2B3C4D5E6F7G8H9J0K",
      "parentName": "Provider Company",
      "status": "active",
      "ownerEmail": "john.doe@abc.com",
      "ownerName": "John Doe",
      "billingType": "prepaid",
      "creditLimit": 0,
      "balance": 850000.00,
      "totalUsage": 150000.00,
      "children": [],
      "roles": []
    },
    {
      "id": "01HZQK6Y9Z0A1B2C3D4E5F6G7H",
      "name": "PT XYZ Corporation",
      "parentId": "01HZQK1A2B3C4D5E6F7G8H9J0K",
      "parentName": "Provider Company",
      "status": "active",
      "ownerEmail": "admin@xyz.com",
      "ownerName": "Jane Smith",
      "billingType": "postpaid",
      "creditLimit": 10000000.00,
      "balance": -250000.00,
      "totalUsage": 250000.00,
      "children": [],
      "roles": []
    }
  ],
  "currPage": 1,
  "haveNext": true,
  "totalPage": 45
}
```

**Response Fields:**
- `code`: Status code (0 = success)
- `messages`: Status message
- `data`: Array of client objects
- `currPage`: Current page number
- `haveNext`: Boolean indicating if there are more pages
- `totalPage`: Total number of records matching the search

**Error Responses:**

403 Forbidden - Insufficient permissions:
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

---

## 5. Delete Client

**Endpoint:** `DELETE /api/v1/instant-link/clients/:id`

**Authorization:** ADMIN, OWNER with ACCESS("DELETE_CLIENT")

**Description:** Delete a client account. This is a soft delete that marks the client as inactive.

**Headers:**
```
Authorization: Bearer <your-token>
```

**Path Parameters:**
- `id`: Client ID to delete

**cURL Example:**
```bash
curl -X DELETE "http://localhost:8080/api/v1/instant-link/clients/01HZQK5X8V9Y2N1P3R4T6W8Z0A" \
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

403 Forbidden - Insufficient permissions:
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

404 Not Found - Client not found:
```json
{
  "code": 404,
  "message": "Not found"
}
```

500 Internal Server Error - Delete failed:
```json
{
  "code": 500,
  "message": "Unable to delete"
}
```

---

## Common Error Codes

| Code | Message | Description |
|------|---------|-------------|
| 409 | Already exist | Client name or owner email already exists |
| 403 | Forbidden | Insufficient permissions to perform this action |
| 404 | Not found | Client or parent client not found |
| 500 | Unable to create | Failed to create client |
| 500 | Unable to update | Failed to update client |
| 500 | Unable to delete | Failed to delete client |

---

## Billing Types

### Prepaid
- Client must maintain positive balance
- Messages are charged against the balance
- Cannot send messages when balance ≤ 0
- Top-up required to continue sending

### Postpaid
- Client has a credit limit
- Balance goes negative as messages are sent
- Can send messages while within credit limit
- Invoice generated periodically

---

## Client Hierarchy Example

```
Provider Company (PROVIDER_ID)
├── PT ABC Indonesia
│   ├── PT ABC Branch Surabaya
│   └── PT ABC Branch Jakarta
└── PT XYZ Corporation
    └── PT XYZ Regional Office
```

**Rules:**
- PROVIDER users can manage all levels
- ADMIN users can view all levels
- OWNER users can only manage their own client and direct sub-clients
- Non-PROVIDER users cannot set parentId to PROVIDER_ID when creating clients

---

## Notes

1. **Client Creation**: The system automatically creates an owner user with OWNER role when creating a client
2. **Unique Constraints**: Client name and owner email must be unique across all clients
3. **Parent-Child Relationship**: Clients can have multiple levels of hierarchy
4. **Billing Configuration**: Each client can have independent billing configuration (prepaid/postpaid)
5. **Access Control**: All operations respect the hierarchical access control model
6. **Soft Delete**: Deleted clients are marked as inactive, not permanently removed

