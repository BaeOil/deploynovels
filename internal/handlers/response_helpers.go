package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"novel-be/internal/dto"
	"novel-be/internal/service"
)

// --- 1. ฟังก์ชันมาตรฐาน (New Standard) ---

// RespondWithJSON สำหรับการส่งข้อมูลกลับแบบปกติ
func RespondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := dto.SuccessResponse{
		Status:  statusCode,
		Data:    data,
		Message: "success",
	}
	json.NewEncoder(w).Encode(response)
}

// RespondWithError สำหรับการแจ้ง Error แบบละเอียด (4 parameters)
func RespondWithError(w http.ResponseWriter, statusCode int, message string, err string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := dto.ErrorResponse{
		Error: dto.ErrorDetail{
			Code:    errorCode(statusCode, err),
			Message: message,
		},
	}
	json.NewEncoder(w).Encode(response)
}

func errorCode(statusCode int, explicitCode string) string {
	if explicitCode != "" {
		return explicitCode
	}

	switch statusCode {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusMethodNotAllowed:
		return "METHOD_NOT_ALLOWED"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	default:
		return "INTERNAL_ERROR"
	}
}

// --- 2. ฟังก์ชันตัวเชื่อม (Adapters) เพื่อให้ Compile ผ่านทุกไฟล์ ---

// RespondWithCreated แก้ปัญหาที่นายเจอใน get_handlers.go:222
// โดยรับ String (Message) และ Data (interface{}) ตามที่ Handler เรียกมา
func RespondWithCreated(w http.ResponseWriter, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := dto.SuccessResponse{
		Status:  http.StatusCreated,
		Data:    data,
		Message: message,
	}
	json.NewEncoder(w).Encode(response)
}

// --- 3. ตัวช่วยสำหรับการจัดการ Error ภายในระบบ (Internal Error Handling) ---

// writeInternalError บันทึก error จริงลง server log เพื่อการ debug
// และตอบกลับเป็นข้อความ generic ไปยัง client เพื่อไม่ให้เปิดเผย
// รายละเอียดภายใน (SQL error, path, endpoint ฯลฯ)
func writeInternalError(w http.ResponseWriter, err error) {
	log.Printf("[ERROR] internal error: %v", err)
	WriteError(w, http.StatusInternalServerError, "internal server error")
}

// respondInternal ใช้กับจุดที่ RespondWithError (4 พารามิเตอร์) รูปแบบเดิม
// โดยยังคง message ที่ handler กำหนดไว้ และซ่อนรายละเอียด error จริงไว้ใน log
func respondInternal(w http.ResponseWriter, message string, err error) {
	log.Printf("[ERROR] internal error: %v", err)
	RespondWithError(w, http.StatusInternalServerError, message, "")
}

// writeUploadError แยก validation error ของไฟล์ (ตอบ 400 ได้เพราะไม่ใช่ข้อมูลภายใน)
// ออกจาก error ภายในระบบ storage (ตอบ 500 แบบ generic + log ฝั่ง server)
func writeUploadError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrImageTooLarge) || errors.Is(err, service.ErrInvalidImageType) {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeInternalError(w, err)
}

// WriteJSON เชื่อมโยงโค้ดเก่าที่เรียกใช้ WriteJSON ให้มาใช้มาตรฐานใหม่
func WriteJSON(w http.ResponseWriter, status int, payload interface{}) {
	RespondWithJSON(w, status, payload)
}

// WriteError เชื่อมโยงโค้ดเก่าที่เรียกใช้ WriteError ให้มาใช้มาตรฐานใหม่
func WriteError(w http.ResponseWriter, status int, message string) {
	RespondWithError(w, status, message, "")
}

// RespondWithError3 (Optional) สำหรับกรณีที่มีการเรียก RespondWithError แบบ 3 ตัวแปร
func RespondWithError3(w http.ResponseWriter, statusCode int, message string) {
	RespondWithError(w, statusCode, message, "")
}
