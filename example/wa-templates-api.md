# WhatsApp Template API Documentation

Base URL: `/api/v1/instant-link`

**Authorization Required:** All endpoints require authentication

**Roles & Permissions:**
- **ADMIN** users can mengelola semua template untuk semua client (sesuai pengaturan di service layer)
- **OWNER** users hanya bisa mengelola template milik client-nya sendiri (`authInfo.ClientId`)

---

## Table of Contents

1. [Overview](#overview)
2. [Template Data Model](#template-data-model)
3. [Upload Image for Template Header](#upload-image-for-template-header)
4. [Upload Image for Template Media (Frontend/Display)](#upload-image-for-template-media-frontenddisplay)
5. [Template CRUD Endpoints](#template-crud-endpoints)
   - [Create Template](#1-create-whatsapp-template)
   - [Update Template](#2-update-whatsapp-template)
   - [Get Template by ID](#3-get-whatsapp-template-by-id)
   - [Search Templates](#4-search-whatsapp-templates)
   - [Delete Template](#5-delete-whatsapp-template)
6. [Template Workflow Endpoints](#template-workflow-endpoints)
   - [Submit Template for Internal Review](#6-submit-template-for-internal-review)
   - [Approve Template (Admin)](#7-approve-template-admin-only)
   - [Reject Template (Admin)](#8-reject-template-admin-only)
   - [Submit Template to WhatsApp](#9-submit-template-to-whatsapp)
7. [Template Examples by Type](#template-examples-by-type)
   - [Text Only Template](#text-only-template)
   - [Template with Header Text](#template-with-header-text)
   - [Template with Header Image](#template-with-header-image)
   - [Template with Header Video](#template-with-header-video)
   - [Template with Footer](#template-with-footer)
   - [Template with Quick Reply Buttons](#template-with-quick-reply-buttons)
   - [Template with Call-to-Action Buttons](#template-with-call-to-action-buttons)
   - [Complete Template with All Components](#complete-template-with-all-components)
8. [Status & Workflow](#status--workflow)
9. [Common Error Codes](#common-error-codes)

---

## Overview

WhatsApp Template API digunakan untuk mengelola template pesan WhatsApp Business yang akan dipakai saat pengiriman pesan berbasis template. Template mengikuti struktur Meta WhatsApp Business API namun disimpan dalam struktur internal sebagai berikut:

- Satu template memiliki beberapa komponen (`components`): HEADER, BODY, FOOTER, BUTTONS
- Komponen BUTTONS berisi array tombol (Quick Reply / Phone Number / URL)
- Template memiliki status internal (`status`) dan status WhatsApp (`waStatus`)

Semua response dibungkus dengan struktur umum:

```json
{
  "code": 0,
  "messages": "Success",
  "data": { /* payload utama */ }
}
```

---

## Template Data Model

### Request: `WATemplateReq`

```json
{
  "name": "string",
  "waTemplate": "string",
  "category": "string",
  "language": "string",
  "clientId": "string",
  "media": "string (optional)",
  "components": [
    {
      "type": "HEADER|BODY|FOOTER|BUTTONS",
      "format": "TEXT|IMAGE|VIDEO|DOCUMENT",
      "text": "string",
      "buttons": [
        {
          "type": "QUICK_REPLY|PHONE_NUMBER|URL",
          "text": "string",
          "url": "string",
          "phoneNumber": "string"
        }
      ],
      "examples": ["string"],
      "headerHandle": ["string"]
    }
  ],
  "status": "string (optional, default: DRAFT)"
}
```

**Field Explanation:**
- `name` (string, required): Display name for the template (max 255 chars, can contain spaces and special characters)
- `waTemplate` (string, required): WhatsApp template name (max 50 chars, **lowercase with underscores only**, e.g., "bill_reminder_v2")
- `category` (string, required): Template category (MARKETING, UTILITY, AUTHENTICATION)
- `language` (string, required): Language code (e.g., "id", "en", "en_US")
- `clientId` (string, required): Client ID who owns this template
- `media` (string, optional): URL media yang telah di-upload melalui endpoint `/api/v1/instant-link/wa-send/upload-image` (untuk keperluan internal/tampilan)
- `components` (array, required): Template components
  - `headerHandle` (array, optional): Sample media URLs for HEADER with IMAGE/VIDEO/DOCUMENT format
- `status` (string, optional): Internal status (default: DRAFT)

> **Important:** The `waTemplate` field is the actual name sent to WhatsApp API and must follow WhatsApp naming rules (lowercase, underscores, no spaces). The `name` field is for display purposes only.

### Response: `WATemplate`

```json
{
  "id": "string",
  "name": "string",
  "waTemplate": "string",
  "category": "string",
  "language": "string",
  "clientId": "string",
  "clientName": "string",
  "components": [ /* sama seperti di request */ ],
  "status": "DRAFT|SUBMITTED|APPROVED|REJECTED",
  "waStatus": "PENDING|APPROVED|REJECTED|" ,
  "waTemplateId": "string",
  "rejectedReason": "string",
  "createdAt": "2025-12-10T10:00:00Z",
  "updatedAt": "2025-12-10T10:00:00Z",
  "submittedAt": "2025-12-10T10:05:00Z",
  "approvedAt": "2025-12-10T10:10:00Z"
}
```

**Component Rules (umum):**

- `HEADER`
  - `format`: `TEXT`, `IMAGE`, `VIDEO`, atau `DOCUMENT`
  - Jika `format = TEXT` maka `text` wajib diisi (maks 60 karakter)
  - Jika `format = IMAGE|VIDEO|DOCUMENT`:
    - **PENTING - Untuk TEMPLATE CREATION:**
      - **Wajib upload media terlebih dahulu** menggunakan endpoint `/api/v1/instant-link/wa-templates/upload-img`
      - Gunakan file handle yang dikembalikan dari upload endpoint di field `example.header_handle` (string)
      - File handle diperlukan untuk approval template di WhatsApp
      - Format: `"example": {"header_handle": "4::base64string:ARxxx..."}`
    - **PENTING - Untuk MESSAGE SENDING:**
      - Saat mengirim pesan menggunakan template ini, Anda mengirimkan media URL atau Media ID yang berbeda
      - Lihat `wa-send-message-api.md` untuk cara mengirim pesan dengan header media
      - Format: `{"type": "image", "image": {"link": "https://url-to-image.jpg"}}` atau `{"type": "image", "image": {"id": "media-id"}}`

- `BODY`
  - Wajib ada minimal satu komponen `BODY`
  - `text` maksimal 1024 karakter
  - Mendukung variabel `{{1}}`, `{{2}}`, dst.
  - Jika menggunakan variabel, isi `examples` harus sesuai jumlah variabel

- `FOOTER`
  - Opsional
  - `text` maksimal 60 karakter, tidak mendukung variabel

- `BUTTONS`
  - Opsional
  - Array tombol (`buttons`):
    - `QUICK_REPLY`: maksimal 3 tombol, masing-masing <= 25 karakter
    - `PHONE_NUMBER`: maksimal 1 tombol, wajib isi `phoneNumber`
    - `URL`: maksimal 2 tombol, wajib isi `url`

---

## Upload Image for Template Header

### Upload Image to WhatsApp

**Endpoint:** `POST /api/v1/instant-link/wa-templates/upload-img`

**Authorization:** ADMIN, OWNER

**Description:** Upload image ke WhatsApp Cloud API untuk mendapatkan file handle yang akan digunakan sebagai `headerHandle` saat membuat template dengan header IMAGE/VIDEO/DOCUMENT. **Upload image harus dilakukan SEBELUM membuat template.**

**Headers:**
```text
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "fileName": "promo-banner.jpg",
  "file": "base64_encoded_image_data_here"
}
```

**Field Explanation:**
- `fileName` (string, required): Nama file dengan ekstensi (e.g., "banner.jpg", "promo.png")
  - **Rekomendasi:** Gunakan nama file sederhana tanpa spasi atau karakter khusus
  - Contoh yang baik: `promo-banner.jpg`, `banner_image.png`, `header01.jpg`
  - Hindari: `Screenshot 2025-12-09 at 00.09.47.png` (mengandung spasi)
  - File name akan otomatis di-encode untuk URL, tapi lebih baik gunakan format sederhana
- `file` (string, required): Base64 encoded image data

**Supported File Types:**
- `.jpg` / `.jpeg` (image/jpeg)
- `.png` (image/png)

**cURL Example:**
```bash
# Contoh dengan base64 encoding menggunakan file lokal
IMAGE_BASE64=$(base64 -i /path/to/your/image.jpg)

curl -X POST "http://localhost:8080/api/v1/instant-link/wa-templates/upload-img" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d "{
    \"fileName\": \"promo-banner.jpg\",
    \"file\": \"$IMAGE_BASE64\"
  }"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": "4::aW1hZ2UvanBlZw==:ARaK7h5VRxxxxxxxxxxxxxxxxxxxxxx:e:1234567890:xxxxxxxxx:ARZxxxxxxxxxxxxx"
}
```

**Response Explanation:**
- `data`: File handle string yang dikembalikan oleh WhatsApp. Ini adalah identifier unik untuk file yang di-upload.
- File handle ini **harus disimpan** dan digunakan di field `example.header_handle` dalam component saat membuat template.

**Error Responses:**

400 Bad Request (failed to upload):
```json
{
  "code": 400,
  "message": "Failed to upload image: [error details from WhatsApp]"
}
```

500 Internal Server Error (WABA not configured):
```json
{
  "code": 500,
  "message": "WhatsApp Business Account ID not configured"
}
```

**Usage Flow:**

1. **Upload Image First:**
   ```bash
   # Step 1: Upload image
   curl -X POST "http://localhost:8080/api/v1/instant-link/wa-templates/upload-img" \
     -H "Authorization: Bearer YOUR_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"fileName": "banner.jpg", "file": "base64_data_here"}'
   
   # Response: {"code":0,"messages":"Success","data":"4::aW1hZ2UvanBlZw==:ARaK7h5VRxxx..."}
   ```

2. **Use File Handle in Template Creation:**
   ```bash
   # Step 2: Create template using the file handle
   curl -X POST "http://localhost:8080/api/v1/instant-link/wa-templates" \
     -H "Authorization: Bearer YOUR_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "name": "promo_dengan_gambar",
       "waTemplate": "promo_dengan_gambar",
       "category": "MARKETING",
       "language": "id",
       "clientId": "client-123",
       "components": [
         {
           "type": "HEADER",
           "format": "IMAGE",
           "example": {
             "header_handle": "4::aW1hZ2UvanBlZw==:ARaK7h5VRxxx..."
           }
         },
         {
           "type": "BODY",
           "text": "Dapatkan diskon spesial untuk Anda!"
         }
       ]
     }'
   ```

---

## Upload Image for Template Media (Frontend/Display)

### Upload Image untuk Keperluan Display Template

**Endpoint:** `POST /api/v1/instant-link/wa-send/upload-image`

**Authorization:** ADMIN, OWNER

**Description:** Upload image ke server internal untuk disimpan dan digunakan sebagai media display/preview template di frontend. Image ini disimpan di server dan dapat diakses melalui URL publik. URL hasil upload ini dapat disimpan di field `media` pada `WATemplateReq` untuk keperluan tampilan di aplikasi.

**Headers:**
```text
Authorization: Bearer <your-token>
Content-Type: multipart/form-data
```

**Request Body (Form Data):**
- `image` (file, required): File image yang akan di-upload

**Field Explanation:**
- `image`: File gambar dalam format multipart/form-data
  - Supported types: `image/jpeg`, `image/jpg`, `image/png`
  - Maximum size: 5MB
  - File akan disimpan dengan nama unik (ULID) di direktori `./images`

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/upload-image" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -F "image=@/path/to/your/image.jpg"
```

**JavaScript/Frontend Example:**
```javascript
// Example menggunakan fetch API
const fileInput = document.getElementById('imageInput');
const file = fileInput.files[0];

const formData = new FormData();
formData.append('image', file);

fetch('http://localhost:8080/api/v1/instant-link/wa-send/upload-image', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer YOUR_TOKEN_HERE'
  },
  body: formData
})
.then(response => response.json())
.then(data => {
  console.log('Image URL:', data.data);
  // Gunakan data.data sebagai nilai untuk field 'media' saat create/update template
})
.catch(error => console.error('Error:', error));
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": "https://ilbedev.kitamandiri.com/api/v1/instant-link/wa-send/image/01JEXAMPLE123456.jpg"
}
```

**Response Explanation:**
- `data`: URL publik untuk mengakses image yang telah di-upload
- URL ini dapat digunakan untuk:
  - Disimpan di field `media` pada WATemplateReq (untuk display di frontend)
  - Preview template sebelum submit
  - Ditampilkan di list template sebagai thumbnail

**Error Responses:**

400 Bad Request (invalid file type):
```json
{
  "code": 400,
  "message": "Invalid request"
}
```

400 Bad Request (file too large):
```json
{
  "code": 400,
  "message": "Image size must not exceed 5MB"
}
```

500 Internal Server Error:
```json
{
  "code": 500,
  "message": "Internal server error"
}
```

### Mengakses Image yang Telah Di-upload

**Endpoint:** `GET /api/v1/instant-link/wa-send/image/:filename`

**Authorization:** Public (tidak memerlukan authorization)

**Description:** Mendapatkan image yang telah di-upload sebelumnya.

**URL Parameters:**
- `filename` (required): Nama file yang dikembalikan dari upload endpoint

**Example:**
```
GET https://ilbedev.kitamandiri.com/api/v1/instant-link/wa-send/image/01JEXAMPLE123456.jpg
```

**Response:**
- Content-Type: `image/jpeg` atau `image/png` (tergantung tipe file)
- Cache-Control: `public, max-age=31536000` (di-cache selama 1 tahun)
- Body: Binary image data

---

### Workflow: Upload Image dan Create Template dengan Media

Berikut adalah alur lengkap untuk membuat template dengan media untuk display:

**Step 1: Upload Image untuk Display**
```bash
# Upload image ke server internal
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/upload-image" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "image=@/path/to/promo-banner.jpg"

# Response:
# {
#   "code": 0,
#   "messages": "Success",
#   "data": "https://ilbedev.kitamandiri.com/api/v1/instant-link/wa-send/image/01JEXAMPLE123456.jpg"
# }
```

**Step 2: (Optional) Upload Image ke WhatsApp untuk Header Handle**

Jika template Anda memiliki header IMAGE/VIDEO/DOCUMENT, Anda perlu upload ke WhatsApp juga:

```bash
# Convert image to base64
IMAGE_BASE64=$(base64 -i /path/to/promo-banner.jpg)

# Upload to WhatsApp API
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-templates/upload-img" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"fileName\": \"promo-banner.jpg\",
    \"file\": \"$IMAGE_BASE64\"
  }"

# Response:
# {
#   "code": 0,
#   "messages": "Success",
#   "data": "4::aW1hZ2UvanBlZw==:ARaK7h5VRxxx..."
# }
```

**Step 3: Create Template dengan Media URL**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-templates" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Promo Special",
    "waTemplate": "promo_special",
    "category": "MARKETING",
    "language": "id",
    "clientId": "client-123",
    "media": "https://ilbedev.kitamandiri.com/api/v1/instant-link/wa-send/image/01JEXAMPLE123456.jpg",
    "components": [
      {
        "type": "HEADER",
        "format": "IMAGE",
        "example": {
          "header_handle": "4::aW1hZ2UvanBlZw==:ARaK7h5VRxxx..."
        }
      },
      {
        "type": "BODY",
        "text": "Dapatkan promo spesial untuk pelanggan setia! Diskon hingga {{1}}%",
        "examples": ["20"]
      },
      {
        "type": "FOOTER",
        "text": "Syarat dan ketentuan berlaku"
      }
    ]
  }'
```

**Penjelasan:**
- **Field `media`**: URL dari upload wa-send/upload-image → untuk display/preview di frontend
- **Field `example.header_handle`**: File handle dari wa-templates/upload-img → untuk submit template ke WhatsApp

**Catatan Penting:**

1. **Dua Jenis Upload:**
   - `/api/v1/instant-link/wa-send/upload-image` → Upload ke server internal untuk display/preview
   - `/api/v1/instant-link/wa-templates/upload-img` → Upload ke WhatsApp API untuk header handle

2. **Perbedaan Kegunaan:**
   - `media` field → Untuk tampilan di aplikasi (list template, preview, dll)
   - `header_handle` → Untuk validasi template di WhatsApp saat submit

3. **Bisa Menggunakan Image yang Berbeda:**
   - Image untuk `media` dan `header_handle` tidak harus sama
   - Anda bisa menggunakan image dengan resolusi lebih rendah untuk display, dan resolusi tinggi untuk WhatsApp

---

## Template CRUD Endpoints

### 1. Create WhatsApp Template

**Endpoint:** `POST /api/v1/instant-link/wa-templates`

**Authorization:** ADMIN, OWNER

**Description:** Membuat template baru yang akan digunakan saat mengirim pesan WhatsApp berbasis template.

**Headers:**
```text
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body (contoh umum):**
```json
{
  "name": "bill_reminder",
  "category": "UTILITY",
  "language": "id",
  "clientId": "client-123",
  "components": [
    {
      "type": "BODY",
      "text": "Halo {{1}}, tagihan Anda untuk periode {{2}} sejumlah Rp {{3}}.",
      "examples": ["Budi", "November 2025", "250.000"]
    }
  ]
}
```

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-templates" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "bill_reminder",
    "category": "UTILITY",
    "language": "id",
    "clientId": "client-123",
    "components": [
      {
        "type": "BODY",
        "text": "Halo {{1}}, tagihan Anda untuk periode {{2}} sejumlah Rp {{3}}.",
        "examples": ["Budi", "November 2025", "250.000"]
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
    "name": "bill_reminder",
    "category": "UTILITY",
    "language": "id",
    "clientId": "client-123",
    "clientName": "PT Contoh",
    "components": [
      {
        "type": "BODY",
        "text": "Halo {{1}}, tagihan Anda untuk periode {{2}} sejumlah Rp {{3}}.",
        "examples": ["Budi", "November 2025", "250.000"]
      }
    ],
    "status": "DRAFT",
    "waStatus": "",
    "createdAt": "2025-12-10T10:00:00Z",
    "updatedAt": "2025-12-10T10:00:00Z"
  }
}
```

**Error Responses:**

400 Bad Request:
```json
{
  "code": 400,
  "message": "Invalid template format"
}
```

403 Forbidden (OWNER membuat template untuk client lain):
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

---

### 2. Update WhatsApp Template

**Endpoint:** `PUT /api/v1/instant-link/wa-templates/:id`

**Authorization:** ADMIN, OWNER

**Description:** Update template yang sudah ada. Hanya field yang dikirim yang akan di-update.

**Headers:**
```text
Authorization: Bearer <your-token>
Content-Type: application/json
```

**URL Parameters:**
- `id` (required): ID template

**Request Body (partial, contoh):**
```json
{
  "name": "bill_reminder_v2",
  "components": [
    {
      "type": "BODY",
      "text": "Halo {{1}}, tagihan Anda untuk periode {{2}} sejumlah Rp {{3}}. Mohon dibayar sebelum {{4}}.",
      "examples": ["Budi", "November 2025", "250.000", "10 Desember 2025"]
    }
  ]
}
```

**cURL Example:**
```bash
curl -X PUT "http://localhost:8080/api/v1/instant-link/wa-templates/template-uuid-12345" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "bill_reminder_v2",
    "components": [
      {
        "type": "BODY",
        "text": "Halo {{1}}, tagihan Anda untuk periode {{2}} sejumlah Rp {{3}}. Mohon dibayar sebelum {{4}}.",
        "examples": ["Budi", "November 2025", "250.000", "10 Desember 2025"]
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
    "name": "bill_reminder_v2",
    "category": "UTILITY",
    "language": "id",
    "clientId": "client-123",
    "components": [
      {
        "type": "BODY",
        "text": "Halo {{1}}, tagihan Anda untuk periode {{2}} sejumlah Rp {{3}}. Mohon dibayar sebelum {{4}}.",
        "examples": ["Budi", "November 2025", "250.000", "10 Desember 2025"]
      }
    ],
    "status": "DRAFT",
    "updatedAt": "2025-12-10T11:00:00Z"
  }
}
```

---

### 3. Get WhatsApp Template by ID

**Endpoint:** `GET /api/v1/instant-link/wa-templates/:id`

**Authorization:** ADMIN, OWNER

**Headers:**
```text
Authorization: Bearer <your-token>
```

**URL Parameters:**
- `id` (required): ID template

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
    "name": "bill_reminder_v2",
    "category": "UTILITY",
    "language": "id",
    "clientId": "client-123",
    "clientName": "PT Contoh",
    "components": [
      {
        "type": "BODY",
        "text": "Halo {{1}}, tagihan Anda untuk periode {{2}} sejumlah Rp {{3}}. Mohon dibayar sebelum {{4}}.",
        "examples": ["Budi", "November 2025", "250.000", "10 Desember 2025"]
      }
    ],
    "status": "APPROVED",
    "waStatus": "APPROVED",
    "waTemplateId": "wa-template-id-from-meta",
    "createdAt": "2025-12-10T10:00:00Z",
    "updatedAt": "2025-12-10T11:00:00Z",
    "submittedAt": "2025-12-10T10:30:00Z",
    "approvedAt": "2025-12-10T10:45:00Z"
  }
}
```

---

### 4. Search WhatsApp Templates

**Endpoint:** `GET /api/v1/instant-link/wa-templates`

**Authorization:** ADMIN, OWNER

**Headers:**
```text
Authorization: Bearer <your-token>
```

**Query Parameters (semua optional):**
- `filter` - Kata kunci pencarian (nama template atau waTemplateId)
- `clientId` - Filter by client ID (untuk PROVIDER; OWNER otomatis dikunci ke client-nya)
- `status` - Status internal: `DRAFT`, `SUBMITTED`, `APPROVED`, `REJECTED`
- `waStatus` - Status WhatsApp: `PENDING`, `APPROVED`, `REJECTED`
- `category` - Kategori template: `MARKETING`, `UTILITY`, `AUTHENTICATION`
- `sortBy` - Sort by field: `name`, `category`, `status`, `wa_status`, `created_at`, `updated_at`, `submitted_at`, `approved_at` (default: `name`)
- `sortOrder` - Sort order: `ASC`, `DESC` (default: `ASC`)
- `pageNo` - Nomor halaman (default: 1)
- `rowPerPage` - Jumlah item per halaman (default: 20)

**Catatan Akses:**
- Jika `auth.ClientId != PROVIDER_ID`, maka `clientId` akan di-set otomatis ke `auth.ClientId` oleh controller. OWNER tidak bisa melihat template client lain.

**cURL Examples:**
```bash
# Basic search
curl -X GET "http://localhost:8080/api/v1/instant-link/wa-templates?filter=bill&status=APPROVED&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"

# With sorting by created date (newest first)
curl -X GET "http://localhost:8080/api/v1/instant-link/wa-templates?sortBy=created_at&sortOrder=DESC&pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"

# With sorting by status (alphabetically)
curl -X GET "http://localhost:8080/api/v1/instant-link/wa-templates?sortBy=status&sortOrder=ASC&pageNo=1&rowPerPage=20" \
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
      "name": "bill_reminder_v2",
      "category": "UTILITY",
      "language": "id",
      "clientId": "client-123",
      "clientName": "PT Contoh",
      "components": [
        {
          "type": "BODY",
          "text": "Halo {{1}}, tagihan Anda untuk periode {{2}} sejumlah Rp {{3}}. Mohon dibayar sebelum {{4}}.",
          "examples": ["Budi", "November 2025", "250.000", "10 Desember 2025"]
        }
      ],
      "status": "APPROVED",
      "waStatus": "APPROVED",
      "createdAt": "2025-12-10T10:00:00Z"
    }
  ],
  "currPage": 1,
  "haveNext": false,
  "totalPage": 1
}
```

---

### 5. Delete WhatsApp Template

**Endpoint:** `DELETE /api/v1/instant-link/wa-templates/:id`

**Authorization:** ADMIN, OWNER

**Description:** Menghapus (soft delete) template. Biasanya hanya diizinkan untuk template dengan status `DRAFT`.

**Headers:**
```text
Authorization: Bearer <your-token>
```

**URL Parameters:**
- `id` (required): ID template

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

## Template Workflow Endpoints

### 6. Submit Template for Internal Review

**Endpoint:** `POST /api/v1/instant-link/wa-templates/:id/submit`

**Authorization:** ADMIN, OWNER

**Description:** Submit template untuk review internal admin. Biasanya mengubah `status` dari `DRAFT` menjadi `SUBMITTED`.

**Headers:**
```text
Authorization: Bearer <your-token>
```

**URL Parameters:**
- `id` (required): ID template

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
    "submittedAt": "2025-12-10T10:30:00Z"
  }
}
```

---

### 7. Approve Template (Admin Only)

**Endpoint:** `POST /api/v1/instant-link/wa-templates/:id/approve`

**Authorization:** ADMIN only

**Description:** Menyetujui template yang sudah di-submit. Mengubah `status` menjadi `APPROVED`. Bisa sekaligus menyimpan `waTemplateId` jika sudah ada di Meta.

**Headers:**
```text
Authorization: Bearer <your-token>
Content-Type: application/json
```

**URL Parameters:**
- `id` (required): ID template

**Request Body:**
```json
{
  "waTemplateId": "optional-wa-template-id-from-meta"
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
    "approvedAt": "2025-12-10T10:45:00Z"
  }
}
```

---

### 8. Reject Template (Admin Only)

**Endpoint:** `POST /api/v1/instant-link/wa-templates/:id/reject`

**Authorization:** ADMIN only

**Description:** Menolak template yang diajukan dengan alasan tertentu. Mengubah `status` menjadi `REJECTED`.

**Headers:**
```text
Authorization: Bearer <your-token>
Content-Type: application/json
```

**URL Parameters:**
- `id` (required): ID template

**Request Body:**
```json
{
  "reason": "Template tidak sesuai dengan kebijakan WhatsApp"
}
```

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-templates/template-uuid-12345/reject" \
  -H "Authorization: Bearer ADMIN_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Konten terlalu promosi tanpa opt-in yang jelas"
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
    "rejectedReason": "Konten terlalu promosi tanpa opt-in yang jelas"
  }
}
```

---

### 9. Submit Template to WhatsApp

**Endpoint:** `POST /api/v1/instant-link/wa-templates/:id/submit-to-whatsapp`

**Authorization:** ADMIN, OWNER

**Description:** Submit template yang sudah disetujui ke WhatsApp Business API (Meta) untuk proses review.

**Headers:**
```text
Authorization: Bearer <your-token>
Content-Type: application/json
```

**URL Parameters:**
- `id` (required): ID template

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

## Template Examples by Type

### Text Only Template

Template sederhana hanya dengan komponen BODY, tanpa header/footer/buttons.

```json
{
  "name": "simple_text_notification",
  "category": "UTILITY",
  "language": "id",
  "clientId": "client-123",
  "components": [
    {
      "type": "BODY",
      "text": "Halo {{1}}, terima kasih telah menggunakan layanan kami.",
      "examples": ["Budi"]
    }
  ]
}
```

### Template with Header Text

Header berupa teks singkat, misalnya judul notifikasi.

```json
{
  "name": "header_text_notification",
  "category": "UTILITY",
  "language": "id",
  "clientId": "client-123",
  "components": [
    {
      "type": "HEADER",
      "format": "TEXT",
      "text": "Pemberitahuan Tagihan"
    },
    {
      "type": "BODY",
      "text": "Halo {{1}}, tagihan Anda untuk periode {{2}} sejumlah Rp {{3}}.",
      "examples": ["Budi", "November 2025", "250.000"]
    }
  ]
}
```

### Template with Header Image

Header berupa gambar (image). **PENTING: Upload image terlebih dahulu menggunakan endpoint `upload-img` untuk mendapatkan file handle.**

**Step 1: Upload Image**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-templates/upload-img" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"fileName": "promo-banner.jpg", "file": "BASE64_IMAGE_DATA"}'

# Response: 
# {
#   "code": 0,
#   "messages": "Success",
#   "data": "4::aW1hZ2UvanBlZw==:ARaK7h5VRxxx..."
# }
```

**Step 2: Create Template with File Handle**
```json
{
  "name": "promo_with_image_header",
  "waTemplate": "promo_with_image_header",
  "category": "MARKETING",
  "language": "id",
  "clientId": "client-123",
  "components": [
    {
      "type": "HEADER",
      "format": "IMAGE",
      "example": {
        "header_handle": "4:dGVzdF9pbWFnZV93YV8xLnBuZw==:aW1hZ2UvcG5n:ARYz3CMQ6pwZEinoSSbAmS3aa0CcrVvUcYgJKIU2zuBPhXPt3D3Dk1PJFPPg3WjyxMqHRcMOwEMR0PoF-6RzjJFZ_APUlocCVCUtMaPPpgd-Nw:e:1765841586:2639195729779153:61583676055393:ARaXipi9Hc_ovKvWICc"
      }
    },
    {
      "type": "BODY",
      "text": "🎉 Promo spesial untuk {{1}}! Dapatkan diskon {{2}}% untuk pembayaran tagihan bulan ini.",
      "examples": ["pelanggan setia", "20"]
    },
    {
      "type": "FOOTER",
      "text": "Syarat dan ketentuan berlaku"
    }
  ]
}
```

> **Catatan Penting - Perbedaan Template Creation vs Message Sending:**
> 
> **1. TEMPLATE CREATION (API ini):**
> - `example.header_handle`: File handle string yang didapat dari endpoint `/api/v1/instant-link/wa-templates/upload-img`
> - Field ini **WAJIB** untuk template dengan header IMAGE/VIDEO/DOCUMENT
> - File handle ini digunakan WhatsApp untuk validasi dan approval template
> - Format: `"example": {"header_handle": "4::base64:ARxxx..."}`
> - File handle hanya perlu di-upload SEKALI saat membuat template
>
> **2. MESSAGE SENDING (Lihat wa-send-message-api.md):**
> - Saat mengirim pesan menggunakan template ini, Anda mengirimkan **media URL atau Media ID yang BERBEDA**
> - Bisa menggunakan image/video yang berbeda untuk setiap pesan
> - Format: `{"type": "image", "image": {"link": "https://your-url.jpg"}}` atau `{"type": "image", "image": {"id": "meta-media-id"}}`
> - Media untuk message sending bisa dari:
>   - URL publik (upload ke server Anda sendiri atau gunakan endpoint `/api/v1/instant-link/wa-send/upload-image`)
>   - Media ID dari Meta (setelah upload ke WhatsApp Business API)
>
> **Contoh:**
> - Template creation menggunakan `header_handle`: `"4::aW1hZ2UvanBlZw==:ARaK7h5VRxxx..."`
> - Message sending menggunakan URL: `"https://ilbedev.kitamandiri.com/api/v1/instant-link/wa-send/image/01KC8AXS5FVB6TXJXFSK5776JK.jpg"`

### Template with Header Video

Header berupa video (video). **PENTING: Upload video terlebih dahulu menggunakan endpoint `upload-img` untuk mendapatkan file handle.**

> **Note:** Meskipun endpoint nya bernama "upload-img", endpoint ini juga mendukung upload video dan document.

```json
{
  "name": "tutorial_video_header",
  "waTemplate": "tutorial_video_header",
  "category": "UTILITY",
  "language": "id",
  "clientId": "client-123",
  "components": [
    {
      "type": "HEADER",
      "format": "VIDEO",
      "example": {
        "header_handle": "4::dmlkZW8vbXA0:ARaK7h5VRxxx..."
      }
    },
    {
      "type": "BODY",
      "text": "Halo {{1}}, berikut tutorial penggunaan layanan kami. Silakan tonton video di atas.",
      "examples": ["Pelanggan"]
    }
  ]
}
```

> **Catatan - Perbedaan Template Creation vs Message Sending:**
> 
> **Untuk TEMPLATE CREATION:**
> - `example.header_handle` adalah **WAJIB** untuk template dengan format VIDEO atau DOCUMENT
> - Upload video/document menggunakan endpoint yang sama: `/api/v1/instant-link/wa-templates/upload-img`
> - File handle digunakan untuk validasi template di WhatsApp
> - Format: `"example": {"header_handle": "4::dmlkZW8vbXA0:ARxxx..."}`
>
> **Untuk MESSAGE SENDING:**
> - Saat mengirim pesan, gunakan video URL atau Media ID yang BERBEDA
> - Format: `{"type": "video", "video": {"link": "https://your-video-url.mp4"}}` atau `{"type": "video", "video": {"id": "meta-media-id"}}`
> - Setiap pesan bisa menggunakan video yang berbeda
> - Lihat `wa-send-message-api.md` untuk detail lengkap

### Template with Footer

Template dengan footer sebagai catatan singkat.

```json
{
  "name": "bill_with_footer",
  "category": "UTILITY",
  "language": "id",
  "clientId": "client-123",
  "components": [
    {
      "type": "BODY",
      "text": "Halo {{1}}, tagihan Anda sebesar Rp {{2}} telah terbit.",
      "examples": ["Budi", "250.000"]
    },
    {
      "type": "FOOTER",
      "text": "Terima kasih atas kepercayaan Anda"
    }
  ]
}
```

### Template with Quick Reply Buttons

Menggunakan tombol quick reply untuk memudahkan respon user.

```json
{
  "name": "customer_support_quick_reply",
  "category": "UTILITY",
  "language": "id",
  "clientId": "client-123",
  "components": [
    {
      "type": "HEADER",
      "format": "TEXT",
      "text": "Layanan Pelanggan"
    },
    {
      "type": "BODY",
      "text": "Halo {{1}}, ada yang bisa kami bantu? Silakan pilih salah satu menu di bawah.",
      "examples": ["Budi"]
    },
    {
      "type": "FOOTER",
      "text": "Tim kami siap membantu Anda"
    },
    {
      "type": "BUTTONS",
      "buttons": [
        {
          "type": "QUICK_REPLY",
          "text": "Cek Tagihan"
        },
        {
          "type": "QUICK_REPLY",
          "text": "Lapor Gangguan"
        },
        {
          "type": "QUICK_REPLY",
          "text": "Hubungi CS"
        }
      ]
    }
  ]
}
```

**Notes:**
- Maksimal 3 tombol `QUICK_REPLY`.
- Teks tombol maksimal 25 karakter.

### Template with Call-to-Action Buttons

Menggunakan tombol URL dan/atau PHONE_NUMBER.

```json
{
  "name": "order_confirmation_cta",
  "category": "UTILITY",
  "language": "id",
  "clientId": "client-123",
  "components": [
    {
      "type": "BODY",
      "text": "Pesanan Anda #{{1}} telah dikonfirmasi. Total: Rp {{2}}.",
      "examples": ["ORD-12345", "150.000"]
    },
    {
      "type": "BUTTONS",
      "buttons": [
        {
          "type": "URL",
          "text": "Lihat Detail",
          "url": "https://example.com/orders/{{1}}"
        },
        {
          "type": "PHONE_NUMBER",
          "text": "Hubungi Kami",
          "phoneNumber": "+6281234567890"
        }
      ]
    }
  ]
}
```

**Notes:**
- Tombol `URL` mendukung placeholder `{{1}}` di URL.
- Kombinasi tombol mengikuti batasan WhatsApp (umumnya maksimal 2 tombol URL atau 1 tombol PHONE_NUMBER).

### Complete Template with All Components

Contoh template yang menggunakan header media, body dengan variabel, footer, dan tombol quick reply.

**Step 1: Upload image terlebih dahulu**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-templates/upload-img" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"fileName": "water-bill-header.jpg", "file": "BASE64_IMAGE_DATA"}'

# Response: {"code":0,"messages":"Success","data":"4::aW1hZ2UvanBlZw==:ARaK7h5VRxxx..."}
```

**Step 2: Create template dengan semua komponen**
```json
{
  "name": "water_bill_complete_template",
  "waTemplate": "water_bill_complete_template",
  "category": "UTILITY",
  "language": "id",
  "clientId": "client-123",
  "components": [
    {
      "type": "HEADER",
      "format": "IMAGE",
      "example": {
        "header_handle": "4::aW1hZ2UvanBlZw==:ARaK7h5VRxxx..."
      }
    },
    {
      "type": "BODY",
      "text": "Halo {{1}},\n\nTagihan air Anda untuk periode {{2}} adalah Rp {{3}} dengan pemakaian {{4}} m³. Jatuh tempo pada {{5}}.",
      "examples": ["Budi Santoso", "November 2025", "250.000", "15", "10 Desember 2025"]
    },
    {
      "type": "FOOTER",
      "text": "PDAM - Melayani dengan hati"
    },
    {
      "type": "BUTTONS",
      "buttons": [
        {
          "type": "QUICK_REPLY",
          "text": "Bayar Sekarang"
        },
        {
          "type": "QUICK_REPLY",
          "text": "Lihat Rincian"
        },
        {
          "type": "QUICK_REPLY",
          "text": "Hubungi CS"
        }
      ]
    }
  ]
}
```

---

## Status & Workflow

### Internal Status (`status`)

1. `DRAFT` → Template baru dibuat, masih bisa diedit
2. `SUBMITTED` → Template diajukan untuk review internal admin
3. `APPROVED` → Disetujui admin, siap dikirim ke WhatsApp
4. `REJECTED` → Ditolak admin, dapat diperbaiki lalu diajukan ulang

### WhatsApp Status (`waStatus`)

1. `PENDING` → Sudah dikirim ke Meta, menunggu review
2. `APPROVED` → Disetujui Meta, bisa digunakan untuk kirim pesan
3. `REJECTED` → Ditolak Meta, perlu revisi konten

---

## Common Error Codes

| Status Code | Description |
|-------------|-------------|
| 400 | Bad Request - Input tidak valid atau field wajib kosong |
| 401 | Unauthorized - Token autentikasi tidak ada / tidak valid |
| 403 | Forbidden - User tidak memiliki hak akses ke resource ini |
| 404 | Not Found - Template tidak ditemukan |
| 409 | Conflict - Duplikasi nama template atau constraint lain |
| 500 | Internal Server Error - Terjadi kesalahan di server |

---

**Last updated:** 2025-12-10
