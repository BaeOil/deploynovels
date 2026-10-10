package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"novel-be/internal/dto"
	"novel-be/internal/middleware"
	"novel-be/internal/repository"
	"novel-be/internal/service"
)

type AuthHandler struct {
	authService  *service.AuthService
	mediaService service.MediaService
	auditService service.AuditService
}

func NewAuthHandler(as *service.AuthService, ms service.MediaService, auditService service.AuditService) *AuthHandler {
	return &AuthHandler{authService: as, mediaService: ms, auditService: auditService}
}

// 📝 1. ท่อสมัครสมาชิก (Register) - รับ Multipart Form เผื่อการอัปโหลดรูปภาพ
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// จำกัดขนาดไฟล์รูปโปรไฟล์รวมไม่เกิน 5MB
	err := r.ParseMultipartForm(5 << 20)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "รูปภาพขนาดใหญ่เกินไป (จำกัด 5MB)")
		return
	}

	// แกะข้อมูลตัวอักษรจาก Form-Data เข้าสู่ DTO
	req := dto.RegisterRequest{
		Username: r.FormValue("username"),
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
	}

	if err := req.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	avatarURL := ""
	file, handler, err := r.FormFile("profileImage")
	if err != nil && err != http.ErrMissingFile {
		WriteError(w, http.StatusBadRequest, "ไม่สามารถอ่านไฟล์รูปภาพได้")
		return
	}
	if err == nil {
		defer file.Close()

		// รูปโปรไฟล์ตอนสมัครสมาชิก: จำกัด 5MB ตาม VAL-09
		uploadedURL, uploadErr := h.mediaService.UploadImage(r.Context(), handler, service.ProfilePictureMaxBytes)
		if uploadErr != nil {
			writeUploadError(w, uploadErr)
			return
		}
		avatarURL = uploadedURL
	}

	res, err := h.authService.Register(r.Context(), req, avatarURL)
	if err != nil {
		if err.Error() == "username already in use" || err.Error() == "email already in use" {
			WriteError(w, http.StatusConflict, err.Error())
			return
		}
		writeInternalError(w, err)
		return
	}
	recordAuditWithBackendActor(r, h.auditService, &res.User.ID, res.User.Role, service.AuditEvent{Action: "REGISTER", TargetType: "user", TargetID: int64Pointer(int(res.User.ID)), Status: "SUCCESS", Metadata: map[string]interface{}{"email": res.User.Email, "username": res.User.Username}})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

// 🔑 2. ท่อเข้าสู่ระบบ (Login) - รับข้อมูลรูปแบบ JSON
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req dto.LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "รูปแบบข้อมูลไม่ถูกต้อง")
		return
	}
	if req.Email == "" || req.Password == "" {
		WriteError(w, http.StatusBadRequest, "กรุณากรอกอีเมลและรหัสผ่าน")
		return
	}

	// เรียกใช้งาน Service ตัวจริง เพื่อเช็คข้อมูลใน DB และออก JWT Token ตัวจริง
	res, err := h.authService.Login(r.Context(), req)
	if err != nil {
		recordAuditWithBackendActor(r, h.auditService, nil, "", service.AuditEvent{
			Action:     "LOGIN_FAILED",
			TargetType: "user",
			TargetID:   nil,
			Status:     "FAILURE",
			Metadata: map[string]interface{}{
				"email":  req.Email,
				"reason": err.Error(),
			},
		})
		// หากรหัสผิดหรือหาไม่เจอ จะเด้งข้อความแจ้งเตือนสีแดงออกไป
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}
	recordAuditWithBackendActor(r, h.auditService, &res.User.ID, res.User.Role, service.AuditEvent{Action: "LOGIN", TargetType: "user", TargetID: int64Pointer(int(res.User.ID)), Status: "SUCCESS", Metadata: map[string]interface{}{"email": res.User.Email}})

	// พ่นข้อมูลตั๋วพร้อมรายละเอียดประวัติผู้ใช้กลับไปให้หน้าบ้านจัดเก็บลง LocalStorage/Cookie
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

// 🔄 3. ท่อรีเฟรชโทเค็น (Refresh Token)
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req dto.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "รูปแบบข้อมูลไม่ถูกต้อง")
		return
	}

	res, err := h.authService.RefreshToken(r.Context(), req)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

