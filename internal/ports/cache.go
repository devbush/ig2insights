package ports

import (
	"context"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
)

// CachedItem represents a cached reel with transcript
type CachedItem struct {
	Reel          *domain.Reel
	Transcript    *domain.Transcript
	AudioPath     string // WAV audio for transcription and --audio
	VideoPath     string // MP4 video for --video
	ThumbnailPath string
	CreatedAt     time.Time
	ExpiresAt     time.Time
}

// CacheStore handles persistent caching of reels and transcripts
type CacheStore interface {
	// Get retrieves a cached item by reel ID
	Get(ctx context.Context, reelID string) (*CachedItem, error)

	// Set stores an item in cache
	Set(ctx context.Context, reelID string, item *CachedItem) error

	// Delete removes a specific item from cache
	Delete(ctx context.Context, reelID string) error

	// CleanExpired removes all expired items
	CleanExpired(ctx context.Context) (int, error)

	// Clear removes all cached items
	Clear(ctx context.Context) error

	// GetCacheDir returns the cache directory path for a reel
	GetCacheDir(reelID string) string

	// Stats returns cache statistics
	Stats(ctx context.Context) (itemCount int, totalSize int64, err error)
}
