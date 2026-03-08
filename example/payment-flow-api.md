# PPOB Payment Flow API Documentation

## Overview
API flow untuk pembayaran tagihan PDAM melalui WhatsApp dengan Virtual Account.

## Flow Diagram
```
User → Inquiry → Payment Button → Bank Selection → Payment Process → VA Generated
```

## 1. Inquiry Flow

### User Input
User mengirim nomor pelanggan PDAM via WhatsApp
```
Example: "Tolong cek tagihan 123456"
```

### System Response
AI mendeteksi inquiry request dan memanggil PPOB Inquiry API

**Endpoint Internal:** `services.PPOBSvc.Inquiry(billID)`

**Response Format:**
```json
{
  "result_cd": "0000",
  "result_msg": "Success",
  "tx_id": "TX123456789",
  "bill_id": "123456",
  "amount": 150000,
  "admin": 2500,
  "total_amount": 152500,
  "detail": {
    "product_name": "PDAM Kabupaten Tangerang",
    "customer_name": "John Doe",
    "customer_address": "Jl. Example No. 123",
    "period": "Januari 2025",
    "detail_billing": [
      {
        "period": "JAN-2025",
        "bill_amount": 150000,
        "fine": 0,
        "admin_merchant": 2500,
        "total": 152500
      }
    ]
  }
}
```

### WhatsApp Message
```
📋 INFORMASI TAGIHAN PDAM

👤 Nama: John Doe
🏠 Alamat: Jl. Example No. 123
🔢 No. Pelanggan: 123456

📅 Periode: Januari 2025
━━━━━━━━━━━━━━━━━━━━
💧 Tagihan Air: Rp 150.000
📝 Biaya Admin: Rp 2.500
━━━━━━━━━━━━━━━━━━━━
💰 TOTAL: Rp 152.500
```

**Interactive Button:**
```
[💰 Bayar Sekarang] → ButtonID: pay_123456
```

---

## 2. Payment Button Click Flow

### User Action
User clicks "Bayar Sekarang" button

### Handler
`handlePaymentButtonClick(sessionID, fromNumber, buttonID, clientId)`

**Parse ButtonID:** `pay_123456` → BillID: `123456`

### System Action
1. Retrieve cached inquiry result from Redis
   - Key: `ppob:inquiry:123456`
   - TTL: 10 minutes

2. Send bank selection list

### WhatsApp Interactive List
```
Body: "💳 Silakan pilih bank untuk mendapatkan Virtual Account:"
Button: "Pilih Bank"

Sections:
  - BCA Virtual Account (ID: bank_123456_bca)
  - Mandiri Virtual Account (ID: bank_123456_mandiri)
  - BRI Virtual Account (ID: bank_123456_bri)
```

---

## 3. Bank Selection Flow

### User Action
User selects bank from list (e.g., "BCA Virtual Account")

### Handler
`handleBankSelection(sessionID, fromNumber, listID, clientId)`

**Parse ListID:** `bank_123456_bca`
- BillID: `123456`
- Bank: `BCA`

### System Process

#### Step 1: Retrieve Inquiry Data
```go
cacheKey := fmt.Sprintf("ppob:inquiry:%s", billID)
inquiryJSON := gocom.KeyVal().Get(cacheKey)
```

#### Step 2: Send Processing Message
```
💳 BCA Virtual Account

⏳ Sedang membuat Virtual Account...
📝 Nomor pelanggan: 123456

_Mohon tunggu sebentar..._
```

#### Step 3: Call Payment API
**Endpoint Internal:** `services.PPOBSvc.Payment(inquiryResp)`

**Request:**
```json
{
  "trx_date": "20241204151530",
  "product_code": "PDAMKABTGR",
  "bill_id": "123456",
  "ref_id": "ref_01KBKS...",
  "merchant_token": "sha256_hash...",
  "user": "ktmippob",
  "merchant_code": "KTMIPPOB001",
  "category": "PDAM0001",
  "method": "PDAMTKR001",
  "tx_id": "TX123456789",
  "amount": 150000,
  "admin": 2500,
  "total_amount": 152500
}
```

**Response:**
```json
{
  "result_cd": "0000",
  "result_msg": "Success",
  "tx_id": "TX123456789",
  "status": "pending",
  "total_amount": 152500,
  "detail": {
    "customer_name": "John Doe"
  }
}
```

#### Step 4: Generate VA Number
```go
vaNumber := generateVANumber(bankName, txID)
// Example: 7777TX123456 (BCA format)
```

#### Step 5: Send Success Message

---

## 4. Payment Success Message

