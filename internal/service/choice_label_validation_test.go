package service

import (
	"errors"
	"strings"
	"testing"

	"novel-be/internal/models"
)

// VAL-04: choice label ไม่เกิน 80 อักขระ ทั้งเส้นทาง create/update และ scene sync
func TestChoiceLabelLengthEnforcedInService(t *testing.T) {
	label80 := strings.Repeat("ก", 80)
	label81 := strings.Repeat("ก", 81)

	t.Run("create 80 runes passes", func(t *testing.T) {
		svc, _ := newChoiceIntegrityService()
		if _, err := svc.CreateChoice(models.Choice{FromSceneID: 1, ToSceneID: 3, Label: label80}); err != nil {
			t.Fatalf("expected 80-rune label to pass, got %v", err)
		}
	})

	t.Run("create 81 runes rejected", func(t *testing.T) {
		svc, _ := newChoiceIntegrityService()
		if _, err := svc.CreateChoice(models.Choice{FromSceneID: 1, ToSceneID: 3, Label: label81}); !errors.Is(err, ErrInvalidChoiceLabel) {
			t.Fatalf("expected ErrInvalidChoiceLabel, got %v", err)
		}
	})

	t.Run("update 81 runes rejected", func(t *testing.T) {
		svc, _ := newChoiceIntegrityService()
		err := svc.UpdateChoice(models.Choice{ChoiceID: 11, ToSceneID: 2, Label: label81})
		if !errors.Is(err, ErrInvalidChoiceLabel) {
			t.Fatalf("expected ErrInvalidChoiceLabel, got %v", err)
		}
	})

	t.Run("sync 81 runes rejected", func(t *testing.T) {
		svc, _ := newChoiceIntegrityService()
		_, err := svc.SyncSceneChoices(1, []interface{}{map[string]interface{}{
			"choice_id": float64(11), "label": label81, "to_scene_id": float64(2),
		}})
		if !errors.Is(err, ErrInvalidChoiceLabel) {
			t.Fatalf("expected ErrInvalidChoiceLabel, got %v", err)
		}
	})
}
