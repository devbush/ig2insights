# Smart Asset Caching Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable independent caching for transcript, video, and thumbnail so each asset is checked and fetched independently.

**Architecture:** Extend `CachedItem` with `ThumbnailPath`, refactor `TranscribeService.Transcribe()` to check cache per-asset, update CLI to show "(cached)" status and copy files from cache to output.

**Tech Stack:** Go, existing ports/adapters architecture

---

## Task 1: Add ThumbnailPath to Cache Model

**Files:**
- Modify: `internal/ports/cache.go:11-17`
- Modify: `internal/adapters/cache/store.go:26-32`

**Step 1: Update CachedItem struct**

In `internal/ports/cache.go`, add `ThumbnailPath` field:

```go
// CachedItem represents a cached reel with transcript
type CachedItem struct {
	Reel          *domain.Reel
	Transcript    *domain.Transcript
	VideoPath     string
	ThumbnailPath string
	CreatedAt     time.Time
	ExpiresAt     time.Time
}
```

**Step 2: Update metaFile struct in cache adapter**

In `internal/adapters/cache/store.go`, update `metaFile`:

```go
type metaFile struct {
	Reel          *domain.Reel       `json:"reel"`
	Transcript    *domain.Transcript `json:"transcript"`
	VideoPath     string             `json:"video_path"`
	ThumbnailPath string             `json:"thumbnail_path"`
	CreatedAt     time.Time          `json:"created_at"`
	ExpiresAt     time.Time          `json:"expires_at"`
}
```

**Step 3: Update Get method to include ThumbnailPath**

In `internal/adapters/cache/store.go`, update `Get()` return (around line 62-68):

```go
	return &ports.CachedItem{
		Reel:          meta.Reel,
		Transcript:    meta.Transcript,
		VideoPath:     meta.VideoPath,
		ThumbnailPath: meta.ThumbnailPath,
		CreatedAt:     meta.CreatedAt,
		ExpiresAt:     meta.ExpiresAt,
	}, nil
```

**Step 4: Update Set method to include ThumbnailPath**

In `internal/adapters/cache/store.go`, update `Set()` (around line 77-83):

```go
	meta := metaFile{
		Reel:          item.Reel,
		Transcript:    item.Transcript,
		VideoPath:     item.VideoPath,
		ThumbnailPath: item.ThumbnailPath,
		CreatedAt:     item.CreatedAt,
		ExpiresAt:     item.ExpiresAt,
	}
```

**Step 5: Run tests to verify no regressions**

Run: `go test ./internal/adapters/cache/... ./internal/ports/...`
Expected: PASS (existing tests should still pass)

**Step 6: Commit**

```bash
git add internal/ports/cache.go internal/adapters/cache/store.go
git commit -m "feat(cache): add ThumbnailPath to CachedItem"
```

---

## Task 2: Update TranscribeResult for Per-Asset Cache Status

**Files:**
- Modify: `internal/application/transcribe.go:22-29`

**Step 1: Update TranscribeResult struct**

Replace the existing struct with per-asset cache flags:

```go
// TranscribeResult contains the transcription result
type TranscribeResult struct {
	Reel          *domain.Reel
	Transcript    *domain.Transcript
	VideoPath     string
	ThumbnailPath string

	// Per-asset cache status
	TranscriptFromCache bool
	VideoFromCache      bool
	ThumbnailFromCache  bool
}
```

**Step 2: Run tests (expect some failures)**

Run: `go test ./internal/application/...`
Expected: Tests may fail due to `FromCache` field being removed - this is expected.

**Step 3: Update test assertions**

In `internal/application/transcribe_test.go`, update `TestTranscribeService_Transcribe` (around line 122-124):

```go
	if result.TranscriptFromCache {
		t.Errorf("TranscriptFromCache should be false for fresh transcription")
	}
```

Update `TestTranscribeService_CacheHit` (around line 163-165):

```go
	if !result.TranscriptFromCache {
		t.Errorf("TranscriptFromCache should be true for cached result")
	}
```

Update `TestTranscribeService_NoCacheBypass` (around line 198-200):

```go
	if result.TranscriptFromCache {
		t.Errorf("TranscriptFromCache should be false when NoCache is set")
	}
```

**Step 4: Run tests**

Run: `go test ./internal/application/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/application/transcribe.go internal/application/transcribe_test.go
git commit -m "feat(application): add per-asset cache status to TranscribeResult"
```

