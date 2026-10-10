package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"novel-be/internal/dto"
	"novel-be/internal/models"
	"novel-be/internal/repository"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday" // go get github.com/microcosm-cc/bluemonday
)

type writerService struct {
	repo repository.WriterRepository
}

// sanitizeBioHTML ทำความสะอาด HTML ของ bio ก่อนเข้า DB เสมอ ไม่ว่าจะมาจาก
// ทาง Quill editor (multipart/form-data) หรือยิง JSON ตรงเข้า /api/writers/apply ก็ตาม
// จำกัด tag ให้ตรงกับ QUILL_MODULES.toolbar ฝั่ง frontend เท่านั้น (ไม่รวม img/script/on* ฯลฯ)
var bioSanitizer = newBioSanitizer()

func newBioSanitizer() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "br", "strong", "em", "u", "s", "blockquote", "ol", "ul", "li", "h1", "h2", "h3")
	p.AllowAttrs("href").OnElements("a")
	p.AllowElements("a")
	p.RequireNoFollowOnLinks(true)
	p.AllowStandardURLs()
	return p
}

func sanitizeBio(raw string) string {
	return bioSanitizer.Sanitize(raw)
}

// NewWriterServiceDirect ส่งมอบบริการและสวมรอยอินเตอร์เฟซหลัก
func NewWriterServiceDirect(repo repository.WriterRepository) *writerService {
	return &writerService{repo: repo}
}

// 🟢 2. ปรับปรุงฟังก์ชันนี้ให้ตรงตามสัญญา (want GetWriterByID(int) (*models.Writer, error))
func (s *writerService) GetWriterByID(id int) (*models.Writer, error) {
	// วิ่งไปเรียกฝั่ง repo ต่อเพื่อดึงข้อมูลนักเขียนและส่งคืนไทป์โมเดลตรง ๆ
	return s.repo.GetWriterByID(id)
}

func (s *writerService) GetWriterByUserID(userID int) (*models.Writer, error) {
	return s.repo.GetWriterByUserID(userID)
}

func (s *writerService) GetLatestWriterApplicationByUserID(userID int) (*models.Writer, error) {
	return s.repo.GetLatestWriterApplicationByUserID(userID)
}

// ✍️ 3. Logic ส่งคำขอสมัครเป็นนักเขียน
var (
	ErrAlreadyWriter = errors.New("คุณเป็นนักเขียนอยู่แล้ว ไม่สามารถสมัครซ้ำได้")
	ErrAlreadyApply  = errors.New("คุณได้สมัครหรือเป็นนักเขียนอยู่แล้ว หากต้องการรอแอดมินตรวจสอบหรือยื่นใหม่หลังถูกปฏิเสธ")

	// Business validation errors — message คงไว้ตามเดิมเพื่อไม่เปลี่ยนข้อความที่ client ได้รับ
	// และใช้ errors.Is เพื่อ map เป็น HTTP 400 ใน handler
	ErrWriterApplyMissingRequired = errors.New("กรุณากรอกข้อมูลนามปากกาและช่องทางติดต่อหลักที่จำเป็นค่ะ")
	ErrWriterApplyInvalidEmail    = errors.New("กรุณากรอกอีเมลภาษาอังกฤษให้ถูกต้อง")
)

