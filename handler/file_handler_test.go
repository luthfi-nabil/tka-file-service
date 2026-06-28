package handler_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"tka-learning-portal/file-service/config"
	"tka-learning-portal/file-service/handler"
	"tka-learning-portal/file-service/middleware"
	"tka-learning-portal/file-service/models"
	"tka-learning-portal/file-service/repository"
	"tka-learning-portal/file-service/router"
)

// ---------------------------------------------------------------------------
// Package-level RSA key pair (generated once for all tests)
// ---------------------------------------------------------------------------

var (
	testPrivateKey *rsa.PrivateKey
	testPublicKey  *rsa.PublicKey
)

func init() {
	var err error
	testPrivateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("test RSA key generation failed: " + err.Error())
	}
	testPublicKey = &testPrivateKey.PublicKey
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func realConfig() *config.Config {
	cfg := config.Load()
	cfg.JWTPublicKey = testPublicKey
	return cfg
}

func connectDB(t *testing.T, cfg *config.Config) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		t.Skipf("cannot open DB (%v) – skipping integration tests", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("DB not reachable (%v) – skipping integration tests", err)
	}
	return db
}

const testAdminUUID = "00000000-0000-0000-0000-000000000001"

func makeAdminToken(t *testing.T) string {
	t.Helper()
	claims := middleware.BuildClaims(
		testAdminUUID, "test_admin", models.RoleAdministrator, "Administrator",
		"", "", "", time.Hour,
	)
	tok, err := middleware.GenerateToken(claims, testPrivateKey)
	if err != nil {
		t.Fatalf("make admin token: %v", err)
	}
	return tok
}

func buildEngine(db *sql.DB, uploadDir string) http.Handler {
	fileRepo := repository.NewFileRepository(db)
	fileHdl := handler.NewFileHandler(fileRepo, uploadDir, 10)
	return router.Setup(fileHdl, testPublicKey)
}

func doRequest(eng http.Handler, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewBuffer(b))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)
	return w
}

