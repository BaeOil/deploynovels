package handlers

import (
	"bytes"
	"context"
	"errors"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"novel-be/internal/middleware"
	"novel-be/internal/models"
	"novel-be/internal/service"
)

// fakeMediaService บันทึกการเรียก upload เพื่อยืนยันว่า handler ไม่ upload เมื่อ authorization fail
type fakeMediaService struct {
	uploads      int
	lastMaxBytes int64
	uploadErr    error
	uploadedURL  string
}

func (f *fakeMediaService) UploadImage(_ context.Context, _ *multipart.FileHeader, maxBytes int64) (string, error) {
	f.uploads++
	f.lastMaxBytes = maxBytes
	if f.uploadErr != nil {
		return "", f.uploadErr
	}
	if f.uploadedURL == "" {
		f.uploadedURL = "http://minio:9000/novel-buckets/uploads/cover.jpg"
	}
	return f.uploadedURL, nil
}
func (f *fakeMediaService) DeleteImage(context.Context, string) error { return nil }
func (f *fakeMediaService) GetPresignedURL(context.Context, string) (string, error) {
	return "", nil
}

// uploadImageRequest สร้าง multipart POST request ตามรูปแบบ /upload/image จริง
func uploadImageRequest(t *testing.T, novelID string, userID uint, role string) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if novelID != "" {
		if err := mw.WriteField("novel_id", novelID); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	part, err := mw.CreateFormFile("image", "cover.jpg")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
	if _, err := part.Write(jpeg); err != nil {
		t.Fatalf("write file: %v", err)
	}
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload/image", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	ctx := req.Context()
	if userID != 0 {
		ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	}
	if role != "" {
		ctx = context.WithValue(ctx, middleware.RoleKey, role)
	}
	return req.WithContext(ctx)
}

// 1. Owner (นักเขียนเจ้าของนิยาย) → upload/update cover สำเร็จ
func TestUploadImageOwnerCanUpdateCover(t *testing.T) {
	media := &fakeMediaService{}
	novel := &mockNovelService{novel: models.Novel{ID: 7, AuthorID: 10}}
	writer := &mockWriterService{writer: &models.Writer{WriterID: 10, UserID: 42}}
	handler := UploadImageHandler(media, novel, writer)

	response := httptest.NewRecorder()
	handler(response, uploadImageRequest(t, "7", 42, ""))

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201 for owner, got %d body=%s", response.Code, response.Body.String())
	}
	if media.uploads != 1 {
		t.Fatalf("expected 1 upload, got %d", media.uploads)
	}
	if media.lastMaxBytes != service.NovelCoverMaxBytes {
		t.Fatalf("expected cover limit %d, got %d", service.NovelCoverMaxBytes, media.lastMaxBytes)
	}
	if novel.coverUpdatedID != 7 || novel.coverUpdatedURL == "" {
		t.Fatalf("expected cover of novel 7 to be updated, got id=%d url=%q", novel.coverUpdatedID, novel.coverUpdatedURL)
	}
}

// 2. Writer A ใช้ novel_id ของ Writer B → 403 และไม่มีการ upload/อัปเดต DB
func TestUploadImageRejectsOtherWritersNovel(t *testing.T) {
	media := &fakeMediaService{}
	novel := &mockNovelService{novel: models.Novel{ID: 7, AuthorID: 99}} // เจ้าของจริงคือคนอื่น
	writer := &mockWriterService{writer: &models.Writer{WriterID: 10, UserID: 42}}
	handler := UploadImageHandler(media, novel, writer)

	response := httptest.NewRecorder()
	handler(response, uploadImageRequest(t, "7", 42, ""))

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-owner writer, got %d body=%s", response.Code, response.Body.String())
	}
	if media.uploads != 0 {
		t.Fatal("upload must not happen when authorization fails")
	}
	if novel.coverUpdatedID != 0 || novel.coverUpdatedURL != "" {
		t.Fatalf("DB must not be updated when authorization fails, got id=%d url=%q", novel.coverUpdatedID, novel.coverUpdatedURL)
	}
}

