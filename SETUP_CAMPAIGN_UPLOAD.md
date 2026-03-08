# Setup Campaign File Upload Feature

## Quick Setup (Run This First)

```bash
cd /Users/yogabagas/Project/gitlab/kilat/il-dashboard

# Add excelize dependency for XLSX support
go get github.com/xuri/excelize/v2@v2.8.0

# Update go.mod and go.sum
go mod tidy

# Update vendor directory (if using vendoring)
go mod vendor

# Test compilation
go build
```

## Alternative: Manual Setup

If `go get` fails due to network issues, update `go.mod` manually:

```go
require (
	// ... existing dependencies ...
	github.com/xuri/excelize/v2 v2.8.0
)
```

Then run:
```bash
GOPROXY=https://goproxy.io,direct go mod tidy
go mod vendor
```

## ✅ Implementation Complete

The feature is now ready with:

### 1. **New API Endpoint**
```
POST /api/v1/instant-link/campaigns/:id/upload-contacts
```

### 2. **Complete Flow Implementation**

✅ Step 1: Check if campaign exists in database  
✅ Step 2: Return 400 if campaign not found  
✅ Step 3: Check Redis with key `campaign:target:{campaignId}:client:{clientId}`  
✅ Step 4: Return 201 if already uploaded (idempotency)  
✅ Step 5: Parse file (CSV/XLSX) and bulk insert with retry (3 attempts)  
✅ Step 6: Publish to NATS topic `campaign.target.{clientId}`  

### 3. **Features Included**

- ✅ CSV and XLSX support
- ✅ Redis deduplication (24-hour TTL)
- ✅ Bulk insert with retry mechanism (3 attempts with exponential backoff)
- ✅ Auto-deduplication of contacts
- ✅ Batch processing (100 records per batch)
- ✅ NATS pubsub notification
- ✅ Comprehensive logging
- ✅ Error handling

## API Usage

### cURL Example

```bash
curl -X POST \
  "http://localhost:8080/api/v1/instant-link/campaigns/YOUR_CAMPAIGN_ID/upload-contacts" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@contacts.csv"
```

### File Format (CSV)

```csv
phone,name,custom_field
628123456789,John Doe,value1
628987654321,Jane Smith,value2
628111222333,Bob Johnson,value3
```

### File Format (XLSX)

| Column A | Column B | Column C |
|----------|----------|----------|
| phone | name | custom_field |
| 628123456789 | John Doe | value1 |
| 628987654321 | Jane Smith | value2 |

**Notes:**
- First row is header (skipped)
- Only first column (phone/email) is required
- Duplicates are automatically removed

## Response Examples

### Success (First Upload - 201 Created)

```json
{
  "success": true,
  "data": {
    "success": true,
    "message": "Contacts uploaded successfully",
    "campaignId": "01GW5Z2TCVH8K9",
    "totalTargets": 150,
    "filename": "contacts.csv"
  }
}
```

### Duplicate Upload (201 Created)

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

### Campaign Not Found (400 Bad Request)

```json
{
  "success": false,
  "error": {
    "code": 400,
    "message": "Campaign not found"
  }
}
```

### Invalid File Type (400 Bad Request)

```json
{
  "success": false,
  "error": {
    "code": 400,
    "message": "Invalid file type. Only .csv and .xlsx files are accepted"
  }
}
```

## Testing

### 1. Create Campaign First

```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/campaigns" \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Campaign",
    "type": "whatsapp",
    "senderId": "YOUR_SENDER_ID",
    "templateId": "YOUR_TEMPLATE_ID",
    "targets": []
  }'
```

### 2. Upload Contacts to Campaign

```bash
curl -X POST \
  "http://localhost:8080/api/v1/instant-link/campaigns/{CAMPAIGN_ID}/upload-contacts" \
  -H "Authorization: Bearer TOKEN" \
  -F "file=@examples/sample_recipients.csv"
```

### 3. Verify Recipients Generated

```bash
curl -X GET \
  "http://localhost:8080/api/v1/instant-link/campaigns/{CAMPAIGN_ID}/recipients" \
  -H "Authorization: Bearer TOKEN"
```

### 4. Test Duplicate Prevention

Upload the same file again - should return cached response with 201 status.

## Technical Details

### Redis Key Pattern
```
campaign:target:{campaignId}:client:{clientId}
```

**TTL:** 24 hours (86400 seconds)

### NATS Topic Pattern
```
campaign.target.{clientId}
```

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

### Retry Mechanism

- **Max Retries:** 3 attempts
- **Backoff:** Exponential (500ms × attempt number)
  - Attempt 1 fails → wait 500ms
  - Attempt 2 fails → wait 1000ms
  - Attempt 3 fails → return error

### Batch Processing

- **Batch Size:** 100 records per batch
- **Strategy:** Insert each target individually within batch
- **Total Batches:** Calculated based on total targets

## Troubleshooting

### Error: "could not import github.com/xuri/excelize/v2"

**Solution:**
```bash
go get github.com/xuri/excelize/v2@v2.8.0
go mod tidy
```

### Error: "inconsistent vendoring"

**Solution:**
```bash
go mod vendor
```

### Error: "Failed to insert contacts after retries"

**Causes:**
- Database connection lost
- Invalid campaign_target table structure
- Constraint violations (unique index)

**Solution:**
- Check database connection
- Verify campaign_targets table exists
- Check for duplicate TargetId in file

## Sample Test File

A sample CSV file is provided at:
```
examples/sample_recipients.csv
```

Test with this file first!

## Files Modified

1. **controllers/campaigns.go**
   - Added `uploadContacts()` function
   - Added `parseContactsFile()` helper
   - Added `bulkInsertTargetsWithRetry()` helper

2. **go.mod**
   - Added `github.com/xuri/excelize/v2 v2.8.0`

3. **Documentation**
   - Created `docs/CAMPAIGN_FILE_UPLOAD_GUIDE.md`
   - Created `CAMPAIGN_FILE_UPLOAD_IMPLEMENTATION.md`
   - Created this setup guide

## Next Steps

1. Run the setup commands above
2. Test with sample file
3. Check logs for detailed execution flow
4. Monitor Redis keys: `redis-cli KEYS "campaign:target:*"`
5. Monitor NATS messages: Subscribe to `campaign.target.*`

## Support

Check logs at: `logs/il-dashboard-YYYY-MM-DD.log`

Search for:
- Component: `CampaignsController`
- Functions: `uploadContacts`, `parseContactsFile`, `bulkInsertTargetsWithRetry`
