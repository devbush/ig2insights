# Audio/Video Download Fix Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `--audio` flag, fix video download format, and cache all downloaded assets.

**Architecture:** Update data model to separate AudioPath (WAV for transcription) from VideoPath (MP4 video). Rename `Download()` to `DownloadAudio()` for clarity. Fix `DownloadVideo()` to use proper yt-dlp format with ffmpeg merging.

**Tech Stack:** Go, yt-dlp, ffmpeg

---

### Task 1: Update Cache Data Model

**Files:**
- Modify: `internal/ports/cache.go:11-18`
- Modify: `internal/adapters/cache/store.go:26-33`
- Modify: `internal/adapters/cache/store.go:63-70`
- Modify: `internal/adapters/cache/store.go:79-86`

**Step 1: Update CachedItem in ports/cache.go**

Change `VideoPath` to `AudioPath` and add new `VideoPath`:

```go
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
```

**Step 2: Update metaFile struct in store.go**

```go
type metaFile struct {
	Reel          *domain.Reel       `json:"reel"`
	Transcript    *domain.Transcript `json:"transcript"`
	AudioPath     string             `json:"audio_path"`
	VideoPath     string             `json:"video_path"`
	ThumbnailPath string             `json:"thumbnail_path"`
	CreatedAt     time.Time          `json:"created_at"`
	ExpiresAt     time.Time          `json:"expires_at"`
}
```

**Step 3: Update Get() return in store.go**

```go
return &ports.CachedItem{
	Reel:          meta.Reel,
	Transcript:    meta.Transcript,
	AudioPath:     meta.AudioPath,
	VideoPath:     meta.VideoPath,
	ThumbnailPath: meta.ThumbnailPath,
	CreatedAt:     meta.CreatedAt,
	ExpiresAt:     meta.ExpiresAt,
}, nil
```

**Step 4: Update Set() in store.go**

```go
meta := metaFile{
	Reel:          item.Reel,
	Transcript:    item.Transcript,
	AudioPath:     item.AudioPath,
	VideoPath:     item.VideoPath,
	ThumbnailPath: item.ThumbnailPath,
	CreatedAt:     item.CreatedAt,
	ExpiresAt:     item.ExpiresAt,
}
```

**Step 5: Run tests**

Run: `go test ./internal/adapters/cache/...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/ports/cache.go internal/adapters/cache/store.go
git commit -m "refactor(cache): add AudioPath, separate from VideoPath"
```

---

### Task 2: Update Downloader Interface and Implementation

**Files:**
- Modify: `internal/ports/downloader.go:9-18`
- Modify: `internal/adapters/ytdlp/downloader.go:131-230`
- Modify: `internal/adapters/ytdlp/downloader.go:663-695`

**Step 1: Update DownloadResult and interface in ports/downloader.go**

```go
// DownloadResult contains the downloaded audio info
type DownloadResult struct {
	AudioPath string       // WAV audio file for transcription
	Reel      *domain.Reel // Populated with metadata from download
}

// VideoDownloader handles video download from Instagram
type VideoDownloader interface {
	// DownloadAudio extracts audio from a reel, returns path to WAV file
	DownloadAudio(ctx context.Context, reelID string, destDir string) (*DownloadResult, error)

	// DownloadVideo downloads the full video file (MP4 with audio)
	DownloadVideo(ctx context.Context, reelID string, destPath string) error
```

**Step 2: Rename Download() to DownloadAudio() in downloader.go**

Change function signature at line 131:

```go
func (d *Downloader) DownloadAudio(ctx context.Context, reelID string, destDir string) (*ports.DownloadResult, error) {
```

**Step 3: Update return statements in DownloadAudio()**

At line 192 (error fallback):
```go
return &ports.DownloadResult{
	AudioPath: matches[0],
	Reel: &domain.Reel{
		ID:        reelID,
		FetchedAt: time.Now(),
	},
}, nil
```

At line 218 (success):
```go
return &ports.DownloadResult{
	AudioPath: audioPath,
	Reel: &domain.Reel{
		ID:              reelID,
		URL:             url,
		Author:          info.Uploader,
		Title:           info.Title,
		DurationSeconds: int(info.Duration),
		ViewCount:       info.ViewCount,
		FetchedAt:       time.Now(),
	},
}, nil
```

