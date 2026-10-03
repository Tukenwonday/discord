package storage

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/cordis/backend/internal/models"
	"github.com/google/uuid"
)

// Local stores attachments under UPLOAD_DIR using a uuid file name that keeps
// the original extension, and serves them from UPLOAD_PUBLIC_BASE.
type Local struct {
	dir        string
	publicBase string
	maxBytes   int64
}

// NewLocal builds the local driver and creates the upload directory.
func NewLocal(dir, publicBase string, maxBytes int64) (*Local, error) {
	if dir == "" {
		return nil, fmt.Errorf("upload dir must not be empty")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve upload dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	base := publicBase
	if base == "" {
		base = "/uploads"
	}
	return &Local{dir: abs, publicBase: base, maxBytes: maxBytes}, nil
}

// Name implements Driver.
func (l *Local) Name() string { return "local" }

// Dir returns the absolute upload directory, used to mount the file server.
func (l *Local) Dir() string { return l.dir }

// Save implements Driver.
func (l *Local) Save(ctx context.Context, f *multipart.FileHeader, contentType string) (models.Attachment, error) {
	if err := ValidateSize(f.Size, l.maxBytes); err != nil {
		return models.Attachment{}, err
	}
	if err := ValidateContentType(contentType); err != nil {
		return models.Attachment{}, err
	}
	original := SafeFilename(f.Filename)
	name := uuid.NewString() + Extension(original)
	path := filepath.Join(l.dir, name)

	src, err := f.Open()
	if err != nil {
		return models.Attachment{}, fmt.Errorf("open upload: %w", err)
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(path)
	if err != nil {
		return models.Attachment{}, fmt.Errorf("create upload file: %w", err)
	}
	written, copyErr := CopyTo(dst, src, l.maxBytes)
	closeErr := dst.Close()
	if copyErr != nil {
		_ = os.Remove(path)
		return models.Attachment{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return models.Attachment{}, fmt.Errorf("close upload file: %w", closeErr)
	}

	return models.Attachment{
		ID:          uuid.NewString(),
		URL:         l.publicBase + "/" + name,
		Filename:    original,
		Size:        written,
		ContentType: contentType,
	}, nil
}