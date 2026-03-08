# WhatsApp Business API Documentation

Base URL: `/api/v1/instant-link`

**Authorization Required:** All endpoints require authentication

**Roles & Permissions:**
- **ADMIN** users can manage templates and send messages for any client
- **OWNER** users can only manage templates and send messages for their own client (based on `authInfo.ClientId`)

---

## Table of Contents

1. [WhatsApp Template Management](#whatsapp-template-management)
   - [Create Template](#1-create-whatsapp-template)
   - [Update Template](#2-update-whatsapp-template)
   - [Get Template by ID](#3-get-whatsapp-template-by-id)
   - [Search Templates](#4-search-whatsapp-templates)
   - [Delete Template](#5-delete-whatsapp-template)
   - [Submit Template for Review](#6-submit-template-for-internal-review)
   - [Approve Template (Admin)](#7-approve-template-admin-only)
   - [Reject Template (Admin)](#8-reject-template-admin-only)
   - [Submit to WhatsApp](#9-submit-template-to-whatsapp)

2. [WhatsApp Message Sending](#whatsapp-message-sending)

3. [Template Examples by Type](#template-examples-by-type)
   - [Text Template](#text-template)
   - [Text with Header Image](#text-template-with-header-image)
   - [Text with Header Video](#text-template-with-header-video)
   - [Text with Quick Reply Buttons](#text-template-with-quick-reply-buttons)
   - [Text with Call-to-Action Buttons](#text-template-with-call-to-action-buttons)
   - [Complete Template with All Components](#complete-template-with-all-components)

---

# WhatsApp Template Management

## 1. Create WhatsApp Template

**Endpoint:** `POST /api/v1/instant-link/wa-templates`

**Authorization:** ADMIN, OWNER

**Description:** Create a new WhatsApp message template. Templates must follow WhatsApp Business API guidelines and will need approval before use.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "welcome_message",
  "category": "MARKETING",
  "language": "id",
  "clientId": "client-123",
  "components": [
    {
      "type": "HEADER",
      "format": "TEXT",
      "text": "Selamat Datang! 🎉"
    },
    {
      "type": "BODY",
      "text": "Halo {{1}}, terima kasih telah bergabung dengan {{2}}. Kami siap melayani Anda!",
      "examples": ["Budi", "PDAM Jakarta"]
    },
    {
      "type": "FOOTER",
      "text": "Powered by InstantLink"
    },
    {
      "type": "BUTTONS",
      "buttons": [
        {
          "type": "QUICK_REPLY",
          "text": "Mulai"
        },
        {
          "type": "QUICK_REPLY",
          "text": "Bantuan"
        }
      ]
    }
  ]
}
```

**Field Descriptions:**

**Template Fields:**
- `name` (required): Template name (lowercase, underscore only, e.g., "welcome_message")
- `category` (required): Template category
  - `MARKETING` - Promotional messages, product updates
  - `UTILITY` - Account updates, order updates, alerts
  - `AUTHENTICATION` - OTP, verification codes
- `language` (required): Language code (e.g., "id" for Indonesian, "en" for English, "en_US" for English US)
- `clientId` (required): Client identifier
- `components` (required): Array of template components
- `status` (optional): Template status (default: "DRAFT")

**Component Types:**

**1. HEADER Component (optional)**
```json
{
  "type": "HEADER",
  "format": "TEXT|IMAGE|VIDEO|DOCUMENT",
  "text": "Header text here"  // For TEXT format
}
```
- `format` options:
  - `TEXT` - Plain text header (max 60 characters)
  - `IMAGE` - Image header (will be provided when sending)
  - `VIDEO` - Video header (will be provided when sending)
  - `DOCUMENT` - Document header (will be provided when sending)

**2. BODY Component (required)**
```json
{
  "type": "BODY",
  "text": "Message body with {{1}} and {{2}} variables",
  "examples": ["value1", "value2"]  // Required if using variables
}
```
- Maximum 1024 characters
- Use `{{1}}`, `{{2}}`, etc. for dynamic variables
- `examples` array must match number of variables

**3. FOOTER Component (optional)**
```json
{
  "type": "FOOTER",
  "text": "Footer text here"
}
```
- Maximum 60 characters
- No variables allowed

**4. BUTTONS Component (optional)**
```json
{
  "type": "BUTTONS",
  "buttons": [
    {
      "type": "QUICK_REPLY|PHONE_NUMBER|URL",
      "text": "Button text",
      "url": "https://example.com",        // For URL type
      "phoneNumber": "+6281234567890"      // For PHONE_NUMBER type
    }
  ]
}
```

Button types:
- `QUICK_REPLY` - Quick reply button (max 3 buttons, max 25 chars each)
- `PHONE_NUMBER` - Call button (max 1, requires phoneNumber field)
- `URL` - Website button (max 2, requires url field)

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-templates" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "welcome_message",
    "category": "MARKETING",
    "language": "id",
    "clientId": "client-123",
    "components": [
      {
        "type": "BODY",
        "text": "Halo {{1}}, selamat datang di layanan kami!"
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
    "id": "template-uuid-12345",
    "name": "welcome_message",
    "category": "MARKETING",
    "language": "id",
    "clientId": "client-123",
    "clientName": "PT Example",
    "components": [
      {
        "type": "BODY",
        "text": "Halo {{1}}, selamat datang di layanan kami!"
      }
    ],
    "status": "DRAFT",
    "waStatus": "",
    "createdAt": "2025-11-27T10:00:00Z",
    "updatedAt": "2025-11-27T10:00:00Z"
  }
}
```

**Error Responses:**

400 Bad Request - Invalid template format:
```json
{
  "code": 400,
  "message": "Invalid template format"
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

## 2. Update WhatsApp Template

**Endpoint:** `PUT /api/v1/instant-link/wa-templates/:id`

**Authorization:** ADMIN, OWNER

**Description:** Update an existing template. Only templates in DRAFT or REJECTED status can be updated.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**URL Parameters:**
- `id` (required): Template ID

**Request Body:**
```json
{
  "name": "welcome_message_v2",
  "category": "UTILITY",
  "components": [
    {
      "type": "BODY",
      "text": "Halo {{1}}, terima kasih telah menghubungi kami!"
    }
  ]
}
```

All fields are optional - only provided fields will be updated.

**cURL Example:**
```bash
curl -X PUT "http://localhost:8080/api/v1/instant-link/wa-templates/template-uuid-12345" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "components": [
      {
        "type": "BODY",
        "text": "Updated message body"
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
    "id": "template-uuid-12345",
    "name": "welcome_message_v2",
    "category": "UTILITY",
    "language": "id",
    "clientId": "client-123",
    "components": [...],
    "status": "DRAFT",
    "updatedAt": "2025-11-27T14:00:00Z"
  }
}
```

---

## 3. Get WhatsApp Template by ID

**Endpoint:** `GET /api/v1/instant-link/wa-templates/:id`

**Authorization:** ADMIN, OWNER

**Headers:**
```
Authorization: Bearer <your-token>
```

**URL Parameters:**
- `id` (required): Template ID

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/wa-templates/template-uuid-12345" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "template-uuid-12345",
    "name": "welcome_message",
    "category": "MARKETING",
    "language": "id",
    "clientId": "client-123",
    "clientName": "PT Example",
    "components": [...],
    "status": "APPROVED",
    "waStatus": "APPROVED",
    "waTemplateId": "wa-template-id-from-meta",
    "createdAt": "2025-11-27T10:00:00Z",
    "updatedAt": "2025-11-27T10:00:00Z",
    "submittedAt": "2025-11-27T11:00:00Z",
    "approvedAt": "2025-11-27T12:00:00Z"
  }
}
```

---

## 4. Search WhatsApp Templates

**Endpoint:** `GET /api/v1/instant-link/wa-templates`

**Authorization:** ADMIN, OWNER

**Headers:**
```
Authorization: Bearer <your-token>
```

**Query Parameters (all optional):**
- `filter` - Search keyword (searches in template name)
- `clientId` - Filter by client ID (OWNER users: automatically filtered to their client)
- `status` - Filter by internal status: `DRAFT`, `SUBMITTED`, `APPROVED`, `REJECTED`
- `waStatus` - Filter by WhatsApp status: `PENDING`, `APPROVED`, `REJECTED`
- `category` - Filter by category: `MARKETING`, `UTILITY`, `AUTHENTICATION`
- `pageNo` - Page number (default: 1)
- `rowPerPage` - Items per page (default: 20)

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/wa-templates?filter=welcome&category=MARKETING&status=APPROVED&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": [
    {
      "id": "template-uuid-12345",
      "name": "welcome_message",
      "category": "MARKETING",
      "language": "id",
      "clientId": "client-123",
      "clientName": "PT Example",
      "components": [...],
      "status": "APPROVED",
      "waStatus": "APPROVED",
      "createdAt": "2025-11-27T10:00:00Z"
    }
  ],
  "currPage": 1,
  "haveNext": false,
  "totalPage": 1
}
```

---

## 5. Delete WhatsApp Template

**Endpoint:** `DELETE /api/v1/instant-link/wa-templates/:id`

**Authorization:** ADMIN, OWNER

**Description:** Soft delete a template. Only templates in DRAFT status can be deleted.

**Headers:**
```
Authorization: Bearer <your-token>
```

**URL Parameters:**
- `id` (required): Template ID

**cURL Example:**
```bash
curl -X DELETE "http://localhost:8080/api/v1/instant-link/wa-templates/template-uuid-12345" \
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

---

## 6. Submit Template for Internal Review

**Endpoint:** `POST /api/v1/instant-link/wa-templates/:id/submit`

**Authorization:** ADMIN, OWNER

**Description:** Submit template for internal review by admin. Changes status from DRAFT to SUBMITTED.

**Headers:**
```
Authorization: Bearer <your-token>
```

**URL Parameters:**
- `id` (required): Template ID

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-templates/template-uuid-12345/submit" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "template-uuid-12345",
    "status": "SUBMITTED",
    "submittedAt": "2025-11-27T11:00:00Z"
  }
}
```

---

## 7. Approve Template (Admin Only)

**Endpoint:** `POST /api/v1/instant-link/wa-templates/:id/approve`

**Authorization:** ADMIN only

**Description:** Approve a submitted template. Changes status to APPROVED.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**URL Parameters:**
- `id` (required): Template ID

**Request Body:**
```json
{
  "waTemplateId": "optional-wa-template-id-if-already-approved-by-meta"
}
```

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-templates/template-uuid-12345/approve" \
  -H "Authorization: Bearer ADMIN_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "waTemplateId": "wa-template-id-from-meta"
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "template-uuid-12345",
    "status": "APPROVED",
    "waTemplateId": "wa-template-id-from-meta",
    "approvedAt": "2025-11-27T12:00:00Z"
  }
}
```

---

## 8. Reject Template (Admin Only)

**Endpoint:** `POST /api/v1/instant-link/wa-templates/:id/reject`

**Authorization:** ADMIN only

**Description:** Reject a submitted template with reason. Changes status to REJECTED.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**URL Parameters:**
- `id` (required): Template ID

**Request Body:**
```json
{
  "reason": "Template does not comply with WhatsApp policies"
}
```

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-templates/template-uuid-12345/reject" \
  -H "Authorization: Bearer ADMIN_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Template contains promotional content without proper opt-in"
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "template-uuid-12345",
    "status": "REJECTED",
    "rejectedReason": "Template contains promotional content without proper opt-in"
  }
}
```

---

## 9. Submit Template to WhatsApp

**Endpoint:** `POST /api/v1/instant-link/wa-templates/:id/submit-to-whatsapp`

**Authorization:** ADMIN, OWNER

**Description:** Submit approved template to WhatsApp Business API for Meta review.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: application/json
```

**URL Parameters:**
- `id` (required): Template ID

**Request Body:**
```json
{
  "wabaId": "your-whatsapp-business-account-id",
  "accessToken": "your-whatsapp-access-token"
}
```

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-templates/template-uuid-12345/submit-to-whatsapp" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "wabaId": "123456789012345",
    "accessToken": "EAABsbCS..."
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "template-uuid-12345",
    "waTemplateId": "wa-template-id-from-meta",
    "waStatus": "PENDING"
  }
}
```

---

# WhatsApp Message Sending

Section ini hanya sebagai overview singkat. Detail lengkap untuk API kirim pesan WhatsApp dipindahkan ke dokumen terpisah supaya lebih rapi.

**Endpoint Utama (dari `controllers/wa_send.go`):**
- `POST /api/v1/instant-link/wa-send/message` – kirim **single** message berbasis template.
- `POST /api/v1/instant-link/wa-send/bulk` – kirim **bulk** message berbasis template.

**Konsep umum:**
- Wajib menggunakan `templateId` yang mengacu ke `WATemplate.id`.
- Variabel di template (mis. `{{1}}`, `{{2}}`) diisi lewat array `parameters`.
- Format request/response mengikuti DTO di `dtos/wa_send.go`.

👉 **Dokumen lengkap Send Message API:** lihat `wa-send-message-api.md` di folder `example/`.

---

# Template Examples by Type

...existing code untuk contoh-contoh template bisa dibiarkan atau digeser ke `wa-templates-api.md` jika ingin full dipisah...