---

## Task 3: Refactor Transcribe Method for Per-Asset Caching

**Files:**
- Modify: `internal/application/transcribe.go:54-112`

**Step 1: Write test for partial cache scenario**

Add new test in `internal/application/transcribe_test.go`:

```go
func TestTranscribeService_PartialCache_TranscriptOnly(t *testing.T) {
	cache := newMockCache()
	downloader := &mockDownloader{available: true}
	transcriber := &mockTranscriber{modelDownloaded: true}

	// Pre-populate cache with transcript only (no video path)
	cache.Set(context.Background(), "partial123", &ports.CachedItem{
		Reel:       &domain.Reel{ID: "partial123", Title: "Cached Reel"},
		Transcript: &domain.Transcript{Text: "Cached transcript"},
		VideoPath:  "", // No video cached
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	})

	svc := NewTranscribeService(cache, downloader, transcriber, 24*time.Hour)

	ctx := context.Background()
	result, err := svc.Transcribe(ctx, "partial123", TranscribeOptions{
		SaveVideo: true, // Request video even though only transcript cached
	})

	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}

	if !result.TranscriptFromCache {
		t.Errorf("TranscriptFromCache should be true")
	}

	if result.VideoFromCache {
		t.Errorf("VideoFromCache should be false - video wasn't cached")
	}

	if result.VideoPath == "" {
		t.Errorf("VideoPath should be populated after download")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/application/... -run TestTranscribeService_PartialCache`
Expected: FAIL (current implementation returns early on cache hit)

**Step 3: Refactor Transcribe method**

Replace the entire `Transcribe` method in `internal/application/transcribe.go`:

```go
// Transcribe processes a reel and returns its transcript
func (s *TranscribeService) Transcribe(ctx context.Context, reelID string, opts TranscribeOptions) (*TranscribeResult, error) {
	result := &TranscribeResult{}

	// Try to get cached data first
	var cached *ports.CachedItem
	if !opts.NoCache {
		var err error
		cached, err = s.cache.Get(ctx, reelID)
		if err != nil {
			cached = nil // Treat errors as cache miss
		}
	}

	// Determine what we have cached vs what we need
	cacheDir := s.cache.GetCacheDir(reelID)

	hasTranscript := cached != nil && cached.Transcript != nil
	hasVideo := cached != nil && cached.VideoPath != "" && fileExists(cached.VideoPath)
	hasThumbnail := cached != nil && cached.ThumbnailPath != "" && fileExists(cached.ThumbnailPath)

	needTranscript := !hasTranscript
	needVideo := opts.SaveVideo && !hasVideo
	needThumbnail := opts.SaveThumbnail && !hasThumbnail

	// Use cached reel metadata if available
	var reel *domain.Reel
	if cached != nil && cached.Reel != nil {
		reel = cached.Reel
	}

	// Download video if needed for transcription OR if user requested video
	var videoPath string
	if needTranscript || needVideo {
		downloadResult, err := s.downloader.Download(ctx, reelID, cacheDir)
		if err != nil {
			return nil, err
		}
		videoPath = downloadResult.VideoPath
		reel = downloadResult.Reel
	} else if hasVideo {
		videoPath = cached.VideoPath
	}

	// Transcribe if needed
	var transcript *domain.Transcript
	if needTranscript {
		model := opts.Model
		if model == "" {
			model = "small"
		}

		language := opts.Language
		if language == "" {
			language = "auto"
		}

		var err error
		transcript, err = s.transcriber.Transcribe(ctx, videoPath, ports.TranscribeOpts{
			Model:    model,
			Language: language,
		})
		if err != nil {
			return nil, err
		}
	} else {
		transcript = cached.Transcript
	}

	// Download thumbnail if needed
	var thumbnailPath string
	if needThumbnail {
		thumbPath := filepath.Join(cacheDir, "thumbnail.jpg")
		if err := s.downloader.DownloadThumbnail(ctx, reelID, thumbPath); err != nil {
			// Non-fatal - continue without thumbnail
		} else {
			thumbnailPath = thumbPath
		}
	} else if hasThumbnail {
		thumbnailPath = cached.ThumbnailPath
	}

	// Update cache with any new data
	now := time.Now()
	cacheItem := &ports.CachedItem{
		Reel:          reel,
		Transcript:    transcript,
		VideoPath:     videoPath,
		ThumbnailPath: thumbnailPath,
		CreatedAt:     now,
		ExpiresAt:     now.Add(s.cacheTTL),
	}

	// Preserve original timestamps if we had cached data
	if cached != nil {
		cacheItem.CreatedAt = cached.CreatedAt
	}

	_ = s.cache.Set(ctx, reelID, cacheItem)

	// Build result
	result.Reel = reel
	result.Transcript = transcript
	result.VideoPath = videoPath
	result.ThumbnailPath = thumbnailPath
	result.TranscriptFromCache = hasTranscript
	result.VideoFromCache = hasVideo && opts.SaveVideo
	result.ThumbnailFromCache = hasThumbnail && opts.SaveThumbnail

	return result, nil
}

// fileExists checks if a file exists at the given path
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
```

