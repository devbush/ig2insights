package domain

import "errors"

var (
	// ErrReelNotFound indicates the reel doesn't exist or is private
	ErrReelNotFound = errors.New("reel not found or is private")

	// ErrAccountNotFound indicates the account doesn't exist
	ErrAccountNotFound = errors.New("account not found")

	// ErrRateLimited indicates Instagram rate limiting
	ErrRateLimited = errors.New("rate limited by Instagram")

	// ErrNetworkFailure indicates a network error
	ErrNetworkFailure = errors.New("network failure")

	// ErrTranscriptionFailed indicates transcription error
	ErrTranscriptionFailed = errors.New("transcription failed")

	// ErrModelNotFound indicates the whisper model isn't downloaded
	ErrModelNotFound = errors.New("model not found")

	// ErrCacheExpired indicates cached item has expired
	ErrCacheExpired = errors.New("cache expired")

	// ErrCacheMiss indicates item not in cache
	ErrCacheMiss = errors.New("cache miss")

	// ErrFFmpegNotFound indicates ffmpeg is not installed
	ErrFFmpegNotFound = errors.New("ffmpeg not found")
)
