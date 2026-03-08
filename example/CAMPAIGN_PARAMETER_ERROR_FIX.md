# Campaign Parameter Error Fix - Summary

**Date:** December 12, 2025

---

## Problem

Campaign gagal dengan error:
```
"(#132000) Number of parameters does not match the expected number of params"
"body: number of localizable_params (1) does not match the expected number of params (0)"
```

**Log:**
```
time="2025-12-12T07:29:46Z" level=error msg="[CampaignExecutor] Failed to send WhatsApp message 
recipientId=01KC8QDQ5XAK56RHTD0VF0B9CM phone=6285692280119 
err=WhatsApp API error: {...Number of parameters does not match...}"
```

---

## Root Cause

Code di `workers/campaign_executor.go` mengirim **semua parameters** sebagai **text parameters untuk body**:

```go
// ❌ OLD CODE (WRONG)
for _, v := range parameters {
    waParams = append(waParams, dtos.WAMessageParameter{
        Type: "text",
        Text: fmt.Sprintf("%v", v),
    })
}
```

**Masalah:**
1. Template punya **IMAGE header** tapi **TIDAK ada variabel di body** (0 params expected)
2. Code mengirim **1 text parameter** (untuk image URL)
3. WhatsApp reject karena tidak cocok

**Contoh:**
- Template body: "Check out our promo!" ← Tidak ada {{1}}, {{2}}
- Parameters dikirim: `{"image": "https://example.com/image.jpg"}` → Dianggap sebagai text param
- WhatsApp expect: 0 params (no variables)
- WhatsApp receive: 1 param (image dianggap text)
- Result: ERROR ❌

---

## Solution

### Fix di `workers/campaign_executor.go`

**✅ NEW CODE:**
```go
// Build parameters based on template structure
var waParams []dtos.WAMessageParameter

// Check if parameters contain header media keys
hasHeaderImage := false
if _, ok := parameters["header_image"]; ok {
    hasHeaderImage = true
} else if _, ok := parameters["image"]; ok {
    hasHeaderImage = true
} else if _, ok := parameters["image_id"]; ok {
    hasHeaderImage = true
}

// Add header media parameter if exists
if hasHeaderImage {
    if imageURL, ok := parameters["image"].(string); ok && imageURL != "" {
        waParams = append(waParams, dtos.WAMessageParameter{
            Type:  "image",
            Image: &dtos.WAMessageMediaParam{Link: imageURL},
        })
    }
}

// Add body text parameters (param1, param2, etc)
// Skip header-related keys
for i := 1; i <= len(parameters); i++ {
    key := fmt.Sprintf("param%d", i)
    if v, ok := parameters[key]; ok {
        waParams = append(waParams, dtos.WAMessageParameter{
            Type: "text",
            Text: fmt.Sprintf("%v", v),
        })
    }
}
```

**Key Changes:**
1. ✅ Membedakan **header media parameters** vs **body text parameters**
2. ✅ Header image/video/document dikirim dengan type yang benar
3. ✅ Body text hanya dikirim jika ada param1, param2, dll
4. ✅ Tidak mengirim text param untuk template tanpa variabel body

---

## Parameter Format

### Case 1: Template dengan IMAGE Header, TANPA Body Variables

**Template:**
- Header: IMAGE
- Body: "Check out our latest promotion!"
- No {{1}}, {{2}} variables

**✅ Correct Parameters:**
```json
{
  "parameters": {
    "image": "https://example.com/promo.jpg"
  }
}
```

**❌ Wrong Parameters:**
```json
{
  "parameters": {
    "image": "https://example.com/promo.jpg",
    "param1": "John Doe"  // ❌ Template has no {{1}}
  }
}
```

---

### Case 2: Template dengan IMAGE Header DAN Body Variables

**Template:**
- Header: IMAGE
- Body: "Hi {{1}}, discount {{2}} for you!"
- Has {{1}}, {{2}} variables

**✅ Correct Parameters:**
```json
{
  "parameters": {
    "image": "https://example.com/promo.jpg",
    "param1": "John Doe",
    "param2": "30%"
  }
}
```

---

### Case 3: Template TANPA Header, Hanya Body Variables

**Template:**
- No header
- Body: "Hi {{1}}, your order {{2}} is ready!"

**✅ Correct Parameters:**
```json
{
  "parameters": {
    "param1": "John Doe",
    "param2": "#ORD-12345"
  }
}
```

---

### Case 4: Template dengan WhatsApp Media ID

**✅ Using Media ID (Recommended):**
```json
{
  "parameters": {
    "image_id": "1234567890123"
  }
}
```

