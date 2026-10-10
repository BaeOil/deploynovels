package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"novel-be/internal/repository"
)

// fakeMediaRepo บันทึกการเรียก upload เพื่อตรวจว่า service ส่ง contentType ที่ตรวจจากไฟล์จริง
type fakeMediaRepo struct {
	uploads     int
	contentType string
	uploadSize  int64
}

func (f *fakeMediaRepo) EnsureBucketExists(context.Context, string) error { return nil }
func (f *fakeMediaRepo) UploadFile(_ context.Context, bucket, object string, _ io.Reader, size int64, contentType string) (string, error) {
	f.uploads++
	f.contentType = contentType
	f.uploadSize = size
	return "http://minio:9000/" + bucket + "/" + object, nil
}
func (f *fakeMediaRepo) DownloadFile(context.Context, string, string) (io.Reader, error) {
	return nil, nil
}
func (f *fakeMediaRepo) DeleteFile(context.Context, string, string) error { return nil }
func (f *fakeMediaRepo) GetPresignedURL(context.Context, string, string) (string, error) {
	return "", nil
}

var _ repository.MediaRepository = (*fakeMediaRepo)(nil)

// newFileHeader สร้าง *multipart.FileHeader จาก content จริง (ผ่าน multipart parse ตามการใช้งานจริง)
func newFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()
	return newFileHeaderWithContentType(t, filename, "application/octet-stream", content)
}

func newFileHeaderWithContentType(t *testing.T, filename, contentType string, content []byte) *multipart.FileHeader {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="image"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	part, err := mw.CreatePart(header)
	if err != nil {
		t.Fatalf("create multipart part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write multipart part: %v", err)
	}
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if err := req.ParseMultipartForm(64 << 20); err != nil {
		t.Fatalf("parse multipart form: %v", err)
	}
	_, fileHeader, err := req.FormFile("image")
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	return fileHeader
}

func imageWithMagic(size int, magic []byte) []byte {
	if size < len(magic) {
		size = len(magic)
	}
	content := make([]byte, size)
	copy(content, magic)
	return content
}

var (
	jpegMagic = []byte{0xFF, 0xD8, 0xFF, 0xE0}
	pngMagic  = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	webpMagic = []byte{'R', 'I', 'F', 'F', 0x24, 0x00, 0x00, 0x00, 'W', 'E', 'B', 'P', 'V', 'P', '8', ' '}
	gifMagic  = []byte{'G', 'I', 'F', '8', '9', 'a'}
	exeMagic  = []byte{'M', 'Z', 0x90, 0x00}
)

// VAL-10: รูปที่ผ่านต้องเป็น JPEG / PNG / WebP เท่านั้น ตรวจจากเนื้อหาไฟล์จริง
func TestUploadImageAcceptsJPEGPNGWebP(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  []byte
		wantMime string
	}{
		{"jpeg", "cover.jpg", imageWithMagic(1024, jpegMagic), "image/jpeg"},
		{"jpeg with jpeg extension", "photo.jpeg", imageWithMagic(1024, jpegMagic), "image/jpeg"},
		{"png", "cover.png", imageWithMagic(1024, pngMagic), "image/png"},
		{"webp", "cover.webp", imageWithMagic(1024, webpMagic), "image/webp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeMediaRepo{}
			svc := NewMediaService(repo)
			fileHeader := newFileHeader(t, tt.filename, tt.content)

			url, err := svc.UploadImage(context.Background(), fileHeader, NovelCoverMaxBytes)
			if err != nil {
				t.Fatalf("expected upload to succeed, got %v", err)
			}
			if url == "" {
				t.Fatal("expected url")
			}
			if repo.uploads != 1 {
				t.Fatalf("expected 1 upload, got %d", repo.uploads)
			}
			if repo.contentType != tt.wantMime {
				t.Fatalf("expected stored content type %q (detected from file content), got %q", tt.wantMime, repo.contentType)
			}
		})
	}
}

// VAL-10: GIF ต้องไม่ผ่าน ไม่ว่าจะใช้ extension อะไร
func TestUploadImageRejectsGIF(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"gif with gif extension", "anim.gif"},
		{"gif renamed to png", "anim.png"},
		{"gif renamed to jpg", "anim.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeMediaRepo{}
			svc := NewMediaService(repo)
			fileHeader := newFileHeader(t, tt.filename, imageWithMagic(1024, gifMagic))

			if _, err := svc.UploadImage(context.Background(), fileHeader, NovelCoverMaxBytes); !errors.Is(err, ErrInvalidImageType) {
				t.Fatalf("expected ErrInvalidImageType, got %v", err)
			}
			if repo.uploads != 0 {
				t.Fatal("rejected file must not be uploaded")
			}
		})
	}
}

