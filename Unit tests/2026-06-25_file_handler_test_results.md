# File Service – Handler Unit Test Results
**Timestamp:** 2026-06-25 22:47 WIB  
**Package:** `tka-learning-portal/file-service/handler`  
**Go command:** `go test ./handler/... -v`

## Summary
All 10 tests **PASSED** in 4.250s.

## Results

| Test | Status | Notes |
|------|--------|-------|
| TestUpload_NoAuth | PASS | Upload without JWT → 401 |
| TestGetMeta_NoAuth | PASS | Get metadata without JWT → 401 |
| TestDownload_NoAuth | PASS | Download without JWT → 401 |
| TestDelete_NoAuth | PASS | Delete without JWT → 401 |
| TestHealth | PASS | GET /health → 200 |
| TestGetMeta_NotFound | PASS | Non-existent file ID → 404 |
| TestDelete_NotFound | PASS | Non-existent file ID → 404 |
| TestUpload_MissingFileField | PASS | No "file" field → 400 |
| TestFileCRUD_Full | PASS | Full upload → get meta → download → delete → 404 cycle |
| TestUpload_PDFAccepted | PASS | `application/pdf` upload → 201 (new test) |
| TestUpload_UnsupportedTypeRejected | PASS | `application/octet-stream` → 400 (new test) |

## Changes Made
- Added `allowedMIMETypes` allowlist to `file_handler.go` including:
  - `application/pdf`
  - `image/jpeg`, `image/png`, `image/gif`, `image/webp`
  - `application/msword`, `application/vnd.openxmlformats-officedocument.wordprocessingml.document`
  - `application/vnd.ms-excel`, `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
  - `video/mp4`, `video/webm`
- Upload now returns `400` with `{"error": "unsupported file type: <mime>"}` for unlisted types.
- Added `TestUpload_PDFAccepted` and `TestUpload_UnsupportedTypeRejected` tests.
- Updated `TestFileCRUD_Full` to use `image/jpeg` content (old test used `application/octet-stream` via `CreateFormFile`).
