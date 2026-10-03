// Package storage defines the attachment Driver interface plus the local
// filesystem and S3 implementations selected by UPLOAD_DRIVER.
package storage

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/cordis/backend/internal/httpx"
	"github.com/cordis/backend/internal/models"
)

// Driver stores an uploaded file and returns the attachment row describing it.
type Driver interface {
	// Name identifies the driver, matching UPLOAD_DRIVER.
	Name() string
	// Save persists the file and returns the attachment without its id so the
	// caller can decide whether it belongs to a message yet.
	Save(ctx context.Context, f *multipart.FileHeader, contentType string) (models.Attachment, error)
}

// AllowedContentTypes is the exact allow list from section 5 of the contract.
var AllowedContentTypes = map[string]bool{
	"image/png":      true,
	"image/jpeg":     true,
	"image/gif":      true,
	"image/webp":     true,
	"application/pdf": true,
	"text/plain":     true,
	"application/zip": true,
	"video/mp4":      true,
	"video/webm":     true,
	"application/x-msdownload": true,
	"application/x-msi": true,
	"application/vnd.microsoft.portable-executable": true,
	"application/x-7z-compressed":                   true,
	"application/vnd.rar":                          true,
	"application/x-msdoc":                           true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":        true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
}

// Errors returned by the drivers.
var (
	ErrUnsupportedType = errors.New("unsupported content type")
	ErrTooLarge        = errors.New("file is too large")
	ErrEmptyFile       = errors.New("file is empty")
)

// ValidateSize rejects files larger than the configured maximum.
func ValidateSize(size, maxBytes int64) error {
	if size <= 0 {
		return ErrEmptyFile
	}
	if size > maxBytes {
		return ErrTooLarge
	}
	return nil
}

// ValidateContentType rejects content types outside the allow list.
func ValidateContentType(contentType string) error {
	base := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if !AllowedContentTypes[base] {
		return ErrUnsupportedType
	}
	return nil
}

// SafeFilename keeps the base name of the client file, dropping any path
// component and control characters the client may have supplied.
func SafeFilename(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == "/" {
		return "file"
	}
	return name
}

// Extension returns the lowercase extension including the leading dot.
func Extension(name string) string { return strings.ToLower(filepath.Ext(name)) }

// MapError converts driver errors into the contract error envelope values.
func MapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrUnsupportedType):
		return httpx.NewUnsupportedMediaType("file type is not allowed")
	case errors.Is(err, ErrTooLarge):
		return httpx.NewPayloadTooLarge("file exceeds the maximum upload size")
	case errors.Is(err, ErrEmptyFile):
		return httpx.NewField("file", "file is empty")
	default:
		return httpx.NewInternalFromError(err)
	}
}

// CopyTo streams a multipart file into w, enforcing the size ceiling so an
// oversized upload is rejected while streaming instead of after the fact.
func CopyTo(dst io.Writer, src io.Reader, maxBytes int64) (int64, error) {
	limited := io.LimitReader(src, maxBytes+1)
	n, err := io.Copy(dst, limited)
	if err != nil {
		return n, err
	}
	if n > maxBytes {
		return n, ErrTooLarge
	}
	return n, nil
}