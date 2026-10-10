package handlers

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// ค่าที่ล็อกไว้ใน docs/SRS.md (Validation Rules) — ห้ามแก้ไขโดยไม่ผ่าน Requirement process
const (
	maxNovelTitleLength    = 100 // VAL-01
	maxNovelCaptionsLength = 200 // VAL-02
	maxNovelCategories     = 3   // VAL-06
)

// validateNovelTitle ตาม VAL-01: ชื่อเรื่องไม่เกิน 100 อักขระ (นับตามตัวอักษรจริง ไม่ใช่ bytes)
func validateNovelTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("title is required")
	}
	if utf8.RuneCountInString(title) > maxNovelTitleLength {
		return errors.New("title must not exceed 100 characters")
	}
	return nil
}

// validateNovelCaptions ตาม VAL-02: คำโปรย (captions) ไม่เกิน 200 อักขระ
func validateNovelCaptions(captions *string) error {
	if captions != nil && utf8.RuneCountInString(*captions) > maxNovelCaptionsLength {
		return errors.New("captions must not exceed 200 characters")
	}
	return nil
}

// validateNovelCategoryIDs ตาม VAL-06: ต่อนิยายหนึ่งเรื่องไม่เกิน 3 หมวดหมู่ และห้ามมี ID ซ้ำ
// ไม่กำหนด minimum เพราะเอกสารล็อกเฉพาะ maximum
func validateNovelCategoryIDs(categoryIDs []int) error {
	if len(categoryIDs) > maxNovelCategories {
		return errors.New("category_ids must not exceed 3 categories")
	}
	seen := make(map[int]bool, len(categoryIDs))
	for _, id := range categoryIDs {
		if seen[id] {
			return errors.New("category_ids must not contain duplicates")
		}
		seen[id] = true
	}
	return nil
}

type CreateNovelRequest struct {
	Title        string  `json:"title"`
	Captions     *string `json:"captions,omitempty"`
	Introduction *string `json:"introduction,omitempty"`
	CoverImage   *string `json:"cover_image,omitempty"`
	Status       string  `json:"status,omitempty"`
	IsPublished  *bool   `json:"is_published,omitempty"`
	IsCompleted  *bool   `json:"is_completed,omitempty"`
	CategoryIDs  []int   `json:"category_ids,omitempty"`
	AuthorID     int     `json:"author_id"`
}

func normalizeNovelStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "publish", "published":
		return "published"
	case "completed", "complete", "finished":
		return "completed"
	case "completed-published", "completed_published", "completed+published":
		return "completed-published"
	case "completed-draft", "completed_draft", "completed+draft":
		return "completed-draft"
	default:
		return "draft"
	}
}

func deriveNovelStatus(isCompleted, isPublished bool) string {
	switch {
	case isCompleted && isPublished:
		return "completed-published"
	case isCompleted:
		return "completed-draft"
	case isPublished:
		return "published"
	default:
		return "draft"
	}
}

func resolveNovelStatus(status string, isPublished *bool, isCompleted *bool) (string, bool, bool) {
	resolvedPublished := false
	resolvedCompleted := false

	if isPublished != nil {
		resolvedPublished = *isPublished
	}
	if isCompleted != nil {
		resolvedCompleted = *isCompleted
	}

	if isPublished == nil && isCompleted == nil {
		switch normalizeNovelStatus(status) {
		case "published":
			resolvedPublished = true
		case "completed-published":
			resolvedPublished = true
			resolvedCompleted = true
		case "completed-draft", "completed":
			resolvedCompleted = true
		}
	}

	return deriveNovelStatus(resolvedCompleted, resolvedPublished), resolvedPublished, resolvedCompleted
}

func (r *CreateNovelRequest) Validate() error {
	if err := validateNovelTitle(r.Title); err != nil {
		return err
	}
	if err := validateNovelCaptions(r.Captions); err != nil {
		return err
	}
	if err := validateNovelCategoryIDs(r.CategoryIDs); err != nil {
		return err
	}
	resolvedStatus, _, _ := resolveNovelStatus(r.Status, r.IsPublished, r.IsCompleted)
	r.Status = resolvedStatus
	return nil
}

type UpdateNovelRequest struct {
	Title        string  `json:"title,omitempty"`
	Captions     *string `json:"captions,omitempty"`
	Introduction *string `json:"introduction,omitempty"`
	CoverImage   *string `json:"cover_image,omitempty"`
	CategoryIDs  []int   `json:"category_ids,omitempty"`
	Status       string  `json:"status,omitempty"`
	IsPublished  *bool   `json:"is_published,omitempty"`
	IsCompleted  *bool   `json:"is_completed,omitempty"`
}

func (r *UpdateNovelRequest) Validate() error {
	// At least one updatable field must be present
	if strings.TrimSpace(r.Title) == "" && strings.TrimSpace(r.Status) == "" && r.Captions == nil && r.Introduction == nil && r.CoverImage == nil && len(r.CategoryIDs) == 0 && r.IsPublished == nil && r.IsCompleted == nil {
		return errors.New("at least one field is required")
	}
	// VAL-01 / VAL-02 / VAL-06 — ตรวจเฉพาะ field ที่ส่งมาเปลี่ยน (ค่าว่าง = ไม่เปลี่ยนค่าเดิม)
	if strings.TrimSpace(r.Title) != "" {
		if utf8.RuneCountInString(r.Title) > maxNovelTitleLength {
			return errors.New("title must not exceed 100 characters")
		}
	}
	if err := validateNovelCaptions(r.Captions); err != nil {
		return err
	}
	if len(r.CategoryIDs) > 0 {
		if err := validateNovelCategoryIDs(r.CategoryIDs); err != nil {
			return err
		}
	}
	return nil
}
