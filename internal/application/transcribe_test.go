package application

import (
	"context"
	"testing"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

// Mock implementations for testing
type mockCache struct {
	items map[string]*ports.CachedItem
}

func newMockCache() *mockCache {
	return &mockCache{items: make(map[string]*ports.CachedItem)}
}

func (m *mockCache) Get(ctx context.Context, reelID string) (*ports.CachedItem, error) {
	if item, ok := m.items[reelID]; ok {
		return item, nil
	}
	return nil, domain.ErrCacheMiss
}

func (m *mockCache) Set(ctx context.Context, reelID string, item *ports.CachedItem) error {
	m.items[reelID] = item
	return nil
}

func (m *mockCache) Delete(ctx context.Context, reelID string) error {
	delete(m.items, reelID)
	return nil
}

func (m *mockCache) CleanExpired(ctx context.Context) (int, error) { return 0, nil }
func (m *mockCache) Clear(ctx context.Context) error              { return nil }
func (m *mockCache) GetCacheDir(reelID string) string             { return "/tmp/" + reelID }
func (m *mockCache) Stats(ctx context.Context) (int, int64, error) {
	return len(m.items), 0, nil
}

type mockDownloader struct {
	available bool
}

func (m *mockDownloader) Download(ctx context.Context, reelID string, destDir string) (*ports.DownloadResult, error) {
	return &ports.DownloadResult{
		VideoPath: destDir + "/video.mp4",
		Reel: &domain.Reel{
			ID:        reelID,
			Title:     "Test Reel",
			Author:    "testuser",
			FetchedAt: time.Now(),
		},
	}, nil
}

func (m *mockDownloader) IsAvailable() bool                                               { return m.available }
func (m *mockDownloader) GetBinaryPath() string                                           { return "/usr/bin/yt-dlp" }
func (m *mockDownloader) Install(ctx context.Context, progress func(int64, int64)) error  { return nil }
func (m *mockDownloader) Update(ctx context.Context) error                                { return nil }
func (m *mockDownloader) IsFFmpegAvailable() bool                                         { return true }
func (m *mockDownloader) GetFFmpegPath() string                                           { return "/usr/bin/ffmpeg" }
func (m *mockDownloader) InstallFFmpeg(ctx context.Context, progress func(int64, int64)) error { return nil }
func (m *mockDownloader) FFmpegInstructions() string                                      { return "" }
func (m *mockDownloader) DownloadVideo(ctx context.Context, reelID string, destPath string) error {
	return nil
}
func (m *mockDownloader) DownloadThumbnail(ctx context.Context, reelID string, destPath string) error {
	return nil
}

type mockTranscriber struct {
	modelDownloaded bool
}

func (m *mockTranscriber) Transcribe(ctx context.Context, videoPath string, opts ports.TranscribeOpts) (*domain.Transcript, error) {
	return &domain.Transcript{
		Text: "Hello world transcription",
		Segments: []domain.Segment{
			{Start: 0, End: 3.5, Text: "Hello world transcription"},
		},
		Model:         opts.Model,
		TranscribedAt: time.Now(),
	}, nil
}

func (m *mockTranscriber) AvailableModels() []ports.Model {
	return []ports.Model{{Name: "small", Size: 462 * 1024 * 1024, Downloaded: m.modelDownloaded}}
}

func (m *mockTranscriber) IsModelDownloaded(model string) bool { return m.modelDownloaded }
func (m *mockTranscriber) DownloadModel(ctx context.Context, model string, progress func(int64, int64)) error {
	return nil
}
func (m *mockTranscriber) DeleteModel(model string) error { return nil }

func TestTranscribeService_Transcribe(t *testing.T) {
	cache := newMockCache()
	downloader := &mockDownloader{available: true}
	transcriber := &mockTranscriber{modelDownloaded: true}

	svc := NewTranscribeService(cache, downloader, transcriber, 24*time.Hour)

	ctx := context.Background()
	result, err := svc.Transcribe(ctx, "test123", TranscribeOptions{
		Model:   "small",
		NoCache: false,
	})

	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}

	if result.Transcript.Text != "Hello world transcription" {
		t.Errorf("Transcript text = %s, want 'Hello world transcription'", result.Transcript.Text)
	}

	if result.FromCache {
		t.Errorf("FromCache should be false for fresh transcription")
	}

	// Verify it was cached
	cached, err := cache.Get(ctx, "test123")
	if err != nil {
		t.Errorf("Item should be cached, got error: %v", err)
	}
	if cached.Transcript.Text != "Hello world transcription" {
		t.Errorf("Cached transcript text mismatch")
	}
}

