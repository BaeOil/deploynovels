package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"novel-be/internal/middleware"
	"novel-be/internal/models"
	"novel-be/internal/service"
)

// captureLog สลับ log output ชั่วคราวเพื่อยืนยันว่า server ยัง log error ภายในได้
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &buf
}

func requestWithUserAndRole(method, path string, body any, userID uint, role string) *http.Request {
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	if role != "" {
		ctx = context.WithValue(ctx, middleware.RoleKey, role)
	}
	return req.WithContext(ctx)
}

// Internal error (เช่น SQL error) ต้องไม่ถูกส่งไปให้ client แต่ยังคง log ฝั่ง server
func TestCreateNovelDoesNotLeakInternalError(t *testing.T) {
	logBuf := captureLog(t)

	novel := &mockNovelService{createErr: errors.New(`pq: relation "novels" does not exist`)}
	writer := &mockWriterService{writer: &models.Writer{WriterID: 10, UserID: 42}}
	handler := NovelsHandler(novel, nil, writer, nil, nil)

	response := httptest.NewRecorder()
	handler(response, requestWithUserAndRole(http.MethodPost, "/novels", map[string]any{"title": "นิยายทดสอบ"}, 42, ""))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if strings.Contains(body, "pq:") || strings.Contains(body, "does not exist") {
		t.Fatalf("internal error leaked to client: %s", body)
	}
	if !strings.Contains(body, "internal server error") {
		t.Fatalf("expected generic message, got %s", body)
	}
	if !strings.Contains(logBuf.String(), "pq:") {
		t.Fatalf("expected internal error to be logged on server, log=%q", logBuf.String())
	}
}

// Business error ของ choice ต้อง map เป็น status ที่ถูกต้องพร้อม message เดิม
func TestCreateChoiceHandlerMapsBusinessErrors(t *testing.T) {
	newHandler := func(choiceErr error) http.HandlerFunc {
		scene := &mockSceneService{
			scene:           models.SceneResponse{SceneID: 9, NovelID: 7, ChapterID: 11},
			createChoiceErr: choiceErr,
		}
		chapter := &mockChapterService{chapter: &models.Chapter{ChapterID: 11, NovelID: 7}}
		novel := &mockNovelService{novel: models.Novel{ID: 7, AuthorID: 10}}
		writer := &mockWriterService{writer: &models.Writer{WriterID: 10, UserID: 42}}
		return CreateChoiceHandler(scene, chapter, novel, writer, nil)
	}

	t.Run("duplicate choice maps to 409 with original message", func(t *testing.T) {
		response := httptest.NewRecorder()
		handler := newHandler(service.ErrChoiceDuplicate)
		handler(response, requestWithUserAndRole(http.MethodPost, "/choices", map[string]any{
			"from_scene_id": 9, "to_scene_id": 10, "label": "ทางเลือก",
		}, 42, ""))

		if response.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d body=%s", response.Code, response.Body.String())
		}
		if !strings.Contains(response.Body.String(), "ทางเลือกนี้มีอยู่แล้ว") {
			t.Fatalf("expected original business message, got %s", response.Body.String())
		}
	})

	t.Run("label too long maps to 400", func(t *testing.T) {
		response := httptest.NewRecorder()
		handler := newHandler(service.ErrInvalidChoiceLabel)
		handler(response, requestWithUserAndRole(http.MethodPost, "/choices", map[string]any{
			"from_scene_id": 9, "to_scene_id": 10, "label": "x",
		}, 42, ""))

		if response.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("unknown internal error becomes generic 500", func(t *testing.T) {
		logBuf := captureLog(t)
		response := httptest.NewRecorder()
		handler := newHandler(errors.New("connection reset by peer"))
		handler(response, requestWithUserAndRole(http.MethodPost, "/choices", map[string]any{
			"from_scene_id": 9, "to_scene_id": 10, "label": "ทางเลือก",
		}, 42, ""))

		if response.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", response.Code, response.Body.String())
		}
		body := response.Body.String()
		if strings.Contains(body, "connection reset") {
			t.Fatalf("internal error leaked to client: %s", body)
		}
		if !strings.Contains(logBuf.String(), "connection reset") {
			t.Fatalf("expected internal error to be logged, log=%q", logBuf.String())
		}
	})
}
