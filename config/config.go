package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	AppPort             string
	DatabaseURL         string // Supabase pooler connection string (ถ้ากำหนดให้ใช้แทน field เดิม)
	DatabaseHost        string
	DatabasePort        int
	DatabaseUser        string
	DatabasePassword    string
	DatabaseName        string
	DatabaseSSLMode     string
	MinIOEndpoint       string
	MinIORegion         string
	MinIOPublicEndpoint string
	MinIOAccessKey      string
	MinIOSecretKey      string
	MinIOUseSSL         bool
	AllowedOrigins      string
	JWTSecret           string
	JWTRefreshSecret    string
}

func LoadConfig() (Config, error) {
	// อ่านไฟล์ .env สำหรับ local dev ถ้ามี (บน Railway/Docker ไม่มีไฟล์นี้ จะอ่านจาก env จริงแทน)
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	_ = viper.ReadInConfig() // ใช้ _ รับ Error ไว้ เพราะถ้ารันใน container อาจไม่มีไฟล์ .env

	// lookup อ่านค่าจาก environment variable ของ container โดยตรง (Railway/Docker)
	// ถ้าไม่พบจึง fallback ไป .env ที่ viper โหลดไว้ (local dev) แล้วถึงค่า default
	//
	// ⚠️ ใช้ flat key เช่น "JWT_SECRET" ไม่ใช้ dotted key "JWT.SECRET"
	//    เพราะ viper.GetString("JWT.SECRET") จะไม่ไปอ่าน env var JWT_SECRET จาก container
	//    (จุด . ทำให้ viper ตีความว่าเป็น nested key) — นี่คือสาเหตุที่ deploy แล้วค่า JWT ว่าง
	//
	// ⚠️ กันเคส Railway Raw editor มี tab/space ติด: สแกน env ทั้งหมดแล้ว TrimSpace
	//    ทั้งชื่อตัวแปรและค่า (เช่น key กลายเป็น "\tJWT_SECRET" หรือค่าเป็น "\thttps://...")
	envMap := make(map[string]string, 64)
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i > 0 {
			key := strings.TrimSpace(kv[:i])
			if key == "" {
				continue
			}
			if _, dup := envMap[key]; !dup {
				envMap[key] = strings.TrimSpace(kv[i+1:])
			}
		}
	}

	lookup := func(key string, def ...string) string {
		if v := envMap[key]; v != "" {
			return v
		}
		if v := strings.TrimSpace(viper.GetString(key)); v != "" {
			return v
		}
		if len(def) > 0 {
			return def[0]
		}
		return ""
	}

	// Port: Railway inject PORT มาเอง ไม่งั้น fallback APP_PORT แล้ว 8080
	appPort := lookup("PORT")
	if appPort == "" {
		appPort = lookup("APP_PORT", "8080")
	}

	jwtSecret := lookup("JWT_SECRET")
	jwtRefreshSecret := lookup("JWT_REFRESH_SECRET")
	if jwtSecret == "" || jwtRefreshSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET and JWT_REFRESH_SECRET must be set")
	}

	databasePort := 5432
	if p := lookup("POSTGRES_PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			databasePort = parsed
		}
	}

	config := Config{
		AppPort:             appPort,
		DatabaseURL:         lookup("DATABASE_URL"),
		DatabaseHost:        lookup("POSTGRES_HOST", "localhost"),
		DatabasePort:        databasePort,
		DatabaseUser:        lookup("POSTGRES_USER", "postgres"),
		DatabasePassword:    lookup("POSTGRES_PASSWORD"),
		DatabaseName:        lookup("POSTGRES_DBNAME", "novel"),
		DatabaseSSLMode:     lookup("POSTGRES_SSLMODE", "disable"),
		MinIOEndpoint:       lookup("MINIO_ENDPOINT", "minio:9000"),
		MinIORegion:         lookup("MINIO_REGION", "ap-northeast-2"),
		MinIOPublicEndpoint: lookup("MINIO_PUBLIC_ENDPOINT", "http://localhost:9000"),
		MinIOAccessKey:      lookup("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:      lookup("MINIO_SECRET_KEY", "minioadmin"),
		MinIOUseSSL:         strings.EqualFold(lookup("MINIO_USE_SSL"), "true"),
		AllowedOrigins:      lookup("ALLOWED_ORIGINS"),
		JWTSecret:           jwtSecret,
		JWTRefreshSecret:    jwtRefreshSecret,
	}

	return config, nil
}

// GetConnectionString คืน connection string สำหรับ database/sql
// ถ้ามี DATABASE_URL ถูกกำหนดไว้ (เช่น Supabase pooler) ให้ใช้ค่านั้นโดยตรง
// มิฉะนั้นสร้าง connection string จาก field แยก (สำหรับ local dev)
func (c *Config) GetConnectionString() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		c.DatabaseHost,
		c.DatabasePort,
		c.DatabaseUser,
		c.DatabasePassword,
		c.DatabaseName,
		c.DatabaseSSLMode)
}

// GetMinIOPublicBaseURL คืน public base URL สำหรับสร้าง URL ที่ browser เข้าถึงได้
// เช่น https://<ref>.storage.supabase.co/storage/v1/object/public
func (c *Config) GetMinIOPublicBaseURL() string {
	endpoint := strings.TrimSuffix(strings.TrimSpace(c.MinIOPublicEndpoint), "/")
	// ถ้า publicEndpoint ยังเป็น localhost ให้ fallback เป็น endpoint เดิม (local dev)
	if strings.HasPrefix(endpoint, "http://localhost") || strings.HasPrefix(endpoint, "http://minio") {
		return endpoint
	}
	return endpoint
}

// Validate ตรวจสอบว่า config จำเป็นครบสำหรับ production
func (c *Config) Validate() error {
	if c.JWTSecret == "" || c.JWTRefreshSecret == "" {
		return fmt.Errorf("JWT secrets must be set")
	}
	return nil
}
