# Image Upload & Serving Setup

## Directory Structure

```
il-dashboard/
├── dir/                    # Temporary files for SendFileBytes (gitignored)
│   └── README.md
├── images/                 # Uploaded images storage
│   └── [ULID].png/jpg
└── scripts/
    └── cleanup-temp-files.sh
```

## How Image Upload & Serving Works

### 1. Upload Image
**Endpoint:** `POST /api/v1/instant-link/wa-send/upload-image`

**Process:**
1. User uploads image via form-data (`image` field)
2. System validates file type (image/jpeg, image/png, image/jpg)
3. System validates file size (max 5MB)
4. System generates unique filename using ULID (e.g., `01KC83HHA91Z98RDAGRHTNXGC2.png`)
5. File saved to `./images/` folder
6. Returns URL: `https://domain.com/api/v1/instant-link/wa-send/image/01KC83HHA91Z98RDAGRHTNXGC2.png`

### 2. Serve Image
**Endpoint:** `GET /api/v1/instant-link/wa-send/image/:filename`

**Process:**
1. User requests image by filename
2. System reads file from `./images/:filename`
3. `SendFileBytes()` creates temporary file in `dir/` folder with pattern:
   ```
   sendFile[timestamp]_[filename]
   Example: sendFile2203171855_01KC83HHA91Z98RDAGRHTNXGC2.png
   ```
4. File is served to client with proper Content-Type header
5. Temporary file is automatically deleted after serving

## Folders

### `./images/`
- Stores actual uploaded images
- Files have ULID names: `01KC83HHA91Z98RDAGRHTNXGC2.png`
- Should be backed up
- Not in `.gitignore` (depends on deployment strategy)

### `./dir/`
- Stores temporary files created by `SendFileBytes()`
- Files have pattern: `sendFile*_*.png`
- Automatically cleaned up after serving
- **In `.gitignore`** - should not be committed
- Can be safely deleted anytime

## Maintenance

### Cleanup Temporary Files
If temporary files accumulate in `dir/` folder:

```bash
# Run cleanup script
./scripts/cleanup-temp-files.sh

# Or manually
rm -f ./dir/sendFile*
```

## Troubleshooting

### Error: "no such file or directory: dir/sendFile..."
**Solution:** Folder `dir/` was created. Make sure application has write permissions.

### Error: "Image not found"
**Solution:** Check if file exists in `./images/` folder with the exact filename.

### Images not loading
**Solution:** 
1. Check `base.url` configuration in `config.properties`
2. Verify file exists: `ls -la ./images/[filename]`
3. Check file permissions: `chmod 644 ./images/[filename]`

## Configuration

Set base URL in `config.properties`:
```properties
base.url=https://your-domain.com
```

This URL is used to generate public image URLs when uploading.

