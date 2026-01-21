package ports

import (
	"context"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
)

// CachedItem represents a cached reel with its associated media and transcript.
type CachedItem struct {
	Reel          *domain.Reel
	Transcript    *domain.Transcript
	AudioPath     string    // WAV audio file path (used for transcription and --audio flag)
	VideoPath     string    // MP4 video file path (used for --video flag)
	ThumbnailPath string    // thumbnail image path
	CreatedAt     time.Time // when this item was cached
	ExpiresAt     time.Time // when this item should be considered stale
}

// CacheStore handles persistent caching of reels and transcripts.
type CacheStore interface {
	// Get retrieves a cached item by reel ID, returning nil if not found.
	Get(ctx context.Context, reelID string) (*CachedItem, error)

	// Set stores an item in the cache.
	Set(ctx context.Context, reelID string, item *CachedItem) error

	// Delete removes a specific item from the cache.
	Delete(ctx context.Context, reelID string) error

	// CleanExpired removes all expired items and returns the count removed.
	CleanExpired(ctx context.Context) (int, error)

	// Clear removes all cached items.
	Clear(ctx context.Context) error

	// GetCacheDir returns the cache directory path for a given reel ID.
	GetCacheDir(reelID string) string

	// Stats returns cache statistics: item count and total size in bytes.
	Stats(ctx context.Context) (itemCount int, totalSize int64, err error)
}
