# WhatsApp Image Upload API Documentation

Base URL: `/api/v1/instant-link/wa-send`

**Authorization Required:** Upload endpoint requires authentication (ADMIN, OWNER). Get image endpoint is public.

---

## Table of Contents

1. [Image Management](#image-management)
   - [Upload Image](#1-upload-image)
   - [Get Image](#2-get-image)

---

# Image Management

## 1. Upload Image

**Endpoint:** `POST /api/v1/instant-link/wa-send/upload-image`

**Authorization:** ADMIN, OWNER

**Description:** Upload an image file to be used in WhatsApp messages. The image will be saved to the server and a public URL will be returned. This URL can then be used when sending WhatsApp messages with images.

**Headers:**
```
Authorization: Bearer <your-token>
Content-Type: multipart/form-data
```

**Form Data:**
- `image` (required): Image file to upload

**Validation Rules:**
- File type: Only JPEG and PNG images are allowed
- File size: Maximum 5MB
- Field name must be `image`

**cURL Example:**

```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/upload-image" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -F "image=@/path/to/your/image.jpg"
```

**Success Response (200 OK):**

```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "imageUrl": "http://localhost:8080/api/v1/instant-link/wa-send/image/01JEPQK3M7EXAMPLE123.jpg"
  }
}
```

**Response Fields:**
- `imageUrl`: Public URL of the uploaded image. This URL can be used in WhatsApp message templates or direct messages

**Error Responses:**

400 Bad Request - Missing file:
```json
{
  "code": 400,
  "message": "Invalid request"
}
```

400 Bad Request - Invalid file type:
```json
{
  "code": 400,
  "message": "Only JPEG and PNG images are allowed"
}
```

400 Bad Request - File too large:
```json
{
  "code": 400,
  "message": "Image size must not exceed 5MB"
}
```

401 Unauthorized:
```json
{
  "code": 401,
  "message": "Unauthorized"
}
```

500 Internal Server Error:
```json
{
  "code": 500,
  "message": "Internal server error"
}
```

**Usage Example:**

After uploading, you can use the returned `imageUrl` when sending WhatsApp messages:

```json
{
  "to": "6281234567890",
  "message": "Check out this image!",
  "imageUrl": "http://localhost:8080/api/v1/instant-link/wa-send/image/01JEPQK3M7EXAMPLE123.jpg"
}
```

---

## 2. Get Image

**Endpoint:** `GET /api/v1/instant-link/wa-send/image/:filename`

**Authorization:** None (Public endpoint)

**Description:** Retrieve an uploaded image by its filename. This endpoint is publicly accessible to allow WhatsApp and other services to display the images.

**Path Parameters:**
- `filename` (required): The filename of the image (e.g., `01JEPQK3M7EXAMPLE123.jpg`)

**Security Features:**
- Filename validation to prevent directory traversal attacks
- Only files from the `images` directory can be accessed
- Filenames with `..` or `/` characters are rejected

**cURL Example:**

```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/wa-send/image/01JEPQK3M7EXAMPLE123.jpg"
```

**Success Response (200 OK):**

Returns the image file directly with appropriate headers:

**Headers:**
```
Content-Type: image/jpeg  # or image/png based on file extension
Cache-Control: public, max-age=31536000
```

**Response Body:** Binary image data

**Error Responses:**

400 Bad Request - Invalid filename:
```json
{
  "code": 400,
  "message": "Invalid filename"
}
```

404 Not Found - Image not found:
```json
{
  "code": 404,
  "message": "Image not found"
}
```

**Browser Access:**

You can also access the image directly in a browser:

```
http://localhost:8080/api/v1/instant-link/wa-send/image/01JEPQK3M7EXAMPLE123.jpg
```

---

## Integration Notes

### Workflow for Sending WhatsApp Messages with Images

1. **Upload the image first:**
   ```bash
   POST /api/v1/instant-link/wa-send/upload-image
   ```
   
2. **Get the imageUrl from the response:**
   ```json
   {
     "imageUrl": "http://localhost:8080/api/v1/instant-link/wa-send/image/01JEPQK3M7EXAMPLE123.jpg"
   }
   ```

3. **Use the imageUrl when sending messages:**
   ```bash
   POST /api/v1/instant-link/wa-send/message
   {
     "to": "6281234567890",
     "message": "Your message here",
     "imageUrl": "http://localhost:8080/api/v1/instant-link/wa-send/image/01JEPQK3M7EXAMPLE123.jpg"
   }
   ```

### Image Storage

- Images are stored in the `./images/` directory on the server
- Each image is assigned a unique filename using ULID (Universally Unique Lexicographically Sortable Identifier)
- Original file extension is preserved (`.jpg`, `.jpeg`, `.png`)
- Images are publicly accessible via the GET endpoint

### Performance & Caching

- Images are served with a 1-year cache header (`max-age=31536000`)
- This reduces server load and improves image loading performance
- Browsers and CDNs will cache the images automatically

### File Naming Convention

Format: `{ULID}{extension}`

Example: `01JEPQK3M7EXAMPLE123.jpg`

Where:
- `01JEPQK3M7EXAMPLE123` is the ULID (time-sortable, unique identifier)
- `.jpg` is the original file extension

### Security Considerations

1. **File Type Validation:** Only JPEG and PNG images are accepted
2. **File Size Limit:** Maximum 5MB per image
3. **Path Traversal Prevention:** Filenames are validated to prevent directory traversal
4. **Authentication:** Upload requires authentication (ADMIN or OWNER role)
5. **Public Access:** Images are publicly accessible once uploaded (no authentication required for viewing)

### Production Deployment Notes

**Important:** Update the `BASE_URL` configuration in your production environment:

```properties
# config-prod.properties
BASE_URL=https://yourdomain.com
```

This ensures that the returned `imageUrl` uses the correct production domain instead of `localhost`.

---

## Example: Complete Upload and Send Flow

### Step 1: Upload Image

**Request:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/upload-image" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -F "image=@/path/to/promo-banner.jpg"
```

**Response:**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "imageUrl": "http://localhost:8080/api/v1/instant-link/wa-send/image/01JEPQK3M7ABC123XYZ.jpg"
  }
}
```

### Step 2: Send Message with Image

**Request:**
```bash
curl -X POST "http://localhost:8080/api/v1/instant-link/wa-send/message" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "senderId": "sender-123",
    "to": "6281234567890",
    "message": "Lihat promo spesial kami!",
    "imageUrl": "http://localhost:8080/api/v1/instant-link/wa-send/image/01JEPQK3M7ABC123XYZ.jpg"
  }'
```

### Step 3: Verify Image is Accessible

Open in browser or use curl:
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/wa-send/image/01JEPQK3M7ABC123XYZ.jpg" \
  --output downloaded-image.jpg
```

The image should download successfully and be viewable.

