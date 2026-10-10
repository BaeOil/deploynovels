package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"novel-be/internal/middleware"
	"novel-be/internal/models"
	"novel-be/internal/service"
	"strings"
)

func toPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func UpdateScenePositionHandler(sceneService service.SceneService, novelService service.NovelService, writerService service.WriterService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sceneID, err := extractIDFromPath(r.URL.Path, "/scenes/")
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid id parameter")
			return
		}

		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok || userID == 0 {
			WriteError(w, http.StatusUnauthorized, "unauthorized: ไม่พบข้อมูลสิทธิ์ผู้ใช้งานหรือโทเคนไม่ถูกต้อง")
			return
		}

		writer, err := writerService.GetWriterByUserID(int(userID))
		if err != nil || writer == nil {
			WriteError(w, http.StatusForbidden, "forbidden: คุณยังไม่ใช่นักเขียนที่ได้รับอนุมัติ")
			return
		}

		existingScene, err := sceneService.GetScene(sceneID)
		if err != nil {
			WriteError(w, http.StatusNotFound, "scene not found")
			return
		}

		novelDetail, err := novelService.GetNovelDetail(existingScene.NovelID)
		if err != nil {
			writeInternalError(w, err)
			return
		}

		novelPtr, ok := novelDetail.(*models.Novel)
		if !ok || novelPtr == nil {
			WriteError(w, http.StatusInternalServerError, "failed to load novel details")
			return
		}

		if novelPtr.AuthorID != writer.WriterID {
			WriteError(w, http.StatusForbidden, "forbidden: คุณไม่มีสิทธิ์แก้ไขตำแหน่งนิยายเรื่องนี้")
			return
		}

		var req UpdateScenePositionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := req.Validate(); err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if err := sceneService.UpdateScenePosition(sceneID, req.NodeX, req.NodeY); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				WriteError(w, http.StatusNotFound, "scene not found")
				return
			}
			writeInternalError(w, err)
			return
		}

		WriteJSON(w, http.StatusOK, map[string]any{
			"message":  "scene position updated",
			"scene_id": sceneID,
			"node_x":   req.NodeX,
			"node_y":   req.NodeY,
		})
	}
}

func CreateSceneHandler(sceneService service.SceneService, notificationService service.NotificationService, chapterService service.ChapterService, novelService service.NovelService, writerService service.WriterService, auditService service.AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateSceneRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := req.Validate(); err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		chapter, err := chapterService.GetChapterByID(req.ChapterID)
		if err != nil || chapter == nil || chapter.NovelID != req.NovelID {
			WriteError(w, http.StatusBadRequest, "chapter does not belong to novel")
			return
		}
		if err := requireWriterOwnsNovel(r.Context(), writerService, novelService, chapter.NovelID); err != nil {
			if writeOwnershipError(w, err) {
				return
			}
			writeInternalError(w, err)
			return
		}

		// ✅ แก้จุดแดง image_48c32e โดยใช้ strPtr() หุ้มค่าที่เป็น string
		sceneID, err := sceneService.CreateScene(models.Scene{
			NovelID:           req.NovelID,
			ChapterID:         req.ChapterID,
			Title:             req.Title,
			Content:           req.Content,
			ImageURL:          req.ImageURL,
			Type:              req.Type,
			Status:            req.Status,
			EndingTitle:       toPtr(req.EndingTitle),
			EndingType:        toPtr(req.EndingType),
			EndingDescription: toPtr(req.EndingDescription),
		})

		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if req.Choices != nil {
			if _, err := sceneService.SyncSceneChoices(sceneID, req.Choices); err != nil {
				writeChoiceServiceError(w, err)
				return
			}
		}

		if err := notificationService.NotifySceneUpdated(req.NovelID, sceneID); err != nil {
			log.Printf("NotifySceneUpdated failed: %v", err)
		}
		createMeta := map[string]interface{}{"novel_id": req.NovelID, "chapter_id": req.ChapterID}
		if ch, err := chapterService.GetChapterByID(req.ChapterID); err == nil && ch != nil {
			createMeta["chapter_title"] = ch.Title
		}
		if nov, err := novelService.GetNovelDetail(req.NovelID); err == nil && nov != nil {
			if n, ok := nov.(*models.Novel); ok && n != nil {
				createMeta["novel_title"] = n.Title
			}
		}
		recordAudit(r, auditService, service.AuditEvent{Action: "CREATE_SCENE", TargetType: "scene", TargetID: int64Pointer(sceneID), Status: "SUCCESS", Metadata: createMeta})

		WriteJSON(w, http.StatusCreated, map[string]any{"message": "scene created", "scene_id": sceneID})
	}
}

