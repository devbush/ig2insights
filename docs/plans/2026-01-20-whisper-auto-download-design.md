# Whisper.cpp Auto-Download Design

Add auto-download functionality for whisper.cpp binary, similar to yt-dlp.

## Download & Extract Flow

When whisper.cpp binary is needed:
1. Download `whisper-bin-x64.zip` to a temp file
2. Extract only `main.exe` from the zip
3. Save as `~/.ig2insights/bin/whisper.exe`
4. Delete temp zip file

## Integration Points

Three places check/install whisper.cpp:

1. **`ig2insights <reel>`** - Auto-install before transcribing (like yt-dlp)
2. **`ig2insights deps status`** - Show whisper.cpp status
3. **`ig2insights deps install`** - Install both yt-dlp and whisper.cpp

The Transcriber gets three new methods (matching Downloader's API):
- `IsAvailable()` - check if binary exists
- `GetBinaryPath()` - return path to binary
- `Install(ctx, progress)` - download and extract

## Platform Support

| Platform | Behavior |
|----------|----------|
| **Windows** | Auto-download `whisper-bin-x64.zip`, extract `main.exe` → `whisper.exe` |
| **macOS** | Check PATH for `whisper`. If missing: "Install with: `brew install whisper-cpp`" |
| **Linux** | Check PATH for `whisper`. If missing: "Build from source: `git clone https://github.com/ggerganov/whisper.cpp && cd whisper.cpp && make`" |

Binary search order (all platforms):
1. `~/.ig2insights/bin/whisper[.exe]` (bundled)
2. PATH system (`whisper`, `whisper-cpp`, `main`)
