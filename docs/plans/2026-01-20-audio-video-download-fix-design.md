# Audio/Video Download Fix Design

## Problem

1. `--video` flag downloads MP4 but there's no actual video (yt-dlp `-f best` selects wrong stream on Instagram)
2. No `--audio` flag to download audio separately
3. Video files not cached (re-downloaded each time)

## Solution

Add `--audio` flag, fix video download format, cache all downloaded assets.

## Data Model Changes

### Cache Structure

```go
// ports/cache.go
type CachedItem struct {
    Reel          *domain.Reel
    Transcript    *domain.Transcript
    AudioPath     string    // WAV audio (for transcription and --audio)
    VideoPath     string    // MP4 video (for --video)
    ThumbnailPath string
    CreatedAt     time.Time
    ExpiresAt     time.Time
}
```

### Download Result

```go
// ports/downloader.go
type DownloadResult struct {
    AudioPath string       // WAV file path
    Reel      *domain.Reel
}
```

## Interface Changes

### VideoDownloader Interface

Rename `Download` to `DownloadAudio` for clarity:

```go
type VideoDownloader interface {
    DownloadAudio(ctx context.Context, reelID string, destDir string) (*DownloadResult, error)
    DownloadVideo(ctx context.Context, reelID string, destPath string) error
    DownloadThumbnail(ctx context.Context, reelID string, destPath string) error
    // ... other methods unchanged
}
```

## yt-dlp Video Download Fix

Change `DownloadVideo()` format from:

```go
args := []string{
    "--no-warnings",
    "-f", "best",
    "-o", destPath,
    url,
}
```

To:

```go
args := []string{
    "--no-warnings",
    "-f", "bv*+ba/b",
    "--merge-output-format", "mp4",
    "-o", destPath,
    url,
}
```

Add ffmpeg availability check, return `domain.ErrFFmpegNotFound` if not available.

## Transcribe Service Changes

### Options

```go
type TranscribeOptions struct {
    Model         string
    Format        string
    NoCache       bool
    Language      string
    SaveAudio     bool   // NEW
    SaveVideo     bool
    SaveThumbnail bool
    OutputDir     string
}
```

### Result

```go
type TranscribeResult struct {
    Reel          *domain.Reel
    Transcript    *domain.Transcript
    AudioPath     string  // Renamed from VideoPath
    VideoPath     string  // NEW
    ThumbnailPath string

    TranscriptFromCache bool
    AudioFromCache      bool  // Renamed from VideoFromCache
    VideoFromCache      bool  // NEW
    ThumbnailFromCache  bool
}
```

### Caching Logic

- Audio is always downloaded for transcription, always cached
- Video downloaded only when `SaveVideo=true`, cached when downloaded
- Check cache for each asset independently before downloading

## CLI Changes

### New Flag

```go
audioFlag bool

rootCmd.PersistentFlags().BoolVar(&audioFlag, "audio", false, "Download the audio file (WAV)")
rootCmd.PersistentFlags().BoolVar(&videoFlag, "video", false, "Download the video file (MP4)")
```

### Output Handling

- `--audio`: copy `result.AudioPath` to `{outputDir}/{name}.wav`
- `--video`: copy `result.VideoPath` to `{outputDir}/{name}.mp4`

### Interactive Menu

Add audio option:

```go
{Label: "Download audio (WAV)", Value: "audio", Checked: false},
```

### runDownloadOnly Signature

```go
func runDownloadOnly(input string, audio, video, thumbnail bool) error
```

## Files to Modify

| File | Changes |
|------|---------|
| `internal/ports/cache.go` | Add `AudioPath`, keep `VideoPath` for MP4 |
| `internal/adapters/cache/store.go` | Update `metaFile` struct to match |
| `internal/ports/downloader.go` | Rename `VideoPath` → `AudioPath`, rename `Download` → `DownloadAudio` |
| `internal/adapters/ytdlp/downloader.go` | Rename method, fix format, add ffmpeg check |
| `internal/application/transcribe.go` | Add `SaveAudio`, rename fields, update caching logic |
| `internal/adapters/cli/root.go` | Add `--audio` flag, update menu, update output logic |
| `internal/application/transcribe_test.go` | Update mocks and tests |

## Verification

1. Build: `go build ./cmd/ig2insights`
2. Test: `go test ./...`
3. Manual test:
   - `./ig2insights --audio --video --thumbnail <reel-url>`
   - Verify output contains `.wav`, `.mp4`, `.jpg`, `.txt`
   - Run again, verify cache is used
   - Play MP4, confirm it has video and audio
