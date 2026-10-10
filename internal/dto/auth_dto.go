package dto

import (
	"errors"
	"mime/multipart"
	"regexp"
	"strings"
)

// RegisterRequest รับข้อมูลจากฟอร์มสมัครสมาชิกแบบฟอร์ม (Form-Data)
type RegisterRequest struct {
	Username string                `form:"username"`
	Email    string                `form:"email"`
	Password string                `form:"password"`
	Avatar   *multipart.FileHeader `form:"avatar"` // รองรับการอัปโหลดไฟล์รูปเข้า MinIO
}

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// ValidateUsername ตรวจสอบชื่อผู้ใช้ตาม VAL-03: 3-50 อักขระ และ charset เดิมของระบบ (letters, numbers, underscores)
func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("username is required")
	}
	if len(username) < 3 || len(username) > 50 {
		return errors.New("username length must be between 3 and 50 characters")
	}
	if !usernameRegex.MatchString(username) {
		return errors.New("username must contain only letters, numbers, and underscores")
	}
	return nil
}

func (r *RegisterRequest) Validate() error {
	r.Username = strings.TrimSpace(r.Username)
	r.Email = strings.TrimSpace(r.Email)
	if err := ValidateUsername(r.Username); err != nil {
		return err
	}
	if r.Email == "" {
		return errors.New("email is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	if len(r.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if !regexp.MustCompile(`[A-Za-z]`).MatchString(r.Password) || !regexp.MustCompile(`\d`).MatchString(r.Password) {
		return errors.New("password must contain both letters and numbers")
	}
	return nil
}

// LoginRequest รับข้อมูล JSON ตอนเข้าสู่ระบบ
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type UpdateUsernameRequest struct {
	Username string `json:"username"`
}

type UpdateEmailRequest struct {
	Email string `json:"email"`
}

func (r *UpdateEmailRequest) Validate() error {
	r.Email = strings.TrimSpace(r.Email)
	if r.Email == "" {
		return errors.New("email is required")
	}
	return nil
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

func (r *ChangePasswordRequest) Validate() error {
	if strings.TrimSpace(r.CurrentPassword) == "" {
		return errors.New("current password is required")
	}
	if strings.TrimSpace(r.NewPassword) == "" {
		return errors.New("new password is required")
	}
	if strings.TrimSpace(r.ConfirmPassword) == "" {
		return errors.New("confirm password is required")
	}
	if len(r.NewPassword) < 8 {
		return errors.New("new password must be at least 8 characters")
	}
	if !regexp.MustCompile(`[A-Z]`).MatchString(r.NewPassword) {
		return errors.New("new password must contain at least 1 uppercase letter")
	}
	if !regexp.MustCompile(`\d`).MatchString(r.NewPassword) {
		return errors.New("new password must contain at least 1 number")
	}
	if r.NewPassword != r.ConfirmPassword {
		return errors.New("passwords do not match")
	}
	return nil
}

// AuthResponse ผลลัพธ์ส่งกลับไปหน้าบ้านพร้อมตั๋ว JWT
type AuthResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	User         struct {
		ID         uint   `json:"id"`
		Username   string `json:"username"`
		Email      string `json:"email"`
		PicProfile string `json:"pic_profile"`
		Role       string `json:"role"`
	} `json:"user"`
}

type NotificationSettingsDTO struct {
	InAppNotifications bool `json:"in_app_notifications"`
	NovelUpdates       bool `json:"novel_updates"`
	Comments           bool `json:"comments"`
	Likes              bool `json:"likes"`
	Follows            bool `json:"follows"`
}

type DeleteOwnAccountRequest struct {
	CurrentPassword string `json:"current_password"`
}
