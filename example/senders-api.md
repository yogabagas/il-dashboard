# Senders API Documentation

Base URL: `/api/v1/instant-link/senders`

**Authorization Required:** All endpoints require authentication

**Important Notes:**
- **ADMIN** and **OWNER** users can mengelola konfigurasi sender
- Sender merepresentasikan identitas pengirim untuk berbagai channel (WhatsApp, SMS, Email)
- Setiap sender memiliki limit harian dan bulanan untuk kontrol penggunaan
- Konfigurasi teknis per sender disimpan di field `config` (struktur fleksibel per channel)

**Access Control (Multi-Tenancy):**
- **PROVIDER (Super Admin)**: Dapat mengelola seluruh sender untuk semua client (jika diaktifkan di service layer)
- **Non-PROVIDER users (Regular Clients)**:
  - Hanya dapat membuat/melihat/mengubah/menghapus sender milik client-nya sendiri
  - `clientId` di-resolve otomatis dari user yang terautentikasi (di-enforce di service)
  - Percobaan akses sender milik client lain akan ditolak (403 Forbidden)

---

## Overview

The Senders API digunakan untuk mengelola daftar "sender" yang dipakai saat mengirim pesan melalui berbagai channel (WhatsApp, SMS, Email). Setiap sender memiliki identitas unik (`identifier`) dan konfigurasi channel yang spesifik.

**Entity Model (High Level):**
- `id`: ID unik sender
- `clientId`: ID client pemilik sender
- `type`: Tipe channel (`whatsapp`, `sms`, `email`)
- `name`: Nama sender yang mudah dibaca (label)
- `identifier`: Identitas teknis pengirim (misal nomor WA, sender ID SMS, alamat email)
- `config`: Konfigurasi tambahan dalam bentuk JSON object (per channel)
- `status`: Status sender (`active`, `inactive`, atau nilai lain sesuai implementasi)
- `isVerified`: Menandakan apakah sender sudah diverifikasi
- `dailyLimit`: Batas jumlah pesan per hari
- `monthlyLimit`: Batas jumlah pesan per bulan
- `createdBy` / `updatedBy`: Informasi audit user
- `createdAt` / `updatedAt`: Timestamp pembuatan dan update terakhir

---

## 1. Create Sender

**Endpoint:** `POST /api/v1/instant-link/senders`

**Authorization:** ADMIN, OWNER

**Description:** Membuat sender baru untuk client yang sedang login. Digunakan untuk mendaftarkan nomor WhatsApp, sender ID SMS, atau alamat email sebagai pengirim resmi.

**Headers:**
```text
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "type": "whatsapp",
  "name": "Main WhatsApp Sender",
  "identifier": "6281234567890",
  "config": {
    "bsp": "meta",
    "businessAccountId": "1234567890",
    "wabaId": "123456789012345",
    "extra": {
      "webhookUrl": "https://your-callback-url.com/whatsapp"
    }
  },
  "dailyLimit": 1000,
  "monthlyLimit": 20000
}
```

**Request Body Fields:**
- `type` (string, required): Tipe channel. Contoh: `whatsapp`, `sms`, `email`.
- `name` (string, required): Nama/label sender untuk keperluan UI.
- `identifier` (string, required): Identitas teknis pengirim (nomor, sender ID, email, dll.).
- `config` (object, optional): Konfigurasi tambahan per channel, bebas tergantung integrasi.
- `dailyLimit` (integer, optional): Batas pesan per hari untuk sender ini.
- `monthlyLimit` (integer, optional): Batas pesan per bulan untuk sender ini.

