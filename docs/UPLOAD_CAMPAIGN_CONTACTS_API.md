# Upload Campaign Contacts API

## Overview

New dedicated endpoint for uploading target contacts to an existing campaign via CSV or XLSX files.

## Endpoint

```
POST /api/v1/instant-link/campaigns/:id/upload-contacts
```

**Authentication:** Required (Bearer Token)
**Authorization:** ADMIN, OWNER roles only

## Request

### Parameters

| Parameter | Type | Location | Required | Description |
|-----------|------|----------|----------|-------------|
| id | string | URL path | ✅ Yes | Campaign ID |
| file | File | Form data | ✅ Yes | CSV or XLSX file with contacts |

### Content-Type

```
multipart/form-data
```

### Supported File Formats

- **.csv** - Comma-separated values
- **.xlsx** - Excel spreadsheet

### File Structure

**CSV Format:**
```csv
phone,name,custom_field
628123456789,John Doe,value1
628987654321,Jane Smith,value2
```

**XLSX Format:**
| Column A | Column B | Column C |
|----------|----------|----------|
| phone | name | custom_field |
| 628123456789 | John Doe | value1 |
| 628987654321 | Jane Smith | value2 |

**Notes:**
- First row is treated as header
- Phone numbers should be in international format (e.g., 628xxx for Indonesia)
- Additional columns are supported for custom fields

## Response

### Success Response (200 OK)

```json
{
  "success": true,
  "data": {
    "success": true,
    "message": "File uploaded successfully",
    "campaignId": "01GW5Z2TCVH8K9QBZJ0X5MWQRS",
    "filename": "contacts.csv",
    "size": 1024,
    "note": "File parsing will be implemented in service layer"
  }
}
```

### Error Responses

#### File Not Provided (400 Bad Request)
```json
{
  "success": false,
  "error": {
    "code": 400,
    "message": "File is required. Upload CSV or XLSX file with 'file' key"
  }
}
```

#### Invalid File Type (400 Bad Request)
```json
{
  "success": false,
  "error": {
    "code": 400,
    "message": "Invalid file type. Only .csv and .xlsx files are accepted"
  }
}
```

#### Campaign Not Found (404 Not Found)
```json
{
  "success": false,
  "error": {
    "code": 404,
    "message": "Campaign not found"
  }
}
```

#### Unauthorized (403 Forbidden)
```json
{
  "success": false,
  "error": {
    "code": 403,
    "message": "Not allowed"
  }
}
```

## Usage Examples

### cURL

```bash
curl -X POST \
  "https://api.example.com/api/v1/instant-link/campaigns/01GW5Z2TCVH8K9/upload-contacts" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@contacts.csv"
```

### JavaScript (Fetch API)

```javascript
const formData = new FormData();
formData.append('file', fileInput.files[0]);

fetch(`/api/v1/instant-link/campaigns/${campaignId}/upload-contacts`, {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer YOUR_TOKEN'
  },
  body: formData
})
.then(response => response.json())
.then(data => console.log(data));
```

### Postman

1. **Method:** POST
2. **URL:** `{{base_url}}/api/v1/instant-link/campaigns/{{campaign_id}}/upload-contacts`
3. **Headers:**
   - `Authorization: Bearer {{token}}`
4. **Body:** form-data
   - Key: `file` (type: File)
   - Value: Select your CSV/XLSX file

### Python (requests)

```python
import requests

url = f"https://api.example.com/api/v1/instant-link/campaigns/{campaign_id}/upload-contacts"
headers = {"Authorization": f"Bearer {token}"}
files = {"file": open("contacts.csv", "rb")}

response = requests.post(url, headers=headers, files=files)
print(response.json())
```

## Workflow

### Typical Usage Flow

1. **Create Campaign** (without targets)
   ```
   POST /api/v1/instant-link/campaigns
   ```

2. **Upload Contacts File**
   ```
   POST /api/v1/instant-link/campaigns/:id/upload-contacts
   ```

3. **Start Campaign**
   ```
   POST /api/v1/instant-link/campaigns/:id/start
   ```

### Alternative: Using Existing Contacts

If you prefer to use contacts from your database instead of uploading:

1. Create campaign with targets (groups, contacts, or "all")
2. Generate recipients
   ```
   POST /api/v1/instant-link/campaigns/:id/generate-recipients
   ```
3. Start campaign

## File Format Best Practices

### ✅ DO

- Use UTF-8 encoding for CSV files
- Include header row for clarity
- Use international phone format (e.g., 628xxx)
- Test with small file first (5-10 rows)
- Remove duplicates before upload
- Validate phone numbers/emails before upload

### ❌ DON'T

- Mix phone numbers and emails in same file
- Use spaces in phone numbers
- Upload files larger than 50 MB
- Use old Excel format (.xls)
- Include sensitive data in custom fields

## Sample Files

### Sample CSV (`contacts.csv`)

```csv
phone,name,discount
628123456789,John Doe,20%
628987654321,Jane Smith,15%
628111222333,Bob Johnson,10%
```

### Sample XLSX

Download template: [coming soon]

## Implementation Status

### ✅ Completed

- [x] Route registration
- [x] Controller endpoint
- [x] File upload validation
- [x] File type checking (.csv, .xlsx)
- [x] Basic response structure

### 🚧 To Be Implemented in Service Layer

- [ ] CSV file parsing
- [ ] XLSX file parsing
- [ ] Contact deduplication
- [ ] Recipient object creation
- [ ] Database insertion
- [ ] Balance checking
- [ ] Cost calculation
- [ ] Campaign recipient count update
- [ ] Error handling for invalid data
- [ ] Async processing for large files

## Next Steps

To complete the implementation, you need to:

1. Create service method: `UploadCampaignContacts(campaignId, file, authInfo)`
2. Parse CSV/XLSX file content
3. Validate and transform rows into campaign recipients
4. Insert recipients into database
5. Update campaign total recipient count
6. Return detailed results with success/failure counts

## Related Endpoints

- **Create Campaign:** `POST /api/v1/instant-link/campaigns`
- **Generate Recipients:** `POST /api/v1/instant-link/campaigns/:id/generate-recipients`
- **Get Recipients:** `GET /api/v1/instant-link/campaigns/:id/recipients`
- **Start Campaign:** `POST /api/v1/instant-link/campaigns/:id/start`

## Support

For implementation questions:
- See: `controllers/campaigns.go` line 143-165
- Contact backend team
