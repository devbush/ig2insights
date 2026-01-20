package tui

import (
	"fmt"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
)

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
