package handlers

import (
	"log"
	"net/http"
	"novel-be/internal/middleware"
	"novel-be/internal/service"
	"strconv"
)

// POST /upload/image - Upload image to MinIO and Update Database
//
// Security: เมื่อ client ส่ง novel_id มาด้วย จะต้องผ่านการตรวจสอบ ownership
// (เจ้าของนิยายที่เป็น writer หรือ admin) ก่อนเสมอ — ห้ามเชื่อ novel_id จาก client
// เพียงอย่างเดียว (ป้องกันการแก้ปกนิยายของคนอื่น / IDOR)
func UploadImageHandler(mediaService service.MediaService, novelService service.NovelService, writerService service.WriterService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			RespondWithError(w, http.StatusMethodNotAllowed, "method not allowed", "only POST is supported")
			return
		}

		if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
			RespondWithError(w, http.StatusBadRequest, "failed to parse form", "")
			return
		}

		novelIDStr := r.FormValue("novel_id")
		novelID := 0
		if novelIDStr != "" {
			parsedID, err := strconv.Atoi(novelIDStr)
			if err != nil || parsedID <= 0 {
				RespondWithError(w, http.StatusBadRequest, "invalid novel_id", "")
				return
			}
			novelID = parsedID

			// ตรวจสอบสิทธิ์ก่อน upload/อัปเดต DB ใด ๆ ทั้งสิ้น
			if !CheckIsOwnerOrAdmin(r, novelID, novelService, writerService) {
				if _, ok := middleware.GetUserIDFromContext(r.Context()); !ok {
					WriteError(w, http.StatusUnauthorized, "unauthorized")
					return
				}
				WriteError(w, http.StatusForbidden, "forbidden: คุณไม่มีสิทธิ์แก้ไขรูปปกของนิยายฉบับนี้")
				return
			}
		}

		file, handler, err := r.FormFile("image")
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "missing image file", "")
			return
		}
		defer file.Close()

		// ปกนิยาย: จำกัด 10MB ตาม VAL-08 (ชนิดไฟล์ JPEG/PNG/WebP ตรวจใน service ตาม VAL-10)
		url, err := mediaService.UploadImage(r.Context(), handler, service.NovelCoverMaxBytes)
		if err != nil {
			writeUploadError(w, err)
			return
		}

		dbPathToSave := url

		var dbStatus string = "not updated"

		if novelID != 0 {
			err = novelService.UpdateNovelCover(novelID, dbPathToSave)
			if err != nil {
				log.Printf("[ERROR] update novel cover failed: novelID=%d: %v", novelID, err)
				dbStatus = "failed to update database"
			} else {
				dbStatus = "successfully updated database"
			}
		}

		RespondWithCreated(w, "process completed", map[string]interface{}{
			"full_url":   url,
			"saved_path": dbPathToSave,
			"filename":   handler.Filename,
			"db_status":  dbStatus,
		})
	}
}
