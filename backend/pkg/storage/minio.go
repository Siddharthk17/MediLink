// Package storage provides MinIO object storage operations.
package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"
)

// StorageClient defines MinIO storage operations.
type StorageClient interface {
	UploadFile(ctx context.Context, patientFHIRID string, file io.Reader, fileName string, contentType string, size int64) (string, error)
	UploadWithKey(ctx context.Context, bucket, key string, data io.Reader, contentType string, size int64) error
	GetPresignedURL(ctx context.Context, bucket, key string) (string, error)
	DownloadFile(ctx context.Context, bucket, key string) ([]byte, error)
	DeleteFile(ctx context.Context, bucket, key string) error
	Health(ctx context.Context) bool
}

// MinIOClient implements StorageClient using MinIO.
type MinIOClient struct {
	client *minio.Client
	bucket string
	logger zerolog.Logger
}

// NewMinIOClient creates a MinIO client and ensures the bucket exists.
func NewMinIOClient(endpoint, accessKey, secretKey, bucket string, useSSL bool, logger zerolog.Logger) (*MinIOClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client init: %w", err)
	}

	mc := &MinIOClient{
		client: client,
		bucket: bucket,
		logger: logger,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("minio bucket check: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio create bucket: %w", err)
		}
		logger.Info().Str("bucket", bucket).Msg("MinIO bucket created")
	}

	return mc, nil
}

// UploadFile stores a file in MinIO. Key format: {patientFHIRID}/{YYYY-MM-DD}/{uuid}.{ext}
func (m *MinIOClient) UploadFile(ctx context.Context, patientFHIRID string, file io.Reader, fileName string, contentType string, size int64) (string, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		ext = extensionFromContentType(contentType)
	}
	key := fmt.Sprintf("%s/%s/%s%s", patientFHIRID, time.Now().Format("2006-01-02"), uuid.New().String(), ext)

	_, err := m.client.PutObject(ctx, m.bucket, key, file, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("minio upload: %w", err)
	}

	m.logger.Info().Str("key", key).Str("bucket", m.bucket).Msg("file uploaded to MinIO")
	return key, nil
}

// UploadWithKey stores data in MinIO with an explicit bucket and key.
func (m *MinIOClient) UploadWithKey(ctx context.Context, bucket, key string, data io.Reader, contentType string, size int64) error {
	_, err := m.client.PutObject(ctx, bucket, key, data, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("minio upload: %w", err)
	}
	m.logger.Info().Str("key", key).Str("bucket", bucket).Msg("file uploaded to MinIO")
	return nil
}

// GetPresignedURL generates a pre-signed download URL with 15-minute expiry.
func (m *MinIOClient) GetPresignedURL(ctx context.Context, bucket, key string) (string, error) {
	url, err := m.client.PresignedGetObject(ctx, bucket, key, 15*time.Minute, nil)
	if err != nil {
		return "", fmt.Errorf("minio presign: %w", err)
	}
	return url.String(), nil
}

// DownloadFile retrieves a file's bytes from MinIO.
func (m *MinIOClient) DownloadFile(ctx context.Context, bucket, key string) ([]byte, error) {
	obj, err := m.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("minio download: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("minio read: %w", err)
	}
	return data, nil
}

// DeleteFile removes a file from MinIO.
func (m *MinIOClient) DeleteFile(ctx context.Context, bucket, key string) error {
	return m.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

// Health returns true if MinIO is reachable.
func (m *MinIOClient) Health(ctx context.Context) bool {
	_, err := m.client.BucketExists(ctx, m.bucket)
	return err == nil
}

func extensionFromContentType(ct string) string {
	switch ct {
	case "application/pdf":
		return ".pdf"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}

// NoopStorageClient is a no-op implementation for testing/dev.
type NoopStorageClient struct{}

func (n *NoopStorageClient) UploadFile(_ context.Context, patientFHIRID string, _ io.Reader, fileName string, contentType string, _ int64) (string, error) {
	return fmt.Sprintf("%s/%s/%s", patientFHIRID, time.Now().Format("2006-01-02"), fileName), nil
}
func (n *NoopStorageClient) UploadWithKey(_ context.Context, _, key string, _ io.Reader, _ string, _ int64) error {
	return nil
}
func (n *NoopStorageClient) GetPresignedURL(_ context.Context, _, key string) (string, error) {
	return "http://localhost:9000/" + key, nil
}
func (n *NoopStorageClient) DownloadFile(_ context.Context, _, _ string) ([]byte, error) {
	return []byte("noop file content"), nil
}
func (n *NoopStorageClient) DeleteFile(_ context.Context, _, _ string) error { return nil }
func (n *NoopStorageClient) Health(_ context.Context) bool                   { return true }