**cURL Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/senders" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "whatsapp",
    "name": "Main WhatsApp Sender",
    "identifier": "6281234567890",
    "config": {"bsp": "meta"},
    "dailyLimit": 1000,
    "monthlyLimit": 20000
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01HZQK1A2B3C4D5E6F7G8H9J0K",
    "clientId": "01CLIENT1234567890ABCDE",
    "type": "whatsapp",
    "name": "Main WhatsApp Sender",
    "identifier": "6281234567890",
    "config": {
      "bsp": "meta"
    },
    "status": "active",
    "isVerified": false,
    "dailyLimit": 1000,
    "monthlyLimit": 20000,
    "createdBy": "admin@example.com",
    "updatedBy": "admin@example.com",
    "createdAt": "2025-11-28T10:00:00Z",
    "updatedAt": "2025-11-28T10:00:00Z"
  }
}
```

**Response Fields (data):**
- `id`: ID unik sender (ULID / string).
- `clientId`: ID client pemilik sender.
- `type`: Tipe channel sender.
- `name`: Nama sender.
- `identifier`: Identitas teknis pengirim.
- `config`: Konfigurasi tambahan (JSON object).
- `status`: Status sender.
- `isVerified`: Apakah sender sudah diverifikasi.
- `dailyLimit`: Batas harian.
- `monthlyLimit`: Batas bulanan.
- `createdBy` / `updatedBy`: Audit user.
- `createdAt` / `updatedAt`: Timestamp ISO8601.

**Error Responses:**

400 Bad Request (contoh):
```json
{
  "code": 400,
  "message": "Invalid request body"
}
```

401 Unauthorized:
```json
{
  "code": 401,
  "message": "Unauthorized"
}
```

403 Forbidden:
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

409 Conflict (contoh duplikasi identifier):
```json
{
  "code": 409,
  "message": "Sender identifier already exists"
}
```

---

## 2. Update Sender

**Endpoint:** `PUT /api/v1/instant-link/senders/:id`

**Authorization:** ADMIN, OWNER

**Description:** Mengubah data sender yang sudah ada. Dapat digunakan untuk mengupdate nama, identifier, konfigurasi, status, flag verifikasi, dan limit.

**Headers:**
```text
Authorization: Bearer <your-token>
Content-Type: application/json
```

**URL Parameters:**
- `id` (required): ID sender yang akan diupdate.

**Request Body:**
```json
{
  "name": "Updated WhatsApp Sender",
  "identifier": "6289876543210",
  "config": {
    "bsp": "meta",
    "businessAccountId": "9876543210"
  },
  "status": "active",
  "isVerified": 1,
  "dailyLimit": 2000,
  "monthlyLimit": 50000
}
```

**Request Body Fields (partial update):**
- `name` (string, optional)
- `identifier` (string, optional)
- `config` (object, optional)
- `status` (string, optional)
- `isVerified` (integer, optional): 1 = verified, 0 = not verified.
- `dailyLimit` (integer, optional)
- `monthlyLimit` (integer, optional)

**cURL Example:**
```bash
curl -X PUT "http://localhost:8080/api/v1/instant-link/senders/01HZQK1A2B3C4D5E6F7G8H9J0K" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated WhatsApp Sender",
    "status": "active",
    "isVerified": 1
  }'
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01HZQK1A2B3C4D5E6F7G8H9J0K",
    "clientId": "01CLIENT1234567890ABCDE",
    "type": "whatsapp",
    "name": "Updated WhatsApp Sender",
    "identifier": "6289876543210",
    "config": {
      "bsp": "meta"
    },
    "status": "active",
    "isVerified": true,
    "dailyLimit": 2000,
    "monthlyLimit": 50000,
    "createdBy": "admin@example.com",
    "updatedBy": "admin@example.com",
    "createdAt": "2025-11-28T10:00:00Z",
    "updatedAt": "2025-11-28T11:00:00Z"
  }
}
```

**Error Responses:**

404 Not Found:
```json
{
  "code": 404,
  "message": "Sender not found"
}
```

403 Forbidden (bukan milik client yang sama):
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

---

## 3. Get Sender by ID

**Endpoint:** `GET /api/v1/instant-link/senders/:id`

**Authorization:** ADMIN, OWNER

**Description:** Mengambil detail satu sender berdasarkan ID-nya.

**Headers:**
```text
Authorization: Bearer <your-token>
```

**URL Parameters:**
- `id` (required): ID sender.

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/senders/01HZQK1A2B3C4D5E6F7G8H9J0K" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "id": "01HZQK1A2B3C4D5E6F7G8H9J0K",
    "clientId": "01CLIENT1234567890ABCDE",
    "type": "whatsapp",
    "name": "Main WhatsApp Sender",
    "identifier": "6281234567890",
    "config": {
      "bsp": "meta"
    },
    "status": "active",
    "isVerified": true,
    "dailyLimit": 1000,
    "monthlyLimit": 20000,
    "createdBy": "admin@example.com",
    "updatedBy": "admin@example.com",
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
  "message": "Sender not found"
}
```