**Step 4: Fix DownloadVideo() format and add ffmpeg check**

Replace the entire DownloadVideo function:

```go
func (d *Downloader) DownloadVideo(ctx context.Context, reelID string, destPath string) error {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return fmt.Errorf("yt-dlp not found")
	}

	// Check for ffmpeg (needed for merging video+audio)
	if !d.IsFFmpegAvailable() {
		return domain.ErrFFmpegNotFound
	}

	url := buildReelURL(reelID)

	// Download best video+audio combined, fallback to best single stream
	args := []string{
		"--no-warnings",
		"-f", "bv*+ba/b",
		"--merge-output-format", "mp4",
		"-o", destPath,
		url,
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "Private video") || strings.Contains(stderr, "Video unavailable") {
				return domain.ErrReelNotFound
			}
			if strings.Contains(stderr, "rate") || strings.Contains(stderr, "429") {
				return domain.ErrRateLimited
			}
			return fmt.Errorf("video download failed: %s", strings.TrimSpace(stderr))
		}
		return fmt.Errorf("video download failed: %w", err)
	}

	return nil
}
```

**Step 5: Run tests**

Run: `go test ./internal/adapters/ytdlp/...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/ports/downloader.go internal/adapters/ytdlp/downloader.go
git commit -m "refactor(downloader): rename Download to DownloadAudio, fix video format"
```

---

### Task 3: Update Transcribe Service

**Files:**
- Modify: `internal/application/transcribe.go:14-35`
- Modify: `internal/application/transcribe.go:60-170`

**Step 1: Update TranscribeOptions**

```go
// TranscribeOptions configures the transcription
type TranscribeOptions struct {
	Model         string
	Format        string // text, srt, json
	NoCache       bool
	Language      string // empty defaults to "auto"
	SaveAudio     bool   // Save WAV audio file
	SaveVideo     bool   // Save MP4 video file
	SaveThumbnail bool
	OutputDir     string // directory for outputs
}
```

**Step 2: Update TranscribeResult**

```go
// TranscribeResult contains the transcription result
type TranscribeResult struct {
	Reel          *domain.Reel
	Transcript    *domain.Transcript
	AudioPath     string // WAV audio path
	VideoPath     string // MP4 video path
	ThumbnailPath string

	// Per-asset cache status
	TranscriptFromCache bool
	AudioFromCache      bool
	VideoFromCache      bool
	ThumbnailFromCache  bool
}
```

**Step 3: Update Transcribe() method**

