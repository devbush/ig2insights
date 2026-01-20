# Enhanced Transcription: Progress Display & Download Options

## Overview

Add a step-based progress display with spinners, optional video/thumbnail downloads, and fix the language auto-detection to preserve original audio language.

## New CLI Flags

```
--language, -l    Language code (default: "auto" - detect and preserve original)
--video           Download the original video file
--thumbnail       Download the video thumbnail
--download-dir    Directory for downloaded assets (default: same as output)
```

### Examples

```bash
# Transcribe, keeping original French
ig2insights https://instagram.com/reel/ABC123

# Transcribe + download video and thumbnail
ig2insights --video --thumbnail https://instagram.com/reel/ABC123

# Force English transcription (hint to whisper)
ig2insights --language en https://instagram.com/reel/ABC123

# Download everything to specific folder
ig2insights --video --thumbnail --download-dir ./downloads https://instagram.com/reel/ABC123
```

### File Naming Convention

- Transcript: `<reel-id>.txt` (or `.srt`/`.json` based on `--format`)
- Video: `<reel-id>.mp4`
- Thumbnail: `<reel-id>.jpg`

When `-o output.txt` is specified, assets go alongside: `output.mp4`, `output.jpg`

## Interactive Menu Changes

### Main Menu (unchanged)

```
┌─────────────────────────────────┐
│  ig2insights                    │
├─────────────────────────────────┤
│  > Transcribe a single reel     │
│    Browse an account's reels    │
│    Manage cache                 │
│    Settings                     │
└─────────────────────────────────┘
```

### New Checkbox Selection (after selecting "Transcribe a single reel")

```
┌─────────────────────────────────┐
│  What would you like to get?    │
├─────────────────────────────────┤
│  [x] Transcript                 │
│  [ ] Download video             │
│  [ ] Download thumbnail         │
│                                 │
│  Press Enter to continue        │
└─────────────────────────────────┘
```

- Transcript is checked by default (but can be unchecked)
- At least one option must be selected
- After confirmation, prompts for reel URL

## Progress Display

### Step-based Hybrid Progress

```
[1/4] Checking dependencies... ✓
[2/4] Downloading video... 45.2% (2.1 MB / 4.7 MB)
[2/4] Downloading video... ✓
[3/4] Extracting audio... ⠹
[3/4] Extracting audio... ✓
[4/4] Transcribing... ⠹
[4/4] Transcribing... ✓

✓ Complete! Saved to reel-ABC123.txt
```

### Steps

1. **Checking dependencies** - Quick check, instant ✓ or error
2. **Downloading video** - Real progress bar (yt-dlp provides size info)
3. **Extracting audio** - Spinner (ffmpeg, usually fast)
4. **Transcribing** - Spinner (whisper.cpp, no progress available)

### When Downloads Enabled

```
[5/6] Saving video... ✓
[6/6] Downloading thumbnail... ✓
```

### Spinner Animation

Cycles through: `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`

## Code Changes

| File | Changes |
|------|---------|
| `internal/ports/transcriber.go` | Already has `Language` in `TranscribeOpts` - no change |
| `internal/ports/downloader.go` | Add `DownloadThumbnail(ctx, reelID, destPath)` method |
| `internal/application/transcribe.go` | Add `Language`, `SaveVideo`, `SaveThumbnail` to `TranscribeOptions`; update orchestration |
| `internal/adapters/whisper/transcriber.go` | Pass `-l auto` by default |
| `internal/adapters/ytdlp/downloader.go` | Add thumbnail download using yt-dlp's `--write-thumbnail` |
| `internal/adapters/cli/root.go` | Add new flags; wire up options |
| `internal/adapters/cli/tui/checkbox.go` | New file - checkbox selection component |
| `internal/adapters/cli/tui/progress.go` | New file - step-based progress display |

### Key Fix: Language Auto-Detection

In `internal/adapters/whisper/transcriber.go`, change the args to always pass language:

```go
args := []string{
    "-m", t.modelPath(model),
    "-f", videoPath,
    "-of", outputBase,
    "-oj",
    "-l", opts.Language,  // Always pass language (default "auto")
}
```

And ensure `opts.Language` defaults to `"auto"` in the application layer.

## Edge Cases

| Scenario | Behavior |
|----------|----------|
| Only thumbnail selected | Download thumbnail only, skip whisper |
| Only video selected | Download and save video, skip transcription |
| `--download-dir` doesn't exist | Create directory automatically |
| Thumbnail not available | Warn but don't fail; continue with other outputs |
| Video already in cache | Copy from cache instead of re-downloading |
| No options selected in interactive mode | Show error "Please select at least one option" |

## Quiet Mode (`-q`)

- Suppresses all progress output
- Still shows errors
- Final "Complete!" message is also suppressed

## Output Summary (non-quiet mode)

```
✓ Complete!
  Transcript: ./reel-ABC123.txt
  Video: ./reel-ABC123.mp4
  Thumbnail: ./reel-ABC123.jpg
```