func (s *writerService) ApplyForWriter(ctx context.Context, userID uint, req dto.WriterApplyRequest) error {
	if req.PenName == "" || req.ContactRequired == "" {
		return ErrWriterApplyMissingRequired
	}
	if !regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`).MatchString(strings.TrimSpace(req.EmailWriter)) {
		return ErrWriterApplyInvalidEmail
	}
	// VAL-07: ต้องเลือกหมวดหมู่ความสนใจ 3-5 รายการ (ตรวจก่อนแตะ repository)
	if err := req.ValidateInterests(); err != nil {
		return err
	}

	userRole, err := s.repo.GetUserRoleByUserID(int(userID))
	if err != nil {
		return err
	}
	if userRole == "writer" {
		return ErrAlreadyWriter
	}

	existingWriter, err := s.repo.GetLatestWriterApplicationByUserID(int(userID))
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if existingWriter != nil && existingWriter.Status != "rejected" && existingWriter.Status != "revoked" {
		return ErrAlreadyApply
	}

	req.Bio = sanitizeBio(req.Bio)

	// มัดรวมข้อมูลการติดต่อและประเภทนิยายทั้งหมดลงใน JSON เดียว
	contacts := map[string]interface{}{
		"primary_contact":   req.ContactRequired,
		"secondary_contact": req.ContactOptional,
		"genres":            req.Genres,
	}

	contactBytes, err := json.Marshal(contacts)
	if err != nil {
		return err
	}

	return s.repo.Apply(ctx, userID, req, string(contactBytes))
}

// 🔍 4. Logic ดึงรายการคำขอที่รอการตรวจสอบ (pending) หรือดูประวัติตั้งแต่ทุกสถานะ
func (s *writerService) GetPendingRequests(ctx context.Context, status string, page, limit int) ([]dto.WriterRequestResponse, error) {
	if page < 0 {
		page = 0
	}
	if limit < 0 {
		limit = 0
	}
	return s.repo.GetPendingRequests(ctx, status, page, limit)
}

// ✅ 5. Logic การกดอนุมัติอัปเกรดฐานะผู้ใช้งาน
func (s *writerService) ApproveWriter(ctx context.Context, writerID uint, adminID uint) error {
	if writerID == 0 {
		return errors.New("รหัสคำขอนักเขียนไม่ถูกต้อง")
	}
	if adminID == 0 {
		return errors.New("ผู้ดูแลระบบไม่ถูกต้อง")
	}
	return s.repo.ApproveWriter(ctx, writerID, adminID)
}

// ❌ Logic การกดปฏิเสธคำขอสมัครนักเขียน
func (s *writerService) RejectWriter(ctx context.Context, writerID uint, adminID uint, rejectionReason string) error {
	if writerID == 0 {
		return errors.New("รหัสคำขอนักเขียนไม่ถูกต้อง")
	}
	if adminID == 0 {
		return errors.New("ผู้ดูแลระบบไม่ถูกต้อง")
	}
	return s.repo.RejectWriter(ctx, writerID, adminID, rejectionReason)
}

// ✏️ Logic อัปเดตโปรไฟล์นักเขียน
func (s *writerService) UpdateWriterProfile(ctx context.Context, writerID int, req dto.UpdateWriterProfileRequest) error {
	if writerID == 0 {
		return errors.New("รหัสนักเขียนไม่ถูกต้อง")
	}
	if req.PenName == "" {
		return errors.New("นามปากกาต้องไม่เป็นค่าว่าง")
	}
	req.Bio = sanitizeBio(req.Bio)

	// ✅ แก้ไข: กำหนดค่าเริ่มต้นเป็น {} ป้องกันการพยายาม insert string ว่างลงคอลัมน์ json
	var contactJSON string = "{}"

	if req.ContactRequired != "" || req.ContactOptional != "" {
		contacts := map[string]string{
			"contact_required": req.ContactRequired,
			"contact_optional": req.ContactOptional,
		}
		bytes, err := json.Marshal(contacts)
		if err != nil {
			return errors.New("รูปแบบข้อมูลช่องทางติดต่อไม่ถูกต้อง")
		}
		contactJSON = string(bytes)
	} else if req.ContactInfo != nil {
		if str, ok := req.ContactInfo.(string); ok {
			// ✅ แก้ไข: เช็กดักอีกชั้นว่าถ้าเป็น "" จะไม่เอาไปทับ "{}"
			if strings.TrimSpace(str) != "" {
				contactJSON = str
			}
		} else {
			bytes, err := json.Marshal(req.ContactInfo)
			if err != nil {
				return errors.New("รูปแบบข้อมูลช่องทางติดต่อไม่ถูกต้อง")
			}
			contactJSON = string(bytes)
		}
	}

	return s.repo.UpdateWriterProfile(ctx, writerID, req, contactJSON)
}
