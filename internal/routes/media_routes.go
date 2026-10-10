package routes

import (
	"net/http"

	"novel-be/internal/handlers"
	"novel-be/internal/middleware"
	"novel-be/internal/service"
)

func registerMediaRoutes(mux *http.ServeMux, media service.MediaService, novel service.NovelService, writer service.WriterService) {
	mux.Handle("/upload/image", middleware.RequestLogger(middleware.RequireAuth(handlers.UploadImageHandler(media, novel, writer))))
}