// VAL-10: เปลี่ยนชื่อ .exe เป็น .png ต้องไม่ผ่าน + Content-Type ปลอมต้องช่วยให้ผ่านไม่ได้
func TestUploadImageRejectsFakeExtensionAndSpoofedContentType(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		contentType string
		content     []byte
	}{
		{"exe renamed to png", "malware.png", "image/png", imageWithMagic(1024, exeMagic)},
		{"exe with spoofed content type and jpg name", "photo.jpg", "image/jpeg", imageWithMagic(1024, exeMagic)},
		{"text renamed to webp", "page.webp", "image/webp", []byte("this is definitely not an image")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeMediaRepo{}
			svc := NewMediaService(repo)
			fileHeader := newFileHeaderWithContentType(t, tt.filename, tt.contentType, tt.content)

			if _, err := svc.UploadImage(context.Background(), fileHeader, NovelCoverMaxBytes); !errors.Is(err, ErrInvalidImageType) {
				t.Fatalf("expected ErrInvalidImageType, got %v", err)
			}
			if repo.uploads != 0 {
				t.Fatal("invalid file must not be uploaded")
			}
		})
	}
}

// VAL-10: extension ไม่ใช่ jpg/jpeg/png/webp ต้องไม่ผ่าน แม้เนื้อหาจะเป็นรูปจริง
func TestUploadImageRejectsDisallowedExtension(t *testing.T) {
	repo := &fakeMediaRepo{}
	svc := NewMediaService(repo)
	fileHeader := newFileHeader(t, "cover.gif", imageWithMagic(1024, jpegMagic))

	if _, err := svc.UploadImage(context.Background(), fileHeader, NovelCoverMaxBytes); !errors.Is(err, ErrInvalidImageType) {
		t.Fatalf("expected ErrInvalidImageType for .gif extension, got %v", err)
	}
	if repo.uploads != 0 {
		t.Fatal("rejected file must not be uploaded")
	}
}

// VAL-08: ปกนิยาย 10MB → ผ่าน / >10MB → ไม่ผ่าน
func TestUploadImageCoverSizeBoundary(t *testing.T) {
	svc := NewMediaService(&fakeMediaRepo{})

	atLimit := newFileHeader(t, "cover.jpg", imageWithMagic(int(NovelCoverMaxBytes), jpegMagic))
	if _, err := svc.UploadImage(context.Background(), atLimit, NovelCoverMaxBytes); err != nil {
		t.Fatalf("expected exactly 10MB to pass, got %v", err)
	}

	overLimit := newFileHeader(t, "cover.jpg", imageWithMagic(int(NovelCoverMaxBytes)+1, jpegMagic))
	if _, err := svc.UploadImage(context.Background(), overLimit, NovelCoverMaxBytes); !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("expected ErrImageTooLarge for >10MB, got %v", err)
	}
}

// VAL-09: รูปโปรไฟล์ 5MB → ผ่าน / >5MB → ไม่ผ่าน และแยก limit จากปกนิยาย
func TestUploadImageProfileSizeBoundary(t *testing.T) {
	svc := NewMediaService(&fakeMediaRepo{})

	atLimit := newFileHeader(t, "avatar.png", imageWithMagic(int(ProfilePictureMaxBytes), pngMagic))
	if _, err := svc.UploadImage(context.Background(), atLimit, ProfilePictureMaxBytes); err != nil {
		t.Fatalf("expected exactly 5MB to pass, got %v", err)
	}

	overLimit := newFileHeader(t, "avatar.png", imageWithMagic(int(ProfilePictureMaxBytes)+1, pngMagic))
	if _, err := svc.UploadImage(context.Background(), overLimit, ProfilePictureMaxBytes); !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("expected ErrImageTooLarge for >5MB, got %v", err)
	}

	// ไฟล์ 6MB ต้องผ่านเมื่อใช้ limit ของปก (10MB) แต่ไม่ผ่านเมื่อใช้ limit ของโปรไฟล์ (5MB)
	sixMB := newFileHeader(t, "photo.jpg", imageWithMagic(6*1024*1024, jpegMagic))
	if _, err := svc.UploadImage(context.Background(), sixMB, ProfilePictureMaxBytes); !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("expected 6MB to fail profile limit, got %v", err)
	}
	sixMB = newFileHeader(t, "photo.jpg", imageWithMagic(6*1024*1024, jpegMagic))
	if _, err := svc.UploadImage(context.Background(), sixMB, NovelCoverMaxBytes); err != nil {
		t.Fatalf("expected 6MB to pass cover limit, got %v", err)
	}
}
