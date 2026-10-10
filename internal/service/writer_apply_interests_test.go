package service

import (
	"context"
	"errors"
	"testing"

	"novel-be/internal/dto"
)

// VAL-07: ApplyForWriter ต้อง reject ความสนใจที่ไม่ใช่ 3-5 หมวดหมู่
// โดยตรวจก่อนเรียก repository (repo เป็น nil ใน test นี้ ถ้า validation ไม่ผ่านจริงจะไม่ panic)
func TestApplyForWriterRejectsInvalidInterestCounts(t *testing.T) {
	s := &writerService{}

	makeIDs := func(n int) []int {
		ids := make([]int, 0, n)
		for i := 1; i <= n; i++ {
			ids = append(ids, i)
		}
		return ids
	}

	tests := []struct {
		name        string
		categoryIDs []int
	}{
		{"0 interests", nil},
		{"2 interests", makeIDs(2)},
		{"6 interests", makeIDs(6)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := dto.WriterApplyRequest{
				NameLastname:    "ผู้สมัคร ทดสอบ",
				PenName:         "penname",
				EmailWriter:     "writer@example.com",
				ContactRequired: "@contact",
				CategoryIDs:     tt.categoryIDs,
			}
			err := s.ApplyForWriter(context.Background(), 1, req)
			if !errors.Is(err, dto.ErrInvalidWriterInterests) {
				t.Fatalf("expected dto.ErrInvalidWriterInterests, got %v", err)
			}
		})
	}
}
