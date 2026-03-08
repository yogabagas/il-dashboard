# Campaign Contact Upload - Implementation Summary

## ✅ Feature Completed

New API endpoint to upload target contacts via CSV/XLSX files to existing campaigns.

---

## 🎯 Implementation

### New API Endpoint

```
POST /api/v1/instant-link/campaigns/:id/upload-contacts
```

### Complete Flow (As Requested)

1. ✅ **Check if campaign exists** in `campaigns` table
2. ✅ **Return 400** if campaign not found or unauthorized
3. ✅ **Check Redis** for key: `campaign:target:{campaignId}:client:{clientId}`
4. ✅ **Return 201** if already uploaded (idempotency)
5. ✅ **Parse file** (CSV/XLSX) and **bulk insert** to `campaign_targets` table
6. ✅ **Retry mechanism** (3 attempts with exponential backoff)
7. ✅ **Publish to NATS** topic: `campaign.target.{clientId}`

---

## 📁 Files Modified

### 1. `controllers/campaigns.go`

**Added:**
- Route: `gocom.POST("/api/v1/instant-link/campaigns/:id/upload-contacts", ...)`
- Function: `uploadContacts()` - Main endpoint handler (lines ~143-305)
- Function: `parseContactsFile()` - Parse CSV/XLSX files (lines ~337-419)
- Function: `bulkInsertTargetsWithRetry()` - Bulk insert with retry (lines ~421-501)

**Added Imports:**
```go
"bytes"
"encoding/csv"
"encoding/json"
"fmt"
"io"
"mime/multipart"
"path/filepath"
"strings"
"time"
"github.com/oklog/ulid/v2"
"github.com/xuri/excelize/v2"
"gitlab.com/anti_metter/switching_common/campaign"
"gitlab.com/anti_metter/switching_common/campaignTarget"
customLogger "gitlab.com/bot3342545/il-dashboard/logger"
```

### 2. `go.mod`

**Added dependency:**
```go
github.com/xuri/excelize/v2 v2.8.0
```

---

## 🔧 Technical Details

### Redis Deduplication

**Key Pattern:**
```
campaign:target:{campaignId}:client:{clientId}
```

**Value:** `"uploaded"`  
**TTL:** 24 hours (86400 seconds)

**Purpose:** Prevents duplicate uploads for the same campaign

### Retry Mechanism

**Max Retries:** 3 attempts per batch  
**Backoff Strategy:** Exponential
- Attempt 1 → No wait
- Attempt 2 → Wait 500ms
- Attempt 3 → Wait 1000ms

**Batch Size:** 100 targets per batch

### NATS Pubsub Event

**Topic:** `campaign.target.{clientId}`

**Payload:**
```json
{
  "campaignId": "01GW5Z2TCVH8K9",
  "clientId": "client-123",
  "totalTargets": 150,
  "uploadedBy": "user-456",
  "uploadedAt": "2026-01-23T12:34:56Z",
  "filename": "contacts.csv"
}
```

---

## 📊 File Format

### CSV Format
```csv
phone,name,field1,field2
628123456789,John Doe,value1,value2
628987654321,Jane Smith,value3,value4
```

### XLSX Format
| A | B | C | D |
|---|---|---|---|
| phone | name | field1 | field2 |
| 628123456789 | John Doe | value1 | value2 |
| 628987654321 | Jane Smith | value3 | value4 |

**Notes:**
- **First row:** Header (automatically skipped)
- **Column A (required):** Phone number or email
- **Other columns:** Optional (currently not used, but preserved for future)
- **Duplicates:** Automatically removed
- **Empty rows:** Automatically skipped

---

## 🚀 Setup & Usage

### 1. Install Dependencies

```bash
cd /Users/yogabagas/Project/gitlab/kilat/il-dashboard

# Install excelize for XLSX support
go get github.com/xuri/excelize/v2@v2.8.0

# Update dependencies
go mod tidy

# Update vendor (if using)
go mod vendor
```

### 2. Test the API

#### Step A: Create Campaign
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/campaigns" \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Campaign",
    "type": "whatsapp",
    "senderId": "SENDER_ID",
    "templateId": "TEMPLATE_ID",
    "targets": []
  }'
```

**Response:** Get `campaignId` from response

#### Step B: Upload Contacts
```bash
curl -X POST \
  "http://localhost:8080/api/v1/instant-link/campaigns/{campaignId}/upload-contacts" \
  -H "Authorization: Bearer TOKEN" \
  -F "file=@examples/sample_recipients.csv"
```

**Expected Response (First Upload):**
```json
{
  "success": true,
  "data": {
    "success": true,
    "message": "Contacts uploaded successfully",
    "campaignId": "01GW5Z2TCVH8K9",
    "totalTargets": 5,
    "filename": "sample_recipients.csv"
  }
}
```

**Expected Response (Duplicate Upload):**
```json
{
  "success": true,
  "data": {
    "success": true,
    "message": "Contacts already uploaded for this campaign",
    "campaignId": "01GW5Z2TCVH8K9",
    "cached": true
  }
}
```

#### Step C: Verify Targets
```bash
# Check campaign_targets table
SELECT * FROM campaign_targets WHERE campaign_id = '{campaignId}';

# Check Redis key
redis-cli GET "campaign:target:{campaignId}:client:{clientId}"

