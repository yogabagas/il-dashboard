# Campaign Executor Worker

## Overview

Campaign Executor adalah background worker yang secara otomatis mengeksekusi campaign yang berstatus `running`. Worker ini berjalan sebagai goroutine terpisah dan secara periodik memeriksa campaign yang perlu diproses.

## How It Works

### 1. **Worker Lifecycle**

Worker dimulai saat aplikasi start di `main.go`:
```go
workers.GetCampaignExecutor().Start()
```

### 2. **Execution Flow**

```
┌─────────────────────────────────────────┐
│  Campaign Executor Worker Started      │
│  (Check every 10 seconds)               │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│  Query all campaigns with               │
│  status = 'running'                     │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│  For each running campaign:             │
│  - Get pending recipients (batch)       │
│  - Send messages                        │
│  - Update recipient status              │
│  - Update campaign stats                │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│  If no more pending recipients:         │
│  - Mark campaign as 'completed'         │
└─────────────────────────────────────────┘
```

### 3. **Processing Details**

#### Batch Processing
- Worker mengambil recipients sesuai `batchSize` yang didefinisikan di campaign
- Default: 100 recipients per batch
- Setiap batch diproses secara sequential

#### Message Sending
- Untuk setiap recipient, worker memanggil `WASendService.SendTemplateMessage()`
- Support untuk WhatsApp (SMS dan Email belum diimplementasi)
- Parameters dari campaign/recipient di-merge dan dikirim ke template

#### Status Updates
**Recipient Status:**
- `pending` → `sent` (jika berhasil dikirim)
- `pending` → `failed` (jika gagal)

**Campaign Status:**
- `running` → `completed` (jika semua recipients sudah diproses)

#### Delay Between Messages
- Worker menerapkan delay sesuai `delaySeconds` di campaign
- Default: 5 detik antar message
- Berguna untuk menghindari rate limiting

### 4. **Pause/Resume Support**

Worker mendukung pause/resume campaign:
- Saat campaign di-pause (`status = paused`), worker akan berhenti memproses campaign tersebut
- Saat di-resume (`status = running`), worker akan melanjutkan dari recipients yang masih `pending`

### 5. **Statistics Tracking**

Worker secara otomatis update statistik campaign:
- `totalSent`: Total messages yang berhasil dikirim
- `totalDelivered`: Total messages yang delivered (update via webhook)
- `totalFailed`: Total messages yang gagal
- `totalRead`: Total messages yang dibaca (update via webhook)

## Configuration

### Worker Settings

| Setting | Value | Description |
|---------|-------|-------------|
| Check Interval | 10 seconds | Seberapa sering worker cek running campaigns |
| Max Campaigns | 100 | Max running campaigns yang diproses per cycle |
| Batch Processing | Yes | Process recipients in batches |

### Campaign Settings

Campaign settings yang mempengaruhi execution:

| Field | Type | Description |
|-------|------|-------------|
| `batchSize` | int | Number of recipients to process per batch |
| `delaySeconds` | int | Delay in seconds between each message |
| `type` | string | Campaign type: `whatsapp`, `sms`, `email` |

## Error Handling

### Recipient-Level Errors

Jika pengiriman ke recipient gagal:
1. Status recipient diubah ke `failed`
2. Error message disimpan di field `errorMessage`
3. `retryCount` di-increment
4. Campaign tetap berjalan untuk recipient lainnya

### Campaign-Level Errors

Jika terjadi error pada campaign level (template/sender tidak ditemukan):
1. Campaign tetap berstatus `running`
2. Error di-log
3. Worker akan mencoba lagi pada cycle berikutnya

## Monitoring

### Logs

Worker menghasilkan log yang detail:

```
[CampaignExecutor] Starting campaign executor worker...
[CampaignExecutor] Worker started, checking for running campaigns every 10 seconds
[CampaignExecutor] Found 2 running campaign(s)
[CampaignExecutor] Processing campaign campaignId=xxx name=xxx
[CampaignExecutor] Found 50 pending recipients (total pending: 500) for campaign campaignId=xxx
[CampaignExecutor] Sending message to recipient recipientId=xxx phone=628xxx campaignId=xxx
[CampaignExecutor] Recipient marked as sent recipientId=xxx messageId=xxx
[CampaignExecutor] Batch completed for campaign campaignId=xxx success=50 failed=0
[CampaignExecutor] Campaign stats updated campaignId=xxx sent=50 delivered=0 failed=0 read=0
```

### Key Metrics to Monitor

1. **Processing Rate**: Berapa recipient yang diproses per menit
2. **Success Rate**: Persentase messages yang berhasil dikirim
3. **Failed Messages**: Jumlah dan alasan kegagalan
4. **Campaign Completion Time**: Waktu yang dibutuhkan untuk menyelesaikan campaign

## Performance Considerations

### Concurrency

- Setiap running campaign diproses di goroutine terpisah
- Messages dalam satu campaign diproses secara sequential (untuk menghormati delay)
- Multiple campaigns bisa diproses parallel

### Resource Usage

- **Memory**: Minimal, karena batch processing
- **CPU**: Ringan, kebanyakan waiting di delay
- **Network**: Tergantung jumlah messages yang dikirim

