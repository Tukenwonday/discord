package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/cordis/backend/internal/models"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// presignTTL bounds how long a presigned download link stays valid, which
// keeps private buckets usable without exposing credentials.
const presignTTL = 7 * 24 * time.Hour

// S3 stores attachments in a MinIO or S3 compatible bucket using minio-go.
type S3 struct {
	client        *minio.Client
	bucket        string
	publicBase    string
	maxBytes      int64
	forcePathStyle bool
}

// S3Options configures the S3 driver.
type S3Options struct {
	Endpoint       string
	Region         string
	Bucket         string
	AccessKey      string
	SecretKey      string
	ForcePathStyle bool
	PublicBase     string
	MaxBytes       int64
}

// NewS3 builds the S3 driver and verifies that the bucket exists.
func NewS3(ctx context.Context, opts S3Options) (*S3, error) {
	if opts.Endpoint == "" || opts.Bucket == "" {
		return nil, fmt.Errorf("s3 endpoint and bucket are required")
	}
	secure := strings.HasPrefix(opts.Endpoint, "https://")
	endpoint := strings.TrimPrefix(strings.TrimPrefix(opts.Endpoint, "https://"), "http://")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(opts.AccessKey, opts.SecretKey, ""),
		Secure:       secure,
		Region:       opts.Region,
		BucketLookup: bucketLookup(opts.ForcePathStyle),
	})
	if err != nil {
		return nil, fmt.Errorf("new s3 client: %w", err)
	}
	if exists, err := client.BucketExists(ctx, opts.Bucket); err != nil {
		return nil, fmt.Errorf("check s3 bucket: %w", err)
	} else if !exists {
		if err := client.MakeBucket(ctx, opts.Bucket, minio.MakeBucketOptions{Region: opts.Region}); err != nil {
			return nil, fmt.Errorf("create s3 bucket: %w", err)
		}
	}
	return &S3{
		client:         client,
		bucket:         opts.Bucket,
		publicBase:     strings.TrimRight(opts.PublicBase, "/"),
		maxBytes:       opts.MaxBytes,
		forcePathStyle: opts.ForcePathStyle,
	}, nil
}

func bucketLookup(forcePathStyle bool) minio.BucketLookupType {
	if forcePathStyle {
		return minio.BucketLookupPath
	}
	return minio.BucketLookupAuto
}

// Name implements Driver.
func (s *S3) Name() string { return "s3" }

// Save implements Driver, returning a presigned URL so private buckets still
// expose the attachment to its recipients.
func (s *S3) Save(ctx context.Context, f *multipart.FileHeader, contentType string) (models.Attachment, error) {
	if err := ValidateSize(f.Size, s.maxBytes); err != nil {
		return models.Attachment{}, err
	}
	if err := ValidateContentType(contentType); err != nil {
		return models.Attachment{}, err
	}
	original := SafeFilename(f.Filename)
	objectName := path.Join(uuid.NewString()+Extension(original))

	src, err := f.Open()
	if err != nil {
		return models.Attachment{}, fmt.Errorf("open upload: %w", err)
	}
	defer func() { _ = src.Close() }()

	var buf bytes.Buffer
	written, copyErr := CopyTo(&buf, src, s.maxBytes)
	if copyErr != nil {
		return models.Attachment{}, copyErr
	}

	info, err := clientPutObject(ctx, s.client, s.bucket, objectName, &buf, written, contentType)
	if err != nil {
		return models.Attachment{}, err
	}

	publicURL, err := s.publicObjectURL(ctx, objectName)
	if err != nil {
		return models.Attachment{}, err
	}

	return models.Attachment{
		ID:          uuid.NewString(),
		URL:         publicURL,
		Filename:    original,
		Size:        info.Size,
		ContentType: contentType,
	}, nil
}

func clientPutObject(ctx context.Context, client *minio.Client, bucket, object string, body io.Reader, size int64, contentType string) (minio.UploadInfo, error) {
	info, err := client.PutObject(ctx, bucket, object, body, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("put object: %w", err)
	}
	return info, nil
}

// publicObjectURL prefers a configured public base URL and otherwise returns a
// presigned GET URL valid for a week.
func (s *S3) publicObjectURL(ctx context.Context, object string) (string, error) {
	if s.publicBase != "" {
		return s.publicBase + "/" + object, nil
	}
	presigned, err := s.client.PresignedGetObject(ctx, s.bucket, object, presignTTL, url.Values{})
	if err != nil {
		return "", fmt.Errorf("presign object url: %w", err)
	}
	return presigned.String(), nil
}