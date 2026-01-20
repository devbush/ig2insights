package tui

import (
	"testing"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
)

func TestFormatCount(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{892, "892"},
		{999, "999"},
		{1000, "1.0K"},
		{1234, "1.2K"},
		{12345, "12.3K"},
		{999999, "1000.0K"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{12345678, "12.3M"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatCount(tt.input)
			if result != tt.expected {
				t.Errorf("FormatCount(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	tests := []struct {
		input    time.Time
		expected string
	}{
		{time.Time{}, "---"},
		{time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), "Jan 15"},
		{time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC), "Dec 25"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatDate(tt.input)
			if result != tt.expected {
				t.Errorf("FormatDate(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatReelLine(t *testing.T) {
	reel := &domain.Reel{
		Title:        "This is a test caption",
		UploadedAt:   time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		ViewCount:    12345,
		LikeCount:    1234,
		CommentCount: 45,
	}

	result := FormatReelLine(reel, 25)

	// Should contain formatted values
	if len(result) == 0 {
		t.Error("FormatReelLine returned empty string")
	}
	// Check it contains the emoji stats
	if !contains(result, "👁") || !contains(result, "❤️") || !contains(result, "💬") {
		t.Errorf("FormatReelLine missing stat emojis: %s", result)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
