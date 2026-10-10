package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// LoggingMiddleware logs HTTP requests with method, path, and duration
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Flush() {
	if flusher, ok := lrw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lrw := &loggingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)
		statusEmoji := "✅"
		if lrw.statusCode >= 400 {
			statusEmoji = "❌"
		}

		fmt.Printf("%s [%s] %s %s - Status: %d - Duration: %v\n",
			statusEmoji,
			r.Method,
			r.RequestURI,
			r.RemoteAddr,
			lrw.statusCode,
			duration,
		)
	})
}

// ค่า default ถ้าไม่ได้ตั้ง ALLOWED_ORIGINS ไว้ใน env เลย (กันไม่ให้ local dev พังถ้าลืมตั้งค่า)
var defaultAllowedOrigins = []string{
	"http://localhost:5173",
	"http://127.0.0.1:5173",
}

// isOriginAllowed เช็คว่า origin ที่ request เข้ามาอยู่ใน whitelist หรือไม่
// รองรับ wildcard แบบ "*.vercel.app" สำหรับ preview deployment ของ Vercel ที่โดเมนสุ่มทุกครั้ง
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range allowedOrigins {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}
		if allowed == origin {
			return true
		}
		if strings.HasPrefix(allowed, "*.") {
			suffix := strings.TrimPrefix(allowed, "*")
			if strings.HasSuffix(origin, suffix) {
				return true
			}
		}
	}
	return false
}

func getAllowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if raw == "" {
		return defaultAllowedOrigins
	}
	origins := strings.Split(raw, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	return origins
}

func CORSMiddleware(next http.Handler) http.Handler {
	allowedOrigins := getAllowedOrigins()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// ตอบ Allow-Origin เฉพาะตอนที่ origin นี้อยู่ใน whitelist จริงๆ เท่านั้น
		// (ห้ามตอบค่า hardcode กลับไปเหมือนโค้ดเดิม เพราะ browser จะเทียบกับ origin จริง ถ้าไม่ตรงจะบล็อกอยู่ดี
		//  และถ้าไม่ตรงแล้วยังตอบไปแบบผิดๆ จะดูเหมือนใช้งานได้แต่จริงๆ ถูกบล็อกเงียบๆ ทำให้ debug ยาก)
		if isOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// WrapHandlerWithLogging wraps a single handler with logging
func WrapHandlerWithLogging(handler http.HandlerFunc) http.HandlerFunc {
	return RequestLogger(handler).ServeHTTP
}