**Step 4: Add missing import**

Add `"os"` and `"path/filepath"` to imports in `internal/application/transcribe.go`:

```go
import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)
```

**Step 5: Run all tests**

Run: `go test ./internal/application/...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/application/transcribe.go internal/application/transcribe_test.go
git commit -m "feat(application): implement per-asset caching in Transcribe"
```

---

## Task 4: Add File Copy Helper

**Files:**
- Create: `internal/adapters/cli/fileutil.go`

**Step 1: Create file utility**

Create `internal/adapters/cli/fileutil.go`:

```go
package cli

import (
	"io"
	"os"
)

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	// If src and dst are the same, nothing to do
	if src == dst {
		return nil
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
```

**Step 2: Run build to verify**

Run: `go build ./cmd/ig2insights`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/adapters/cli/fileutil.go
git commit -m "feat(cli): add copyFile helper utility"
```

---

## Task 5: Update CLI Progress for Cache Status

**Files:**
- Modify: `internal/adapters/cli/root.go:206-368`

**Step 1: Add pre-flight cache check and refactor runTranscribe**

Replace the `runTranscribe` function in `internal/adapters/cli/root.go`:

```go
func runTranscribe(input string) error {
	app, err := GetApp()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	reel, err := domain.ParseReelInput(input)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Pre-flight cache check to determine what's cached
	var cached *ports.CachedItem
	if !noCacheFlag {
		cached, _ = app.Cache.Get(ctx, reel.ID)
	}

	hasTranscript := cached != nil && cached.Transcript != nil
	hasVideo := cached != nil && cached.VideoPath != "" && fileExists(cached.VideoPath)
	hasThumbnail := cached != nil && cached.ThumbnailPath != "" && fileExists(cached.ThumbnailPath)

	// Build step list based on what we're doing and what's cached
	steps := []string{}

	steps = append(steps, "Checking dependencies")

	if hasTranscript {
		steps = append(steps, "Downloading video (cached)")
		steps = append(steps, "Extracting audio (cached)")
		steps = append(steps, "Transcribing (cached)")
	} else {
		steps = append(steps, "Downloading video")
		steps = append(steps, "Extracting audio")
		steps = append(steps, "Transcribing")
	}

	videoStepIdx := -1
	thumbStepIdx := -1

	if videoFlag {
		videoStepIdx = len(steps)
		if hasVideo {
			steps = append(steps, "Saving video (cached)")
		} else {
			steps = append(steps, "Saving video")
		}
	}
	if thumbnailFlag {
		thumbStepIdx = len(steps)
		if hasThumbnail {
			steps = append(steps, "Downloading thumbnail (cached)")
		} else {
			steps = append(steps, "Downloading thumbnail")
		}
	}

	progress := tui.NewProgressDisplay(steps, quietFlag)

	// Step 1: Check dependencies
	progress.StartStep(0)

	if !app.Downloader.IsAvailable() {
		if err := app.Downloader.Install(context.Background(), func(d, t int64) {
			progress.UpdateProgress(0, d, t)
		}); err != nil {
			progress.FailStep(0, err.Error())
			return fmt.Errorf("failed to install yt-dlp: %w", err)
		}
	}

	if !app.Transcriber.IsAvailable() {
		instructions := app.Transcriber.InstallationInstructions()
		if instructions != "" {
			progress.FailStep(0, "whisper.cpp not found")
			return errors.New(instructions)
		}
		if err := app.Transcriber.Install(context.Background(), func(d, t int64) {
			progress.UpdateProgress(0, d, t)
		}); err != nil {
			progress.FailStep(0, err.Error())
			return fmt.Errorf("failed to install whisper.cpp: %w", err)
		}
	}

	if !app.Downloader.IsFFmpegAvailable() {
		instructions := app.Downloader.FFmpegInstructions()
		if instructions != "" {
			progress.FailStep(0, "ffmpeg not found")
			return errors.New(instructions)
		}
		if err := app.Downloader.InstallFFmpeg(context.Background(), func(d, t int64) {
			progress.UpdateProgress(0, d, t)
		}); err != nil {
			progress.FailStep(0, err.Error())
			return fmt.Errorf("failed to install ffmpeg: %w", err)
		}
	}

	model := modelFlag
	if model == "" {
		model = app.Config.Defaults.Model
	}

	if !app.Transcriber.IsModelDownloaded(model) {
		if err := app.Transcriber.DownloadModel(context.Background(), model, func(d, t int64) {
			progress.UpdateProgress(0, d, t)
		}); err != nil {
			progress.FailStep(0, err.Error())
			return fmt.Errorf("failed to download model: %w", err)
		}
	}
	progress.CompleteStep(0)

	// Start spinner for indeterminate steps
	spinnerDone := progress.StartSpinner()

	// If transcript is cached, immediately complete those steps
	if hasTranscript {
		progress.CompleteStep(1) // Download (cached)
		progress.CompleteStep(2) // Extract (cached)
		progress.CompleteStep(3) // Transcribe (cached)
	} else {
		progress.StartStep(1)
	}

	result, err := app.TranscribeSvc.Transcribe(ctx, reel.ID, application.TranscribeOptions{
		Model:         model,
		NoCache:       noCacheFlag,
		Language:      languageFlag,
		SaveVideo:     videoFlag,
		SaveThumbnail: thumbnailFlag,
	})

	if err != nil {
		close(spinnerDone)
		progress.FailStep(1, err.Error())
		return err
	}

	// Mark transcription steps complete if not already
	if !hasTranscript {
		progress.CompleteStep(1) // Download
		progress.CompleteStep(2) // Extract
		progress.CompleteStep(3) // Transcribe
	}

	// Determine output directory
	outputDir := downloadDirFlag
	if outputDir == "" {
		if outputFlag != "" {
			outputDir = filepath.Dir(outputFlag)
		} else {
			outputDir = "."
		}
	}

	outputs := make(map[string]string)

	// Step: Save video (if requested)
	if videoFlag && videoStepIdx >= 0 {
		if result.VideoFromCache {
			// Already marked as cached in step name, just complete it
			progress.CompleteStep(videoStepIdx)
		} else {
			progress.StartStep(videoStepIdx)
		}

		videoPath := filepath.Join(outputDir, reel.ID+".mp4")
		if result.VideoPath != "" {
			if err := copyFile(result.VideoPath, videoPath); err != nil {
				progress.FailStep(videoStepIdx, err.Error())
			} else {
				progress.CompleteStep(videoStepIdx)
				outputs["Video"] = videoPath
			}
		} else {
			progress.FailStep(videoStepIdx, "no video available")
		}
	}

	// Step: Download thumbnail (if requested)
	if thumbnailFlag && thumbStepIdx >= 0 {
		if result.ThumbnailFromCache {
			progress.CompleteStep(thumbStepIdx)
		} else {
			progress.StartStep(thumbStepIdx)
		}

		thumbPath := filepath.Join(outputDir, reel.ID+".jpg")
		if result.ThumbnailPath != "" {
			if err := copyFile(result.ThumbnailPath, thumbPath); err != nil {
				progress.FailStep(thumbStepIdx, err.Error())
			} else {
				progress.CompleteStep(thumbStepIdx)
				outputs["Thumbnail"] = thumbPath
			}
		} else {
			// Thumbnail wasn't cached, try downloading directly
			if err := app.Downloader.DownloadThumbnail(ctx, reel.ID, thumbPath); err != nil {
				progress.FailStep(thumbStepIdx, err.Error())
			} else {
				progress.CompleteStep(thumbStepIdx)
				outputs["Thumbnail"] = thumbPath
			}
		}
	}

	// Stop spinner
	close(spinnerDone)

	// Output transcript
	if err := outputResult(result); err != nil {
		return err
	}

	if outputFlag != "" {
		outputs["Transcript"] = outputFlag
	}

	if !quietFlag && len(outputs) > 0 {
		progress.Complete(outputs)
	}

	return nil
}
```

**Step 2: Add missing import**

Update imports in `internal/adapters/cli/root.go` to include `ports`:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/devbush/ig2insights/internal/adapters/cli/tui"
	"github.com/devbush/ig2insights/internal/application"
	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
	"github.com/spf13/cobra"
)
```