// 3. ผู้ใช้ที่ไม่มีสิทธิ์ (ไม่ใช่นักเขียน) → 403 ไม่มีการ upload/อัปเดต DB
func TestUploadImageRejectsNonWriterUser(t *testing.T) {
	media := &fakeMediaService{}
	novel := &mockNovelService{novel: models.Novel{ID: 7, AuthorID: 10}}
	writer := &mockWriterService{} // ไม่มี record นักเขียน
	handler := UploadImageHandler(media, novel, writer)

	response := httptest.NewRecorder()
	handler(response, uploadImageRequest(t, "7", 42, ""))

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for user without writer record, got %d body=%s", response.Code, response.Body.String())
	}
	if media.uploads != 0 {
		t.Fatal("upload must not happen when authorization fails")
	}
	if novel.coverUpdatedID != 0 {
		t.Fatal("DB must not be updated when authorization fails")
	}
}

// 4. ไม่มี user ใน context → 401 และไม่มีการ upload
func TestUploadImageRejectsUnauthenticatedUser(t *testing.T) {
	media := &fakeMediaService{}
	novel := &mockNovelService{novel: models.Novel{ID: 7, AuthorID: 10}}
	writer := &mockWriterService{writer: &models.Writer{WriterID: 10, UserID: 42}}
	handler := UploadImageHandler(media, novel, writer)

	response := httptest.NewRecorder()
	handler(response, uploadImageRequest(t, "7", 0, ""))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing user, got %d body=%s", response.Code, response.Body.String())
	}
	if media.uploads != 0 {
		t.Fatal("upload must not happen when authorization fails")
	}
}

// Admin ยังใช้สิทธิ์ได้ตาม architecture เดิม (CheckIsOwnerOrAdmin มี admin bypass)
func TestUploadImageAllowsAdmin(t *testing.T) {
	media := &fakeMediaService{}
	novel := &mockNovelService{novel: models.Novel{ID: 7, AuthorID: 99}}
	writer := &mockWriterService{} // admin ไม่จำเป็นต้องมี record นักเขียน
	handler := UploadImageHandler(media, novel, writer)

	response := httptest.NewRecorder()
	handler(response, uploadImageRequest(t, "7", 999, "admin"))

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201 for admin, got %d body=%s", response.Code, response.Body.String())
	}
	if novel.coverUpdatedID != 7 {
		t.Fatalf("expected cover update for admin, got id=%d", novel.coverUpdatedID)
	}
}

// novel_id ที่ไม่ใช่ตัวเลข → 400 และไม่มีการ upload
func TestUploadImageRejectsInvalidNovelID(t *testing.T) {
	media := &fakeMediaService{}
	novel := &mockNovelService{}
	writer := &mockWriterService{writer: &models.Writer{WriterID: 10, UserID: 42}}
	handler := UploadImageHandler(media, novel, writer)

	response := httptest.NewRecorder()
	handler(response, uploadImageRequest(t, "not-a-number", 42, ""))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid novel_id, got %d", response.Code)
	}
	if media.uploads != 0 {
		t.Fatal("upload must not happen for invalid novel_id")
	}
}

// error ภายในระบบ storage ต้องไม่ถูกส่งไปให้ client (log ไว้ฝั่ง server แทน)
func TestUploadImageHidesStorageErrorFromClient(t *testing.T) {
	var logBuf bytes.Buffer
	prevWriter := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(prevWriter)

	media := &fakeMediaService{uploadErr: errors.New("Put \"http://minio:9000/\": dial tcp: connection refused")}
	novel := &mockNovelService{novel: models.Novel{ID: 7, AuthorID: 10}}
	writer := &mockWriterService{writer: &models.Writer{WriterID: 10, UserID: 42}}
	handler := UploadImageHandler(media, novel, writer)

	response := httptest.NewRecorder()
	handler(response, uploadImageRequest(t, "7", 42, ""))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for storage failure, got %d", response.Code)
	}
	body := response.Body.String()
	if strings.Contains(body, "minio") || strings.Contains(body, "connection refused") {
		t.Fatalf("internal error leaked to client: %s", body)
	}
	if !strings.Contains(body, "internal server error") {
		t.Fatalf("expected generic message, got %s", body)
	}
	if !strings.Contains(logBuf.String(), "connection refused") {
		t.Fatalf("expected internal error to be logged, log=%q", logBuf.String())
	}
	if novel.coverUpdatedID != 0 {
		t.Fatal("DB must not be updated when upload fails")
	}
}
