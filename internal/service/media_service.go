package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"novel-be/internal/repository"
	"path/filepath"
	"strings"
	"time"
)

const (
	NovelImagesBucket = "novel-buckets"

	// NovelCoverMaxBytes ตาม VAL-08: รูปปกนิยายไม่เกิน 10MB
	NovelCoverMaxBytes int64 = 10 * 1024 * 1024
	// ProfilePictureMaxBytes ตาม VAL-09: รูปโปรไฟล์/avatar ไม่เกิน 5MB (แยก limit จากปกนิยาย)
	ProfilePictureMaxBytes int64 = 5 * 1024 * 1024
)

// ข้อผิดพลาดด้าน validation ของไฟล์อัปโหลด — เป็น business error
// ที่ handler ตอบกลับเป็น 400 ได้โดยไม่เปิดเผยข้อมูลภายในระบบ
var (
	ErrImageTooLarge    = errors.New("image file size exceeds the allowed limit")
	ErrInvalidImageType = errors.New("invalid image file: only JPEG, PNG, and WebP are allowed")
)

type mediaService struct {
	mediaRepo repository.MediaRepository
}

func NewMediaService(mediaRepo repository.MediaRepository) MediaService {
	return &mediaService{
		mediaRepo: mediaRepo,
	}
}

// UploadImage อัปโหลดรูปภาพพร้อมบังคับ limit ตามชนิดการใช้งาน (maxBytes)
// และตรวจสอบชนิดไฟล์จากเนื้อหาไฟล์จริง (magic bytes) ไม่เชื่อ extension
// หรือ Content-Type ที่ client ส่งมา (VAL-08 / VAL-09 / VAL-10)
func (s *mediaService) UploadImage(ctx context.Context, file *multipart.FileHeader, maxBytes int64) (string, error) {
	if maxBytes <= 0 {
		maxBytes = NovelCoverMaxBytes
	}
	if file.Size > maxBytes {
		return "", fmt.Errorf("%w: max %d bytes", ErrImageTooLarge, maxBytes)
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// อ่านหัวไฟล์เพื่อตรวจชนิดจริง (JPEG/PNG/WebP เท่านั้น ไม่รับ GIF)
	head := make([]byte, 512)
	n, readErr := io.ReadFull(src, head)
	if readErr != nil && !errors.Is(readErr, io.ErrUnexpectedEOF) && !errors.Is(readErr, io.EOF) {
		return "", readErr
	}
	if n == 0 {
		return "", ErrInvalidImageType
	}
	contentType := detectImageContentType(head[:n])
	if contentType == "" {
		return "", ErrInvalidImageType
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !isAllowedImageType(ext) {
		return "", ErrInvalidImageType
	}

	// ย้าย cursor กลับไปจุดเริ่มต้นก่อนส่งเข้า storage
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	objectName := fmt.Sprintf("uploads/%d-%s", time.Now().Unix(), file.Filename)

	if err := s.mediaRepo.EnsureBucketExists(ctx, NovelImagesBucket); err != nil {
		return "", err
	}

	// ใช้ชนิดที่ตรวจได้จากเนื้อหาไฟล์จริง ไม่ใช่ค่าที่ client ประกาศมา
	url, err := s.mediaRepo.UploadFile(ctx, NovelImagesBucket, objectName, src, file.Size, contentType)
	if err != nil {
		return "", err
	}

	return url, nil
}

func (s *mediaService) DeleteImage(ctx context.Context, filename string) error {
	return s.mediaRepo.DeleteFile(ctx, NovelImagesBucket, filename)
}

func (s *mediaService) GetPresignedURL(ctx context.Context, filename string) (string, error) {
	return s.mediaRepo.GetPresignedURL(ctx, NovelImagesBucket, filename)
}

// isAllowedImageType ตาม VAL-10: มีเฉพาะ .jpg/.jpeg/.png/.webp (ไม่รับ .gif)
func isAllowedImageType(ext string) bool {
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	return allowed[ext]
}

// detectImageContentType ตรวจ magic bytes จริงของไฟล์
// คืนค่าว่างถ้าไม่ใช่ JPEG/PNG/WebP (ไฟล์อื่น เช่น GIF, EXE จะถูกปฏิเสธ)
func detectImageContentType(head []byte) string {
	switch {
	case len(head) >= 3 && head[0] == 0xFF && head[1] == 0xD8 && head[2] == 0xFF:
		return "image/jpeg"
	case len(head) >= 8 && bytes.Equal(head[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return "image/png"
	case len(head) >= 12 && bytes.Equal(head[:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")):
		return "image/webp"
	default:
		return ""
	}
}
