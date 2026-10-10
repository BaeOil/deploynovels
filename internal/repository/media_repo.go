package repository

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

type MediaRepository interface {
	EnsureBucketExists(ctx context.Context, bucketName string) error
	UploadFile(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, contentType string) (string, error)
	DownloadFile(ctx context.Context, bucketName, objectName string) (io.Reader, error)
	DeleteFile(ctx context.Context, bucketName, objectName string) error
	GetPresignedURL(ctx context.Context, bucketName, objectName string) (string, error)
}

// S3MediaRepository เชื่อมต่อ Supabase Storage (S3-compatible) ผ่าน AWS SDK for Go v2
// เลิกใช้ minio-go เพราะ minio-go ไม่รับ endpoint ที่มี path (/storage/v1/s3) ทำให้เชื่อม Supabase ไม่ได้
type S3MediaRepository struct {
	client         *s3.Client
	presignClient  *s3.PresignClient
	publicEndpoint string // public base URL ที่ browser เรียกได้ เช่น https://<ref>.storage.supabase.co/storage/v1/object/public
}

// NewS3MediaRepository สร้าง repository สำหรับ Supabase Storage (S3-compatible)
// client: s3 client ที่ config BaseEndpoint + UsePathStyle ไว้แล้ว (ทำใน main.go)
// publicEndpoint: public base URL สำหรับสร้าง URL ที่ browser เข้าถึงได้
func NewS3MediaRepository(client *s3.Client, publicEndpoint string) MediaRepository {
	return &S3MediaRepository{
		client:         client,
		presignClient:  s3.NewPresignClient(client),
		publicEndpoint: strings.TrimSuffix(strings.TrimSpace(publicEndpoint), "/"),
	}
}

func (m *S3MediaRepository) EnsureBucketExists(ctx context.Context, bucketName string) error {
	_, err := m.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucketName)})
	if err == nil {
		return nil // bucket มีอยู่แล้ว
	}

	// ถ้าเป็น 404 NotFound แปลว่ายังไม่มี bucket → ลองสร้าง
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		if code != "NotFound" && code != "404" && code != "NoSuchBucket" {
			return fmt.Errorf("failed to check bucket existence: %w", err)
		}
	} else {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	_, err = m.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucketName)})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}
	return nil
}

// UploadFile อัปโหลดไฟล์และคืน URL เต็มแบบ public สำหรับบันทึกลง Database
func (m *S3MediaRepository) UploadFile(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, contentType string) (string, error) {
	_, err := m.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucketName),
		Key:           aws.String(objectName),
		Body:          reader,
		ContentLength: aws.Int64(objectSize),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// สร้าง URL เต็มจาก publicEndpoint: https://<ref>.storage.supabase.co/storage/v1/object/public/image/uploads/xxx.jpg
	url := fmt.Sprintf("%s/%s/%s", m.publicEndpoint, bucketName, objectName)
	return url, nil
}

func (m *S3MediaRepository) DownloadFile(ctx context.Context, bucketName, objectName string) (io.Reader, error) {
	out, err := m.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	return out.Body, nil
}

func (m *S3MediaRepository) DeleteFile(ctx context.Context, bucketName, objectName string) error {
	_, err := m.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

func (m *S3MediaRepository) GetPresignedURL(ctx context.Context, bucketName, objectName string) (string, error) {
	req, err := m.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectName),
	}, s3.WithPresignExpires(time.Hour*24))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return req.URL, nil
}

func (r *postgresNovelRepository) UpdateCoverImage(id int, url string) error {
	query := `UPDATE novels SET cover_image = $1 WHERE novel_id = $2`
	_, err := r.db.Exec(query, url, id)
	if err != nil {
		return fmt.Errorf("failed to update db: %w", err)
	}
	return nil
}