# Check NATS topic
# Subscribe to: campaign.target.{clientId}
```

---

## 📝 Response Codes

| Code | Scenario | Message |
|------|----------|---------|
| 201 | Success - First upload | "Contacts uploaded successfully" |
| 201 | Success - Duplicate | "Contacts already uploaded for this campaign" |
| 400 | Campaign not found | "Campaign not found" |
| 400 | Missing file | "File is required..." |
| 400 | Invalid file type | "Invalid file type..." |
| 400 | Empty file | "No valid contacts found in file" |
| 403 | Unauthorized | "Not allowed" |
| 500 | Insert failed after retries | "Failed to insert contacts..." |

---

## 🔍 Logging

All operations logged with:
- **Component:** `CampaignsController`
- **Functions:**
  - `uploadContacts` - Main entry point
  - `parseContactsFile` - File parsing
  - `bulkInsertTargetsWithRetry` - Database insertion

**Example log:**
```json
{
  "level": "info",
  "component": "CampaignsController",
  "function": "uploadContacts",
  "campaign_id": "01GW5Z2TCVH8K9",
  "total_targets": 150,
  "message": "Upload contacts completed successfully"
}
```

---

## 🛡️ Error Handling

### Retry Mechanism

If database insert fails:
1. **Attempt 1** → Immediate retry
2. **Attempt 2** → Wait 500ms, retry
3. **Attempt 3** → Wait 1000ms, retry
4. **All failed** → Return 500 error

### Graceful Degradation

- **Redis failure:** Logs warning, continues processing (no deduplication)
- **Pubsub failure:** Logs error, request still succeeds (notification skipped)
- **Parse error:** Returns 400 immediately (no partial insert)

---

## 🎯 Data Flow Diagram

```
Client uploads file
       ↓
[1] Check campaign exists → 400 if not found
       ↓
[2] Check Redis key → 201 if exists
       ↓
[3] Parse CSV/XLSX file
       ↓
[4] Deduplicate contacts (in-memory map)
       ↓
[5] Bulk insert (batches of 100)
    ├─ Retry on failure (max 3 attempts)
    └─ Exponential backoff
       ↓
[6] Publish to NATS
       ↓
[7] Set Redis key (TTL: 24h)
       ↓
Return 201 Created
```

---

## 🧪 Testing Checklist

- [ ] Upload valid CSV file
- [ ] Upload valid XLSX file
- [ ] Upload duplicate file (verify 201 cached response)
- [ ] Upload to non-existent campaign (verify 400)
- [ ] Upload invalid file type (.txt, .pdf) (verify 400)
- [ ] Upload empty file (verify 400)
- [ ] Check campaign_targets table populated
- [ ] Check Redis key created
- [ ] Check NATS message published
- [ ] Test with large file (1000+ rows)
- [ ] Test retry mechanism (disconnect DB during upload)

---

## 📦 Dependencies

### New Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| github.com/xuri/excelize/v2 | v2.8.0 | Parse XLSX files |

### Why excelize?
- Pure Go (no C dependencies)
- Well-maintained (active development)
- Memory efficient
- Supports streaming for large files
- BSD-3-Clause license (permissive)

---

## 🚨 Important Notes

### Idempotency

- Uploading the same file multiple times returns **201 Created** (not 409 Conflict)
- Redis key expires after **24 hours**
- After expiry, same file can be uploaded again

### Performance

- **Batch Size:** 100 targets per database transaction
- **Memory:** File loaded into memory (limit: ~50MB recommended)
- **Recommended Max:** 50,000 rows per file
- **Large files:** Split into multiple uploads

### Security

- ✅ Authentication required (Bearer token)
- ✅ Authorization check (ADMIN, OWNER roles only)
- ✅ Campaign ownership validation (client_id check)
- ✅ File type validation (.csv, .xlsx only)
- ⚠️ No virus scanning (consider adding)
- ⚠️ No file size limit enforced (relies on framework default)

---

## 📚 Related Documentation

- **API Guide:** `docs/UPLOAD_CAMPAIGN_CONTACTS_API.md`
- **Setup Guide:** `SETUP_CAMPAIGN_UPLOAD.md`
- **Sample File:** `examples/sample_recipients.csv`

---

## 🔗 Integration Points

### Tables Used
- `campaigns` - Validate campaign exists
- `campaign_targets` - Store uploaded contacts

### External Services
- **Redis** - Deduplication cache (`gocom.KeyVal()`)
- **NATS** - Event notification (`gocom.PubSub()`)

### Error Codes Used
- `common.ERR_NOT_ALLOWED` - Unauthorized access
- `gocom.NewError(400, ...)` - Validation errors
- `gocom.NewError(500, ...)` - Server errors

---

## ✅ Next Steps (For You)

1. **Install dependency:**
   ```bash
   go get github.com/xuri/excelize/v2@v2.8.0
   go mod tidy
   ```

2. **Test the endpoint:**
   - Use `examples/sample_recipients.csv`
   - Check logs for detailed execution flow
   - Verify data in `campaign_targets` table

3. **Monitor:**
   - Redis keys: `redis-cli KEYS "campaign:target:*"`
   - NATS topics: Subscribe to `campaign.target.*`
   - Database: `SELECT * FROM campaign_targets WHERE campaign_id = '...'`

4. **Production checklist:**
   - [ ] Add file size validation
   - [ ] Add rate limiting
   - [ ] Monitor memory usage with large files
   - [ ] Set up alerts for failed uploads
   - [ ] Document API for frontend team

---

## 🎉 Ready to Use!

After running `go get` and `go mod tidy`, your API is ready to accept file uploads!
