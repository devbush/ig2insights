package domain

import (
	"fmt"
	"regexp"
	"strings"
)

// SortOrder defines how to sort reels
type SortOrder string

const (
	SortLatest     SortOrder = "latest"
	SortMostViewed SortOrder = "most_viewed"
)

// Account represents an Instagram account
type Account struct {
	Username  string
	ReelCount int
}

// AccountURL builds the full Instagram URL for an account
func (a *Account) AccountURL() string {
	return fmt.Sprintf("https://www.instagram.com/%s/", a.Username)
}

var (
	// Matches instagram.com/username patterns (not /p/ or /reel/)
	accountURLPattern = regexp.MustCompile(`instagram\.com/([A-Za-z0-9_.]+)/?$`)
	// Valid username pattern
	usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.]+$`)
)

// ParseAccountInput extracts an Account from a URL or username string
func ParseAccountInput(input string) (*Account, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Remove @ prefix if present
	input = strings.TrimPrefix(input, "@")

	// Try to match URL pattern
	if matches := accountURLPattern.FindStringSubmatch(input); len(matches) > 1 {
		return &Account{
			Username: matches[1],
		}, nil
	}

	// Check if it's a valid username
	if usernamePattern.MatchString(input) {
		return &Account{
			Username: input,
		}, nil
	}

	return nil, fmt.Errorf("invalid account URL or username: %s", input)
}
