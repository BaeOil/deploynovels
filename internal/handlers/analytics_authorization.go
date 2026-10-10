package handlers

import (
	"net/http"

	"novel-be/internal/middleware"
	"novel-be/internal/models"
	"novel-be/internal/service"
)

func authorizeAnalyticsAccess(r *http.Request, novelID int, novelSvc service.NovelService, writerSvc service.WriterService) (int, string) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		return http.StatusUnauthorized, "unauthorized: ไม่พบข้อมูลสิทธิ์ผู้ใช้งาน"
	}

	role, _ := middleware.GetRoleFromContext(r.Context())
	if role == "admin" {
		return 0, ""
	}

	writer, err := writerSvc.GetWriterByUserID(int(userID))
	if err != nil || writer == nil {
		return http.StatusForbidden, "forbidden: คุณไม่มีสิทธิ์ดูสถิตินิยายนี้"
	}

	novelDetail, err := novelSvc.GetNovelDetail(novelID)
	if err != nil {
		return http.StatusNotFound, "novel not found"
	}
	novel, ok := novelDetail.(*models.Novel)
	if !ok || novel == nil {
		return http.StatusNotFound, "novel not found"
	}
	if novel.AuthorID != writer.WriterID {
		return http.StatusForbidden, "forbidden: คุณไม่ใช่เจ้าของนิยายนี้"
	}
	return 0, ""
}