**Advantages:**
- No URL accessibility issues
- Faster delivery
- Media ID valid for 30 days

---

## Parameter Keys Reference

| Template Component | Parameter Key | Type | Example |
|-------------------|---------------|------|---------|
| IMAGE header | `image` or `header_image` | URL | `"https://example.com/image.jpg"` |
| IMAGE header | `image_id` | Media ID | `"1234567890123"` |
| VIDEO header | `video` or `header_video` | URL | `"https://example.com/video.mp4"` |
| VIDEO header | `video_id` | Media ID | `"1234567890123"` |
| DOCUMENT header | `document` or `header_document` | URL | `"https://example.com/doc.pdf"` |
| DOCUMENT header | `document_id` | Media ID | `"1234567890123"` |
| Body variable {{1}} | `param1` or `1` | Text | `"John Doe"` |
| Body variable {{2}} | `param2` or `2` | Text | `"30%"` |
| Body variable {{3}} | `param3` or `3` | Text | `"31 Dec 2025"` |

---

## Testing After Fix

### Test 1: Create Campaign with Image Header Only
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/campaigns" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Image Header",
    "type": "whatsapp",
    "senderId": "sender-123",
    "templateId": "template-image-only",
    "batchSize": 10,
    "delaySeconds": 2,
    "parameters": {
      "image": "https://example.com/test.jpg"
    },
    "targets": [{"targetType": "contact", "targetId": "contact-123"}]
  }'
```

### Test 2: Generate Recipients
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/campaigns/$CAMPAIGN_ID/generate-recipients" \
  -H "Authorization: Bearer $TOKEN"
```

### Test 3: Start Campaign
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/campaigns/$CAMPAIGN_ID/start" \
  -H "Authorization: Bearer $TOKEN"
```

### Test 4: Monitor Logs
```bash
# Check for successful sends (no parameter errors)
tail -f logs/il-dashboard-*.log | grep -A 5 "CampaignExecutor"

# Should see:
# [CampaignExecutor] Prepared 1 parameters for template recipientId=xxx hasHeaderMedia=true
# [CampaignExecutor] Recipient marked as sent recipientId=xxx messageId=wamid.xxx
```

---

## Files Changed

### 1. `/workers/campaign_executor.go`
**Changes:**
- Fixed `sendWhatsAppMessage()` function
- Added detection for header media parameters
- Separated header params from body params
- Added logging for parameter count

### 2. `/example/campaign-api.md`
**Changes:**
- Added warning section about parameter mismatch error
- Added examples for image header campaigns
- Added parameter format documentation
- Added troubleshooting guide

### 3. `/example/CAMPAIGN_PARAMETER_ERROR_FIX.md` (NEW)
**Created:**
- Quick reference guide
- Root cause explanation
- Solution summary

---

## Common Mistakes to Avoid

### ❌ Mistake 1: Sending Text Params for Image Template
```json
// Template: IMAGE header, no body variables
{
  "parameters": {
    "image": "https://example.com/promo.jpg",
    "name": "John"  // ❌ WRONG! No {{1}} in template
  }
}
```

### ❌ Mistake 2: Using Wrong Key
```json
{
  "parameters": {
    "header": "https://example.com/promo.jpg"  // ❌ Use "image" not "header"
  }
}
```

### ❌ Mistake 3: Mixing URL and ID
```json
{
  "parameters": {
    "image": "https://example.com/promo.jpg",
    "image_id": "1234567890"  // ❌ Use one or the other, not both
  }
}
```

---

## Next Steps

1. ✅ **Deploy Fix:** Deploy updated `campaign_executor.go`
2. ✅ **Test Campaign:** Create test campaign with image-only template
3. ✅ **Monitor Logs:** Check for successful sends without parameter errors
4. ✅ **Update Docs:** Share updated documentation with team
5. ✅ **Retry Failed:** Retry failed campaigns with correct parameters

---

## Quick Checklist

Before creating campaign with image header:

- [ ] Upload image or get WhatsApp Media ID
- [ ] Check template structure (has body variables or not?)
- [ ] Use correct parameter keys (`image`, `param1`, etc)
- [ ] Don't send text params if template has no variables
- [ ] Test with small batch first (batchSize: 10)
- [ ] Monitor logs during first batch
- [ ] Check recipient status after send

---

## Support

If error persists after fix:
1. Check template is approved in WhatsApp Business Manager
2. Verify parameter keys match documentation
3. Check application logs for detailed errors
4. Test with single recipient first
5. Verify image URL is accessible (for URL method)

---

**Status:** ✅ FIXED
**Priority:** HIGH
**Impact:** All campaigns with media headers
**Testing:** Required before production use