func TestTranscribeService_CacheHit(t *testing.T) {
	cache := newMockCache()
	downloader := &mockDownloader{available: true}
	transcriber := &mockTranscriber{modelDownloaded: true}

	// Pre-populate cache
	cache.Set(context.Background(), "cached123", &ports.CachedItem{
		Reel: &domain.Reel{ID: "cached123"},
		Transcript: &domain.Transcript{
			Text: "Cached result",
		},
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	svc := NewTranscribeService(cache, downloader, transcriber, 24*time.Hour)

	ctx := context.Background()
	result, err := svc.Transcribe(ctx, "cached123", TranscribeOptions{})

	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}

	if result.Transcript.Text != "Cached result" {
		t.Errorf("Should return cached result, got: %s", result.Transcript.Text)
	}

	if !result.FromCache {
		t.Errorf("FromCache should be true for cached result")
	}
}

func TestTranscribeService_NoCacheBypass(t *testing.T) {
	cache := newMockCache()
	downloader := &mockDownloader{available: true}
	transcriber := &mockTranscriber{modelDownloaded: true}

	// Pre-populate cache with existing data
	cache.Set(context.Background(), "cached123", &ports.CachedItem{
		Reel: &domain.Reel{ID: "cached123"},
		Transcript: &domain.Transcript{
			Text: "Old cached result",
		},
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	svc := NewTranscribeService(cache, downloader, transcriber, 24*time.Hour)

	ctx := context.Background()
	result, err := svc.Transcribe(ctx, "cached123", TranscribeOptions{
		NoCache: true,
	})

	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}

	// Should get fresh transcription, not cache
	if result.Transcript.Text != "Hello world transcription" {
		t.Errorf("Expected fresh transcription, got: %s", result.Transcript.Text)
	}

	if result.FromCache {
		t.Errorf("FromCache should be false when NoCache is set")
	}
}

func TestTranscribeService_DownloadError(t *testing.T) {
	cache := newMockCache()
	downloader := &mockDownloaderWithError{}
	transcriber := &mockTranscriber{modelDownloaded: true}

	svc := NewTranscribeService(cache, downloader, transcriber, 24*time.Hour)

	ctx := context.Background()
	_, err := svc.Transcribe(ctx, "test123", TranscribeOptions{})

	if err == nil {
		t.Error("Expected error from failed download")
	}
}

type mockDownloaderWithError struct{}

func (m *mockDownloaderWithError) Download(ctx context.Context, reelID string, destDir string) (*ports.DownloadResult, error) {
	return nil, domain.ErrReelNotFound
}

func (m *mockDownloaderWithError) IsAvailable() bool                                               { return true }
func (m *mockDownloaderWithError) GetBinaryPath() string                                           { return "/usr/bin/yt-dlp" }
func (m *mockDownloaderWithError) Install(ctx context.Context, progress func(int64, int64)) error  { return nil }
func (m *mockDownloaderWithError) Update(ctx context.Context) error                                { return nil }
func (m *mockDownloaderWithError) IsFFmpegAvailable() bool                                         { return true }
func (m *mockDownloaderWithError) GetFFmpegPath() string                                           { return "/usr/bin/ffmpeg" }
func (m *mockDownloaderWithError) InstallFFmpeg(ctx context.Context, progress func(int64, int64)) error { return nil }
func (m *mockDownloaderWithError) FFmpegInstructions() string                                      { return "" }
func (m *mockDownloaderWithError) DownloadVideo(ctx context.Context, reelID string, destPath string) error {
	return nil
}
func (m *mockDownloaderWithError) DownloadThumbnail(ctx context.Context, reelID string, destPath string) error {
	return nil
}
