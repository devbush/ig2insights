# ig2insights

A CLI tool that transcribes Instagram Reels to text using local Whisper AI and yt-dlp.

## Features

- Transcribe Instagram Reels to text, SRT, or JSON
- Batch process multiple reels concurrently
- Download audio, video, and thumbnails
- Local processing with whisper.cpp (no API keys needed)
- Smart caching to avoid re-downloading
- Interactive menu or command-line usage

## Installation

### Prerequisites

- Go 1.21+
- FFmpeg (auto-installed on Windows)

### Build from source

```bash
git clone https://github.com/devbush/ig2insights.git
cd ig2insights
go build -o ig2insights ./cmd/ig2insights
```

## Usage

### Interactive Mode

Run without arguments for an interactive menu:

```bash
./ig2insights
```

### Single Reel

```bash
# By URL
./ig2insights https://www.instagram.com/reel/ABC123/

# By ID
./ig2insights ABC123

# With options
./ig2insights ABC123 --format srt --video --thumbnail
```

### Batch Processing

Process multiple reels concurrently:

```bash
# Multiple IDs
./ig2insights batch ABC123 DEF456 GHI789

# From file (one URL/ID per line)
./ig2insights batch --file reels.txt

# With options
./ig2insights batch --file reels.txt --concurrency 5 --dir ./output
```

### Output Options

| Flag | Description |
|------|-------------|
| `--format` | Output format: `text`, `srt`, `json` |
| `--dir, -d` | Output directory (default: `./{reelID}`) |
| `--name, -n` | Base filename (default: `{reelID}`) |
| `--audio` | Download audio file (WAV) |
| `--video` | Download video file (MP4) |
| `--thumbnail` | Download thumbnail (JPG) |
| `--quiet, -q` | Suppress progress output |

### Model Selection

```bash
# Use a specific Whisper model
./ig2insights ABC123 --model large

# Available models: tiny, base, small (default), medium, large
```

### Language

```bash
# Auto-detect (default)
./ig2insights ABC123 --language auto

# Specify language
./ig2insights ABC123 --language es
```

### Cache Management

```bash
# View cache stats
./ig2insights cache stats

# Clear all cache
./ig2insights cache clear

# Clean expired entries
./ig2insights cache clean
```

### Model Management

```bash
# List available models
./ig2insights model list

# Download a model
./ig2insights model download large

# Delete a model
./ig2insights model delete large
```

## Batch Processing Details

The batch command processes reels concurrently with a configurable worker pool:

```bash
./ig2insights batch --file reels.txt --concurrency 10 --no-save-media
```

| Flag | Description |
|------|-------------|
| `--file, -f` | Input file with URLs/IDs (one per line, `#` for comments) |
| `--concurrency, -c` | Max concurrent workers (default: 10, max: 50) |
| `--no-save-media` | Don't keep audio/video in cache after processing |

Progress is displayed in real-time:

```
Batch processing 47/150 reels [=====>          ] 31%

✓ ABC123 (3.2s)
✓ XYZ789 (2.8s) [cached]
✗ BAD456: reel not found or is private
```

## Configuration

User config is stored at `~/.ig2insights/config.yaml`.

Data directories:
- Models: `~/.ig2insights/models/`
- Cache: `~/.ig2insights/cache/`

## Dependencies

Dependencies are auto-managed:
- **yt-dlp** - Video downloading (auto-installed)
- **whisper.cpp** - Local transcription (auto-installed)
- **FFmpeg** - Audio extraction (auto-installed on Windows)

Check status:
```bash
./ig2insights deps status
```

## License

MIT
