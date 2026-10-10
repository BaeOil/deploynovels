package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"novel-be/internal/dto"
	"novel-be/internal/middleware"
	"novel-be/internal/models"
	"novel-be/internal/service"
)

type NotificationHandler struct {
	service service.NotificationService
}

func NewNotificationHandler(service service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondWithError3(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		RespondWithError3(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.ListNotifications(int(userID), page, limit)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	payloads := make([]map[string]any, 0, len(items))
	for _, item := range items {
		payloads = append(payloads, notificationPayloadFromModel(item))
	}
	RespondWithJSON(w, http.StatusOK, payloads)
}

func notificationPayloadFromModel(n models.Notification) map[string]any {
	cover := any(nil)
	if n.CoverImage != nil {
		cover = *n.CoverImage
	}
	body := n.Message
	if n.Type == "comment" {
		if parts := strings.SplitN(n.Message, "|", 2); len(parts) == 2 {
			body = strings.TrimSpace(parts[1])
		}
	}
	if n.Type == "like" {
		body = ""
	}
	return map[string]any{
		"id":              n.ID,
		"user_id":         n.UserID,
		"type":            n.Type,
		"title":           nilOrString(n.Title),
		"message":         body,
		"body":            body,
		"cover_image":     cover,
		"cover":           cover,
		"reference_id":    n.ReferenceID,
		"reference_type":  n.ReferenceType,
		"reference_title": n.ReferenceTitle,
		"is_read":         n.IsRead,
		"created_at":      n.CreatedAt.Format(time.RFC3339),
		"actor_id":        n.ActorID,
		"actor_name":      n.ActorName,
		"actor_avatar":    n.ActorAvatar,
		"actor_color":     n.ActorColor,
		"actor": map[string]any{
			"id":     n.ActorID,
			"name":   n.ActorName,
			"avatar": n.ActorAvatar,
			"color":  n.ActorColor,
		},
	}
}

func nilOrString(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func (h *NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondWithError3(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		RespondWithError3(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	count, err := h.service.GetUnreadCount(int(userID))
	if err != nil {
		writeInternalError(w, err)
		return
	}
	RespondWithJSON(w, http.StatusOK, map[string]any{"unread_count": count})
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		RespondWithError3(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		RespondWithError3(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, ok := extractNotificationID(r.URL.Path)
	if !ok || id == 0 {
		RespondWithError3(w, http.StatusBadRequest, "invalid notification id")
		return
	}
	if err := h.service.MarkRead(id, int(userID)); err != nil {
		if err == sql.ErrNoRows {
			RespondWithError3(w, http.StatusNotFound, "notification not found")
			return
		}
		writeInternalError(w, err)
		return
	}
	RespondWithJSON(w, http.StatusOK, map[string]any{"message": "notification marked as read"})
}

func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		RespondWithError3(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		RespondWithError3(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.service.MarkAllRead(int(userID)); err != nil {
		writeInternalError(w, err)
		return
	}
	RespondWithJSON(w, http.StatusOK, map[string]any{"message": "all notifications marked as read"})
}

func (h *NotificationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		RespondWithError3(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		RespondWithError3(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, ok := extractNotificationID(r.URL.Path)
	if !ok || id == 0 {
		RespondWithError3(w, http.StatusBadRequest, "invalid notification id")
		return
	}
	if err := h.service.Delete(id, int(userID)); err != nil {
		if err == sql.ErrNoRows {
			RespondWithError3(w, http.StatusNotFound, "notification not found")
			return
		}
		writeInternalError(w, err)
		return
	}
	RespondWithJSON(w, http.StatusOK, map[string]any{"message": "notification deleted"})
}

func (h *NotificationHandler) DeleteAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		RespondWithError3(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		RespondWithError3(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.service.DeleteAll(int(userID)); err != nil {
		writeInternalError(w, err)
		return
	}
	RespondWithJSON(w, http.StatusOK, map[string]any{"message": "all notifications deleted"})
}

func (h *NotificationHandler) Stream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondWithError3(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		RespondWithError3(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		RespondWithError3(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	ch, cleanup := service.SubscribeNotifications(int(userID))
	defer cleanup()

	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case payload, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: notification\ndata: %s\n\n", payload)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (h *NotificationHandler) CreateFromPayload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondWithError3(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var payload struct {
		UserID        int    `json:"user_id"`
		Type          string `json:"type"`
		Title         string `json:"title"`
		Message       string `json:"message"`
		CoverImage    string `json:"cover_image"`
		ReferenceID   int    `json:"reference_id"`
		ReferenceType string `json:"reference_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError3(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var cover *string
	if payload.CoverImage != "" {
		cover = &payload.CoverImage
	}
	var referenceID *int
	if payload.ReferenceID != 0 {
		referenceID = &payload.ReferenceID
	}
	var referenceType *string
	if payload.ReferenceType != "" {
		referenceType = &payload.ReferenceType
	}
	_, err := h.service.CreateNotification(payload.UserID, 0, payload.Type, payload.Title, payload.Message, cover, referenceID, referenceType)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	RespondWithJSON(w, http.StatusCreated, map[string]any{"message": "notification created"})
}

func (h *NotificationHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondWithError3(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		RespondWithError3(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	settings, err := h.service.GetNotificationSettings(int(userID))
	if err != nil {
		writeInternalError(w, err)
		return
	}
	RespondWithJSON(w, http.StatusOK, settings)
}

func (h *NotificationHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPut {
		RespondWithError3(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		RespondWithError3(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req dto.NotificationSettingsDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError3(w, http.StatusBadRequest, "รูปแบบข้อมูลไม่ถูกต้อง")
		return
	}
	if err := h.service.UpdateNotificationSettings(int(userID), req); err != nil {
		writeInternalError(w, err)
		return
	}
	RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "notification settings updated successfully",
	})
}

func extractNotificationID(path string) (int, bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 {
		return 0, false
	}
	candidate := parts[len(parts)-1]
	if candidate == "read" || candidate == "delete" {
		if len(parts) < 2 {
			return 0, false
		}
		candidate = parts[len(parts)-2]
	}
	id, err := strconv.Atoi(candidate)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