// 🔓 4. ท่อออกจากระบบ (Logout) - ปิดเซสชันฝั่ง frontend
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if userID, ok := middleware.GetUserIDFromContext(r.Context()); ok && userID != 0 {
		if user, err := h.authService.GetUserByID(r.Context(), userID); err == nil && user != nil {
			recordAudit(r, h.auditService, service.AuditEvent{Action: "LOGOUT", TargetType: "user", TargetID: int64Pointer(int(userID)), Status: "SUCCESS", Metadata: map[string]interface{}{"email": user.Email}})
		}
	}

	// ในระบบ JWT ปัจจุบัน เราใช้ token แบบ stateless จึงไม่มีการเก็บสถานะเซสชันใน server
	// endpoint นี้ออกแบบมาให้ frontend เรียกแล้วตอบว่าออกจากระบบสำเร็จ
	// ส่วนการล้าง token จะต้องทำที่ฝั่ง client (เช่น localStorage.removeItem('token'))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "ออกจากระบบเรียบร้อยแล้ว",
	})
}

// 👤 5. ท่อดึงข้อมูลผู้ใช้ปัจจุบัน (Get Current User) - ต้องมี Token ที่ถูกต้อง
func (h *AuthHandler) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// ดึง user_id จาก Context (ถูกใส่โดย RequireAuth middleware)
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		WriteError(w, http.StatusUnauthorized, "ไม่สามารถตรวจสอบตัวตนผู้ใช้ได้")
		return
	}

	// เรียก Service ตัวจริงเพื่อดึงข้อมูลผู้ใช้จากฐานข้อมูล
	user, err := h.authService.GetUserByID(r.Context(), userID)
	if err != nil {
		fmt.Printf("❌ ERROR in GetUserByID: %v\n", err)
		writeInternalError(w, err)
		return
	}
	if user == nil {
		fmt.Printf("❌ ERROR: User not found for ID: %d\n", uint(userID))
		WriteError(w, http.StatusNotFound, "ไม่พบข้อมูลผู้ใช้งาน")
		return
	}

	fmt.Printf("✅ DEBUG: User found - ID: %d, Username: %s, Email: %s\n", user.ID, user.Username, user.Email)

	// ตอบกลับข้อมูลผู้ใช้โดยไม่ให้ password hash
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": map[string]interface{}{
			"id":               user.ID,
			"username":         user.Username,
			"email":            user.Email,
			"pic_profile":      user.PicProfile,
			"role":             user.Role,
			"status":           user.Status,
			"created_at":       user.CreatedAt,
			"suspended_reason": user.SuspendedReason,
		},
	})
}

func (h *AuthHandler) UpdateOwnUsername(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.UpdateUsernameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "รูปแบบข้อมูลไม่ถูกต้อง")
		return
	}

	err := h.authService.UpdateUsername(r.Context(), userID, req.Username)
	if err != nil {
		recordAudit(r, h.auditService, service.AuditEvent{
			Action:     "UPDATE_PROFILE",
			TargetType: "user",
			TargetID:   int64Pointer(int(userID)),
			Status:     "FAILURE",
			Metadata:   map[string]interface{}{"field": "username", "reason": err.Error()},
		})
		if errors.Is(err, repository.ErrUsernameTaken) {
			WriteError(w, http.StatusConflict, err.Error())
			return
		}
		if err.Error() == "username is required" ||
			err.Error() == "username length must be between 3 and 50 characters" ||
			err.Error() == "username must contain only letters, numbers, and underscores" {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeInternalError(w, err)
		return
	}
	recordAudit(r, h.auditService, service.AuditEvent{Action: "UPDATE_PROFILE", TargetType: "user", TargetID: int64Pointer(int(userID)), Status: "SUCCESS", Metadata: map[string]interface{}{"field": "username"}})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "username updated successfully"})
}

func (h *AuthHandler) UpdateOwnEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.UpdateEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "รูปแบบข้อมูลไม่ถูกต้อง")
		return
	}

	if err := req.Validate(); err != nil {
		recordAudit(r, h.auditService, service.AuditEvent{
			Action:     "UPDATE_PROFILE",
			TargetType: "user",
			TargetID:   int64Pointer(int(userID)),
			Status:     "FAILURE",
			Metadata:   map[string]interface{}{"field": "email", "reason": err.Error()},
		})
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	err := h.authService.UpdateEmail(r.Context(), userID, req.Email)
	if err != nil {
		recordAudit(r, h.auditService, service.AuditEvent{
			Action:     "UPDATE_PROFILE",
			TargetType: "user",
			TargetID:   int64Pointer(int(userID)),
			Status:     "FAILURE",
			Metadata:   map[string]interface{}{"field": "email", "reason": err.Error()},
		})
		if err.Error() == "email already in use" {
			WriteError(w, http.StatusConflict, err.Error())
			return
		}
		if err.Error() == "email is required" || err.Error() == "invalid user id" {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err.Error() == "user not found" {
			WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		writeInternalError(w, err)
		return
	}
	recordAudit(r, h.auditService, service.AuditEvent{Action: "UPDATE_PROFILE", TargetType: "user", TargetID: int64Pointer(int(userID)), Status: "SUCCESS", Metadata: map[string]interface{}{"field": "email"}})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "email updated successfully"})
}