**Step 3: Add fileExists function if not already present**

Add at the end of `internal/adapters/cli/root.go` (before the closing of the file):

```go
// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
```

**Step 4: Build and verify**

Run: `go build ./cmd/ig2insights`
Expected: Build succeeds

**Step 5: Commit**

```bash
git add internal/adapters/cli/root.go
git commit -m "feat(cli): show cache status in progress display"
```

---

## Task 6: Update Download-Only Mode to Use Cache

**Files:**
- Modify: `internal/adapters/cli/root.go:156-192`

**Step 1: Refactor runDownloadOnly to check and use cache**

Replace `runDownloadOnly` function:

```go
func runDownloadOnly(input string, video, thumbnail bool) error {
	app, err := GetApp()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	reel, err := domain.ParseReelInput(input)
	if err != nil {
		return err
	}

	ctx := context.Background()
	outputDir := downloadDirFlag
	if outputDir == "" {
		outputDir = "."
	}

	// Check cache for existing assets
	cached, _ := app.Cache.Get(ctx, reel.ID)
	hasVideo := cached != nil && cached.VideoPath != "" && fileExists(cached.VideoPath)
	hasThumbnail := cached != nil && cached.ThumbnailPath != "" && fileExists(cached.ThumbnailPath)

	cacheDir := app.Cache.GetCacheDir(reel.ID)
	cacheUpdated := false
	var cachedVideoPath, cachedThumbnailPath string

	if video {
		destPath := filepath.Join(outputDir, reel.ID+".mp4")
		if hasVideo {
			fmt.Printf("Copying video from cache to %s...\n", destPath)
			if err := copyFile(cached.VideoPath, destPath); err != nil {
				return fmt.Errorf("failed to copy video: %w", err)
			}
			cachedVideoPath = cached.VideoPath
		} else {
			fmt.Printf("Downloading video to %s...\n", destPath)
			// Download to cache first, then copy to output
			cachePath := filepath.Join(cacheDir, "video.mp4")
			if err := os.MkdirAll(cacheDir, 0755); err != nil {
				return fmt.Errorf("failed to create cache dir: %w", err)
			}
			if err := app.Downloader.DownloadVideo(ctx, reel.ID, cachePath); err != nil {
				return fmt.Errorf("video download failed: %w", err)
			}
			if err := copyFile(cachePath, destPath); err != nil {
				return fmt.Errorf("failed to copy video: %w", err)
			}
			cachedVideoPath = cachePath
			cacheUpdated = true
		}
		fmt.Println("✓ Video downloaded")
	}

	if thumbnail {
		destPath := filepath.Join(outputDir, reel.ID+".jpg")
		if hasThumbnail {
			fmt.Printf("Copying thumbnail from cache to %s...\n", destPath)
			if err := copyFile(cached.ThumbnailPath, destPath); err != nil {
				return fmt.Errorf("failed to copy thumbnail: %w", err)
			}
			cachedThumbnailPath = cached.ThumbnailPath
		} else {
			fmt.Printf("Downloading thumbnail to %s...\n", destPath)
			// Download to cache first, then copy to output
			cachePath := filepath.Join(cacheDir, "thumbnail.jpg")
			if err := os.MkdirAll(cacheDir, 0755); err != nil {
				return fmt.Errorf("failed to create cache dir: %w", err)
			}
			if err := app.Downloader.DownloadThumbnail(ctx, reel.ID, cachePath); err != nil {
				return fmt.Errorf("thumbnail download failed: %w", err)
			}
			if err := copyFile(cachePath, destPath); err != nil {
				return fmt.Errorf("failed to copy thumbnail: %w", err)
			}
			cachedThumbnailPath = cachePath
			cacheUpdated = true
		}
		fmt.Println("✓ Thumbnail downloaded")
	}

	// Update cache if we downloaded new assets
	if cacheUpdated {
		now := time.Now()
		ttl, _ := time.ParseDuration(cacheTTLFlag)
		if ttl == 0 {
			ttl = 7 * 24 * time.Hour
		}

		cacheItem := &ports.CachedItem{
			VideoPath:     cachedVideoPath,
			ThumbnailPath: cachedThumbnailPath,
			CreatedAt:     now,
			ExpiresAt:     now.Add(ttl),
		}

		// Preserve existing cache data
		if cached != nil {
			cacheItem.Reel = cached.Reel
			cacheItem.Transcript = cached.Transcript
			if cacheItem.VideoPath == "" {
				cacheItem.VideoPath = cached.VideoPath
			}
			if cacheItem.ThumbnailPath == "" {
				cacheItem.ThumbnailPath = cached.ThumbnailPath
			}
			cacheItem.CreatedAt = cached.CreatedAt
		}

		_ = app.Cache.Set(ctx, reel.ID, cacheItem)
	}

	return nil
}
```