// buildUploadRequest builds a multipart/form-data request with a "file" field.
func buildUploadRequest(t *testing.T, filename string, content []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	fw.Write(content)
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

// ---------------------------------------------------------------------------
// Auth guard tests
// ---------------------------------------------------------------------------

func TestUpload_NoAuth(t *testing.T) {
	cfg := realConfig()
	db := connectDB(t, cfg)
	defer db.Close()

	dir := t.TempDir()
	eng := buildEngine(db, dir)

	req := buildUploadRequest(t, "test.txt", []byte("hello"))
	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetMeta_NoAuth(t *testing.T) {
	cfg := realConfig()
	db := connectDB(t, cfg)
	defer db.Close()

	dir := t.TempDir()
	eng := buildEngine(db, dir)

	w := doRequest(eng, http.MethodGet, "/api/v1/files/00000000-0000-0000-0000-000000000000", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDownload_NoAuth(t *testing.T) {
	cfg := realConfig()
	db := connectDB(t, cfg)
	defer db.Close()

	dir := t.TempDir()
	eng := buildEngine(db, dir)

	w := doRequest(eng, http.MethodGet, "/api/v1/files/00000000-0000-0000-0000-000000000000/download", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDelete_NoAuth(t *testing.T) {
	cfg := realConfig()
	db := connectDB(t, cfg)
	defer db.Close()

	dir := t.TempDir()
	eng := buildEngine(db, dir)

	w := doRequest(eng, http.MethodDelete, "/api/v1/files/00000000-0000-0000-0000-000000000000", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

func TestHealth(t *testing.T) {
	cfg := realConfig()
	db := connectDB(t, cfg)
	defer db.Close()

	dir := t.TempDir()
	eng := buildEngine(db, dir)

	w := doRequest(eng, http.MethodGet, "/health", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// 404 cases (non-existent IDs)
// ---------------------------------------------------------------------------

func TestGetMeta_NotFound(t *testing.T) {
	cfg := realConfig()
	db := connectDB(t, cfg)
	defer db.Close()

	dir := t.TempDir()
	eng := buildEngine(db, dir)
	tok := makeAdminToken(t)

	w := doRequest(eng, http.MethodGet, "/api/v1/files/00000000-0000-0000-0000-000000000000", tok, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDelete_NotFound(t *testing.T) {
	cfg := realConfig()
	db := connectDB(t, cfg)
	defer db.Close()

	dir := t.TempDir()
	eng := buildEngine(db, dir)
	tok := makeAdminToken(t)

	w := doRequest(eng, http.MethodDelete, "/api/v1/files/00000000-0000-0000-0000-000000000000", tok, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Upload missing file field
// ---------------------------------------------------------------------------

func TestUpload_MissingFileField(t *testing.T) {
	cfg := realConfig()
	db := connectDB(t, cfg)
	defer db.Close()

	dir := t.TempDir()
	eng := buildEngine(db, dir)
	tok := makeAdminToken(t)

	// Send JSON body instead of multipart — no "file" field.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)

	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Full CRUD integration
// ---------------------------------------------------------------------------

func TestFileCRUD_Full(t *testing.T) {
	cfg := realConfig()
	db := connectDB(t, cfg)
	defer db.Close()

	dir := t.TempDir()
	eng := buildEngine(db, dir)
	tok := makeAdminToken(t)

	// Use a minimal valid JPEG (FF D8 FF E0 header) so MIME validation passes.
	fileContent := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}

	// ── Upload ──
	req := buildUploadRequestWithContentType(t, "test-upload.jpg", "image/jpeg", fileContent)
	req.Header.Set("Authorization", "Bearer "+tok)

	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("upload: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var uploadResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &uploadResp)
	fileID := uploadResp["file_id"]
	if fileID == "" {
		t.Fatal("upload: expected file_id in response")
	}

	// Cleanup: soft-delete from DB after test
	defer db.Exec(`UPDATE uploaded_files SET soft_delete = 1 WHERE file_id = $1::uuid`, fileID)

	// ── GetMeta ──
	w = doRequest(eng, http.MethodGet, "/api/v1/files/"+fileID, tok, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get meta: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var meta map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &meta)
	if meta["file_id"] != fileID {
		t.Errorf("get meta: file_id mismatch, got %v", meta["file_id"])
	}
	if meta["original_name"] != "test-upload.jpg" {
		t.Errorf("get meta: original_name mismatch, got %v", meta["original_name"])
	}

	// ── Download ──
	w = doRequest(eng, http.MethodGet, "/api/v1/files/"+fileID+"/download", tok, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("download: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Equal(w.Body.Bytes(), fileContent) {
		t.Errorf("download: content mismatch")
	}

	// Verify the file is on disk.
	storedFiles, _ := filepath.Glob(filepath.Join(dir, fileID+"*"))
	if len(storedFiles) == 0 {
		t.Error("upload: expected a file on disk named after file_id")
	}

	// ── Delete ──
	w = doRequest(eng, http.MethodDelete, "/api/v1/files/"+fileID, tok, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Double-delete → 404
	w = doRequest(eng, http.MethodDelete, "/api/v1/files/"+fileID, tok, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("double-delete: expected 404, got %d", w.Code)
	}

	// GetMeta after delete → 404
	w = doRequest(eng, http.MethodGet, "/api/v1/files/"+fileID, tok, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("get after delete: expected 404, got %d", w.Code)
	}

	// Cleanup temp dir
	for _, f := range storedFiles {
		os.Remove(f)
	}
}

// ---------------------------------------------------------------------------
// MIME type validation
// ---------------------------------------------------------------------------

// pdfMagicBytes is the PDF file signature (%PDF-).
var pdfMagicBytes = []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34}

func buildUploadRequestWithContentType(t *testing.T, filename, contentType string, content []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename)}
	h["Content-Type"] = []string{contentType}

	fw, err := mw.CreatePart(h)
	if err != nil {
		t.Fatalf("create form part: %v", err)
	}
	fw.Write(content)
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestUpload_PDFAccepted(t *testing.T) {
	cfg := realConfig()
	db := connectDB(t, cfg)
	defer db.Close()

	dir := t.TempDir()
	eng := buildEngine(db, dir)
	tok := makeAdminToken(t)

	req := buildUploadRequestWithContentType(t, "document.pdf", "application/pdf", pdfMagicBytes)
	req.Header.Set("Authorization", "Bearer "+tok)

	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("pdf upload: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	fileID := resp["file_id"]
	if fileID == "" {
		t.Fatal("pdf upload: expected file_id in response")
	}

	// Cleanup
	db.Exec(`UPDATE uploaded_files SET soft_delete = 1 WHERE file_id = $1::uuid`, fileID)
	storedFiles, _ := filepath.Glob(filepath.Join(dir, fileID+"*"))
	for _, f := range storedFiles {
		os.Remove(f)
	}
}

func TestUpload_UnsupportedTypeRejected(t *testing.T) {
	cfg := realConfig()
	db := connectDB(t, cfg)
	defer db.Close()

	dir := t.TempDir()
	eng := buildEngine(db, dir)
	tok := makeAdminToken(t)

	req := buildUploadRequestWithContentType(t, "script.exe", "application/octet-stream", []byte{0x4D, 0x5A})
	req.Header.Set("Authorization", "Bearer "+tok)

	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("unsupported type: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
