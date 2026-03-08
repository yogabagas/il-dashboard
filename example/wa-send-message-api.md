# WhatsApp Send Message API Documentation

Base URL: `/api/v1/instant-link/wa-send`

**Authorization Required:** All endpoints require authentication

**Roles & Permissions:**
- **ADMIN** and **OWNER** users can send WhatsApp messages using approved templates

---

## Table of Contents

1. [Overview](#overview)
2. [Important Notes](#important-notes)
3. [Message Flow](#message-flow)
4. [Endpoints](#endpoints)
   - [Send Single Message](#1-send-single-message)
   - [Send Bulk Messages](#2-send-bulk-messages)
   - [Upload Image](#3-upload-image-for-messages)
   - [Get Image](#4-get-uploaded-image)
5. [Complete Message Examples](#complete-message-examples)
   - [Text Only Message](#text-only-message)
   - [Message with Header Image](#message-with-header-image)
   - [Message with Header Video](#message-with-header-video)
   - [Message with Body Variables](#message-with-body-variables)
   - [Complex Message with Multiple Parameters](#complex-message-with-multiple-parameters)
6. [Error Handling](#error-handling)
7. [Best Practices](#best-practices)

---

## Overview

WhatsApp Send Message API allows you to send WhatsApp messages using pre-approved templates. This API integrates with Meta's WhatsApp Business API and handles:

- Template-based message sending
- Multi-recipient bulk messaging
- Media attachment support (images, videos, documents)
- Message logging and tracking
- Automatic balance deduction and billing
- Campaign tracking

All messages are sent using approved WhatsApp templates and parameters can be dynamically filled.

**Response Format:**
```json
{
  "code": 0,
  "messages": "Success",
  "data": { /* payload */ }
}
```

---

## Important Notes

### Template Requirements
- Only **APPROVED** templates with `waTemplateId` can be used for sending messages
- Template name used internally is the `waTemplate` field (lowercase with underscores)
- The `name` field is only for display purposes

### Balance & Billing
- Each message sends checks client balance before sending
- Message cost is calculated based on template category:
  - **MARKETING**: Higher cost
  - **UTILITY**: Standard cost
  - **AUTHENTICATION**: Lower cost
- Balance is deducted only after successful message delivery

### Message Parameters
Parameters are divided into:
- **Header Parameters**: For templates with IMAGE, VIDEO, or DOCUMENT headers
- **Body Parameters**: For templates with text variables like {{1}}, {{2}}, etc.

### Media Support
- **Images**: JPEG, PNG
- **Videos**: MP4 and supported formats
- **Documents**: PDF and other document types
- Media can be provided via:
  - **URL (link)**: Publicly accessible URL to the media
  - **Media ID (id)**: Previously uploaded media ID from Meta

### 🚨 Common Error: Media Upload Error (Code 131053)

Jika Anda mengirim pesan berhasil tapi mendapatkan webhook callback dengan status `failed`:

```json
{
  "status": "failed",
  "errors": [{
    "code": 131053,
    "title": "Media upload error",
    "message": "Media upload error"
  }]
}
```

**Penyebab:**
WhatsApp **tidak bisa download media** dari URL yang Anda kirim karena:
- URL tidak publicly accessible (butuh authentication)
- SSL certificate invalid/expired atau bukan HTTPS
- File size melebihi limit (Image: 5MB, Video: 16MB, Document: 100MB)
- Server terlalu lambat (timeout > 5 detik)
- Content-Type header tidak sesuai

**Cara Debug:**
```bash
# Test apakah URL accessible dari luar (tanpa authentication)
curl -I https://your-domain.com/api/v1/instant-link/wa-send/image/your-image.jpg

# Harus return:
# HTTP/2 200 
# content-type: image/jpeg
# content-length: < 5242880 (5MB)
```

**Solusi (Pilih Salah Satu):**

**Opsi 1: Upload ke WhatsApp Media API (RECOMMENDED)**
```bash
# Upload media ke WhatsApp dan dapatkan Media ID
curl -X POST "https://graph.facebook.com/v24.0/YOUR_PHONE_NUMBER_ID/media" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -F "file=@/path/to/image.jpg" \
  -F "type=image/jpeg" \
  -F "messaging_product=whatsapp"

# Response: {"id":"1234567890"}
```

Gunakan Media ID saat send message:
```json
{
  "templateId": "template-123",
  "to": "6281234567890",
  "parameters": [
    {
      "type": "image",
      "image": {
        "id": "1234567890"  // Media ID dari WhatsApp
      }
    }
  ]
}
```

**Opsi 2: Fix URL yang Bermasalah**

Pastikan URL Anda memenuhi syarat:
- ✅ **HTTPS** dengan SSL certificate valid (bukan self-signed)
- ✅ **Publicly accessible** tanpa authentication/authorization
- ✅ **Response time < 5 detik**
- ✅ **File size sesuai limit**
- ✅ **Content-Type header benar** (image/jpeg, video/mp4, dll)
- ✅ **Cache-Control header** untuk performance (optional)

Contoh implementasi endpoint image yang benar (lihat `/api/v1/instant-link/wa-send/image/:filename`):
```go
// Set proper headers
ctx.SetHeader("Content-Type", "image/jpeg")
ctx.SetHeader("Cache-Control", "public, max-age=31536000")
ctx.SetHeader("Access-Control-Allow-Origin", "*")

// Return file bytes langsung tanpa redirect
```

**Opsi 3: Upload via Internal API**
```bash
# Upload ke internal storage (akan dapat URL yang sudah proper)
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/upload-image" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "image=@/path/to/image.jpg"

# Response: {"code":0,"data":"01KC8AXS5FVB6TXJXFSK5776JK.jpg"}
```

Gunakan URL lengkap:
```json
{
  "templateId": "template-123",
  "to": "6281234567890",
  "parameters": [
    {
      "type": "image",
      "image": {
        "link": "https://your-domain.com/api/v1/instant-link/wa-send/image/01KC8AXS5FVB6TXJXFSK5776JK.jpg"
      }
    }
  ]
}
```

**Media Requirements (WhatsApp Limits):**
- **Image**: Max 5MB, format JPG/PNG/WebP
- **Video**: Max 16MB, format MP4/3GPP (H.264 video codec, AAC audio codec)
- **Document**: Max 100MB, format PDF/DOC/DOCX/XLS/XLSX/PPT/PPTX/TXT
- **Audio**: Max 16MB, format AAC/MP4/AMR/OGG

---

## Message Flow

1. **Create and Approve Template** (via wa-templates API)
2. **Upload Media** (if needed) - Get URL or Media ID
3. **Send Message** with template ID and parameters
4. **System Validates**:
   - Template exists and is approved
   - Client has sufficient balance
   - Parameters match template structure
5. **Message Sent** to WhatsApp API
6. **Message Log Created**
7. **Balance Deducted** (on success)

---

## Endpoints

### 1. Send Single Message

**Endpoint:** `POST /api/v1/instant-link/wa-send/message`

**Authorization:** ADMIN, OWNER

**Description:** Send a single WhatsApp message to one recipient using an approved template.

**Headers:**
```text
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "senderId": "01HZQK1A2B3C4D5E6F7G8H9J0K",
  "templateId": "01KC83DKP9GQ7J5H6PK3E048PG",
  "to": "6281234567890",
  "parameters": [
    {
      "type": "text",
      "text": "John Doe"
    }
  ],
  "campaignId": "01CAMPAIGN1234567890"
}
```

**Request Body Fields:**
- `senderId` (string, optional): ID of the sender configuration
- `templateId` (string, required): ID of the approved WhatsApp template
- `to` (string, required): Recipient phone number with country code (e.g., "6281234567890")
- `parameters` (array, optional): Template parameters to fill variables
  - `type` (string): Parameter type - "text", "image", "video", "document"
  - `text` (string): Text value (for body parameters)
  - `image` (object): Image parameter (for header)
    - `link` (string): Public URL to image
    - `id` (string): Meta media ID
  - `video` (object): Video parameter (for header)
    - `link` (string): Public URL to video
    - `id` (string): Meta media ID
  - `document` (object): Document parameter (for header)
    - `link` (string): Public URL to document
    - `id` (string): Meta media ID
- `campaignId` (string, optional): Campaign ID for tracking

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/message" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "01KC83DKP9GQ7J5H6PK3E048PG",
    "to": "6281234567890",
    "parameters": [
      {
        "type": "text",
        "text": "John Doe"
      },
      {
        "type": "text",
        "text": "Rp 500.000"
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
    "messageId": "wamid.HBgLNjI4MTIzNDU2Nzg5MBUCABIYIDNBN0E0RjY2RjQxQzQ4NkE5MjM4",
    "to": "6281234567890",
    "status": "sent",
    "messagingProduct": "whatsapp"
  }
}
```

**Response Fields:**
- `messageId` (string): WhatsApp message ID for tracking
- `to` (string): Recipient phone number
- `status` (string): Message status ("sent", "failed")
- `messagingProduct` (string): Always "whatsapp"

**Error Responses:**

400 Bad Request - Template not approved:
```json
{
  "code": 400,
  "message": "Template must be approved before sending"
}
```

402 Payment Required - Insufficient balance:
```json
{
  "code": 402,
  "message": "Insufficient balance"
}
```

404 Not Found - Template not found:
```json
{
  "code": 404,
  "message": "Not found"
}
```

403 Forbidden - No access to template:
```json
{
  "code": 403,
  "message": "Not allowed"
}
```

404 WhatsApp API Error - Template name mismatch:
```json
{
  "code": 404,
  "message": "WhatsApp API error: {\"error\":{\"message\":\"(#132001) Template name does not exist in the translation\",\"type\":\"OAuthException\",\"code\":132001}}"
}
```

---

### 2. Send Bulk Messages

**Endpoint:** `POST /api/v1/instant-link/wa-send/bulk`

**Authorization:** ADMIN, OWNER

**Description:** Send WhatsApp messages to multiple recipients using the same template. Maximum 100 recipients per request.

**Headers:**
```text
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "senderId": "01HZQK1A2B3C4D5E6F7G8H9J0K",
  "templateId": "01KC83DKP9GQ7J5H6PK3E048PG",
  "recipients": [
    {
      "to": "6281234567890",
      "parameters": [
        {
          "type": "text",
          "text": "John Doe"
        },
        {
          "type": "text",
          "text": "Rp 500.000"
        }
      ]
    },
    {
      "to": "6289876543210",
      "parameters": [
        {
          "type": "text",
          "text": "Jane Smith"
        },
        {
          "type": "text",
          "text": "Rp 750.000"
        }
      ]
    }
  ]
}
```

**Request Body Fields:**
- `senderId` (string, optional): ID of the sender configuration
- `templateId` (string, required): ID of the approved WhatsApp template
- `recipients` (array, required): List of recipients (max 100)
  - `to` (string, required): Recipient phone number
  - `parameters` (array, optional): Template parameters specific to this recipient

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/bulk" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "01KC83DKP9GQ7J5H6PK3E048PG",
    "recipients": [
      {
        "to": "6281234567890",
        "parameters": [
          {"type": "text", "text": "John Doe"},
          {"type": "text", "text": "Rp 500.000"}
        ]
      },
      {
        "to": "6289876543210",
        "parameters": [
          {"type": "text", "text": "Jane Smith"},
          {"type": "text", "text": "Rp 750.000"}
        ]
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
    "totalRequested": 2,
    "totalSuccess": 2,
    "totalFailed": 0,
    "results": [
      {
        "messageId": "wamid.HBgLNjI4MTIzNDU2Nzg5MBUCABIYIDNBNzA0RjY2RjQxQzQ4NkE5MjM4",
        "to": "6281234567890",
        "status": "sent",
        "messagingProduct": "whatsapp"
      },
      {
        "messageId": "wamid.HBgLNjI4OTg3NjU0MzIxMBUCABIYIDNCNzA0RjY2RjQxQzQ4NkE5MjM4",
        "to": "6289876543210",
        "status": "sent",
        "messagingProduct": "whatsapp"
      }
    ]
  }
}
```

**Response Fields:**
- `totalRequested` (integer): Total number of recipients
- `totalSuccess` (integer): Number of successful sends
- `totalFailed` (integer): Number of failed sends
- `results` (array): Individual results for each recipient
  - Each result follows the same structure as single message response

**Partial Success Example:**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "totalRequested": 3,
    "totalSuccess": 2,
    "totalFailed": 1,
    "results": [
      {
        "messageId": "wamid.HBgLNjI4MTIzNDU2Nzg5MBUCABIYIDNBNzA0RjY2RjQxQzQ4NkE5MjM4",
        "to": "6281234567890",
        "status": "sent",
        "messagingProduct": "whatsapp"
      },
      {
        "messageId": "",
        "to": "6289999999999",
        "status": "failed"
      },
      {
        "messageId": "wamid.HBgLNjI4OTg3NjU0MzIxMBUCABIYIDNCNzA0RjY2RjQxQzQ4NkE5MjM4",
        "to": "6289876543210",
        "status": "sent",
        "messagingProduct": "whatsapp"
      }
    ]
  }
}
```

**Error Responses:**

400 Bad Request - Too many recipients:
```json
{
  "code": 400,
  "message": "Maximum 100 recipients per bulk request"
}
```

400 Bad Request - No recipients:
```json
{
  "code": 400,
  "message": "Recipients are required"
}
```

---

### 3. Upload Image for Messages

**Endpoint:** `POST /api/v1/instant-link/wa-send/upload-image`

**Authorization:** ADMIN, OWNER

**Description:** Upload an image to get a public URL that can be used in message parameters. This is useful for templates with image headers.

**Headers:**
```text
Authorization: Bearer <your-token>
Content-Type: multipart/form-data
```

**Form Data:**
- `image` (file, required): Image file to upload

**Supported Formats:**
- JPEG (.jpg, .jpeg)
- PNG (.png)

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/upload-image" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -F "image=@/path/to/your/image.jpg"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "url": "https://ilbedev.kitamandiri.com/api/v1/instant-link/wa-send/image/01KC8AXS5FVB6TXJXFSK5776JK.png",
    "filename": "01KC8AXS5FVB6TXJXFSK5776JK.png"
  }
}
```

**Response Fields:**
- `url` (string): Public URL to access the uploaded image
- `filename` (string): Generated filename (ULID + extension)

**Error Responses:**

400 Bad Request - No file:
```json
{
  "code": 400,
  "message": "Invalid request"
}
```

500 Internal Server Error:
```json
{
  "code": 500,
  "message": "Failed to upload image"
}
```

---

### 4. Get Uploaded Image

**Endpoint:** `GET /api/v1/instant-link/wa-send/image/:filename`

**Authorization:** None (Public access)

**Description:** Retrieve an uploaded image by filename. This endpoint returns the actual image file.

**URL Parameters:**
- `filename` (required): Image filename (e.g., "01KC8AXS5FVB6TXJXFSK5776JK.png")

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/wa-send/image/01KC8AXS5FVB6TXJXFSK5776JK.png" \
  --output downloaded-image.png
```

**Success Response:**
- Status: 200 OK
- Content-Type: image/jpeg or image/png
- Body: Binary image data

**Headers:**
```text
Content-Type: image/png
Cache-Control: public, max-age=31536000
```

**Error Responses:**

404 Not Found:
```json
{
  "code": 404,
  "message": "Image not found"
}
```

---

## Complete Message Examples

### Text Only Message

**Template Structure:**
- Body: "Hello {{1}}, your order {{2}} has been confirmed!"

**Request:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/message" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "01KC83DKORDER12345",
    "to": "6281234567890",
    "parameters": [
      {
        "type": "text",
        "text": "John Doe"
      },
      {
        "type": "text",
        "text": "#ORD-12345"
      }
    ]
  }'
```

**Expected Message:**
```
Hello John Doe, your order #ORD-12345 has been confirmed!
```

---

### Message with Header Image

**Template Structure:**
- Header: IMAGE
- Body: "Check out our latest promotion!"

**Step 1 - Upload Image:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/upload-image" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "image=@promo-banner.jpg"

# Response:
# {
#   "code": 0,
#   "messages": "Success",
#   "data": {
#     "url": "https://ilbedev.kitamandiri.com/api/v1/instant-link/wa-send/image/01KC8AXS5FVB6TXJXFSK5776JK.jpg"
#   }
# }
```

**Step 2 - Send Message with Image:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/message" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "01KC83DKPROMO12345",
    "to": "6281234567890",
    "parameters": [
      {
        "type": "image",
        "image": {
          "link": "https://ilbedev.kitamandiri.com/api/v1/instant-link/wa-send/image/01KC8AXS5FVB6TXJXFSK5776JK.jpg"
        }
      }
    ]
  }'
```

**Using Meta Media ID (Alternative):**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/message" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "01KC83DKPROMO12345",
    "to": "6281234567890",
    "parameters": [
      {
        "type": "image",
        "image": {
          "id": "1234567890"
        }
      }
    ]
  }'
```

---

### Message with Header Video

**Template Structure:**
- Header: VIDEO
- Body: "Watch our new product demonstration!"

**Request:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/message" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "01KC83DKVIDEO12345",
    "to": "6281234567890",
    "parameters": [
      {
        "type": "video",
        "video": {
          "link": "https://example.com/videos/product-demo.mp4"
        }
      }
    ]
  }'
```

**Using Meta Media ID:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/message" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "01KC83DKVIDEO12345",
    "to": "6281234567890",
    "parameters": [
      {
        "type": "video",
        "video": {
          "id": "9876543210"
        }
      }
    ]
  }'
```

---

### Message with Body Variables

**Template Structure:**
- Body: "Dear {{1}}, your payment of {{2}} for invoice {{3}} has been received. Thank you!"

**Request:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/message" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "01KC83DKPAYMENT123",
    "to": "6281234567890",
    "parameters": [
      {
        "type": "text",
        "text": "John Doe"
      },
      {
        "type": "text",
        "text": "Rp 1.500.000"
      },
      {
        "type": "text",
        "text": "INV-2025-001"
      }
    ]
  }'
```

**Expected Message:**
```
Dear John Doe, your payment of Rp 1.500.000 for invoice INV-2025-001 has been received. Thank you!
```

---

### Complex Message with Multiple Parameters

**Template Structure:**
- Header: IMAGE
- Body: "Hi {{1}}! Your order {{2}} worth {{3}} is ready for pickup at {{4}}."
- Footer: "Thank you for shopping with us!"
- Buttons: [Quick Reply: "Confirm Pickup"]

**Step 1 - Upload Header Image:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/upload-image" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "image=@order-ready.jpg"
```

**Step 2 - Send Complete Message:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/message" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "01KC83DKORDER12345",
    "to": "6281234567890",
    "parameters": [
      {
        "type": "image",
        "image": {
          "link": "https://ilbedev.kitamandiri.com/api/v1/instant-link/wa-send/image/01KC8AXORDER123.jpg"
        }
      },
      {
        "type": "text",
        "text": "Sarah"
      },
      {
        "type": "text",
        "text": "#ORD-98765"
      },
      {
        "type": "text",
        "text": "Rp 350.000"
      },
      {
        "type": "text",
        "text": "Store Central Park"
      }
    ]
  }'
```

**Expected Message:**
```
[IMAGE: order-ready.jpg]

Hi Sarah! Your order #ORD-98765 worth Rp 350.000 is ready for pickup at Store Central Park.

Thank you for shopping with us!

[Button: Confirm Pickup]
```

---

### Bulk Message with Different Parameters

**Template Structure:**
- Body: "Hi {{1}}, your bill for {{2}} is {{3}}. Payment due date: {{4}}."

**Request:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/bulk" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "01KC83DKBILL123456",
    "recipients": [
      {
        "to": "6281234567890",
        "parameters": [
          {"type": "text", "text": "Alice"},
          {"type": "text", "text": "January 2025"},
          {"type": "text", "text": "Rp 500.000"},
          {"type": "text", "text": "15 Jan 2025"}
        ]
      },
      {
        "to": "6289876543210",
        "parameters": [
          {"type": "text", "text": "Bob"},
          {"type": "text", "text": "January 2025"},
          {"type": "text", "text": "Rp 750.000"},
          {"type": "text", "text": "15 Jan 2025"}
        ]
      },
      {
        "to": "6285555555555",
        "parameters": [
          {"type": "text", "text": "Charlie"},
          {"type": "text", "text": "January 2025"},
          {"type": "text", "text": "Rp 1.000.000"},
          {"type": "text", "text": "15 Jan 2025"}
        ]
      }
    ]
  }'
```

**Expected Messages:**
- To Alice: "Hi Alice, your bill for January 2025 is Rp 500.000. Payment due date: 15 Jan 2025."
- To Bob: "Hi Bob, your bill for January 2025 is Rp 750.000. Payment due date: 15 Jan 2025."
- To Charlie: "Hi Charlie, your bill for January 2025 is Rp 1.000.000. Payment due date: 15 Jan 2025."

---

### Message with Document Header

**Template Structure:**
- Header: DOCUMENT
- Body: "Here is your invoice. Please review and make payment accordingly."

**Request:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/message" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "01KC83DKINVOICE123",
    "to": "6281234567890",
    "parameters": [
      {
        "type": "document",
        "document": {
          "link": "https://example.com/invoices/INV-2025-001.pdf"
        }
      }
    ]
  }'
```

---

## Error Handling

### Common Error Scenarios

#### 1. Template Not Approved
**Error:**
```json
{
  "code": 400,
  "message": "Template must be approved before sending"
}
```
**Solution:** Ensure the template status is "APPROVED" and has a valid `waTemplateId`.

#### 2. Template Name Mismatch
**Error:**
```json
{
  "code": 404,
  "message": "WhatsApp API error: {\"error\":{\"message\":\"(#132001) Template name does not exist in the translation\",\"type\":\"OAuthException\",\"code\":132001}}"
}
```
**Solution:** This occurs when the template name sent to WhatsApp doesn't match what's registered. Make sure:
- Template is properly submitted to WhatsApp
- Template is approved by WhatsApp (waStatus = "APPROVED")
- The `waTemplate` field matches exactly what's in WhatsApp Business Manager

#### 3. Insufficient Balance
**Error:**
```json
{
  "code": 402,
  "message": "Insufficient balance"
}
```
**Solution:** Top up client balance or check billing settings.

#### 4. Invalid Phone Number
**Error:**
```json
{
  "code": 404,
  "message": "WhatsApp API error: {\"error\":{\"message\":\"Invalid phone number\",\"code\":1006}}"
}
```
**Solution:** 
- Use phone number with country code (e.g., "6281234567890")
- Remove spaces, dashes, or special characters
- Ensure phone number has WhatsApp installed

#### 5. Parameter Count Mismatch
**Error:**
```json
{
  "code": 404,
  "message": "WhatsApp API error: {\"error\":{\"message\":\"Parameter count does not match placeholder count\",\"code\":100}}"
}
```
**Solution:** Ensure the number of parameters matches the number of variables in the template.

#### 6. Missing Required Media
**Error:**
```json
{
  "code": 404,
  "message": "WhatsApp API error: {\"error\":{\"message\":\"Missing required component\",\"code\":132015}}"
}
```
**Solution:** If template has IMAGE/VIDEO/DOCUMENT header, you must provide the corresponding parameter.

---

## Best Practices

### 1. Template Management
- Always test templates in sandbox before production
- Use clear, descriptive template names
- Follow WhatsApp template guidelines for approval
- Keep template list updated and delete unused ones

### 2. Parameter Handling
- Validate parameters before sending
- Use appropriate parameter types (text, image, video, document)
- Ensure parameter order matches template variables
- Sanitize user input in parameters

### 3. Media Handling
- Use CDN URLs for better performance
- Ensure media URLs are publicly accessible
- Compress images to reduce load time
- Consider using Meta Media IDs for frequently used images

### 4. Bulk Sending
- Use bulk endpoint for multiple recipients
- Respect rate limits (max 100 per request)
- Implement retry logic for failed messages
- Monitor success/failure rates

### 5. Error Handling
- Log all API responses
- Implement proper error handling
- Notify users of delivery failures
- Store failed messages for retry

### 6. Cost Optimization
- Use UTILITY templates instead of MARKETING when possible
- Monitor message costs per category
- Set up balance alerts
- Track ROI per campaign

### 7. Testing
- Test with small batches first
- Verify template rendering with all parameter combinations
- Check media URLs before bulk sending
- Monitor message logs for issues

### 8. Compliance
- Get user consent before sending marketing messages
- Respect opt-out requests
- Follow WhatsApp Business Policy
- Include opt-out instructions in marketing templates

---

## Integration Example

### Node.js Example

```javascript
const axios = require('axios');

async function sendWhatsAppMessage(token, templateId, to, parameters) {
  try {
    const response = await axios.post(
      'http://localhost:8080/api/v1/instant-link/wa-send/message',
      {
        templateId: templateId,
        to: to,
        parameters: parameters
      },
      {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      }
    );
    
    console.log('Message sent:', response.data);
    return response.data;
  } catch (error) {
    console.error('Failed to send message:', error.response?.data || error.message);
    throw error;
  }
}

// Usage
sendWhatsAppMessage(
  'your-token-here',
  '01KC83DKP9GQ7J5H6PK3E048PG',
  '6281234567890',
  [
    { type: 'text', text: 'John Doe' },
    { type: 'text', text: 'Rp 500.000' }
  ]
);
```

### Python Example

```python
import requests

def send_whatsapp_message(token, template_id, to, parameters):
    url = 'http://localhost:8080/api/v1/instant-link/wa-send/message'
    headers = {
        'Authorization': f'Bearer {token}',
        'Content-Type': 'application/json'
    }
    payload = {
        'templateId': template_id,
        'to': to,
        'parameters': parameters
    }
    
    response = requests.post(url, json=payload, headers=headers)
    response.raise_for_status()
    return response.json()

# Usage
result = send_whatsapp_message(
    'your-token-here',
    '01KC83DKP9GQ7J5H6PK3E048PG',
    '6281234567890',
    [
        {'type': 'text', 'text': 'John Doe'},
        {'type': 'text', 'text': 'Rp 500.000'}
    ]
)
print(result)
```

---

## Troubleshooting Guide

### 🚨 Media Upload Error (Error Code 131053) - DETAILED GUIDE

Ini adalah error paling umum saat mengirim pesan dengan media (image/video/document).

#### Symptom:
Pesan terkirim (status "sent"), tapi beberapa detik kemudian Anda mendapat webhook callback dengan error:

```
time="2025-12-12T04:05:49Z" level=error msg="[handleActionWaWebHook] Message error - Code: 131053, Title: Media upload error, Message: Media upload error"
```

Webhook payload:
```json
{
  "object": "whatsapp_business_account",
  "entry": [{
    "changes": [{
      "value": {
        "statuses": [{
          "id": "wamid.HBgNNjI4MTIxOTgzNjU4MRU...",
          "status": "failed",
          "recipient_id": "6281219836581",
          "errors": [{
            "code": 131053,
            "title": "Media upload error",
            "message": "Media upload error"
          }]
        }]
      }
    }]
  }]
}
```

#### Root Cause:
WhatsApp **tidak bisa download media** dari URL yang Anda kirim. Setelah menerima request send message dari Anda, WhatsApp API akan:
1. Terima request Anda (return 200 OK dengan message ID)
2. **Asynchronously download media** dari URL yang Anda kirimkan
3. Jika download gagal, kirim webhook callback dengan status "failed"

#### Common Causes:

1. **URL Not Publicly Accessible**
   ```bash
   # URL butuh authentication
   curl -I https://your-domain.com/image.jpg
   # Returns: 401 Unauthorized atau 403 Forbidden
   ```

2. **SSL Certificate Issues**
   ```bash
   # Certificate expired atau self-signed
   curl -I https://your-domain.com/image.jpg
   # Returns: SSL certificate problem
   ```

3. **Wrong Content-Type Header**
   ```bash
   curl -I https://your-domain.com/image.jpg
   # Returns: Content-Type: text/html (SALAH!)
   # Harus: Content-Type: image/jpeg
   ```

4. **File Size Exceeded**
   ```bash
   curl -I https://your-domain.com/image.jpg
   # Returns: Content-Length: 8388608 (8MB - TOO LARGE!)
   # Limit: 5MB untuk image
   ```

5. **Server Timeout**
   ```bash
   # Server terlalu lambat respond (>5 detik)
   time curl -I https://your-domain.com/image.jpg
   # Takes: 7.2s (TOO SLOW!)
   ```

6. **HTTP Instead of HTTPS**
   ```bash
   # WhatsApp hanya accept HTTPS
   curl -I http://your-domain.com/image.jpg  # SALAH!
   # Harus HTTPS: https://your-domain.com/image.jpg
   ```

#### How to Debug:

**Step 1: Test URL dari luar**
```bash
# Test apakah URL accessible dari internet
curl -v https://your-domain.com/api/v1/instant-link/wa-send/image/01KC8AXS5FVB6TXJXFSK5776JK.jpg

# Check:
# ✅ HTTP/2 200 (or HTTP/1.1 200)
# ✅ content-type: image/jpeg (atau image/png)
# ✅ content-length: < 5242880 (< 5MB)
# ✅ Response time: < 5 seconds
# ✅ No authentication required
```

**Step 2: Check Headers**
```bash
# Pastikan header-nya benar
curl -I https://your-domain.com/api/v1/instant-link/wa-send/image/01KC8AXS5FVB6TXJXFSK5776JK.jpg

# Expected output:
HTTP/2 200 
content-type: image/jpeg
content-length: 245123
cache-control: public, max-age=31536000
access-control-allow-origin: *
```

**Step 3: Test dari Server Lain**
```bash
# Test dari server/VPS lain (bukan dari local computer)
# WhatsApp download dari server mereka, bukan dari komputer Anda

# Dari server lain:
ssh user@your-server
curl -I https://your-domain.com/api/v1/instant-link/wa-send/image/xxx.jpg
```

**Step 4: Check File Size**
```bash
# Check actual file size
ls -lh /path/to/uploaded/images/01KC8AXS5FVB6TXJXFSK5776JK.jpg

# Atau via HTTP
curl -I https://your-domain.com/api/v1/instant-link/wa-send/image/xxx.jpg | grep content-length
```

#### Solutions:

**✅ Solution 1: Upload ke WhatsApp Media API (RECOMMENDED)**

Ini adalah cara paling reliable karena media sudah ada di server WhatsApp.

```bash
# Step 1: Upload media ke WhatsApp
curl -X POST "https://graph.facebook.com/v24.0/YOUR_PHONE_NUMBER_ID/media" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -F "file=@/path/to/image.jpg" \
  -F "type=image/jpeg" \
  -F "messaging_product=whatsapp"

# Response:
# {"id":"1234567890123"}
```

```bash
# Step 2: Send message dengan Media ID
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/message" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "01KC83DKP9GQ7J5H6PK3E048PG",
    "to": "6281234567890",
    "parameters": [
      {
        "type": "image",
        "image": {
          "id": "1234567890123"
        }
      },
      {
        "type": "text",
        "text": "John Doe"
      }
    ]
  }'
```

**Advantages:**
- ✅ No URL accessibility issues
- ✅ Faster delivery
- ✅ No bandwidth cost dari server Anda
- ✅ Media ID bisa di-reuse untuk multiple messages

**Disadvantages:**
- ❌ Media ID expire setelah 30 hari
- ❌ Extra step untuk upload

---

**✅ Solution 2: Fix Your Image Endpoint**

Pastikan endpoint `/api/v1/instant-link/wa-send/image/:filename` sudah benar:

```go
// di controllers/wa_send.go - getImage function
func (o *WASendController) getImage(ctx gocom.Context) error {
    filename := ctx.Param("filename")

    // Get image data
    data, err := services.GetWASendSvc().GetImages(filename, a.Get(ctx))
    if err != nil {
        return ctx.SendError(err)
    }

    // Detect content type from extension
    ext := filepath.Ext(filename)
    contentType := "image/jpeg"
    if ext == ".png" {
        contentType = "image/png"
    } else if ext == ".jpg" || ext == ".jpeg" {
        contentType = "image/jpeg"
    }

    // Set proper headers (IMPORTANT!)
    ctx.SetHeader("Content-Type", contentType)
    ctx.SetHeader("Cache-Control", "public, max-age=31536000")
    ctx.SetHeader("Access-Control-Allow-Origin", "*")
    
    // Return raw bytes (NO redirect, NO HTML)
    return ctx.SendFileBytes(data, filename)
}
```

**Checklist:**
- ✅ Content-Type header sesuai (image/jpeg, image/png, video/mp4, dll)
- ✅ Return binary data langsung (bukan JSON atau HTML)
- ✅ HTTPS dengan SSL valid
- ✅ No authentication required untuk public images
- ✅ Response time < 5 detik
- ✅ File size sesuai limit

---

**✅ Solution 3: Use CDN**

Upload image ke CDN seperti CloudFlare, AWS S3, Google Cloud Storage:

```bash
# Example: Upload ke S3 dan set public access
aws s3 cp image.jpg s3://your-bucket/images/image.jpg --acl public-read

# Get URL:
# https://your-bucket.s3.amazonaws.com/images/image.jpg
```

```bash
# Send message dengan CDN URL
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/message" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "01KC83DKP9GQ7J5H6PK3E048PG",
    "to": "6281234567890",
    "parameters": [
      {
        "type": "image",
        "image": {
          "link": "https://your-bucket.s3.amazonaws.com/images/image.jpg"
        }
      }
    ]
  }'
```

**Advantages:**
- ✅ Fast response time
- ✅ Highly available
- ✅ No load on your server
- ✅ Proper headers automatically set

---

**✅ Solution 4: Optimize Your Server**

Jika tetap mau pakai URL dari server Anda:

1. **Enable Compression** (untuk transfer cepat)
   ```nginx
   # nginx.conf
   gzip on;
   gzip_types image/jpeg image/png image/gif;
   ```

2. **Enable Caching** (untuk speed)
   ```nginx
   # nginx.conf
   location /api/v1/instant-link/wa-send/image/ {
       expires 1y;
       add_header Cache-Control "public, immutable";
   }
   ```

3. **Optimize Images** (untuk size)
   ```bash
   # Compress JPEG (quality 85%)
   jpegoptim --max=85 image.jpg
   
   # Compress PNG
   optipng -o7 image.png
   ```

4. **Use Fast Storage** (untuk I/O)
   - SSD instead of HDD
   - Local storage instead of network storage
   - Memory cache untuk frequent images

---

#### Testing After Fix:

```bash
# Test 1: Accessibility
curl -I https://your-domain.com/api/v1/instant-link/wa-send/image/test.jpg
# Expected: 200 OK

# Test 2: Content Type
curl -I https://your-domain.com/api/v1/instant-link/wa-send/image/test.jpg | grep content-type
# Expected: content-type: image/jpeg

# Test 3: File Size
curl -I https://your-domain.com/api/v1/instant-link/wa-send/image/test.jpg | grep content-length
# Expected: content-length: < 5242880 (for images)

# Test 4: Response Time
time curl -I https://your-domain.com/api/v1/instant-link/wa-send/image/test.jpg
# Expected: < 5 seconds

# Test 5: Full Download
curl -o test.jpg https://your-domain.com/api/v1/instant-link/wa-send/image/test.jpg
# Expected: File downloaded successfully
```

#### Monitoring:

Tambahkan logging untuk track issues:

```go
// di services/wa_send.go
logger.Infof("[WASendService] Sending message with media URL: %s", imageURL)

// Check webhook callback
// di pubsub/updateStatus.go
if status == "failed" {
    logger.Errorf("[Webhook] Message failed - ID: %s, Errors: %+v", messageId, errors)
}
```

Monitor webhook callback untuk detect pattern:
- Jika semua gagal → Masalah di endpoint image
- Jika random gagal → Masalah performance/timeout
- Jika specific image gagal → Masalah di specific file

---

## Appendix

### Message Log Structure
Every sent message creates a log entry with:
- `id`: Unique message log ID
- `clientId`: Client who sent the message
- `senderId`: Sender configuration used
- `campaignId`: Campaign tracking (if applicable)
- `type`: "whatsapp"
- `direction`: "outbound"
- `recipientType`: "phone"
- `recipientValue`: Recipient phone number
- `templateId`: Template used
- `messageId`: WhatsApp message ID
- `status`: "sent" or "failed"
- `cost`: Message cost
- `sentAt`: Timestamp
- `errorMessage`: Error details (if failed)

### Billing Details
Messages are charged based on template category:
- **MARKETING**: Promotional content, higher cost
- **UTILITY**: Transactional/utility messages, standard cost
- **AUTHENTICATION**: OTP/verification messages, lower cost

Balance check happens before sending, deduction happens after successful delivery.

### Rate Limits
- Single message: No specific limit
- Bulk message: Max 100 recipients per request
- Concurrent bulk requests: Limited to 10 parallel sends per bulk request
- WhatsApp Business API rate limits apply (varies by tier)

---

**Last Updated:** December 12, 2025
**API Version:** v1
**WhatsApp API Version:** v24.0