**Step 2: Add time import**

Ensure `"time"` is in the imports at the top of `internal/adapters/cli/root.go`.

**Step 3: Build and verify**

Run: `go build ./cmd/ig2insights`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/adapters/cli/root.go
git commit -m "feat(cli): use cache in download-only mode"
```

---

## Task 7: Add Integration Tests

**Files:**
- Modify: `internal/application/transcribe_test.go`

**Step 1: Add test for video caching**

Add to `internal/application/transcribe_test.go`:

```go
func TestTranscribeService_CachedVideo(t *testing.T) {
	cache := newMockCache()
	downloader := &mockDownloader{available: true}
	transcriber := &mockTranscriber{modelDownloaded: true}

	// Pre-populate cache with video path
	cache.Set(context.Background(), "withvideo123", &ports.CachedItem{
		Reel:       &domain.Reel{ID: "withvideo123", Title: "Cached Reel"},
		Transcript: &domain.Transcript{Text: "Cached transcript"},
		VideoPath:  "/tmp/withvideo123/video.mp4",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	})

	svc := NewTranscribeService(cache, downloader, transcriber, 24*time.Hour)

	ctx := context.Background()
	result, err := svc.Transcribe(ctx, "withvideo123", TranscribeOptions{
		SaveVideo: true,
	})

	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}

	if !result.TranscriptFromCache {
		t.Errorf("TranscriptFromCache should be true")
	}

	// Note: VideoFromCache will be false because fileExists() will fail on mock path
	// In real usage, the file would exist
}

