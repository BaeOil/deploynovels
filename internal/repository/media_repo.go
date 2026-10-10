package repository

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

type MediaRepository interface {
	EnsureBucketExists(ctx context.Context, bucketName string) error
	UploadFile(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, contentType string) (string, error)
	DownloadFile(ctx context.Context, bucketName, objectName string) (io.Reader, error)
	DeleteFile(ctx context.Context, bucketName, objectName string) error
	GetPresignedURL(ctx context.Context, bucketName, objectName string) (string, error)
}

type MinIOMediaRepository struct {
	client         *minio.Client
	endpoint       string
	publicEndpoint string // URL เต็มที่ browser เรียกได้จริง (เช่น https://<ref>.storage.supabase.co/storage/v1/object/public/image/...)
}

// NewMinIOMediaRepository สร้าง repository สำหรับ MinIO/S3-compatible storage
// endpoint: internal endpoint สำหรับ client (เช่น https://<ref>.storage.supabase.co/storage/v1/s3)
// publicEndpoint: public base URL ที่ browser เรียกได้ (เช่น https://<ref>.storage.supabase.co/storage/v1/object/public)
func NewMinIOMediaRepository(client *minio.Client, endpoint, publicEndpoint string) MediaRepository {
	return &MinIOMediaRepository{
		client:         client,
		endpoint:       endpoint,
		publicEndpoint: strings.TrimSuffix(publicEndpoint, "/"),
	}
}

func (m *MinIOMediaRepository) EnsureBucketExists(ctx context.Context, bucketName string) error {
	exists, err := m.client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		err := m.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	return nil
}

// UploadFile อัปโหลดไฟล์และคืน URL เต็มแบบ public สำหรับบันทึกลง Database
func (m *MinIOMediaRepository) UploadFile(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, contentType string) (string, error) {
	_, err := m.client.PutObject(ctx, bucketName, objectName, reader, objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// สร้าง URL เต็มจาก publicEndpoint: https://<ref>.storage.supabase.co/storage/v1/object/public/image/uploads/xxx.jpg
	url := fmt.Sprintf("%s/%s/%s", m.publicEndpoint, bucketName, objectName)
	return url, nil
}

func (m *MinIOMediaRepository) DownloadFile(ctx context.Context, bucketName, objectName string) (io.Reader, error) {
	object, err := m.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	return object, nil
}

func (m *MinIOMediaRepository) DeleteFile(ctx context.Context, bucketName, objectName string) error {
	err := m.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

func (m *MinIOMediaRepository) GetPresignedURL(ctx context.Context, bucketName, objectName string) (string, error) {
	expiry := time.Hour * 24
	presignedURL, err := m.client.PresignedGetObject(ctx, bucketName, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return presignedURL.String(), nil
}

func (r *postgresNovelRepository) UpdateCoverImage(id int, url string) error {
	query := `UPDATE novels SET cover_image = $1 WHERE novel_id = $2`
	_, err := r.db.Exec(query, url, id)
	if err != nil {
		return fmt.Errorf("failed to update db: %w", err)
	}
	return nil
}