### Scaling

Untuk menangani volume tinggi:
1. Increase `batchSize` untuk process lebih banyak recipients per cycle
2. Decrease `delaySeconds` jika provider mengizinkan
3. Run multiple instances dengan distributed locking (future enhancement)

## Troubleshooting

### Campaign Tidak Berjalan

**Symptoms:**
- Campaign berstatus `running` tapi tidak ada message yang terkirim

**Solutions:**
1. Cek worker logs untuk error messages
2. Verify template exists dan approved
3. Verify sender exists dan configured
4. Cek balance client cukup
5. Verify ada recipients dengan status `pending`

### Messages Terkirim Terlalu Lambat

**Symptoms:**
- Campaign berjalan tapi sangat lambat

**Solutions:**
1. Reduce `delaySeconds` di campaign settings
2. Increase `batchSize` di campaign settings
3. Cek apakah ada rate limiting dari provider

### High Failure Rate

**Symptoms:**
- Banyak recipients berstatus `failed`

**Solutions:**
1. Review error messages di recipient records
2. Validate phone number format
3. Cek template approval status
4. Verify sender configuration
5. Cek WhatsApp API credentials

## Future Enhancements

### Planned Features

1. **Retry Logic**: Auto-retry failed messages with exponential backoff
2. **Rate Limiting**: Smart rate limiting per provider
3. **Priority Queue**: Process high-priority campaigns first
4. **Distributed Execution**: Support multiple worker instances with locking
5. **SMS/Email Support**: Implement SMS dan Email sending
6. **Webhook Integration**: Real-time status updates via webhooks
7. **Scheduled Execution**: Honor `scheduledAt` timestamp
8. **Dynamic Delay**: Adjust delay based on provider response time

### Nice-to-Have

1. Real-time dashboard untuk monitoring campaigns
2. Campaign analytics dan insights
3. A/B testing support
4. Personalized send time optimization
5. Automatic list segmentation

## API Integration

Worker terintegrasi dengan services yang sudah ada:

```go
// Send WhatsApp message
services.GetWASendSvc().SendTemplateMessage(req, authInfo)

// Check balance (called before starting)
services.GetBillingService().CanClientSend(clientId, estimatedCost)

// Deduct balance (done in WASendService)
services.GetBillingService().DeductBalance(clientId, cost, description)
```

## Testing

### Manual Testing

1. **Create a test campaign:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/campaigns" \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Campaign",
    "type": "whatsapp",
    "senderId": "sender-id",
    "templateId": "template-id",
    "batchSize": 10,
    "delaySeconds": 2,
    "targets": [{"targetType": "contact", "targetId": "contact-id"}]
  }'
```

2. **Generate recipients:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/campaigns/{id}/generate-recipients" \
  -H "Authorization: Bearer TOKEN"
```

3. **Start campaign:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/campaigns/{id}/start" \
  -H "Authorization: Bearer TOKEN"
```

4. **Monitor progress:**
```bash
# Watch logs
tail -f logs/il-dashboard-*.log | grep CampaignExecutor

# Check stats
curl -X GET "http://localhost:8080/api/v1/instant-link/campaigns/{id}/stats" \
  -H "Authorization: Bearer TOKEN"
```

5. **Test pause/resume:**
```bash
# Pause
curl -X POST "http://localhost:8080/api/v1/instant-link/campaigns/{id}/pause" \
  -H "Authorization: Bearer TOKEN"

# Resume
curl -X POST "http://localhost:8080/api/v1/instant-link/campaigns/{id}/resume" \
  -H "Authorization: Bearer TOKEN"
```

### Expected Behavior

1. Worker starts automatically when application starts
2. Every 10 seconds, worker checks for running campaigns
3. For each running campaign, worker processes one batch of recipients
4. Messages are sent with the specified delay between each
5. Recipient and campaign stats are updated in real-time
6. When all recipients are processed, campaign is marked as completed

## Code Structure

```
workers/
└── campaign_executor.go
    ├── CampaignExecutor struct
    ├── Start() - Start worker
    ├── Stop() - Graceful shutdown
    ├── run() - Main worker loop
    ├── processRunningCampaigns() - Find & process campaigns
    ├── processCampaign() - Process single campaign
    ├── sendToRecipient() - Send to single recipient
    ├── sendWhatsAppMessage() - WhatsApp-specific sending
    ├── markRecipientSent() - Update sent status
    ├── markRecipientFailed() - Update failed status
    ├── updateCampaignStats() - Update campaign statistics
    └── completeCampaign() - Mark campaign as completed
```

## Summary

Campaign Executor Worker adalah komponen critical yang mengeksekusi bulk messaging campaigns secara otomatis. Worker ini:

✅ Runs automatically in background  
✅ Processes campaigns in batches  
✅ Supports pause/resume  
✅ Updates statistics in real-time  
✅ Handles errors gracefully  
✅ Integrates with billing system  
✅ Provides detailed logging  

Dengan worker ini, setelah campaign di-start, sistem akan secara otomatis mengirim messages ke semua recipients sesuai dengan konfigurasi batch size dan delay.

