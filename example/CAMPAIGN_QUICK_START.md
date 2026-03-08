# Campaign Quick Start Guide

## 🚀 Cara Kirim Campaign dalam 3 Langkah

### Step 1: Buat Campaign
```bash
POST /api/v1/instant-link/campaigns
{
  "name": "Notifikasi Tagihan PDAM",
  "type": "whatsapp",
  "senderId": "YOUR_SENDER_ID",
  "templateId": "YOUR_TEMPLATE_ID",
  "targets": [
    {
      "targetType": "contact_group",
      "targetId": "GROUP_ID"
    }
  ],
  "batchSize": 10,
  "delaySeconds": 2
}
```

### Step 2: Generate Recipients
```bash
POST /api/v1/instant-link/campaigns/:campaignId/generate-recipients
```

### Step 3: Start Campaign
```bash
POST /api/v1/instant-link/campaigns/:campaignId/start
```

**Done!** Worker akan otomatis kirim semua messages.

---

## 📊 Monitoring Campaign

### Check Progress
```bash
GET /api/v1/instant-link/campaigns/:campaignId/stats
```

Response:
```json
{
  "totalRecipients": 100,
  "totalSent": 100,
  "totalDelivered": 95,  // ✅ Real-time update via webhook
  "totalRead": 60,       // ✅ Real-time update via webhook
  "totalFailed": 0,
  "successRate": 100.0,
  "deliveryRate": 95.0,
  "readRate": 60.0
}
```

### View All Messages
```bash
GET /api/v1/instant-link/message-logs?campaignId=:campaignId
```

### View Recipients Detail
```bash
GET /api/v1/instant-link/campaigns/:campaignId/recipients?status=sent
```

Filter status: `pending`, `sent`, `delivered`, `read`, `failed`

---

## 🎮 Campaign Controls

### Pause Campaign
```bash
POST /api/v1/instant-link/campaigns/:campaignId/pause
```

### Resume Campaign
```bash
POST /api/v1/instant-link/campaigns/:campaignId/resume
```

---

## ⚙️ Konfigurasi

### Batch Size
Jumlah message yang dikirim per cycle (default: 10)
- Terlalu besar: risk rate limit
- Terlalu kecil: lama selesai

**Recommended:** 5-20 messages per batch

### Delay Seconds
Jeda antar message dalam detik (default: 2)
- Minimum: 1 detik
- Maximum: 10 detik

**Recommended:** 2-3 detik untuk avoid WhatsApp rate limit

---

## 💰 Message Cost (by Template Category)

| Category | Cost per Message |
|----------|------------------|
| Marketing | Rp 600 |
| Utility | Rp 350 |
| Authentication | Rp 175 |

System akan auto-check balance sebelum start campaign.

---

## ✅ Best Practices

1. **Test dulu dengan group kecil**
   - Buat test campaign dengan 5-10 recipients
   - Verifikasi template dan parameters
   
2. **Monitor during sending**
   - Check stats setiap 1-2 menit
   - Pause jika ada masalah
   
3. **Optimal batch settings**
   - BatchSize: 10
   - DelaySeconds: 2
   - = 5 messages per detik (safe rate)

4. **Balance management**
   - Ensure balance > estimated cost
   - System akan auto-check sebelum start

---

## 🔧 Troubleshooting

### Campaign tidak auto-send setelah start

**Check:**
1. Worker sudah jalan? → Check logs untuk `[CampaignExecutor]`
2. Status = "running"? → GET campaign detail
3. Ada recipients? → Check totalRecipients > 0
4. Balance cukup? → Check client balance

**Fix:**
```bash
# Restart service jika perlu
# atau pause/resume campaign
POST /api/v1/instant-link/campaigns/:id/pause
POST /api/v1/instant-link/campaigns/:id/resume
```

### TotalDelivered masih 0 padahal sudah kirim

**Reason:** WhatsApp webhook delay (normal)

**Wait:** 30-60 detik, akan auto-update

**Check webhook:** Logs untuk `[updateCampaignRecipientStatus]`

### Message logs tidak ada campaignId

**Reason:** Old messages sebelum fix

**Fix:** Sudah diperbaiki, messages baru akan include campaignId

---

## 📝 Example: Campaign Notifikasi Tagihan

```bash
# 1. Buat campaign
curl -X POST http://localhost:8080/api/v1/instant-link/campaigns \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Notifikasi Tagihan PDAM Desember 2025",
    "type": "whatsapp",
    "senderId": "sender-pdam-001",
    "templateId": "template-tagihan-id",
    "targets": [
      {
        "targetType": "contact_group",
        "targetId": "pelanggan-area-a"
      },
      {
        "targetType": "contact_group",
        "targetId": "pelanggan-area-b"
      }
    ],
    "parameters": ["Desember 2025"],
    "batchSize": 15,
    "delaySeconds": 2
  }'

# Response: campaign_id = "01KB..."

# 2. Generate recipients
curl -X POST http://localhost:8080/api/v1/instant-link/campaigns/01KB.../generate-recipients \
  -H "Authorization: Bearer TOKEN"

# Response: totalRecipients = 500

# 3. Start campaign
curl -X POST http://localhost:8080/api/v1/instant-link/campaigns/01KB.../start \
  -H "Authorization: Bearer TOKEN"

# 4. Monitor (setiap 30 detik)
curl http://localhost:8080/api/v1/instant-link/campaigns/01KB.../stats \
  -H "Authorization: Bearer TOKEN"

# 5. Tunggu sampai status = "completed"
```

**Timeline:**
- Total: 500 recipients
- Batch size: 15
- Delay: 2 seconds
- Worker cycle: 10 seconds

**Calculation:**
- Per batch: 15 messages × 2s = 30s
- Total batches: 500 / 15 = 34 batches
- Total time: 34 batches × (30s + 10s wait) = ~23 menit

---

## 🎯 Campaign Status Flow

```
draft → scheduled → running → completed
  ↓         ↓          ↓
cancelled  paused   paused
```

**Status:**
- `draft`: Baru dibuat
- `scheduled`: Ada scheduledAt (future implementation)
- `running`: Sedang dikirim oleh worker
- `paused`: Di-pause manual
- `completed`: Semua recipients sudah diproses
- `cancelled`: Di-cancel manual

---

## 📞 Support

Untuk pertanyaan atau masalah:
1. Check logs di `/logs/il-dashboard-YYYY-MM-DD.log`
2. Cari keyword: `[CampaignExecutor]` atau campaign_id
3. Review `/example/campaign-flow.md` untuk detail

**Files penting:**
- `/services/campaigns.go` - Business logic
- `/workers/campaign_executor.go` - Worker yang kirim messages
- `/controllers/campaigns.go` - API endpoints
- `/example/campaign-flow.md` - Full documentation

