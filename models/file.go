package models

const (
	RoleAdministrator = int16(-1)
	RoleDefault       = int16(0)
	RoleGuru          = int16(1)
	RoleSiswa         = int16(2)
)

// UploadedFile is the DB record for a stored file.
type UploadedFile struct {
	FileID       string
	OriginalName string
	StoredName   string
	MimeType     string
	FileSize     int64
	UploadedBy   string
	CreateDate   string
	SoftDelete   int16
}

// FileMetaResponse is what the API returns when describing a file.
type FileMetaResponse struct {
	FileID       string `json:"file_id"`
	OriginalName string `json:"original_name"`
	MimeType     string `json:"mime_type"`
	FileSize     int64  `json:"file_size"`
	UploadedBy   string `json:"uploaded_by"`
	CreateDate   string `json:"create_date"`
}
