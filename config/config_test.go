package config

import (
	"os"
	"strings"
	"testing"
)

// TestLoadConfig_RailwayTabs จำลองเคส Railway Raw editor ที่มี tab ติดในชื่อตัวแปร/ค่า
// เพื่อยืนยันว่า lookup ยังอ่านได้ถูกต้อง (v14 fix)
func TestLoadConfig_RailwayTabs(t *testing.T) {
	t.Setenv("PORT", "\t9000")
	t.Setenv("DATABASE_URL", "postgresql://u:p@pooler.supabase.com:5432/postgres")
	t.Setenv("MINIO_ENDPOINT", "\thttps://x.storage.supabase.co/storage/v1/s3")
	t.Setenv("MINIO_PUBLIC_ENDPOINT", "https://x.storage.supabase.co/storage/v1/object/public\t")
	t.Setenv("MINIO_ACCESS_KEY", "  access123  ")
	t.Setenv("MINIO_SECRET_KEY", "secret456")
	t.Setenv("MINIO_USE_SSL", "true")
	t.Setenv("ALLOWED_ORIGINS", "https://deploynovels.vercel.app")
	t.Setenv("JWT_SECRET", "\tXxZQRyyaLOyyQMA/60zX+TLmvN0EreoXsqKAxpq0LT8=")
	t.Setenv("JWT_REFRESH_SECRET", "\txOqQrRKBh/T3VAMUnU2qHde+REbN6ZVF6kunMX2oj8A=")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.AppPort != "9000" {
		t.Errorf("AppPort = %q, want 9000", cfg.AppPort)
	}
	if cfg.MinIOEndpoint != "https://x.storage.supabase.co/storage/v1/s3" {
		t.Errorf("MinIOEndpoint = %q (มี whitespace ติด)", cfg.MinIOEndpoint)
	}
	if cfg.MinIOPublicEndpoint != "https://x.storage.supabase.co/storage/v1/object/public" {
		t.Errorf("MinIOPublicEndpoint = %q (มี whitespace ติด)", cfg.MinIOPublicEndpoint)
	}
	if cfg.MinIOAccessKey != "access123" {
		t.Errorf("MinIOAccessKey = %q, want access123", cfg.MinIOAccessKey)
	}
	if cfg.JWTSecret != "XxZQRyyaLOyyQMA/60zX+TLmvN0EreoXsqKAxpq0LT8=" {
		t.Errorf("JWTSecret = %q (ยังมี tab)", cfg.JWTSecret)
	}
	if cfg.JWTRefreshSecret != "xOqQrRKBh/T3VAMUnU2qHde+REbN6ZVF6kunMX2oj8A=" {
		t.Errorf("JWTRefreshSecret = %q (ยังมี tab)", cfg.JWTRefreshSecret)
	}
	if cfg.DatabaseURL != "postgresql://u:p@pooler.supabase.com:5432/postgres" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if !cfg.MinIOUseSSL {
		t.Error("MinIOUseSSL = false, want true")
	}
	if got := cfg.GetMinIOPublicBaseURL(); got != "https://x.storage.supabase.co/storage/v1/object/public" {
		t.Errorf("GetMinIOPublicBaseURL = %q", got)
	}
}

// TestLoadConfig_MissingJWT ต้อง error ชัดเจนเมื่อไม่มี JWT secret
func TestLoadConfig_MissingJWT(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("JWT_REFRESH_SECRET")

	if _, err := LoadConfig(); err == nil {
		t.Error("want error when JWT secrets missing, got nil")
	} else if !strings.Contains(err.Error(), "JWT") {
		t.Errorf("error should mention JWT, got: %v", err)
	}
}