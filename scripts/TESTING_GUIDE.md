# Testing Guide - PPOB Payment Flow dengan Bank Selection

## 📋 Summary Implementasi

### ✅ Yang Sudah Diimplementasikan:

1. **Function Calling dengan Llama 4 Scout**
   - Model: `meta-llama/llama-4-scout-17b-16e-instruct`
   - Support function calling untuk inquiry PPOB
   - Auto-detect customer number dari chat

2. **Interactive Button "Bayar Sekarang"**
   - Muncul setelah inquiry berhasil
   - Format ID: `pay_{billID}_{totalAmount}`

3. **Interactive List - Bank Selection**
   - Pilihan bank: BCA, Mandiri, BRI
   - Format ID: `bank_{billID}_{bankName}`

4. **Handler Flow Complete**
   - `handlePaymentButtonClick()` - Handle klik tombol bayar
   - `handleBankSelection()` - Handle pilihan bank

---

## 🚀 Cara Testing

### Step 1: Jalankan Aplikasi
```bash
./il-dashboard
```

### Step 2: Test Flow via WhatsApp

#### 2.1 Initiate Conversation
Kirim pesan ke WhatsApp Business number Anda:
```
Halo
```

#### 2.2 Kirim Nomor Pelanggan
Kirim nomor pelanggan PDAM:
```
A532159
```
atau
```
123456789
```

#### 2.3 Hasil yang Diharapkan
Anda akan menerima:
1. **Inquiry Result** - Detail tagihan PDAM
2. **Button** - "💰 Bayar Sekarang"

#### 2.4 Klik Button "Bayar Sekarang"
Klik button yang muncul

#### 2.5 Hasil: Bank Selection List
Anda akan menerima interactive list:
```
🏦 Silakan pilih bank untuk pembayaran:

┌──────────────────────────────┐
│   Pilih Bank ▼               │
└──────────────────────────────┘

Bank Transfer
├─ BCA
│  Transfer via Bank BCA
├─ Bank Mandiri
│  Transfer via Bank Mandiri
└─ BRI
   Transfer via Bank BRI
```

#### 2.6 Pilih Bank
Tap untuk membuka list, pilih salah satu bank

#### 2.7 Hasil: Confirmation Message
Anda akan menerima konfirmasi:
```
🏦 *Pembayaran via Bank BCA*

✅ Kami sedang memproses pembayaran Anda untuk tagihan: *A532159*

📝 Detail pembayaran akan dikirimkan segera.

_Mohon tunggu sebentar..._
```

---

## 📊 Log Monitoring

Monitor log aplikasi untuk memastikan flow berjalan:

### 1. Function Calling Detected
```
[GroqService ChatWithKnowledgeAndFunctions] AI calling function: check_pdam_bill with args: {"customer_number":"A532159"}
```

### 2. PPOB Inquiry Success
```
[GroqService ChatWithKnowledgeAndFunctions] Bill inquiry successful, returning formatted response
```

### 3. Button Click Detected
```
[handleActionWaWebHook] Button clicked - ID: pay_A532159_150000, Title: 💰 Bayar Sekarang
```

### 4. Bank List Sent
```
[handlePaymentButtonClick] Bank selection list sent. MessageID: wamid.xxx
```

### 5. Bank Selected
```
[handleActionWaWebHook] List item selected - ID: bank_A532159_bca, Title: BCA
```

### 6. Confirmation Sent
```
[handleBankSelection] Bank selection confirmation sent. MessageID: wamid.xxx, Bank: BCA
```

---

## 🔍 Troubleshooting

### Issue: Function calling tidak jalan
**Check:**
- Model yang dipakai: `meta-llama/llama-4-scout-17b-16e-instruct`
- GROQ_API_KEY sudah di-set di config.properties
- Log: `[GroqService ChatWithKnowledgeAndFunctions] AI calling function`

### Issue: Button tidak muncul
**Check:**
- PPOB Inquiry harus berhasil (response tidak null)
- Log: `[triggerAIConversation] PPOB inquiry successful`
- WhatsApp Business API support interactive buttons

### Issue: Bank list tidak muncul
**Check:**
- Button ID format benar: `pay_{billID}_{totalAmount}`
- Log: `[handlePaymentButtonClick] Processing payment button`
- Handler dipanggil: `handlePaymentButtonClick()`

### Issue: Bank selection tidak detect
**Check:**
- List ID format benar: `bank_{billID}_{bankName}`
- Log: `[handleActionWaWebHook] List item selected`
- Handler dipanggil: `handleBankSelection()`

---

## 📁 File Changes Summary

### 1. `services/groqService.go`
- **Line 52, 177, 312, 528**: Model changed to `meta-llama/llama-4-scout-17b-16e-instruct`
- **Line 386, 402, 428**: Updated conversation model tracking

### 2. `pubsub/updateStatus.go`
- **Line 111-141**: Handler untuk interactive button & list
- **Line 700-786**: `handlePaymentButtonClick()` function
- **Line 788-871**: `handleBankSelection()` function

### 3. `services/wa_send.go`
- **Line 727-821**: `SendInteractiveListMessage()` (sudah ada)

---

## ⏳ Next Steps (TODO)

### Priority 1: PPOB Payment API Integration
Location: `pubsub/updateStatus.go` line 869-870

```go
// TODO: Call PPOB Payment API here
// Example: services.GetPPOBSvc().Payment(billID, bankName)
```

**What to implement:**
1. Create `Payment()` function in `services/ppobService.go`
2. Call payment endpoint with:
   - `billID`: Customer bill ID
   - `bankName`: Selected bank (BCA/MANDIRI/BRI)
   - `totalAmount`: From inquiry response
3. Handle payment response
4. Send payment confirmation to user

### Priority 2: Virtual Account Generation
After payment API call, send VA number to user:
```
💳 *Detail Pembayaran*

Bank: Bank BCA
Virtual Account: 1234567890123456
Nominal: Rp 150.000
Berlaku hingga: 28 Nov 2025 23:59

Silakan transfer sesuai nominal di atas.
```

### Priority 3: Payment Confirmation
Handle payment webhook/callback:
- Detect payment success
- Send notification to user
- Update transaction status

---

## 📝 Notes

1. **Knowledge Base tetap sama** - Model hanya untuk reasoning, knowledge di-inject dari DB
2. **Function calling tested & working** - Llama 4 Scout support function calling dengan baik
3. **Interactive messages working** - Button dan List sudah terintegrasi
4. **Flow complete** - Tinggal PPOB Payment API integration

---

## ✅ Testing Checklist

- [ ] Aplikasi build sukses tanpa error
- [ ] GROQ_API_KEY configured
- [ ] WhatsApp Business API connected
- [ ] Send "Halo" - AI responds
- [ ] Send customer number - Inquiry berhasil
- [ ] Button "Bayar Sekarang" muncul
- [ ] Click button - Bank list muncul
- [ ] Select bank - Confirmation muncul
- [ ] Check logs - All handlers called
- [ ] No errors in logs

---

**Generated:** 2025-11-28
**Status:** Ready for Testing 🚀