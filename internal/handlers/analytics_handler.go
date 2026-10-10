package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"novel-be/internal/service"
)

// NovelAnalyticsHandler จัดการ GET /api/v1/writer/novels/:id/analytics
//
// Authorization:
//   - RequireAuth (บังคับโดย route layer — middleware ผ่านมาก่อนถึง handler นี้)
//   - writer เจ้าของ novel → 200
//   - admin → 200
//   - writer ที่ไม่ใช่เจ้าของ → 403
//   - reader/guest → 401 (จาก RequireAuth middleware)
func NovelAnalyticsHandler(
	analyticsSvc service.AnalyticsService,
	novelSvc service.NovelService,
	writerSvc service.WriterService,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			RespondWithError(w, http.StatusMethodNotAllowed, "method not allowed", "only GET is supported")
			return
		}

		// ─── 1. Parse novel_id จาก URL path ────────────────────────────────
		// Path: /api/v1/writer/novels/{id}/analytics
		novelID, err := extractIDFromPath(r.URL.Path, "/api/v1/writer/novels/")
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "invalid novel_id: ต้องเป็นตัวเลขจำนวนเต็มมากกว่า 0", err.Error())
			return
		}

		if status, message := authorizeAnalyticsAccess(r, novelID, novelSvc, writerSvc); status != 0 {
			RespondWithError(w, status, message, message)
			return
		}

		// ─── 3. เรียก Analytics Service ───────────────────────────────────
		stats, err := analyticsSvc.GetNovelOverview(novelID)
		if err != nil {
			if errors.Is(err, service.ErrNovelNotFound) {
				RespondWithError(w, http.StatusNotFound, "novel not found", err.Error())
				return
			}
			respondInternal(w, "เกิดข้อผิดพลาดในการดึงข้อมูลสถิติ", err)
			return
		}

		RespondWithJSON(w, http.StatusOK, stats)
	}
}

// SceneAnalyticsHandler จัดการ GET /api/v1/writer/novels/:id/analytics/scenes/:sceneId
//
// Authorization:
//   - RequireAuth (บังคับโดย route layer)
//   - writer เจ้าของ novel → 200
//   - admin → 200
//   - writer ที่ไม่ใช่เจ้าของ → 403
//   - reader/guest → 401 (จาก RequireAuth middleware)
func SceneAnalyticsHandler(
	analyticsSvc service.AnalyticsService,
	novelSvc service.NovelService,
	writerSvc service.WriterService,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			RespondWithError(w, http.StatusMethodNotAllowed, "method not allowed", "only GET is supported")
			return
		}

		// ─── 1. Parse novel_id และ scene_id จาก URL path ─────────────────────
		// Path: /api/v1/writer/novels/{id}/analytics/scenes/{sceneId}
		novelID, sceneID, err := extractNovelAndSceneIDFromPath(r.URL.Path)
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "invalid novel_id or scene_id: ต้องเป็นตัวเลขจำนวนเต็มมากกว่า 0", err.Error())
			return
		}

		if status, message := authorizeAnalyticsAccess(r, novelID, novelSvc, writerSvc); status != 0 {
			RespondWithError(w, status, message, message)
			return
		}

		// ─── 3. เรียก Analytics Service ───────────────────────────────────
		stats, err := analyticsSvc.GetSceneAnalytics(novelID, sceneID)
		if err != nil {
			if errors.Is(err, service.ErrSceneNotFound) || errors.Is(err, service.ErrNovelNotFound) {
				RespondWithError(w, http.StatusNotFound, "scene not found in novel", err.Error())
				return
			}
			respondInternal(w, "เกิดข้อผิดพลาดในการดึงข้อมูลสถิติฉาก", err)
			return
		}

		RespondWithJSON(w, http.StatusOK, stats)
	}
}

// SceneChoiceAnalyticsHandler จัดการ GET /api/v1/writer/novels/:id/analytics/scenes/:sceneId/choices
//
// Authorization:
//   - RequireAuth (บังคับโดย route layer)
//   - writer เจ้าของ novel → 200
//   - admin → 200
//   - writer ที่ไม่ใช่เจ้าของ → 403
//   - reader/guest → 401 (จาก RequireAuth middleware)
func SceneChoiceAnalyticsHandler(
	analyticsSvc service.AnalyticsService,
	novelSvc service.NovelService,
	writerSvc service.WriterService,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			RespondWithError(w, http.StatusMethodNotAllowed, "method not allowed", "only GET is supported")
			return
		}

		// ─── 1. Parse novel_id และ scene_id จาก URL path ─────────────────────
		// Path: /api/v1/writer/novels/{id}/analytics/scenes/{sceneId}/choices
		novelID, sceneID, err := extractNovelAndSceneIDFromChoicePath(r.URL.Path)
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "invalid novel_id or scene_id: ต้องเป็นตัวเลขจำนวนเต็มมากกว่า 0", err.Error())
			return
		}

		if status, message := authorizeAnalyticsAccess(r, novelID, novelSvc, writerSvc); status != 0 {
			RespondWithError(w, status, message, message)
			return
		}

		// ─── 3. เรียก Analytics Service ───────────────────────────────────
		stats, err := analyticsSvc.GetSceneChoiceAnalytics(novelID, sceneID)
		if err != nil {
			if errors.Is(err, service.ErrSceneNotFound) || errors.Is(err, service.ErrNovelNotFound) {
				RespondWithError(w, http.StatusNotFound, "scene not found in novel", err.Error())
				return
			}
			respondInternal(w, "เกิดข้อผิดพลาดในการดึงข้อมูลสถิติทางเลือก", err)
			return
		}

		RespondWithJSON(w, http.StatusOK, stats)
	}
}