func UpdateSceneHandler(sceneService service.SceneService, notificationService service.NotificationService, chapterService service.ChapterService, novelService service.NovelService, writerService service.WriterService, auditService service.AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sceneID, err := extractIDFromPath(r.URL.Path, "/scenes/")
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid id parameter")
			return
		}

		existingScene, err := sceneService.GetScene(sceneID)
		if err != nil {
			WriteError(w, http.StatusNotFound, "scene not found")
			return
		}
		chapter, err := chapterService.GetChapterByID(existingScene.ChapterID)
		if err != nil || chapter == nil || chapter.NovelID != existingScene.NovelID {
			WriteError(w, http.StatusNotFound, "scene not found")
			return
		}
		if err := requireWriterOwnsNovel(r.Context(), writerService, novelService, existingScene.NovelID); err != nil {
			if writeOwnershipError(w, err) {
				return
			}
			writeInternalError(w, err)
			return
		}

		var req UpdateSceneRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := req.Validate(); err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		scene := models.Scene{
			SceneID:           sceneID,
			NovelID:           existingScene.NovelID,
			ChapterID:         existingScene.ChapterID,
			Title:             req.Title,
			Content:           req.Content,
			Type:              req.Type,
			Status:            req.Status,
			EndingTitle:       toPtr(req.EndingTitle),
			EndingType:        toPtr(req.EndingType),
			EndingDescription: toPtr(req.EndingDescription),
		}

		if req.IsEnding {
			scene.Type = "ending"
		}

		var validationResult *service.PublishValidationResult
		isPublishing := !strings.EqualFold(existingScene.Status, "published") && strings.EqualFold(req.Status, "published")
		if isPublishing {
			val := sceneService.ValidateNovelPublishability(existingScene.NovelID)
			validationResult = &val
			if !val.CanPublish {
				WriteJSON(w, http.StatusBadRequest, map[string]any{
					"error":       "ไม่สามารถเผยแพร่ฉากได้เนื่องจากมีปัญหาโครงสร้างนิยาย",
					"can_publish": false,
					"issues":      val.Issues,
				})
				return
			}
		}

		if err := sceneService.UpdateScene(scene); err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if err := notificationService.NotifySceneUpdated(scene.NovelID, sceneID); err != nil {
			log.Printf("NotifySceneUpdated failed: %v", err)
			writeInternalError(w, err)
			return
		}

		var choicesDiff *models.ChoiceDiff
		if req.Choices != nil {
			diff, err := sceneService.SyncSceneChoices(sceneID, req.Choices)
			if err != nil {
				writeChoiceServiceError(w, err)
				return
			}
			choicesDiff = diff
		}
		updateMeta := map[string]interface{}{"novel_id": scene.NovelID, "chapter_id": scene.ChapterID}
		// เก็บ diff เฉพาะ field ที่เปลี่ยนจริง (title/type/status) เทียบ existingScene (ค่าก่อนแก้) กับ scene (ค่าหลังแก้)
		if existingScene.Title != scene.Title {
			updateMeta["old_title"] = existingScene.Title
			updateMeta["new_title"] = scene.Title
		}
		if existingScene.Type != scene.Type {
			updateMeta["old_type"] = existingScene.Type
			updateMeta["new_type"] = scene.Type
		}
		if existingScene.Status != scene.Status {
			updateMeta["old_status"] = existingScene.Status
			updateMeta["new_status"] = scene.Status
		}
		if choicesDiff != nil && (choicesDiff.CreatedCount > 0 || choicesDiff.UpdatedCount > 0 || choicesDiff.DeletedCount > 0) {
			updateMeta["choices_diff"] = map[string]interface{}{
				"created_count": choicesDiff.CreatedCount,
				"updated_count": choicesDiff.UpdatedCount,
				"deleted_count": choicesDiff.DeletedCount,
			}
		}
		if ch, err := chapterService.GetChapterByID(scene.ChapterID); err == nil && ch != nil {
			updateMeta["chapter_title"] = ch.Title
		}
		if nov, err := novelService.GetNovelDetail(scene.NovelID); err == nil && nov != nil {
			if n, ok := nov.(*models.Novel); ok && n != nil {
				updateMeta["novel_title"] = n.Title
			}
		}

		action := "UPDATE_SCENE"
		if strings.EqualFold(existingScene.Status, "draft") && strings.EqualFold(scene.Status, "published") {
			action = "PUBLISH_SCENE"
		} else if strings.EqualFold(existingScene.Status, "published") && strings.EqualFold(scene.Status, "draft") {
			action = "UNPUBLISH_SCENE"
		}

		recordAudit(r, auditService, service.AuditEvent{Action: action, TargetType: "scene", TargetID: int64Pointer(sceneID), Status: "SUCCESS", Metadata: updateMeta})

		responsePayload := map[string]any{
			"message": "scene updated",
		}
		if validationResult != nil && len(validationResult.Issues) > 0 {
			responsePayload["issues"] = validationResult.Issues
		}

		WriteJSON(w, http.StatusOK, responsePayload)
	}
}

