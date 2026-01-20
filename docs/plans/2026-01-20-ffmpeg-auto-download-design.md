# FFmpeg Auto-Download Design

Add auto-download functionality for ffmpeg on Windows, needed for audio extraction from Instagram reels.

## Why FFmpeg is Needed

- Instagram provides audio as AAC (m4a container)
- whisper.cpp only supports: flac, mp3, ogg, wav
- yt-dlp uses ffmpeg to extract and convert audio
- No pure-Go AAC decoder available (CGO options defeat zero-config goal)

## Download & Extract Flow

**Search order (all platforms):**
1. System PATH → use existing installation
2. `~/.ig2insights/bin/ffmpeg[.exe]` → use bundled
3. Not found → platform-specific behavior

**Platform behavior when not found:**

| Platform | Behavior |
|----------|----------|
| **Windows** | Auto-download `ffmpeg-release-essentials.7z` (~31MB), extract `ffmpeg.exe` + `ffprobe.exe` → `~/.ig2insights/bin/` |
| **macOS** | Show instructions: `brew install ffmpeg` |
| **Linux** | Show instructions: `sudo apt install ffmpeg` / `sudo dnf install ffmpeg` |

## Download Source

**URL:** `https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.7z`
- ~31MB download (7z compressed)
- Extracts to ~100MB
- Contains ffmpeg.exe + ffprobe.exe in `ffmpeg-X.X.X-essentials_build/bin/`

**7z extraction:** Use [github.com/bodgit/sevenzip](https://github.com/bodgit/sevenzip) - pure Go, no CGO.

## Integration Points

**Three places check/install ffmpeg:**

1. **`ig2insights <reel>`** - Auto-install before downloading
2. **`ig2insights deps status`** - Show ffmpeg status
3. **`ig2insights deps install`** - Install all dependencies

**New methods on Downloader:**

```go
IsFFmpegAvailable() bool
GetFFmpegPath() string
InstallFFmpeg(ctx context.Context, progress func(downloaded, total int64)) error
FFmpegInstructions() string
extractFFmpegFrom7z(archivePath, binDir string) error
```

## CLI Changes

**Updated transcribe flow:**
1. Check yt-dlp → auto-install if missing
2. Check whisper.cpp → auto-install (Windows) or show instructions
3. Check ffmpeg → auto-install (Windows) or show instructions ← NEW
4. Check model → auto-download if missing
5. Download audio → yt-dlp extracts audio
6. Transcribe → whisper.cpp processes audio

**Updated `deps status` output:**
```
Dependency Status:

  yt-dlp:        installed (path)
  whisper.cpp:   installed (path)
  ffmpeg:        installed (path)  ← NEW
  whisper models: 1/5 downloaded
```

## Files to Modify

- `internal/adapters/ytdlp/downloader.go` - Add ffmpeg methods
- `internal/adapters/cli/root.go` - Add ffmpeg check before transcribe
- `internal/adapters/cli/deps.go` - Add ffmpeg to status/install
- `internal/ports/downloader.go` - Add interface methods
- `internal/domain/errors.go` - Add ErrFFmpegNotFound
- `go.mod` - Add `github.com/bodgit/sevenzip` dependency
