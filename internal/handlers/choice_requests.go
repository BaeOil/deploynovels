package handlers

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// maxChoiceLabelLength ตาม VAL-04 ใน docs/SRS.md: label ไม่เกิน 80 อักขระ
const maxChoiceLabelLength = 80

func validateChoiceLabel(label string) error {
	if strings.TrimSpace(label) == "" {
		return errors.New("label is required")
	}
	if utf8.RuneCountInString(label) > maxChoiceLabelLength {
		return errors.New("label must not exceed 80 characters")
	}
	return nil
}

type CreateChoiceRequest struct {
	FromSceneID int    `json:"from_scene_id"`
	ToSceneID   int    `json:"to_scene_id"`
	Label       string `json:"label"`
}

func (r *CreateChoiceRequest) Validate() error {
	if r.FromSceneID == 0 {
		return errors.New("from_scene_id is required")
	}
	if r.ToSceneID == 0 {
		return errors.New("to_scene_id is required")
	}
	return validateChoiceLabel(r.Label)
}

type UpdateChoiceRequest struct {
	Label     string `json:"label"`
	ToSceneID int    `json:"to_scene_id"`
}

func (r *UpdateChoiceRequest) Validate() error {
	if r.ToSceneID == 0 {
		return errors.New("to_scene_id is required")
	}
	return validateChoiceLabel(r.Label)
}