403 Forbidden:
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

---

## 4. Search Senders

**Endpoint:** `GET /api/v1/instant-link/senders`

**Authorization:** ADMIN, OWNER

**Description:** Mencari dan melakukan listing sender dengan dukungan filter dan paginasi.

**Headers:**
```text
Authorization: Bearer <your-token>
```

**Query Parameters:**
- `filter` (optional): General text filter (misal cari berdasarkan name atau identifier).
- `type` (optional): Filter berdasarkan tipe channel (`whatsapp`, `sms`, `email`).
- `status` (optional): Filter berdasarkan status sender (`active`, `inactive`, dll.).
- `pageNo` (optional): Nomor halaman (default biasanya 1).
- `rowPerPage` (optional): Jumlah data per halaman (default biasanya 20).

**cURL Example - Semua Sender (Halaman Pertama):**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/senders?pageNo=1&rowPerPage=20" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Filter by Type:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/senders?type=whatsapp&pageNo=1&rowPerPage=10" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**cURL Example - Filter by Status:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/senders?status=active&pageNo=1&rowPerPage=10" \
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
      "clientId": "01CLIENT1234567890ABCDE",
      "type": "whatsapp",
      "name": "Main WhatsApp Sender",
      "identifier": "6281234567890",
      "config": {
        "bsp": "meta"
      },
      "status": "active",
      "isVerified": true,
      "dailyLimit": 1000,
      "monthlyLimit": 20000,
      "createdBy": "admin@example.com",
      "updatedBy": "admin@example.com",
      "createdAt": "2025-11-28T10:00:00Z",
      "updatedAt": "2025-11-28T10:00:00Z"
    }
  ],
  "currPage": 1,
  "haveNext": false,
  "totalPage": 1
}
```

**Response Fields:**
- `code`: Status code (0 = success).
- `messages`: Status message.
- `data`: Array of sender objects (lihat struktur di atas).
- `currPage`: Halaman saat ini.
- `haveNext`: Boolean, apakah masih ada halaman berikutnya.
- `totalPage`: Total halaman (atau total data, tergantung implementasi `SendPaged`).

---

## 5. Delete Sender

**Endpoint:** `DELETE /api/v1/instant-link/senders/:id`

**Authorization:** ADMIN, OWNER

**Description:** Menghapus sender berdasarkan ID. Implementasi bisa berupa soft delete atau hard delete tergantung layer repository/service.

**Headers:**
```text
Authorization: Bearer <your-token>
```

**URL Parameters:**
- `id` (required): ID sender yang akan dihapus.

**cURL Example:**
```bash
curl -X DELETE "http://localhost:8080/api/v1/instant-link/senders/01HZQK1A2B3C4D5E6F7G8H9J0K" \
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
  "message": "Sender not found"
}
```

403 Forbidden:
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

---

## Additional Notes

- Pastikan limit harian dan bulanan (`dailyLimit`, `monthlyLimit`) disesuaikan dengan kontrak dan kapasitas infrastruktur Anda.
- Field `config` dapat berbeda per integrasi (WhatsApp BSP, SMS gateway, Email provider, dll.), sesuaikan struktur JSON sesuai kebutuhan.
- Perubahan pada sender dapat berdampak pada pengiriman pesan yang sedang berjalan; lakukan perubahan di window maintenance bila diperlukan.

