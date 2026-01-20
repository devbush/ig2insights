package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Reel represents an Instagram Reel
type Reel struct {
	ID              string
	URL             string
	Author          string
	Title           string
	DurationSeconds int
	ViewCount       int64
	LikeCount       int64     // Number of likes on the reel
	CommentCount    int64     // Number of comments on the reel
	UploadedAt      time.Time // When the reel was posted
	FetchedAt       time.Time
}

// ReelURL builds the full Instagram URL for a reel
func (r *Reel) ReelURL() string {
	if r.URL != "" {
		return r.URL
	}
	return fmt.Sprintf("https://www.instagram.com/p/%s/", r.ID)
}

var (
	// Matches /p/ID or /reel/ID patterns
	reelURLPattern = regexp.MustCompile(`instagram\.com/(?:p|reel)/([A-Za-z0-9_-]+)`)
	// Valid reel ID pattern (alphanumeric, dash, underscore)
	reelIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

// ParseReelInput extracts a Reel from a URL or ID string
func ParseReelInput(input string) (*Reel, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Try to match URL pattern
	if matches := reelURLPattern.FindStringSubmatch(input); len(matches) > 1 {
		return &Reel{
			ID:  matches[1],
			URL: input,
		}, nil
	}

	// Check if it's a valid reel ID
	if reelIDPattern.MatchString(input) {
		return &Reel{
			ID: input,
		}, nil
	}

	return nil, fmt.Errorf("invalid reel URL or ID: %s", input)
}
