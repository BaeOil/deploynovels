package handlers

import "testing"

func TestNormalizeAdminNovelStatusDefaultsToAll(t *testing.T) {
	if got := normalizeAdminNovelStatus(""); got != "all" {
		t.Fatalf("empty status should default to all, got %q", got)
	}
}

func TestNormalizeAdminNovelStatusAcceptsAdminFilters(t *testing.T) {
	cases := map[string]string{
		"all":       "all",
		"published": "published",
		"suspended": "suspended",
	}

	for input, want := range cases {
		if got := normalizeAdminNovelStatus(input); got != want {
			t.Fatalf("status %q => %q, want %q", input, got, want)
		}
	}
}
