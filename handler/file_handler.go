package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"tka-learning-portal/file-service/middleware"
	"tka-learning-portal/file-service/repository"
)

type FileHandler struct {
	fileRepo      *repository.FileRepository
	uploadDir     string
	maxFileSizeMB int64
}

func NewFileHandler(fileRepo *repository.FileRepository, uploadDir string, maxFileSizeMB int64) *FileHandler {
	return &FileHandler{
		fileRepo:      fileRepo,
		uploadDir:     uploadDir,
		maxFileSizeMB: maxFileSizeMB,
	}
}

// Upload handles POST /api/v1/files/upload
// Accepts a multipart/form-data request with a "file" field.
// Stores the file on disk and records metadata in the DB.
//
// Request: multipart/form-data
//   - file: the file to upload (required)
//
// Response 201:
//
//	{ "file_id": "550e8400-e29b-41d4-a716-446655440000" }
//
// Response 400: missing file or file too large
// Response 401: unauthorized
// Response 500: storage error
func (h *FileHandler) Upload(c *gin.Context) {
	claims, _ := middleware.GetClaims(c)

	maxBytes := h.maxFileSizeMB << 20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		if strings.Contains(err.Error(), "too large") {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("file exceeds %dMB limit", h.maxFileSizeMB)})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field is required"})
		return
	}
	defer file.Close()

	if header.Size > maxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("file exceeds %dMB limit", h.maxFileSizeMB)})
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}

	ext := filepath.Ext(header.Filename)
	originalName := header.Filename

	// Insert DB record first to get the UUID, then name the file after it.
	fileID, err := h.fileRepo.Create(originalName, "", mimeType, int64(len(data)), claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register file"})
		return
	}

	storedName := fileID + ext
	destPath := filepath.Join(h.uploadDir, storedName)

	if err := os.MkdirAll(h.uploadDir, 0755); err != nil {
		// Rollback DB record
		_ = h.fileRepo.Delete(fileID, claims.UserID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare upload directory"})
		return
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		// Rollback DB record
		_ = h.fileRepo.Delete(fileID, claims.UserID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	// Update stored_name in DB now that we have the file saved.
	if err := h.fileRepo.UpdateStoredName(fileID, storedName); err != nil {
		// Non-fatal: file is on disk, metadata is partially correct.
		// Log and continue — the file_id is still usable.
		_ = err
	}

	c.JSON(http.StatusCreated, gin.H{"file_id": fileID})
}

// GetMeta handles GET /api/v1/files/:id
// Returns metadata for the given file.
//
// Response 200:
//
//	{
//	  "file_id": "...",
//	  "original_name": "photo.jpg",
//	  "mime_type": "image/jpeg",
//	  "file_size": 204800,
//	  "uploaded_by": "...",
//	  "create_date": "2026-06-24T..."
//	}
//
// Response 404: file not found
// Response 401: unauthorized
func (h *FileHandler) GetMeta(c *gin.Context) {
	meta, err := h.fileRepo.Get(c.Param("id"))
	if err == repository.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch file metadata"})
		return
	}
	c.JSON(http.StatusOK, meta)
}

// Download handles GET /api/v1/files/:id/download
// Serves the raw file with its original MIME type and filename.
//
// Response 200: file bytes with Content-Type and Content-Disposition headers
// Response 404: file not found
// Response 401: unauthorized
func (h *FileHandler) Download(c *gin.Context) {
	meta, err := h.fileRepo.Get(c.Param("id"))
	if err == repository.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch file metadata"})
		return
	}

	storedName, _, err := h.fileRepo.GetStoredName(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to locate file"})
		return
	}

	filePath := filepath.Join(h.uploadDir, storedName)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, meta.OriginalName))
	c.File(filePath)
}

// Delete handles DELETE /api/v1/files/:id
// Soft-deletes the file record. The physical file is retained on disk.
//
// Response 200:
//
//	{ "message": "file deleted" }
//
// Response 404: file not found
// Response 401: unauthorized
func (h *FileHandler) Delete(c *gin.Context) {
	claims, _ := middleware.GetClaims(c)

	err := h.fileRepo.Delete(c.Param("id"), claims.UserID)
	if err == repository.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete file"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "file deleted"})
}
