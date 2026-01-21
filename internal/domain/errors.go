package domain

import "errors"

var (
	// Instagram content errors
	ErrReelNotFound             = errors.New("reel not found or is private")
	ErrAccountNotFound          = errors.New("account not found")
	ErrInstagramScrapingBlocked = errors.New("Instagram is blocking profile access - browse feature temporarily unavailable")

	// Network and rate limiting errors
	ErrRateLimited    = errors.New("rate limited by Instagram")
	ErrNetworkFailure = errors.New("network failure")

	// Transcription errors
	ErrTranscriptionFailed = errors.New("transcription failed")
	ErrModelNotFound       = errors.New("model not found")

	// Cache errors
	ErrCacheExpired = errors.New("cache expired")
	ErrCacheMiss    = errors.New("cache miss")

	// Dependency errors
	ErrFFmpegNotFound = errors.New("ffmpeg not found")
)
