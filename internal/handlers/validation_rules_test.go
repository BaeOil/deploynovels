package handlers

import (
	"strings"
	"testing"
)

// VAL-01: ชื่อเรื่องไม่เกิน 100 อักขระ (boundary 100 ผ่าน / 101 ไม่ผ่าน)
func TestCreateNovelTitleBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		title  string
		wantOK bool
	}{
		{"100 ascii", strings.Repeat("a", 100), true},
		{"101 ascii", strings.Repeat("a", 101), false},
		{"100 thai runes", strings.Repeat("น", 100), true},
		{"101 thai runes", strings.Repeat("น", 101), false},
		{"empty", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := CreateNovelRequest{Title: tt.title}
			err := req.Validate()
			if tt.wantOK && err != nil {
				t.Fatalf("expected title to pass, got %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Fatalf("expected title (%d runes) to be rejected", len([]rune(tt.title)))
			}
		})
	}
}

// VAL-01: ฝั่ง update ตรวจเฉพาะตอนส่ง title มาเปลี่ยน
func TestUpdateNovelTitleBoundaries(t *testing.T) {
	if err := (&UpdateNovelRequest{Title: strings.Repeat("a", 100)}).Validate(); err != nil {
		t.Fatalf("expected 100-char title update to pass, got %v", err)
	}
	if err := (&UpdateNovelRequest{Title: strings.Repeat("a", 101)}).Validate(); err == nil {
		t.Fatal("expected 101-char title update to be rejected")
	}
	// ไม่ส่ง title = ไม่เปลี่ยนค่า → ไม่ต้องตรวจความยาว
	if err := (&UpdateNovelRequest{Status: "draft"}).Validate(); err != nil {
		t.Fatalf("expected status-only update to pass, got %v", err)
	}
}

// VAL-02: captions (tagline) ไม่เกิน 200 อักขระ (boundary 200 ผ่าน / 201 ไม่ผ่าน)
func TestNovelCaptionsBoundaries(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name    string
		content string
		wantOK  bool
	}{
		{"200 ascii", strings.Repeat("c", 200), true},
		{"201 ascii", strings.Repeat("c", 201), false},
		{"200 thai runes", strings.Repeat("ก", 200), true},
		{"201 thai runes", strings.Repeat("ก", 201), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createReq := CreateNovelRequest{Title: "t", Captions: strPtr(tt.content)}
			err := createReq.Validate()
			if tt.wantOK && err != nil {
				t.Fatalf("expected captions to pass, got %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Fatalf("expected captions (%d runes) to be rejected", len([]rune(tt.content)))
			}

			updateReq := UpdateNovelRequest{Captions: strPtr(tt.content)}
			err = updateReq.Validate()
			if tt.wantOK && err != nil {
				t.Fatalf("expected captions update to pass, got %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Fatalf("expected captions update (%d runes) to be rejected", len([]rune(tt.content)))
			}
		})
	}
}

// VAL-06: หมวดหมู่ต่อนิยายหนึ่งเรื่องไม่เกิน 3 และห้ามซ้ำ (minimum ไม่ได้ล็อก → 0 ยังผ่าน)
func TestNovelCategoryIDsLimits(t *testing.T) {
	tests := []struct {
		name   string
		ids    []int
		wantOK bool
	}{
		{"0 categories keeps existing behavior", nil, true},
		{"1 category", []int{1}, true},
		{"3 categories", []int{1, 2, 3}, true},
		{"4 categories", []int{1, 2, 3, 4}, false},
		{"duplicate ids", []int{1, 2, 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createReq := CreateNovelRequest{Title: "t", CategoryIDs: tt.ids}
			err := createReq.Validate()
			if tt.wantOK && err != nil {
				t.Fatalf("expected category_ids %v to pass, got %v", tt.ids, err)
			}
			if !tt.wantOK && err == nil {
				t.Fatalf("expected category_ids %v to be rejected", tt.ids)
			}

			// update: ส่ง category_ids มา = ต้องการเปลี่ยน → ตรวจเหมือนกัน
			updateReq := UpdateNovelRequest{Title: "t", CategoryIDs: tt.ids}
			err = updateReq.Validate()
			if tt.wantOK && err != nil {
				t.Fatalf("expected category_ids update %v to pass, got %v", tt.ids, err)
			}
			if !tt.wantOK && err == nil {
				t.Fatalf("expected category_ids update %v to be rejected", tt.ids)
			}
		})
	}
}

// VAL-04: choice label ไม่เกิน 80 อักขระ (boundary 80 ผ่าน / 81 ไม่ผ่าน)
func TestChoiceLabelBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		label  string
		wantOK bool
	}{
		{"80 ascii", strings.Repeat("l", 80), true},
		{"81 ascii", strings.Repeat("l", 81), false},
		{"80 thai runes", strings.Repeat("ก", 80), true},
		{"81 thai runes", strings.Repeat("ก", 81), false},
		{"empty", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createReq := CreateChoiceRequest{FromSceneID: 1, ToSceneID: 2, Label: tt.label}
			err := createReq.Validate()
			if tt.wantOK && err != nil {
				t.Fatalf("expected label to pass, got %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Fatalf("expected label (%d runes) to be rejected", len([]rune(tt.label)))
			}

			updateReq := UpdateChoiceRequest{ToSceneID: 2, Label: tt.label}
			err = updateReq.Validate()
			if tt.wantOK && err != nil {
				t.Fatalf("expected label update to pass, got %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Fatalf("expected label update (%d runes) to be rejected", len([]rune(tt.label)))
			}
		})
	}
}

// VAL-11: เนื้อหาฉาก/คำนำนิยายแบบยาว ไม่จำกัดจำนวนอักขระ
func TestLongFormContentIsNotLimited(t *testing.T) {
	longContent := strings.Repeat("นิยายยาว ", 20000) // ~180k+ runes

	sceneReq := CreateSceneRequest{
		NovelID:   1,
		ChapterID: 2,
		Title:     "ฉากทดสอบ",
		Content:   longContent,
	}
	if err := sceneReq.Validate(); err != nil {
		t.Fatalf("expected long scene content to pass, got %v", err)
	}

	updateSceneReq := UpdateSceneRequest{Content: longContent}
	if err := updateSceneReq.Validate(); err != nil {
		t.Fatalf("expected long scene content update to pass, got %v", err)
	}

	introduction := longContent
	novelReq := CreateNovelRequest{Title: "t", Introduction: &introduction}
	if err := novelReq.Validate(); err != nil {
		t.Fatalf("expected long introduction to pass, got %v", err)
	}
}
