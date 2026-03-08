# Payment Function API Documentation

## Overview
Fungsi `ChatWithKnowledgeAndPaymentFunction` adalah fungsi baru untuk memproses pembayaran tagihan PDAM menggunakan AI function calling, mengikuti pola yang sama dengan fungsi inquiry.

## Function Signature

```go
func (o *GroqSvcImpl) ChatWithKnowledgeAndPaymentFunction(
    userMessage string,
    clientId string, 
    from string,
    inquiryResp *dtos.PPOBInquiryResp
) (string, *dtos.PPOBPaymentResp, *gocom.CodedError)
```

## Parameters

- `userMessage` (string): Pesan dari user yang mengindikasikan konfirmasi pembayaran
- `clientId` (string): ID client untuk mendapatkan knowledge base
- `from` (string): Nomor WhatsApp pengirim
- `inquiryResp` (*dtos.PPOBInquiryResp): Response dari inquiry sebelumnya yang berisi detail tagihan

## Return Values

1. `string`: Response message yang akan dikirim ke user
2. `*dtos.PPOBPaymentResp`: Response dari PPOB payment API (nil jika tidak ada payment yang diproses)
3. `*gocom.CodedError`: Error jika terjadi kesalahan

## Features

### AI Function Calling
Fungsi ini menggunakan AI untuk mendeteksi intent pembayaran dengan mendefinisikan function tool:
- **Function Name**: `pay_pdam_bill`
- **Description**: Bayar tagihan air PDAM
- **Parameters**: 
  - `bank_code` (optional): Kode bank untuk pembayaran

### Error Handling
Menangani berbagai jenis error dengan pesan yang user-friendly:
- Saldo tidak mencukupi
- Tagihan sudah dibayar
- Error generic lainnya

### Response Formatting
Format response menggunakan `formatPaymentResponse()` yang menampilkan:
- Status pembayaran berhasil
- ID Transaksi dan tanggal
- Detail customer (nama, alamat)
- Detail pembayaran (tagihan, admin, total)
- Informasi saldo (saldo awal dan akhir)
- Nomor referensi

## Usage Example

```go
// Setelah user melakukan inquiry dan mendapat response
inquiryResp, err := GetGroqSvc().ChatWithKnowledgeAndFunctions(
    "cek tagihan A532159",
    clientId,
    from,
)

// User konfirmasi pembayaran
if inquiryResp != nil {
    responseMsg, paymentResp, err := GetGroqSvc().ChatWithKnowledgeAndPaymentFunction(
        "ya, bayar sekarang",
        clientId,
        from,
        inquiryResp,
    )
    
    if err != nil {
        // Handle error
        return err
    }
    
    // Send response message to user via WhatsApp
    // paymentResp contains the payment details if successful
}
```

## Flow Diagram

```
User Input: "bayar" / "ya bayar" / "proses pembayaran"
    ↓
AI Function Calling detects payment intent
    ↓
Call pay_pdam_bill function
    ↓
Validate inquiryResp
    ↓
Call PPOB Payment API
    ↓
Format Success/Error Response
    ↓
Return formatted message to user
```

## Response Format Examples

### Success Response
```
✅ *PEMBAYARAN BERHASIL*

🎉 Tagihan PDAM Anda telah dibayarkan!

🔖 ID Transaksi: TXN123456789
📅 Tanggal: 20241204150000
🏷️ Nomor Pelanggan: A532159

👤 Nama: JOHN DOE
📍 Alamat: JL. EXAMPLE NO. 123
📦 Produk: PDAM KABUPATEN TANGERANG

💰 *Detail Pembayaran:*
   Tagihan: Rp 150.000
   Admin: Rp 2.500
   *Total Dibayar: Rp 152.500*

💳 *Informasi Saldo:*
   Saldo Awal: Rp 1.000.000
   Saldo Akhir: Rp 847.500

📌 Referensi: REF123456789

Terima kasih telah menggunakan layanan kami! 🙏
```

### Error Response (Saldo Tidak Cukup)
```
Maaf, pembayaran tidak dapat diproses. ❌

⚠️ *Saldo tidak mencukupi*

Silakan hubungi customer service kami untuk informasi lebih lanjut.

📞 *Call Center PDAM Kabupaten Tangerang*
☎️ (021) 5951234
```

## Integration Notes

1. **Validation**: Fungsi ini memvalidasi bahwa `inquiryResp` tidak nil sebelum melanjutkan
2. **Context**: System prompt dilengkapi dengan konteks pembayaran (nomor pelanggan dan total tagihan)
3. **Timeout**: HTTP client timeout 60 detik untuk mengakomodasi proses pembayaran
4. **Logging**: Comprehensive logging untuk debugging dan monitoring
5. **Error Messages**: Pesan error yang informatif dan ramah pengguna

### PPOBPaymentResp Fields Coverage

Fungsi `formatPaymentResponse` menampilkan semua field yang tersedia di `PPOBPaymentResp`:

**Basic Information:**
- `ResultCD` - Result code
- `ResultMsg` - Result message
- `TxID` - Transaction ID
- `TxDate` - Transaction date
- `Status` - Payment status
- `BillID` - Customer number

**Customer Details (from Detail object):**
- `CustomerName` - Nama pelanggan
- `CustomerAddress` - Alamat pelanggan
- `ProductName` - Nama produk
- `Period` - Periode tagihan

**Payment Details:**
- `Amount` - Jumlah tagihan
- `Admin` - Biaya admin
- `TotalAmount` - Total yang dibayar
- `DeductedBalance` - Saldo yang terdebet
- `Qty` - Jumlah/quantity

**Balance Information:**
- `FirstBalance` - Saldo awal
- `LastBalance` - Saldo akhir

**Virtual Account Information (if applicable):**
- `VaNo` - Nomor Virtual Account
- `BankCode` - Kode bank
- `ValidUntil` - Berlaku hingga

**Additional Information:**
- `QrCode` - QR Code untuk pembayaran
- `RefID` - Referensi ID
- `Sign` - Digital signature

**Merchant Information (optional display):**
- `User` - Merchant user
- `MerchantCode` - Kode merchant
- `ProductCode` - Kode produk
- `Category` - Kategori
- `Method` - Metode pembayaran

## Pattern Consistency

Fungsi ini mengikuti pola yang sama dengan `ChatWithKnowledgeAndFunctions`:
- Struktur kode yang konsisten
- Error handling yang sama
- Logging pattern yang sama
- Response formatting yang konsisten
- AI function calling dengan tool definition
- Timeout dan HTTP client configuration yang sama

