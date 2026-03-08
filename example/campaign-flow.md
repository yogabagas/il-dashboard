# Campaign Flow Documentation

## Overview
Sistem campaign memungkinkan pengiriman bulk message ke banyak penerima dengan tracking dan statistik lengkap.

## Campaign Workflow

### 1. Create Campaign
```http
POST /api/v1/instant-link/campaigns
```

**Request:**
```json
{
  "name": "Campaign PDAM November",
  "type": "whatsapp",
  "senderId": "sender-id-here",
  "templateId": "template-id-here",
  "targets": [
    {
      "targetType": "contact_group",
      "targetId": "group-id-1"
    },
    {
      "targetType": "contact_group",
      "targetId": "group-id-2"
    }
  ],
  "parameters": ["John", "Doe"],
  "scheduledAt": "2025-11-28T10:00:00Z",
  "batchSize": 10,
  "delaySeconds": 2
}
```

**Response:**
```json
{
  "id": "campaign-id",
  "name": "Campaign PDAM November",
  "type": "whatsapp",
  "status": "draft",
  "senderId": "sender-id",
  "templateId": "template-id",
  "totalRecipients": 0,
  "totalSent": 0,
  "totalDelivered": 0,
  "totalFailed": 0,
  "totalRead": 0,
  "batchSize": 10,
  "delaySeconds": 2
}
```

### 2. Generate Recipients
```http
POST /api/v1/instant-link/campaigns/:id/generate-recipients
```

Akan generate daftar penerima berdasarkan targets yang dipilih (contact groups).

**Response:**
```json
{
  "success": true,
  "totalRecipients": 150
}
```

### 3. Start Campaign
```http
POST /api/v1/instant-link/campaigns/:id/start
```

Akan:
- Mengubah status campaign menjadi "running"
- Set `startedAt` timestamp
- Check balance apakah cukup untuk mengirim semua message
- Campaign executor worker akan mulai memproses

**Response:**
```json
{
  "id": "campaign-id",
  "status": "running",
  "startedAt": "2025-11-27T10:00:00Z"
}
```

### 4. Campaign Executor Worker

Worker ini berjalan setiap 10 detik dan akan:

1. **Mencari running campaigns**
   - Query database untuk campaign dengan status "running"
   
2. **Process setiap campaign**
   - Ambil batch recipients dengan status "pending" (sesuai batchSize)
   - Untuk setiap recipient:
     - Kirim message via WhatsApp
     - Update recipient status menjadi "sent"
     - Set `sentAt` timestamp
     - Simpan `messageId` dari WhatsApp
     - Apply delay antar message (delaySeconds)
   
3. **Update campaign statistics**
   - Count recipients by status:
     - TotalSent = sent + delivered + read
     - TotalDelivered = delivered + read
     - TotalFailed = failed
     - TotalRead = read
   
4. **Auto complete campaign**
   - Jika tidak ada lagi pending recipients
   - Set status = "completed"
   - Set `completedAt` timestamp

### 5. Message Logs

Setiap message yang dikirim akan:

1. **Create message log** dengan:
   ```
   - CampaignId (untuk tracking)
   - MessageId (dari WhatsApp)
   - Status: "sent"
   - Cost: calculated based on template category
   - SentAt: timestamp
   ```

2. **Message log akan ter-update** via webhook:
   - Status "delivered" → set `deliveredAt`
   - Status "read" → set `readAt`
   - Status "failed" → set `failedAt` dan `errorMessage`

### 6. Campaign Recipient Updates via Webhook

Ketika WhatsApp webhook menerima status update:

1. **Cari message log** by messageId
2. **Update message log** status dan timestamp
3. **Cari campaign recipient** by messageId
4. **Update campaign recipient**:
   - Status → "delivered" / "read" / "failed"
   - Set timestamp sesuai status
5. **Trigger campaign stats update**:
   - Async update statistik campaign
   - Recalculate TotalSent, TotalDelivered, TotalRead, TotalFailed

## Campaign Control Endpoints

### Pause Campaign
```http
POST /api/v1/instant-link/campaigns/:id/pause
```

