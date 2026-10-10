package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"novel-be/config"
	"novel-be/internal/db"
	"novel-be/internal/middleware"
	"novel-be/internal/repository"
	"novel-be/internal/routes"
	"novel-be/internal/service"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {

	// -----------------------
	// 1. Load Config
	// -----------------------
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ load config fail: %v", err)
	}
	middleware.SetJWTSecret(cfg.JWTSecret)

	// Debug summary (ไม่โชว์ค่า secret แค่บอกว่ามีค่าหรือว่าง) — ใช้วินิจฉัยตอน deploy บน Railway
	mask := func(s string) string {
		if strings.TrimSpace(s) == "" {
			return "<EMPTY>"
		}
		return fmt.Sprintf("<set:%d chars>", len(strings.TrimSpace(s)))
	}
	fmt.Printf("🔧 config: PORT=%s DB=%s MINIO_ENDPOINT=%s REGION=%s PUBLIC=%s KEY=%s SECRET=%s SSL=%v ORIGINS=%s JWT=%s JWT_REFRESH=%s\n",
		cfg.AppPort, mask(cfg.GetConnectionString()[:min(len(cfg.GetConnectionString()), 24)]),
		mask(cfg.MinIOEndpoint), cfg.MinIORegion, mask(cfg.MinIOPublicEndpoint),
		mask(cfg.MinIOAccessKey), mask(cfg.MinIOSecretKey), cfg.MinIOUseSSL,
		mask(cfg.AllowedOrigins), mask(cfg.JWTSecret), mask(cfg.JWTRefreshSecret))

	// -----------------------
	// 2. Connect DB
	// -----------------------
	dbConn, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("❌ DB connect fail: %v", err)
	}
	defer dbConn.Close()

	if err := dbConn.Ping(); err != nil {
		log.Fatalf("❌ DB ping fail: %v", err)
	}
	fmt.Println("✅ DB Connected")

	// -----------------------
	// 3. Connect Supabase Storage (S3-compatible) via AWS SDK for Go v2
	// -----------------------
	// minio-go ใช้ไม่ได้กับ Supabase เพราะไม่รับ endpoint ที่มี path (/storage/v1/s3)
	// AWS SDK v2 รองรับ BaseEndpoint ที่มี path + UsePathStyle ได้ จึงเลิกใช้ minio-go แล้ว
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.MinIORegion),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.MinIOAccessKey, cfg.MinIOSecretKey, "")),
	)
	if err != nil {
		log.Fatalf("❌ AWS config load fail: %v", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.MinIOEndpoint) // https://<ref>.storage.supabase.co/storage/v1/s3
		o.UsePathStyle = true                          // path-style addressing (บังคับสำหรับ Supabase / MinIO-style S3)
	})

	// ทดสอบการเชื่อมต่อจริงด้วยการเรียก ListBuckets (ถ้า key ผิด/หมดอายุจะเจอ error ตรงนี้)
	if _, err := s3Client.ListBuckets(context.Background(), &s3.ListBucketsInput{}); err != nil {
		log.Fatalf("❌ Supabase Storage ping fail: %v", err)
	}
	fmt.Println("✅ Supabase Storage Connected")

	// -----------------------
	// 4. Repositories
	// -----------------------
	novelRepo := repository.NewNovelRepository(dbConn)
	sceneRepo := repository.NewSceneRepository(dbConn)
	chapterRepo := repository.NewChapterRepository(dbConn)
	socialRepo := repository.NewSocialRepository(dbConn)
	readingRepo := repository.NewReadingRepository(dbConn)
	mediaRepo := repository.NewS3MediaRepository(s3Client, cfg.GetMinIOPublicBaseURL())
	categoryRepo := repository.NewCategoryRepository(dbConn)
	authRepo := repository.NewAuthRepository(dbConn)
	writerRepo := repository.NewWriterRepository(dbConn) // 👈 ผูกเชื่อมตารางสมัครนักเขียนเข้าฐานข้อมูลจริง
	reportRepo := repository.NewReportRepository(dbConn)
	auditRepo := repository.NewAuditRepository(dbConn)
	dashboardRepo := repository.NewDashboardRepository(dbConn)

	ctx := context.Background()
	if err := auditRepo.EnsureIndexes(ctx); err != nil {
		log.Printf("⚠️ warning: failed to ensure audit log indexes: %v", err)
	} else {
		fmt.Println("✅ Audit Log Indexes Ready")
	}

	// Ensure MinIO bucket exists (Supabase Storage bucket: image)
	if err := mediaRepo.EnsureBucketExists(ctx, service.NovelImagesBucket); err != nil {
		log.Fatalf("❌ failed to ensure MinIO bucket: %v", err)
	}
	fmt.Println("✅ MinIO Bucket Ready")

	// -----------------------
	// 5. Services
	// -----------------------
	// 🟢 ย้าย mediaService ขึ้นมาสร้างก่อน เพราะ novelService ต้องใช้งาน
	mediaService := service.NewMediaService(mediaRepo)

	// 🟢 ตอนนี้ส่ง mediaService เข้าไปได้แล้ว
	novelService := service.NewNovelService(novelRepo, mediaService, categoryRepo)

	sceneService := service.NewSceneService(sceneRepo, dbConn)
	chapterService := service.NewChapterService(chapterRepo)
	socialService := service.NewSocialService(socialRepo)
	readingService := service.NewReadingService(readingRepo)
	flowService := service.NewFlowService(sceneService)
	categoryService := service.NewCategoryService(categoryRepo)

	// 🟢 สร้างบริการระบบคำขอสมัครนักเขียนแบบสิทธิ์รออนุมัติ
	writerService := service.NewWriterServiceDirect(writerRepo)

	// 🟢 สร้างบริการระบบ Authentication สมาชิกของแท้
	authService := service.NewAuthService(authRepo, cfg.JWTSecret, cfg.JWTRefreshSecret)
	notificationService := service.NewNotificationService(dbConn)
	reportService := service.NewReportService(reportRepo)
	auditService := service.NewAuditService(auditRepo)
	dashboardService := service.NewDashboardService(dashboardRepo)

	// 🟢 ผูก audit service เข้ากับ middleware แบบ decoupled (ผ่าน function type ไม่ import service เข้า middleware ตรงๆ
	// เพราะ audit_service.go import middleware อยู่แล้ว ถ้า middleware import service กลับไปด้วยจะเกิด import cycle)
	// ทำให้ RequireAuth/RequireRole ใน auth_middleware.go บันทึก audit log ได้ตอน token หาย/ผิด/หมดอายุ หรือ role ไม่ตรง
	middleware.SetUnauthorizedRecorder(func(r *http.Request, reason, role string, uid uint, hasUID bool) {
		var actor *uint
		if hasUID {
			actor = &uid
		}
		if err := service.RecordWithBackendActor(auditService, r.Context(), actor, role, service.AuditEvent{
			Action:     "UNAUTHORIZED_ACCESS",
			TargetType: "route",
			Status:     "FAILURE",
			Metadata: map[string]interface{}{
				"path":   r.URL.Path,
				"method": r.Method,
				"reason": reason, // "missing_token" | "invalid_or_expired_token" | "role_mismatch"
			},
		}); err != nil {
			log.Printf("audit log (unauthorized access) write failed: %v", err)
		}
	})

	// 🟢 Analytics Service
	analyticsRepo := repository.NewAnalyticsRepository(dbConn)
	analyticsService := service.NewAnalyticsService(analyticsRepo)

	// สร้าง ServeMux ใหม่
	mux := http.NewServeMux()

	// -----------------------
	// 6. Routes
	// -----------------------
	routes.RegisterRoutes(
		mux,
		flowService,
		novelService,
		chapterService,
		sceneService,
		socialService,
		readingService,
		writerService, // 👈 ส่งมอบบริการคำขอสมัครนักเขียนเข้าสู่กลุ่มเส้นทาง API
		mediaService,
		categoryService,
		*authService, // 👈 ส่งบริการ Authen เข้าพ่วงท้ายแถวโดยใช้ * แกะ Pointer ออกตามโครงสร้างเดิม
		notificationService,
		reportService,
		analyticsService,
		auditService,
		dashboardService,
	)

	// Health check endpoint สำหรับ Railway / uptime monitor
	// mux จับคู่ pattern ที่เฉพาะเจาะจงกว่า "/" ก่อน จึงไม่ทับ route เดิม (แม้จะมี register ตัวที่ขึ้นต้นด้วย / อยู่)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// -----------------------
	// 7. Start Server
	// -----------------------
	fmt.Printf("🚀 Server running on port %s\n", cfg.AppPort)
	fmt.Println("📚 Novel Interactive Platform Backend Ready!")

	handler := middleware.CORSMiddleware(mux) // 👈 ใช้ mux ที่เราสร้าง
	err = http.ListenAndServe(":"+cfg.AppPort, handler)
	if err != nil {
		log.Fatalf("❌ server start fail: %v", err)
	}
}