Replace the entire Transcribe method:

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
	hasAudio := cached != nil && cached.AudioPath != "" && fileExists(cached.AudioPath)
	hasVideo := cached != nil && cached.VideoPath != "" && fileExists(cached.VideoPath)
	hasThumbnail := cached != nil && cached.ThumbnailPath != "" && fileExists(cached.ThumbnailPath)

	needTranscript := !hasTranscript
	needAudio := (opts.SaveAudio || needTranscript) && !hasAudio
	needVideo := opts.SaveVideo && !hasVideo
	needThumbnail := opts.SaveThumbnail && !hasThumbnail

	// Use cached reel metadata if available
	var reel *domain.Reel
	if cached != nil && cached.Reel != nil {
		reel = cached.Reel
	}

	// Download audio if needed for transcription OR if user requested audio
	var audioPath string
	if needAudio {
		downloadResult, err := s.downloader.DownloadAudio(ctx, reelID, cacheDir)
		if err != nil {
			return nil, err
		}
		audioPath = downloadResult.AudioPath
		reel = downloadResult.Reel
	} else if hasAudio {
		audioPath = cached.AudioPath
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
		transcript, err = s.transcriber.Transcribe(ctx, audioPath, ports.TranscribeOpts{
			Model:    model,
			Language: language,
		})
		if err != nil {
			return nil, err
		}
	} else {
		transcript = cached.Transcript
	}

	// Download video if needed (separate from audio)
	var videoPath string
	if needVideo {
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return nil, err
		}
		vidPath := filepath.Join(cacheDir, "video.mp4")
		if err := s.downloader.DownloadVideo(ctx, reelID, vidPath); err != nil {
			// Non-fatal - continue without video
		} else {
			videoPath = vidPath
		}
	} else if hasVideo {
		videoPath = cached.VideoPath
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
		AudioPath:     audioPath,
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
	result.AudioPath = audioPath
	result.VideoPath = videoPath
	result.ThumbnailPath = thumbnailPath
	result.TranscriptFromCache = hasTranscript
	result.AudioFromCache = hasAudio && (opts.SaveAudio || needTranscript)
	result.VideoFromCache = hasVideo && opts.SaveVideo
	result.ThumbnailFromCache = hasThumbnail && opts.SaveThumbnail

	return result, nil
}
```

**Step 4: Commit (tests will fail until Task 4)**

```bash
git add internal/application/transcribe.go
git commit -m "refactor(transcribe): add SaveAudio, separate audio/video caching"
```

---

### Task 4: Update Tests

**Files:**
- Modify: `internal/application/transcribe_test.go`

**Step 1: Update mockDownloader.Download to DownloadAudio**

```go
func (m *mockDownloader) DownloadAudio(ctx context.Context, reelID string, destDir string) (*ports.DownloadResult, error) {
	return &ports.DownloadResult{
		AudioPath: destDir + "/audio.wav",
		Reel: &domain.Reel{
			ID:        reelID,
			Title:     "Test Reel",
			Author:    "testuser",
			FetchedAt: time.Now(),
		},
	}, nil
}
```

**Step 2: Update mockDownloaderWithError.Download to DownloadAudio**

```go
func (m *mockDownloaderWithError) DownloadAudio(ctx context.Context, reelID string, destDir string) (*ports.DownloadResult, error) {
	return nil, domain.ErrReelNotFound
}
```

**Step 3: Update TestTranscribeService_PartialCache_TranscriptOnly**

Change `VideoPath` references to `AudioPath`:

```go
func TestTranscribeService_PartialCache_TranscriptOnly(t *testing.T) {
	cache := newMockCache()
	downloader := &mockDownloader{available: true}
	transcriber := &mockTranscriber{modelDownloaded: true}

	// Pre-populate cache with transcript only (no audio or video)
	cache.Set(context.Background(), "partial123", &ports.CachedItem{
		Reel:       &domain.Reel{ID: "partial123", Title: "Cached Reel"},
		Transcript: &domain.Transcript{Text: "Cached transcript"},
		AudioPath:  "", // No audio cached
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
}
```

**Step 4: Run tests**

Run: `go test ./internal/application/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/application/transcribe_test.go
git commit -m "test(application): update mocks for DownloadAudio rename"
```

---

### Task 5: Update CLI Flags

**Files:**
- Modify: `internal/adapters/cli/root.go:19-31`
- Modify: `internal/adapters/cli/root.go:55-57`

**Step 1: Add audioFlag variable**

```go
var (
	// Global flags
	formatFlag    string
	modelFlag     string
	cacheTTLFlag  string
	noCacheFlag   bool
	dirFlag       string
	nameFlag      string
	quietFlag     bool
	languageFlag  string
	audioFlag     bool
	videoFlag     bool
	thumbnailFlag bool
)
```

**Step 2: Add --audio flag definition**

After line 55, add:

```go
rootCmd.PersistentFlags().BoolVar(&audioFlag, "audio", false, "Download the audio file (WAV)")
rootCmd.PersistentFlags().BoolVar(&videoFlag, "video", false, "Download the video file (MP4)")
```

**Step 3: Commit**

```bash
git add internal/adapters/cli/root.go
git commit -m "feat(cli): add --audio flag"
```

---

### Task 6: Update CLI Interactive Menu and Download Logic

**Files:**
- Modify: `internal/adapters/cli/root.go:111-156` (runTranscribeInteractive)
- Modify: `internal/adapters/cli/root.go:158-291` (runDownloadOnly)

**Step 1: Update runTranscribeInteractive checkbox options**

```go
func runTranscribeInteractive() error {
	// Show output options
	checkboxOpts := []tui.CheckboxOption{
		{Label: "Transcript", Value: "transcript", Checked: true},
		{Label: "Download audio (WAV)", Value: "audio", Checked: false},
		{Label: "Download video (MP4)", Value: "video", Checked: false},
		{Label: "Download thumbnail", Value: "thumbnail", Checked: false},
	}

	selected, err := tui.RunCheckbox("What would you like to get?", checkboxOpts)
	if err != nil {
		return err
	}
	if selected == nil {
		fmt.Println("Cancelled")
		return nil
	}

	// Parse selections
	wantTranscript := false
	wantAudio := false
	wantVideo := false
	wantThumbnail := false
	for _, s := range selected {
		switch s {
		case "transcript":
			wantTranscript = true
		case "audio":
			wantAudio = true
		case "video":
			wantVideo = true
		case "thumbnail":
			wantThumbnail = true
		}
	}

	// Get reel URL
	fmt.Print("Enter reel URL or ID: ")
	var input string
	fmt.Scanln(&input)

	// Set flags based on selections
	audioFlag = wantAudio
	videoFlag = wantVideo
	thumbnailFlag = wantThumbnail

	if wantTranscript {
		return runTranscribe(input)
	}

	// Download only (no transcription)
	return runDownloadOnly(input, wantAudio, wantVideo, wantThumbnail)
}
```

**Step 2: Update runDownloadOnly signature and add audio handling**

```go
func runDownloadOnly(input string, audio, video, thumbnail bool) error {
	app, err := GetApp()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	reel, err := domain.ParseReelInput(input)
	if err != nil {
		return err
	}

	ctx := context.Background()
	outputDir := dirFlag
	if outputDir == "" {
		outputDir = reel.ID
	}
	baseName := nameFlag
	if baseName == "" {
		baseName = reel.ID
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	cached, _ := app.Cache.Get(ctx, reel.ID)
	hasAudio := cached != nil && cached.AudioPath != "" && fileExists(cached.AudioPath)
	hasVideo := cached != nil && cached.VideoPath != "" && fileExists(cached.VideoPath)
	hasThumbnail := cached != nil && cached.ThumbnailPath != "" && fileExists(cached.ThumbnailPath)

	cacheDir := app.Cache.GetCacheDir(reel.ID)
	cacheUpdated := false
	var cachedAudioPath, cachedVideoPath, cachedThumbnailPath string

	if audio {
		destPath := filepath.Join(outputDir, baseName+".wav")
		if hasAudio {
			if !quietFlag {
				fmt.Printf("Copying audio from cache to %s...\n", destPath)
			}
			if err := copyFile(cached.AudioPath, destPath); err != nil {
				return fmt.Errorf("failed to copy audio: %w", err)
			}
			cachedAudioPath = cached.AudioPath
		} else {
			if !quietFlag {
				fmt.Printf("Downloading audio to %s...\n", destPath)
			}
			if err := os.MkdirAll(cacheDir, 0755); err != nil {
				return fmt.Errorf("failed to create cache dir: %w", err)
			}
			downloadResult, err := app.Downloader.DownloadAudio(ctx, reel.ID, cacheDir)
			if err != nil {
				return fmt.Errorf("audio download failed: %w", err)
			}
			if err := copyFile(downloadResult.AudioPath, destPath); err != nil {
				return fmt.Errorf("failed to copy audio: %w", err)
			}
			cachedAudioPath = downloadResult.AudioPath
			cacheUpdated = true
		}
		if !quietFlag {
			fmt.Println("Audio downloaded")
		}
	}

	if video {
		destPath := filepath.Join(outputDir, baseName+".mp4")
		if hasVideo {
			if !quietFlag {
				fmt.Printf("Copying video from cache to %s...\n", destPath)
			}
			if err := copyFile(cached.VideoPath, destPath); err != nil {
				return fmt.Errorf("failed to copy video: %w", err)
			}
			cachedVideoPath = cached.VideoPath
		} else {
			if !quietFlag {
				fmt.Printf("Downloading video to %s...\n", destPath)
			}
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
		if !quietFlag {
			fmt.Println("Video downloaded")
		}
	}

	if thumbnail {
		destPath := filepath.Join(outputDir, baseName+".jpg")
		if hasThumbnail {
			if !quietFlag {
				fmt.Printf("Copying thumbnail from cache to %s...\n", destPath)
			}
			if err := copyFile(cached.ThumbnailPath, destPath); err != nil {
				return fmt.Errorf("failed to copy thumbnail: %w", err)
			}
			cachedThumbnailPath = cached.ThumbnailPath
		} else {
			if !quietFlag {
				fmt.Printf("Downloading thumbnail to %s...\n", destPath)
			}
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
		if !quietFlag {
			fmt.Println("Thumbnail downloaded")
		}
	}

	if cacheUpdated {
		now := time.Now()
		ttl, _ := time.ParseDuration(cacheTTLFlag)
		if ttl == 0 {
			ttl = 7 * 24 * time.Hour
		}

		cacheItem := &ports.CachedItem{
			AudioPath:     cachedAudioPath,
			VideoPath:     cachedVideoPath,
			ThumbnailPath: cachedThumbnailPath,
			CreatedAt:     now,
			ExpiresAt:     now.Add(ttl),
		}

		if cached != nil {
			cacheItem.Reel = cached.Reel
			cacheItem.Transcript = cached.Transcript
			if cacheItem.AudioPath == "" {
				cacheItem.AudioPath = cached.AudioPath
			}
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

**Step 3: Commit**

```bash
git add internal/adapters/cli/root.go
git commit -m "feat(cli): add audio to interactive menu and download-only"
```

---

### Task 7: Update CLI Transcribe Function

**Files:**
- Modify: `internal/adapters/cli/root.go:305-535` (runTranscribe)

**Step 1: Update cache checks to include audio**

Around line 324-326, add hasAudio:

```go
hasTranscript := cached != nil && cached.Transcript != nil
hasAudio := cached != nil && cached.AudioPath != "" && fileExists(cached.AudioPath)
hasVideo := cached != nil && cached.VideoPath != "" && fileExists(cached.VideoPath)
hasThumbnail := cached != nil && cached.ThumbnailPath != "" && fileExists(cached.ThumbnailPath)
```

**Step 2: Add audio step tracking**

After line 344, add audioStepIdx:

```go
audioStepIdx := -1
videoStepIdx := -1
thumbStepIdx := -1

if audioFlag {
	audioStepIdx = len(steps)
	if hasAudio {
		steps = append(steps, "Saving audio (cached)")
	} else {
		steps = append(steps, "Saving audio")
	}
}
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
```

**Step 3: Update TranscribeOptions call**

```go
result, err := app.TranscribeSvc.Transcribe(ctx, reel.ID, application.TranscribeOptions{
	Model:         model,
	NoCache:       noCacheFlag,
	Language:      languageFlag,
	SaveAudio:     audioFlag,
	SaveVideo:     videoFlag,
	SaveThumbnail: thumbnailFlag,
})
```

**Step 4: Add audio output handling**

After line 469, before the video step:

```go
// Step: Save audio (if requested)
if audioFlag && audioStepIdx >= 0 {
	if result.AudioFromCache {
		progress.CompleteStep(audioStepIdx)
	} else {
		progress.StartStep(audioStepIdx)
	}

	audioPath := filepath.Join(outputDir, baseName+".wav")
	if result.AudioPath != "" {
		if err := copyFile(result.AudioPath, audioPath); err != nil {
			progress.FailStep(audioStepIdx, err.Error())
		} else {
			progress.CompleteStep(audioStepIdx)
			outputs["Audio"] = audioPath
		}
	} else {
		progress.FailStep(audioStepIdx, "no audio available")
	}
}
```

**Step 5: Build and run tests**

Run: `go build ./cmd/ig2insights && go test ./...`
Expected: Build succeeds, all tests pass

**Step 6: Commit**

```bash
git add internal/adapters/cli/root.go
git commit -m "feat(cli): add audio output handling in transcribe"
```

---

### Task 8: Final Verification

**Step 1: Build**

Run: `go build -o ig2insights ./cmd/ig2insights`
Expected: Success

**Step 2: Run all tests**

Run: `go test ./...`
Expected: All pass

**Step 3: Manual test (optional)**

Run: `./ig2insights --audio --video --thumbnail <reel-url>`
Expected:
- Output directory contains `.wav`, `.mp4`, `.jpg`, `.txt` files
- Second run uses cache (faster)
- MP4 plays with both video and audio

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat: audio/video download fix complete"
```
