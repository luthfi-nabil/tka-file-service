package repository

import (
	"database/sql"
	"fmt"

	"tka-learning-portal/file-service/models"
)

type FileRepository struct{ db *sql.DB }

func NewFileRepository(db *sql.DB) *FileRepository { return &FileRepository{db: db} }

// Create inserts a file record and returns the generated file_id.
func (r *FileRepository) Create(originalName, storedName, mimeType string, fileSize int64, uploadedBy string) (string, error) {
	var id string
	err := r.db.QueryRow(
		`INSERT INTO uploaded_files (original_name, stored_name, mime_type, file_size, uploaded_by, create_date, soft_delete)
		 VALUES ($1, $2, $3, $4, $5::uuid, NOW(), 0)
		 RETURNING file_id`,
		originalName, storedName, mimeType, fileSize, uploadedBy,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create file record: %w", err)
	}
	return id, nil
}

// Get returns file metadata by file_id.
func (r *FileRepository) Get(id string) (*models.FileMetaResponse, error) {
	var f models.FileMetaResponse
	err := r.db.QueryRow(
		`SELECT file_id::text, original_name, mime_type, file_size, uploaded_by::text, create_date::text
		 FROM uploaded_files
		 WHERE file_id = $1::uuid AND soft_delete = 0`,
		id,
	).Scan(&f.FileID, &f.OriginalName, &f.MimeType, &f.FileSize, &f.UploadedBy, &f.CreateDate)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get file: %w", err)
	}
	return &f, nil
}

// GetStoredName returns the stored filename for serving/downloading.
func (r *FileRepository) GetStoredName(id string) (string, string, error) {
	var storedName, mimeType string
	err := r.db.QueryRow(
		`SELECT stored_name, mime_type FROM uploaded_files WHERE file_id = $1::uuid AND soft_delete = 0`,
		id,
	).Scan(&storedName, &mimeType)
	if err == sql.ErrNoRows {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("get stored name: %w", err)
	}
	return storedName, mimeType, nil
}

// UpdateStoredName sets the stored_name after the file is written to disk.
func (r *FileRepository) UpdateStoredName(id, storedName string) error {
	_, err := r.db.Exec(
		`UPDATE uploaded_files SET stored_name = $1 WHERE file_id = $2::uuid`,
		storedName, id,
	)
	return err
}

// Delete soft-deletes a file record.
func (r *FileRepository) Delete(id, deletedBy string) error {
	res, err := r.db.Exec(
		`UPDATE uploaded_files SET soft_delete = 1 WHERE file_id = $1::uuid AND soft_delete = 0`,
		id,
	)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