func DeleteSceneHandler(sceneService service.SceneService, chapterService service.ChapterService, novelService service.NovelService, writerService service.WriterService, auditService service.AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sceneID, err := extractIDFromPath(r.URL.Path, "/scenes/")
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid id parameter")
			return
		}
		existingScene, err := sceneService.GetScene(sceneID)
		if err != nil {
			WriteError(w, http.StatusNotFound, "scene not found")
			return
		}
		chapter, err := chapterService.GetChapterByID(existingScene.ChapterID)
		if err != nil || chapter == nil || chapter.NovelID != existingScene.NovelID {
			WriteError(w, http.StatusNotFound, "scene not found")
			return
		}
		if err := requireWriterOwnsNovel(r.Context(), writerService, novelService, existingScene.NovelID); err != nil {
			if writeOwnershipError(w, err) {
				return
			}
			writeInternalError(w, err)
			return
		}

		if err := sceneService.DeleteScene(sceneID); err != nil {
			// FR-STRUCT-08: ไม่อนุญาตให้ลบ Start Scene โดยตรง → ตอบ 409 Conflict
			// (สอดคล้องกับ DeleteChapter ที่ตอบ 409 สำหรับ repository.ErrChapterHasStartScene)
			// หมายเหตุ: scene service ใช้ errors.New แบบ inline ยังไม่มี sentinel จึงตรวจจับด้วยข้อความที่ตรงกัน
			if err.Error() == "Start scene cannot be deleted" {
				RespondWithError3(w, http.StatusConflict, err.Error())
				return
			}
			writeInternalError(w, err)
			return
		}
		delMeta := map[string]interface{}{"novel_id": existingScene.NovelID, "chapter_id": existingScene.ChapterID}
		if ch, err := chapterService.GetChapterByID(existingScene.ChapterID); err == nil && ch != nil {
			delMeta["chapter_title"] = ch.Title
		}
		if nov, err := novelService.GetNovelDetail(existingScene.NovelID); err == nil && nov != nil {
			if n, ok := nov.(*models.Novel); ok && n != nil {
				delMeta["novel_title"] = n.Title
			}
		}
		recordAudit(r, auditService, service.AuditEvent{Action: "DELETE_SCENE", TargetType: "scene", TargetID: int64Pointer(sceneID), Status: "SUCCESS", Metadata: delMeta})

		WriteJSON(w, http.StatusOK, map[string]any{"message": "scene deleted"})
	}
}

// ... GetSceneHandler คงเดิม ...
func GetSceneHandler(sceneService service.SceneService, chapterService service.ChapterService, novelService service.NovelService, writerService service.WriterService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sceneID, err := extractIDFromPath(r.URL.Path, "/scenes/")
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid id parameter")
			return
		}

		scene, err := sceneService.GetScene(sceneID)
		if err != nil {
			WriteError(w, http.StatusNotFound, "scene not found")
			return
		}

		// ดึง chapter เพื่อตรวจสอบ status
		chapter, err := chapterService.GetChapterByID(scene.ChapterID)
		if err != nil || chapter == nil {
			WriteError(w, http.StatusNotFound, "scene not found")
			return
		}

		// ดึง novel เพื่อตรวจสอบ is_published
		novelDetail, err := novelService.GetNovelDetail(scene.NovelID)
		if err != nil {
			WriteError(w, http.StatusNotFound, "scene not found")
			return
		}
		novelPtr, ok := novelDetail.(*models.Novel)
		if !ok || novelPtr == nil {
			WriteError(w, http.StatusInternalServerError, "failed to load novel details")
			return
		}

		isOwnerOrAdmin := false
		var currentWriterID int
		var ctxRole string
		ctxUserID, ok := middleware.GetUserIDFromContext(r.Context())
		if ok && ctxUserID != 0 {
			if role, roleOk := middleware.GetRoleFromContext(r.Context()); roleOk && role == "admin" {
				isOwnerOrAdmin = true
				ctxRole = role
			} else if writerService != nil {
				writer, wErr := writerService.GetWriterByUserID(int(ctxUserID))
				if wErr == nil && writer != nil {
					currentWriterID = writer.WriterID
					if writer.WriterID == novelPtr.AuthorID {
						isOwnerOrAdmin = true
					}
				}
			}
		}

		log.Printf("[DEBUG GetSceneHandler] sceneID=%d sceneStatus=%s chapterID=%d chapterStatus=%s novelID=%d novelAuthorID=%d userID=%d writerID=%d role=%s isOwnerOrAdmin=%v novelIsPublished=%v",
			sceneID, scene.Status, scene.ChapterID, chapter.Status, scene.NovelID, novelPtr.AuthorID, ctxUserID, currentWriterID, ctxRole, isOwnerOrAdmin, novelPtr.IsPublished)

		if !isOwnerOrAdmin {
			// ตรวจ 3 ระดับ: novel published + chapter published + scene published
			if !novelPtr.IsPublished || chapter.Status != "published" || scene.Status != "published" {
				log.Printf("[DEBUG GetSceneHandler REJECT 404] sceneID=%d isOwnerOrAdmin=false novelIsPublished=%v chapterStatus=%s sceneStatus=%s",
					sceneID, novelPtr.IsPublished, chapter.Status, scene.Status)
				WriteError(w, http.StatusNotFound, "scene not found")
				return
			}
		}

		WriteJSON(w, http.StatusOK, scene)
	}
}
