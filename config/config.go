package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	AppPort             string
	DatabaseURL         string // Supabase pooler connection string (มีประโยชน์ถ้ากำหนดให้ใช้แทน field เดิม)
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
	// ✅ 1. สั่งให้ Viper ลองหาไฟล์ .env และอ่านค่ามา (ถ้ามี)
	viper.SetConfigFile(".env")
	_ = viper.ReadInConfig() // ใช้ _ รับ Error ไว้ เพราะถ้ารันใน Docker อาจจะไม่มีไฟล์นี้ ก็ปล่อยผ่านได้

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// ✅ 2. เพิ่ม Default ให้ App Port ป้องกันเซิร์ฟเวอร์บึ้ม
	// ถ้า Railway inject PORT env มา (default behavior) ให้ใช้ค่านั้นก่อน
	// มิฉะนั้น fallback เป็น APP_PORT หรือ 8080
	appPort := strings.TrimSpace(os.Getenv("PORT"))
	if appPort == "" {
		appPort = strings.TrimSpace(viper.GetString("APP.PORT"))
		if appPort == "" {
			appPort = "8080"
		}
	}
	viper.Set("APP.PORT", appPort)

	viper.SetDefault("POSTGRES.HOST", "localhost")
	viper.SetDefault("POSTGRES.PORT", 5432)
	viper.SetDefault("POSTGRES.USER", "postgres")
	viper.SetDefault("POSTGRES.PASSWORD", "")
	viper.SetDefault("POSTGRES.DBNAME", "novel")
	viper.SetDefault("POSTGRES.SSLMODE", "disable")

	// MinIO / Supabase Storage defaults
	viper.SetDefault("MINIO.ENDPOINT", "minio:9000")
	viper.SetDefault("MINIO.REGION", "ap-northeast-2")
	viper.SetDefault("MINIO.PUBLIC_ENDPOINT", "http://localhost:9000")
	viper.SetDefault("MINIO.ACCESS_KEY", "minioadmin")
	viper.SetDefault("MINIO.SECRET_KEY", "minioadmin")
	viper.SetDefault("MINIO.USE_SSL", false)

	jwtSecret := strings.TrimSpace(viper.GetString("JWT.SECRET"))
	jwtRefreshSecret := strings.TrimSpace(viper.GetString("JWT.REFRESH_SECRET"))
	if jwtSecret == "" || jwtRefreshSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET and JWT_REFRESH_SECRET must be set")
	}

	config := Config{
		AppPort:             viper.GetString("APP.PORT"),
		DatabaseURL:         strings.TrimSpace(viper.GetString("DATABASE_URL")),
		DatabaseHost:        viper.GetString("POSTGRES.HOST"),
		DatabasePort:        viper.GetInt("POSTGRES.PORT"),
		DatabaseUser:        viper.GetString("POSTGRES.USER"),
		DatabasePassword:    viper.GetString("POSTGRES.PASSWORD"),
		DatabaseName:        viper.GetString("POSTGRES.DBNAME"),
		DatabaseSSLMode:     viper.GetString("POSTGRES.SSLMODE"),
		MinIOEndpoint:       viper.GetString("MINIO.ENDPOINT"),
		MinIORegion:         viper.GetString("MINIO.REGION"),
		MinIOPublicEndpoint: strings.TrimSpace(viper.GetString("MINIO.PUBLIC_ENDPOINT")),
		MinIOAccessKey:      viper.GetString("MINIO.ACCESS_KEY"),
		MinIOSecretKey:      viper.GetString("MINIO.SECRET_KEY"),
		MinIOUseSSL:         viper.GetBool("MINIO.USE_SSL"),
		AllowedOrigins:      strings.TrimSpace(viper.GetString("ALLOWED_ORIGINS")),
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
	endpoint := strings.TrimSuffix(c.MinIOPublicEndpoint, "/")
	// ถ้า publicEndpoint ยังเป็น localhost ให้ fallback เป็น endpoint เดิม (local dev)
	if strings.HasPrefix(endpoint, "http://localhost") || strings.HasPrefix(endpoint, "http://minio") {
		return endpoint
	}
	return endpoint
}

// ตรวจสอบว่า config ถูกต้องสำหรับ production (Optional)
func (c *Config) Validate() error {
	if c.JWTSecret == "" || c.JWTRefreshSecret == "" {
		return fmt.Errorf("JWT secrets must be set")
	}
	return nil
}