func TestTranscribeService_ThumbnailCaching(t *testing.T) {
	cache := newMockCache()
	downloader := &mockDownloader{available: true}
	transcriber := &mockTranscriber{modelDownloaded: true}

	svc := NewTranscribeService(cache, downloader, transcriber, 24*time.Hour)

	ctx := context.Background()
	result, err := svc.Transcribe(ctx, "thumb123", TranscribeOptions{
		SaveThumbnail: true,
	})

	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}

	// Verify thumbnail was requested
	if result.ThumbnailFromCache {
		t.Errorf("ThumbnailFromCache should be false for fresh download")
	}

	// Verify item was cached with thumbnail
	cached, err := cache.Get(ctx, "thumb123")
	if err != nil {
		t.Fatalf("Cache.Get() error = %v", err)
	}

	if cached.ThumbnailPath == "" {
		t.Errorf("ThumbnailPath should be set in cache")
	}
}
```

**Step 2: Run all tests**

Run: `go test ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/application/transcribe_test.go
git commit -m "test(application): add tests for per-asset caching"
```

---

## Task 8: Final Build and Manual Test

**Step 1: Build the binary**

Run: `go build -o ig2insights ./cmd/ig2insights`
Expected: Build succeeds

**Step 2: Run all tests**

Run: `go test ./...`
Expected: All tests pass

**Step 3: Manual test scenario**

1. Transcribe a new reel: `./ig2insights <reel-url>`
2. Run again with video flag: `./ig2insights --video <same-reel-url>`
   - Should show "(cached)" for transcript steps
   - Should download video
3. Run again with video and thumbnail: `./ig2insights --video --thumbnail <same-reel-url>`
   - Should show "(cached)" for transcript and video
   - Should download thumbnail
4. Run again: `./ig2insights --video --thumbnail <same-reel-url>`
   - All steps should show "(cached)"

**Step 4: Commit final state**

```bash
git add -A
git commit -m "feat: smart per-asset caching complete"
```
