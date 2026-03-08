# WhatsApp Media Upload Error (131053) - Quick Guide

**Date:** December 12, 2025

---

## Problem

Anda mengirim WhatsApp message dengan image/video, message terkirim (status "sent"), tapi beberapa detik kemudian mendapat webhook callback dengan error:

```json
{
  "status": "failed",
  "errors": [{
    "code": 131053,
    "title": "Media upload error",
    "message": "Media upload error"
  }]
}
```

**Log:**
```
time="2025-12-12T04:05:49Z" level=error msg="[handleActionWaWebHook] Message error - Code: 131053, Title: Media upload error, Message: Media upload error"
```

---

## Root Cause

WhatsApp **tidak bisa download media** dari URL yang Anda kirim.

**Flow:**
1. Anda kirim request → API return 200 OK dengan message ID
2. WhatsApp **asynchronously download** media dari URL Anda
3. Download gagal → WhatsApp kirim webhook callback "failed"

**Why Download Fails?**
- ❌ URL not publicly accessible (authentication required)
- ❌ SSL certificate invalid/expired
- ❌ HTTP instead of HTTPS
- ❌ Wrong Content-Type header
- ❌ File size exceeded (Image: >5MB, Video: >16MB)
- ❌ Server timeout (>5 seconds)

---

## Quick Debug

```bash
# Test your image URL
curl -v https://your-domain.com/api/v1/instant-link/wa-send/image/xxx.jpg

# Check:
# ✅ HTTP/2 200 OK
# ✅ content-type: image/jpeg (not text/html)
# ✅ content-length: < 5242880 (< 5MB for images)
# ✅ Response time < 5 seconds
# ✅ No 401/403 (no authentication)
```

---

## Solutions (Pick One)

### ✅ Solution 1: Upload to WhatsApp (RECOMMENDED)

**Pros:** Most reliable, no URL issues, faster delivery
**Cons:** Media ID expires after 30 days

```bash
# Step 1: Upload to WhatsApp
curl -X POST "https://graph.facebook.com/v24.0/YOUR_PHONE_NUMBER_ID/media" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -F "file=@image.jpg" \
  -F "type=image/jpeg" \
  -F "messaging_product=whatsapp"

# Response: {"id":"1234567890"}
```

```bash
# Step 2: Send with Media ID
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/message" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "template-123",
    "to": "6281234567890",
    "parameters": [{
      "type": "image",
      "image": {"id": "1234567890"}
    }]
  }'
```

---

### ✅ Solution 2: Fix Your Image Endpoint

Pastikan endpoint return proper headers:

```go
// controllers/wa_send.go
func (o *WASendController) getImage(ctx gocom.Context) error {
    filename := ctx.Param("filename")
    data, err := services.GetWASendSvc().GetImages(filename, a.Get(ctx))
    if err != nil {
        return ctx.SendError(err)
    }

    ext := filepath.Ext(filename)
    contentType := "image/jpeg"
    if ext == ".png" {
        contentType = "image/png"
    }

    // IMPORTANT: Set proper headers
    ctx.SetHeader("Content-Type", contentType)
    ctx.SetHeader("Cache-Control", "public, max-age=31536000")
    ctx.SetHeader("Access-Control-Allow-Origin", "*")
    
    return ctx.SendFileBytes(data, filename)
}
```

**Checklist:**
- ✅ HTTPS with valid SSL certificate
- ✅ Correct Content-Type (image/jpeg, image/png, video/mp4)
- ✅ No authentication for public images
- ✅ Response time < 5 seconds
- ✅ File size within limits

---

### ✅ Solution 3: Use CDN

Upload to CloudFlare, AWS S3, or Google Cloud Storage:

```bash
# Upload to S3 with public access
aws s3 cp image.jpg s3://bucket/images/image.jpg --acl public-read

# Use CDN URL
{
  "parameters": [{
    "type": "image",
    "image": {
      "link": "https://bucket.s3.amazonaws.com/images/image.jpg"
    }
  }]
}
```

---

## Testing After Fix

```bash
# Test 1: Accessibility (must be 200)
curl -I https://your-domain.com/api/v1/instant-link/wa-send/image/test.jpg

# Test 2: Content Type (must be image/jpeg or image/png)
curl -I https://your-domain.com/api/v1/instant-link/wa-send/image/test.jpg | grep content-type

# Test 3: File Size (must be < 5MB for images)
curl -I https://your-domain.com/api/v1/instant-link/wa-send/image/test.jpg | grep content-length

# Test 4: Response Time (must be < 5 seconds)
time curl -I https://your-domain.com/api/v1/instant-link/wa-send/image/test.jpg

# Test 5: Full Download (must work without auth)
curl -o downloaded.jpg https://your-domain.com/api/v1/instant-link/wa-send/image/test.jpg
```

---

## Media Limits (WhatsApp)

| Type | Max Size | Formats |
|------|----------|---------|
| Image | 5 MB | JPG, PNG, WebP |
| Video | 16 MB | MP4, 3GPP (H.264 codec) |
| Document | 100 MB | PDF, DOC, DOCX, XLS, XLSX, PPT, PPTX |
| Audio | 16 MB | AAC, MP4, AMR, OGG |

---

## Common Mistakes

### ❌ Wrong: Return JSON instead of binary
```go
return ctx.SendResult(map[string]string{"url": imageURL})  // WRONG!
```

### ✅ Correct: Return binary data
```go
ctx.SetHeader("Content-Type", "image/jpeg")
return ctx.SendFileBytes(data, filename)  // CORRECT!
```

---

### ❌ Wrong: HTTP URL
```json
{
  "image": {
    "link": "http://domain.com/image.jpg"  // WRONG! Must be HTTPS
  }
}
```

### ✅ Correct: HTTPS URL
```json
{
  "image": {
    "link": "https://domain.com/image.jpg"  // CORRECT!
  }
}
```

---

### ❌ Wrong: Authentication Required
```bash
# Returns 401 Unauthorized
curl -I https://domain.com/api/image/protected.jpg
# WRONG! WhatsApp can't authenticate
```

### ✅ Correct: Public Access
```bash
# Returns 200 OK without auth
curl -I https://domain.com/api/v1/instant-link/wa-send/image/public.jpg
# CORRECT!
```

---

## Monitoring

Add logging to track issues:

```go
// Before sending
logger.Infof("[WASendService] Media URL: %s", imageURL)

// In webhook handler
if status == "failed" && errorCode == 131053 {
    logger.Errorf("[Webhook] Media upload failed for URL: %s", mediaURL)
}
```

**Pattern Analysis:**
- All messages fail → Problem with image endpoint
- Random failures → Performance/timeout issue
- Specific image fails → Problem with specific file

---

## Quick Reference

| Issue | Check | Fix |
|-------|-------|-----|
| URL not accessible | `curl -I <url>` returns 401/403 | Remove auth or use Media ID |
| Wrong content-type | Header shows text/html | Set proper Content-Type |
| File too large | Content-Length > 5MB | Compress image or use video limits |
| Slow response | Takes > 5 seconds | Optimize server or use CDN |
| SSL error | Certificate invalid | Fix SSL certificate |
| HTTP instead HTTPS | URL starts with http:// | Use HTTPS |

---

## More Info

- Full API Documentation: `wa-send-message-api.md`
- Template Documentation: `wa-templates-api.md`
- WhatsApp Business API: https://developers.facebook.com/docs/whatsapp

---

**TL;DR:** Upload media ke WhatsApp terlebih dahulu dan gunakan Media ID, atau pastikan image URL Anda publicly accessible via HTTPS dengan response time < 5 detik.