func (h *AuthHandler) UpdateOwnPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "รูปแบบข้อมูลไม่ถูกต้อง")
		return
	}

	if err := req.Validate(); err != nil {
		recordAudit(r, h.auditService, service.AuditEvent{
			Action:     "UPDATE_PROFILE",
			TargetType: "user",
			TargetID:   int64Pointer(int(userID)),
			Status:     "FAILURE",
			Metadata:   map[string]interface{}{"field": "password", "reason": err.Error()},
		})
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	err := h.authService.ChangePassword(r.Context(), userID, req)
	if err != nil {
		recordAudit(r, h.auditService, service.AuditEvent{
			Action:     "UPDATE_PROFILE",
			TargetType: "user",
			TargetID:   int64Pointer(int(userID)),
			Status:     "FAILURE",
			Metadata:   map[string]interface{}{"field": "password", "reason": err.Error()},
		})
		if err.Error() == "current password is incorrect" ||
			err.Error() == "current password is required" ||
			err.Error() == "new password is required" ||
			err.Error() == "confirm password is required" ||
			err.Error() == "new password must be at least 8 characters" ||
			err.Error() == "new password must contain at least 1 uppercase letter" ||
			err.Error() == "new password must contain at least 1 number" ||
			err.Error() == "passwords do not match" ||
			err.Error() == "invalid user id" {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err.Error() == "user not found" {
			WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		writeInternalError(w, err)
		return
	}
	recordAudit(r, h.auditService, service.AuditEvent{Action: "UPDATE_PROFILE", TargetType: "user", TargetID: int64Pointer(int(userID)), Status: "SUCCESS", Metadata: map[string]interface{}{"field": "password"}})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "password updated successfully"})
}

func (h *AuthHandler) UpdateOwnProfilePicture(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPut {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := r.ParseMultipartForm(5 << 20); err != nil {
		WriteError(w, http.StatusBadRequest, "รูปภาพขนาดใหญ่เกินไป (จำกัด 5MB)")
		return
	}

	file, handler, err := r.FormFile("profileImage")
	if err != nil {
		file, handler, err = r.FormFile("profile_image")
	}
	if err != nil {
		file, handler, err = r.FormFile("avatar")
	}
	if err != nil {
		WriteError(w, http.StatusBadRequest, "ไม่พบไฟล์รูปภาพโปรไฟล์")
		return
	}
	defer file.Close()

	// รูปโปรไฟล์: จำกัด 5MB ตาม VAL-09 (แยกจากปกนิยายที่ 10MB)
	uploadedURL, uploadErr := h.mediaService.UploadImage(r.Context(), handler, service.ProfilePictureMaxBytes)
	if uploadErr != nil {
		writeUploadError(w, uploadErr)
		return
	}

	err = h.authService.UpdateProfilePicture(r.Context(), userID, uploadedURL)
	if err != nil {
		if err.Error() == "profile picture URL is required" || err.Error() == "invalid user id" {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err.Error() == "user not found" {
			WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		writeInternalError(w, err)
		return
	}
	recordAuditWithBackendActor(r, h.auditService, nil, "", service.AuditEvent{Action: "DELETE_ACCOUNT", TargetType: "user", TargetID: nil, Status: "SUCCESS", Metadata: map[string]interface{}{"deleted_user_id": userID}})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "profile picture updated successfully",
		"pic_profile": uploadedURL,
	})
}

func (h *AuthHandler) DeleteOwnAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.DeleteOwnAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "รูปแบบข้อมูลไม่ถูกต้อง")
		return
	}

	err := h.authService.DeleteOwnAccount(r.Context(), userID, req.CurrentPassword)
	if err != nil {
		if err.Error() == "current password is required" || err.Error() == "invalid user id" {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err.Error() == "current password is incorrect" {
			WriteError(w, http.StatusUnauthorized, err.Error())
			return
		}
		if err.Error() == "user not found" {
			WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		if err.Error() == "ต้องระงับบัญชีแทนการลบ เนื่องจากมีนิยายอยู่ในระบบ" {
			WriteError(w, http.StatusConflict, err.Error())
			return
		}
		writeInternalError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "account deleted successfully",
	})
}

func (h *AuthHandler) SuspendOwnAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	err := h.authService.SuspendOwnAccount(r.Context(), userID)
	if err != nil {
		if err.Error() == "account is already suspended" ||
			err.Error() == "invalid user id" {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err.Error() == "user not found" {
			WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		writeInternalError(w, err)
		return
	}

	recordAudit(r, h.auditService, service.AuditEvent{
		Action:     "SUSPEND_OWN_ACCOUNT",
		TargetType: "user",
		TargetID:   int64Pointer(int(userID)),
		Status:     "SUCCESS",
		Metadata:   map[string]interface{}{"reason": "Self-deactivated"},
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "account suspended successfully",
	})
}
