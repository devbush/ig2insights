package cache

import (
	"context"
	"testing"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

func TestFileCache_SetGet(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewFileCache(tmpDir)

	ctx := context.Background()
	item := &ports.CachedItem{
		Reel: &domain.Reel{
			ID:    "test123",
			Title: "Test Reel",
		},
		Transcript: &domain.Transcript{
			Text: "Hello world",
		},
		VideoPath: "/tmp/video.mp4",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := cache.Set(ctx, "test123", item)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := cache.Get(ctx, "test123")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Reel.ID != "test123" {
		t.Errorf("Get() reel ID = %s, want test123", got.Reel.ID)
	}
	if got.Transcript.Text != "Hello world" {
		t.Errorf("Get() transcript text = %s, want Hello world", got.Transcript.Text)
	}
}

func TestFileCache_GetMiss(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewFileCache(tmpDir)

	ctx := context.Background()
	_, err := cache.Get(ctx, "nonexistent")

	if err != domain.ErrCacheMiss {
		t.Errorf("Get() error = %v, want ErrCacheMiss", err)
	}
}

func TestFileCache_GetExpired(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewFileCache(tmpDir)

	ctx := context.Background()
	item := &ports.CachedItem{
		Reel:      &domain.Reel{ID: "expired123"},
		CreatedAt: time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-24 * time.Hour),
	}

	_ = cache.Set(ctx, "expired123", item)

	_, err := cache.Get(ctx, "expired123")
	if err != domain.ErrCacheExpired {
		t.Errorf("Get() error = %v, want ErrCacheExpired", err)
	}
}

func TestFileCache_CleanExpired(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewFileCache(tmpDir)

	ctx := context.Background()

	item := &ports.CachedItem{
		Reel:      &domain.Reel{ID: "willexpire"},
		CreatedAt: time.Now().Add(-1 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}
	_ = cache.Set(ctx, "willexpire", item)

	time.Sleep(10 * time.Millisecond)

	cleaned, err := cache.CleanExpired(ctx)
	if err != nil {
		t.Fatalf("CleanExpired() error = %v", err)
	}

	if cleaned != 1 {
		t.Errorf("CleanExpired() = %d, want 1", cleaned)
	}
}
