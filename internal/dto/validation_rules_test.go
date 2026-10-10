package dto

import (
	"errors"
	"strings"
	"testing"
)

// VAL-03: username ต้องมี 3-50 อักขระ และใช้ charset เดิมของระบบ (letters, numbers, underscores)
func TestValidateUsernameBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantOK   bool
	}{
		{"2 characters", "ab", false},
		{"3 characters", "abc", true},
		{"50 characters", strings.Repeat("a", 50), true},
		{"51 characters", strings.Repeat("a", 51), false},
		{"invalid charset (space)", "user name", false},
		{"invalid charset (thai)", "ผู้ใช้", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.username)
			if tt.wantOK && err != nil {
				t.Fatalf("expected username %q to pass, got %v", tt.username, err)
			}
			if !tt.wantOK && err == nil {
				t.Fatalf("expected username %q to be rejected", tt.username)
			}
		})
	}
}

// VAL-03 ต้องถูกบังคับที่ท่อ Register ด้วย (ไม่ใช่แค่ตอนเปลี่ยนชื่อผู้ใช้)
func TestRegisterRequestValidateUsernameBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantOK   bool
	}{
		{"2 characters", "ab", false},
		{"3 characters", "abc", true},
		{"50 characters", strings.Repeat("a", 50), true},
		{"51 characters", strings.Repeat("a", 51), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := RegisterRequest{
				Username: tt.username,
				Email:    "user@example.com",
				Password: "password1",
			}
			err := req.Validate()
			if tt.wantOK && err != nil {
				t.Fatalf("expected register to pass, got %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Fatalf("expected register with username %q to be rejected", tt.username)
			}
		})
	}
}

// VAL-07: ความสนใจตอนสมัครนักเขียน ต้องเลือก 3-5 หมวดหมู่
func TestWriterApplyValidateInterests(t *testing.T) {
	makeIDs := func(n int) []int {
		ids := make([]int, 0, n)
		for i := 1; i <= n; i++ {
			ids = append(ids, i)
		}
		return ids
	}
	makeGenres := func(n int) []string {
		genres := make([]string, 0, n)
		for i := 1; i <= n; i++ {
			genres = append(genres, "genre")
		}
		return genres
	}

	tests := []struct {
		name        string
		categoryIDs []int
		genres      []string
		wantOK      bool
	}{
		{"0 categories", nil, nil, false},
		{"2 categories", makeIDs(2), nil, false},
		{"3 categories", makeIDs(3), nil, true},
		{"5 categories", makeIDs(5), nil, true},
		{"6 categories", makeIDs(6), nil, false},
		{"genres fallback 2", nil, makeGenres(2), false},
		{"genres fallback 3", nil, makeGenres(3), true},
		{"genres fallback 5", nil, makeGenres(5), true},
		{"genres fallback 6", nil, makeGenres(6), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := WriterApplyRequest{CategoryIDs: tt.categoryIDs, Genres: tt.genres}
			err := req.ValidateInterests()
			if tt.wantOK && err != nil {
				t.Fatalf("expected interests to pass, got %v", err)
			}
			if !tt.wantOK {
				if !errors.Is(err, ErrInvalidWriterInterests) {
					t.Fatalf("expected ErrInvalidWriterInterests, got %v", err)
				}
			}
		})
	}
}