Mengubah status menjadi "paused". Worker akan skip campaign ini.

### Resume Campaign
```http
POST /api/v1/instant-link/campaigns/:id/resume
```

Mengubah status kembali ke "running". Worker akan melanjutkan pengiriman.

## Monitoring & Statistics

### Get Campaign Details
```http
GET /api/v1/instant-link/campaigns/:id
```

### Get Campaign Statistics
```http
GET /api/v1/instant-link/campaigns/:id/stats
```

**Response:**
```json
{
  "totalRecipients": 150,
  "totalSent": 145,
  "totalDelivered": 140,
  "totalRead": 85,
  "totalFailed": 5,
  "pendingCount": 0,
  "successRate": 96.67,
  "deliveryRate": 96.55,
  "readRate": 60.71
}
```

### Get Campaign Recipients
```http
GET /api/v1/instant-link/campaigns/:id/recipients?status=sent&pageNo=1&rowPerPage=20
```

Status filter: `pending`, `sent`, `delivered`, `read`, `failed`

### Get Message Logs by Campaign
```http
GET /api/v1/instant-link/message-logs?campaignId=campaign-id&pageNo=1&rowPerPage=20
```

## Data Flow Diagram

```
[Create Campaign] → [Draft Status]
       ↓
[Generate Recipients] → [Create CampaignRecipients with status=pending]
       ↓
[Start Campaign] → [Status = running]
       ↓
[Campaign Executor Worker] (every 10s)
       ↓
   [Get Batch of Pending Recipients]
       ↓
   [For Each Recipient]
       ↓
   [Send WhatsApp Message] → [Create MessageLog with CampaignId]
       ↓                            ↓
   [Update Recipient]          [Save to DB]
   - status = "sent"
   - messageId = xxx
   - sentAt = now
       ↓
   [Apply Delay]
       ↓
   [Update Campaign Stats]
       ↓
[WhatsApp Webhook] → [Status Update]
       ↓
   [Update MessageLog]
   - delivered/read/failed
   - timestamp
       ↓
   [Update CampaignRecipient]
   - delivered/read/failed
   - timestamp
       ↓
   [Trigger Campaign Stats Update]
   - Recalculate TotalDelivered
   - Recalculate TotalRead
       ↓
[All Recipients Processed] → [Status = completed]
```

## Important Notes

1. **Campaign ID Tracking**
   - Setiap message yang dikirim dari campaign akan memiliki `campaignId` di message log
   - Ini memungkinkan tracking semua message dari satu campaign

2. **Real-time Statistics**
   - Campaign statistics di-update secara async setiap ada perubahan status recipient
   - Worker juga update stats setelah selesai process batch

3. **Balance Check**
   - Sebelum start campaign, system akan check apakah balance cukup
   - Estimate cost = (pending recipients count) × (message cost)
   - Message cost berbeda berdasarkan template category:
     - Marketing: 600
     - Utility: 350
     - Authentication: 175

4. **Batch Processing**
   - `batchSize`: jumlah message yang dikirim per cycle
   - `delaySeconds`: jeda antar message untuk avoid rate limit
   - Default delay: 2 detik

5. **Error Handling**
   - Failed messages akan ditandai dengan status "failed"
   - ErrorMessage akan disimpan di recipient dan message log
   - Campaign tetap lanjut meski ada yang failed

## Troubleshooting

### TotalDelivered masih 0 setelah kirim
**Penyebab:**
- Webhook belum menerima status update dari WhatsApp
- Atau ada delay dari WhatsApp side

**Solusi:**
- Check webhook logs untuk status update
- Tunggu beberapa saat, WhatsApp kadang delay
- Verify webhook URL sudah terdaftar di Meta Dashboard

### Message logs tidak ada campaignId
**Penyebab:**
- Old code sebelum fix

**Solusi:**
- Code sudah diperbaiki
- CampaignId sekarang otomatis di-set saat send dari campaign executor

### Campaign stuck di running
**Penyebab:**
- Worker tidak berjalan
- Atau ada error di worker

**Solusi:**
- Check logs untuk "[CampaignExecutor]"
- Restart service jika perlu
- Atau manual pause/resume campaign

