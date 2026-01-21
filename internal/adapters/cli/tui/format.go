package tui

import (
	"fmt"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
)

// Byte size constants for formatting
const (
	KB = 1024
	MB = KB * 1024
	GB = MB * 1024
)

// FormatSize formats a byte count as a human-readable string
// Examples: 1024 -> "1 KB", 1536000 -> "1.5 MB", 3221225472 -> "3.0 GB"
func FormatSize(bytes int64) string {
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.0f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.0f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatCount formats a number with K/M suffix
// Examples: 892 -> "892", 1234 -> "1.2K", 1500000 -> "1.5M"
func FormatCount(count int64) string {
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	}
	if count >= 1000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%d", count)
}

// FormatDate formats a date as "Jan 15" style
func FormatDate(t time.Time) string {
	if t.IsZero() {
		return "---"
	}
	return t.Format("Jan 2")
}

// FormatReelLine formats a reel as a single line for display
// Example: "Had an amazing day at..."  Jan 15  👁 12.3K  ❤️ 1.2K  💬 45
func FormatReelLine(reel *domain.Reel, maxCaptionLen int) string {
	caption := reel.Title
	if len(caption) > maxCaptionLen {
		caption = caption[:maxCaptionLen-3] + "..."
	}

	// Pad caption to fixed width
	captionFmt := fmt.Sprintf("%%-%ds", maxCaptionLen)
	paddedCaption := fmt.Sprintf(captionFmt, caption)

	date := FormatDate(reel.UploadedAt)
	views := FormatCount(reel.ViewCount)
	likes := FormatCount(reel.LikeCount)
	comments := FormatCount(reel.CommentCount)

	return fmt.Sprintf("%s  %s  👁 %6s  ❤️ %6s  💬 %5s",
		paddedCaption, date, views, likes, comments)
}