### WhatsApp Message Format
```
💳 7777TX123456 BERHASIL DIBUAT

✅ Virtual Account Anda sudah siap!

━━━━━━━━━━━━━━━━━━━━
📱 NOMOR VIRTUAL ACCOUNT
7777TX123456

💰 Total Pembayaran
Rp 152.500

👤 Nama Pelanggan
John Doe

🔢 No. Pelanggan
123456

⏰ Berlaku Hingga
05 Dec 2024, 15:15

🆔 ID Transaksi
TX123456789
━━━━━━━━━━━━━━━━━━━━

📝 Cara Pembayaran:

1️⃣ Buka aplikasi/ATM BCA
2️⃣ Pilih menu Transfer/Bayar
3️⃣ Masukkan nomor VA: 7777TX123456
4️⃣ Masukkan nominal: Rp 152.500
5️⃣ Konfirmasi pembayaran
6️⃣ Simpan bukti transfer

⚠️ PENTING:
• VA hanya berlaku sampai 05 Dec 2024, 15:15
• Pastikan nominal sesuai dengan tagihan
• Pembayaran akan diproses otomatis
• Simpan bukti pembayaran

💬 Butuh bantuan? Ketik bantuan
```

---

## Error Scenarios

### 1. Cache Expired
**Condition:** Inquiry data not found in Redis (>10 minutes)

**Response:**
```
⚠️ Sesi pembayaran telah berakhir.

Silakan cek tagihan kembali dengan mengirim nomor pelanggan Anda.
```

### 2. Payment API Error
**Condition:** PPOB Payment API returns error

**Response:**
```
❌ Pembayaran Gagal

[Error message from API]

Silakan coba lagi atau hubungi customer service.
```

### 3. Invalid Button/List ID
**Condition:** Malformed button or list ID

**Action:** Log error and skip processing

---

## Data Storage

### Redis Cache
- **Key:** `ppob:inquiry:{billID}`
- **Value:** JSON-encoded `PPOBInquiryResp`
- **TTL:** 10 minutes

### Message Conversation
All user/assistant messages saved to `message_conversations` table:
- User clicks button → Role: "user", Message: "[User clicked: Bayar Sekarang]"
- Bank selection → Role: "user", Message: "[User selected bank: BCA]"
- AI responses → Role: "assistant", Message: actual response text

### Message Logs
All WhatsApp messages logged to `message_logs` table with:
- Direction: inbound/outbound
- Status: sent/delivered/read
- Cost: calculated based on message type
- Session ID: links to conversation

---

## Configuration Required

### Environment Variables
```properties
# PPOB API Configuration
app.ppob.base.url=http://148.230.97.174:9302/api/v1
app.ppob.auth.url=http://148.230.97.174:9301/auth/login
app.ppob.login.email=user@kitamandiri.com
app.ppob.login.password=***
app.ppob.secret.code=***
app.ppob.merchant.user=ktmippob
app.ppob.merchant.code=KTMIPPOB001
app.ppob.product.code=PDAMKABTGR
app.ppob.category=PDAM0001
app.ppob.method=PDAMTKR001
```

---

## API Sequence Diagram

```
User          WhatsApp       System         Redis         PPOB API
 |               |             |              |               |
 |--"123456"---->|             |              |               |
 |               |--webhook--->|              |               |
 |               |             |--Inquiry---->|-------------->|
 |               |             |<-------------|<--------------|
 |               |             |--cache------>|               |
 |               |<--message---|              |               |
 |<--"Tagihan"---|             |              |               |
 |<--[Button]----|             |              |               |
 |               |             |              |               |
 |--[Click]----->|             |              |               |
 |               |--webhook--->|              |               |
 |               |<--list------|              |               |
 |<--[Banks]-----|             |              |               |
 |               |             |              |               |
 |--[BCA]------->|             |              |               |
 |               |--webhook--->|              |               |
 |               |             |--get cache-->|               |
 |               |             |<-------------|               |
 |               |             |--Payment-----|-------------->|
 |               |             |<-------------|<--------------|
 |               |<--VA msg----|              |               |
 |<--"VA: 777"---|             |              |               |
```

---

## Testing Guide

### 1. Test Inquiry
```
Input: "cek tagihan 123456"
Expected: Inquiry response + payment button
```

### 2. Test Payment Button
```
Action: Click "Bayar Sekarang"
Expected: Bank selection list
```

### 3. Test Bank Selection
```
Action: Select "BCA Virtual Account"
Expected: Processing message → VA details
```

### 4. Test Cache Expiry
```
1. Click payment button
2. Wait 11 minutes
3. Select bank
Expected: Session expired message
```

### 5. Test Error Handling
```
Simulate API error
Expected: Error message with retry instruction
```

---

## Notes

1. **VA Number Generation:** Currently using dummy format. Need to integrate actual VA generation from payment gateway.

2. **Payment Confirmation:** Need to implement webhook for payment confirmation from bank.

3. **Transaction History:** Consider adding transaction history query feature.

4. **Refund Flow:** Need to implement refund/cancellation flow for failed payments.

5. **Multi-Bank Support:** Currently supports BCA, Mandiri, BRI. Can be extended to other banks.