// AllScenesAnalyticsHandler จัดการ GET /api/v1/writer/novels/:id/analytics/scenes
func AllScenesAnalyticsHandler(analyticsSvc service.AnalyticsService, novelSvc service.NovelService, writerSvc service.WriterService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			RespondWithError(w, http.StatusMethodNotAllowed, "method not allowed", "only GET is supported")
			return
		}
		novelID, err := extractIDFromPath(r.URL.Path, "/api/v1/writer/novels/")
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "invalid novel_id", err.Error())
			return
		}
		if status, message := authorizeAnalyticsAccess(r, novelID, novelSvc, writerSvc); status != 0 {
			RespondWithError(w, status, message, message)
			return
		}
		stats, err := analyticsSvc.GetAllScenesAnalytics(novelID)
		if err != nil {
			respondInternal(w, "เกิดข้อผิดพลาดในการดึงข้อมูลสถิติทุกฉาก", err)
			return
		}
		RespondWithJSON(w, http.StatusOK, stats)
	}
}

func EdgeAnalyticsHandler(analyticsSvc service.AnalyticsService, novelSvc service.NovelService, writerSvc service.WriterService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			RespondWithError(w, http.StatusMethodNotAllowed, "method not allowed", "only GET is supported")
			return
		}
		novelID, err := extractIDFromPath(r.URL.Path, "/api/v1/writer/novels/")
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "invalid novel_id", err.Error())
			return
		}
		if status, message := authorizeAnalyticsAccess(r, novelID, novelSvc, writerSvc); status != 0 {
			RespondWithError(w, status, message, message)
			return
		}
		stats, err := analyticsSvc.GetEdgeAnalytics(novelID)
		if err != nil {
			respondInternal(w, "เกิดข้อผิดพลาดในการดึงข้อมูลสถิติของเส้นทางเลือก", err)
			return
		}
		RespondWithJSON(w, http.StatusOK, stats)
	}
}

// extractNovelAndSceneIDFromPath ถอด novelID และ sceneID จาก path เช่น /api/v1/writer/novels/1/analytics/scenes/5
func extractNovelAndSceneIDFromPath(urlPath string) (int, int, error) {
	prefix := "/api/v1/writer/novels/"
	idx := strings.Index(urlPath, prefix)
	if idx == -1 {
		return 0, 0, errors.New("invalid path prefix")
	}

	sub := urlPath[idx+len(prefix):] // e.g. "1/analytics/scenes/5"
	parts := strings.Split(strings.Trim(sub, "/"), "/")
	// Expected parts: ["1", "analytics", "scenes", "5"]
	if len(parts) < 4 || parts[1] != "analytics" || parts[2] != "scenes" {
		return 0, 0, errors.New("invalid path format")
	}

	novelID, err := strconv.Atoi(parts[0])
	if err != nil || novelID <= 0 {
		return 0, 0, errors.New("invalid novel_id")
	}

	sceneID, err := strconv.Atoi(parts[3])
	if err != nil || sceneID <= 0 {
		return 0, 0, errors.New("invalid scene_id")
	}

	return novelID, sceneID, nil
}

// extractNovelAndSceneIDFromChoicePath ถอด novelID และ sceneID จาก path เช่น /api/v1/writer/novels/1/analytics/scenes/5/choices
func extractNovelAndSceneIDFromChoicePath(urlPath string) (int, int, error) {
	prefix := "/api/v1/writer/novels/"
	idx := strings.Index(urlPath, prefix)
	if idx == -1 {
		return 0, 0, errors.New("invalid path prefix")
	}

	sub := urlPath[idx+len(prefix):] // e.g. "1/analytics/scenes/5/choices"
	parts := strings.Split(strings.Trim(sub, "/"), "/")
	// Expected parts: ["1", "analytics", "scenes", "5", "choices"]
	if len(parts) < 5 || parts[1] != "analytics" || parts[2] != "scenes" || parts[4] != "choices" {
		return 0, 0, errors.New("invalid path format")
	}

	novelID, err := strconv.Atoi(parts[0])
	if err != nil || novelID <= 0 {
		return 0, 0, errors.New("invalid novel_id")
	}

	sceneID, err := strconv.Atoi(parts[3])
	if err != nil || sceneID <= 0 {
		return 0, 0, errors.New("invalid scene_id")
	}

	return novelID, sceneID, nil
}
