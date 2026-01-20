# ig2insights Design Document

A CLI tool to get transcripts of Instagram videos (Reels).

## Overview

ig2insights allows users to transcribe Instagram Reels by providing:
- A reel URL (e.g., `https://www.instagram.com/p/DToLsd-EvGJ/`)
- A reel ID (e.g., `DToLsd-EvGJ`)
- An account URL (e.g., `https://www.instagram.com/npcfaizan/`)
- An account name (e.g., `npcfaizan`)

The tool uses yt-dlp for video downloading and local Whisper for transcription, making it completely free to use.

---

## CLI Interface

### Commands

```
ig2insights                              # Interactive menu
ig2insights <reel-url|reel-id>           # Transcribe a single reel
ig2insights account <username|url>       # Browse account reels interactively
ig2insights account <username> --latest 5    # Transcribe 5 most recent reels
ig2insights account <username> --top 5       # Transcribe 5 most viewed reels
ig2insights cache clear                  # Clear expired cache entries
ig2insights cache clear --all            # Clear all cache
ig2insights model list                   # Show available models + download status
ig2insights model download <name>        # Pre-download a model
ig2insights model remove <name>          # Delete a downloaded model
ig2insights deps status                  # Show yt-dlp and model status
ig2insights deps update                  # Update yt-dlp to latest version
```

### Global Flags

```
--format <text|srt|json>    # Output format (prompts if omitted)
--model <tiny|base|small|medium|large>  # Whisper model (default: small)
--cache-ttl <duration>      # Cache lifetime (default: 7d)
--no-cache                  # Skip cache, always re-download/transcribe
--output <path>             # Write to file instead of stdout
-q, --quiet                 # Suppress progress output
```

### Interactive Menu (no arguments)

```
? What would you like to do?
  > Transcribe a single reel
    Browse an account's reels
    Manage cache
    Settings
```

- **Transcribe single reel** - prompts for URL/ID, asks format, transcribes
- **Browse account** - prompts for username, shows scrollable reel list with checkboxes
- **Manage cache** - shows cache size, offers clear options
- **Settings** - configure default model, TTL, output format

### Account Browsing Flow

When running `ig2insights account @username` without flags:
1. Fetches account info, shows reel count
2. Asks: "Sort by: Latest / Most Viewed"
3. Shows scrollable list with checkboxes (title, date, views)
4. User selects reels with space, confirms with enter
5. Asks output format if `--format` not provided
6. Transcribes selected reels, shows progress

---

## Architecture (DDD)

```
ig2insights/
├── cmd/                    # Entry point
│   └── ig2insights/
│       └── main.go
├── internal/
│   ├── domain/             # Core business logic (no external deps)
│   │   ├── reel.go         # Reel entity (ID, URL, metadata)
│   │   ├── transcript.go   # Transcript entity (text, segments, format)
│   │   ├── account.go      # Account entity (username, reel count)
│   │   └── errors.go       # Domain errors
│   │
│   ├── application/        # Use cases / orchestration
│   │   ├── transcribe.go   # TranscribeReel use case
│   │   ├── browse.go       # BrowseAccount use case
│   │   └── cache.go        # CacheManagement use case
│   │
│   ├── ports/              # Interfaces (abstractions)
│   │   ├── transcriber.go  # Transcriber interface
│   │   ├── downloader.go   # VideoDownloader interface
│   │   ├── fetcher.go      # AccountFetcher interface
│   │   └── cache.go        # CacheStore interface
│   │
│   └── adapters/           # Implementations
│       ├── whisper/        # Whisper transcriber (implements Transcriber)
│       ├── ytdlp/          # yt-dlp downloader (implements VideoDownloader, AccountFetcher)
│       ├── cache/          # File-based cache (implements CacheStore)
│       └── cli/            # CLI handlers, prompts, TUI components
│
├── pkg/                    # Shared utilities (if needed)
└── configs/                # Default config, model paths
```

### Key Interfaces (ports)

```go
type Transcriber interface {
    Transcribe(ctx context.Context, videoPath string, opts TranscribeOpts) (*Transcript, error)
    AvailableModels() []Model
    DownloadModel(ctx context.Context, model Model) error
}

type VideoDownloader interface {
    Download(ctx context.Context, reelID string) (*DownloadResult, error)
}

type AccountFetcher interface {
    GetAccount(ctx context.Context, username string) (*Account, error)
    ListReels(ctx context.Context, username string, sort SortOrder, limit int) ([]Reel, error)
}

type CacheStore interface {
    Get(ctx context.Context, reelID string) (*CachedItem, error)
    Set(ctx context.Context, reelID string, item *CachedItem) error
    Delete(ctx context.Context, reelID string) error
    CleanExpired(ctx context.Context) error
    Clear(ctx context.Context) error
}
```

---

## Data Flow

```
User Input (URL/ID)
       │
       ▼
┌─────────────────┐
│  Parse Input    │  → Extract reel ID from URL or use directly
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Check Cache    │  → If transcript exists & not expired, return it
└────────┬────────┘
         │ (cache miss)
         ▼
┌─────────────────┐
│ Download Video  │  → yt-dlp fetches video to cache dir
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Transcribe    │  → Whisper processes audio, returns segments
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Store & Format │  → Cache transcript, format to user's choice
└────────┬────────┘
         │
         ▼
      Output
```

---

## Caching

### Directory Structure

```
~/.ig2insights/
├── config.yaml              # User settings (default model, TTL, format)
├── bin/
│   └── yt-dlp               # Auto-downloaded binary
├── models/                  # Downloaded Whisper models
│   └── ggml-small.bin
└── cache/
    └── <reel-id>/
        ├── meta.json        # Metadata + timestamps + expiry
        ├── video.mp4        # Downloaded video
        └── transcript.json  # Full transcript (all formats derivable)
```

### Cache TTL

- Default: 7 days
- Configurable via `--cache-ttl` flag or `config.yaml`
- Supports duration formats: `24h`, `7d`, `30d`

### Lazy Cleanup

On every CLI invocation:
1. Scan `cache/` directory (async, non-blocking)
2. Read `meta.json` for each entry
3. If `created_at + ttl < now`, delete the folder

---

## Model Management

### Available Models

| Model  | Size    | Accuracy | Speed     |
|--------|---------|----------|-----------|
| tiny   | ~75MB   | Basic    | Very fast |
| base   | ~140MB  | Good     | Fast      |
| small  | ~462MB  | Better   | Moderate  |
| medium | ~1.5GB  | Great    | Slower    |
| large  | ~3GB    | Best     | Slow      |

Default: `small`

### On-Demand Download

When transcription is requested:
1. Check if required model exists in `~/.ig2insights/models/`
2. If missing, prompt user:
   ```
   ? Model "small" not found. Download now? (462 MB) [Y/n]
   ```
3. Download from Hugging Face with progress bar
4. Store in models folder, continue with transcription

### Model Source

```
https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-{model}.bin
```

---

## Dependency Management

### yt-dlp Auto-Download

When yt-dlp is needed:
1. Check if `~/.ig2insights/bin/yt-dlp` exists
2. If not, check if `yt-dlp` is in system PATH
3. If neither found, prompt and download from GitHub releases:
   ```
   https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp[.exe]
   ```
4. Store in `~/.ig2insights/bin/`, make executable

### Go Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/charmbracelet/bubbletea` - Interactive TUI
- `github.com/charmbracelet/bubbles` - TUI components (list, spinner, progress)
- `gopkg.in/yaml.v3` - Config file parsing
- `github.com/ggerganov/whisper.cpp/bindings/go` - Whisper bindings (or shell out to CLI)

---

## Output Formats

### Plain Text (`--format text`)

```
Hey everyone, welcome back to another video. Today we're going to talk about something really interesting...
```

### SRT Format (`--format srt`)

```
1
00:00:00,000 --> 00:00:03,500
Hey everyone, welcome back to another video.

2
00:00:03,500 --> 00:00:07,200
Today we're going to talk about something really interesting...
```

### JSON Format (`--format json`)

```json
{
  "reel": {
    "id": "DToLsd-EvGJ",
    "url": "https://www.instagram.com/p/DToLsd-EvGJ/",
    "author": "npcfaizan",
    "title": "POV: NPC at the gym",
    "duration_seconds": 45,
    "view_count": 1250000,
    "fetched_at": "2026-01-20T10:30:00Z"
  },
  "transcript": {
    "text": "Hey everyone, welcome back...",
    "segments": [
      {
        "start": 0.0,
        "end": 3.5,
        "text": "Hey everyone, welcome back to another video."
      }
    ],
    "model": "small",
    "language": "en",
    "transcribed_at": "2026-01-20T10:30:15Z"
  }
}
```

### Output Destination

- Default: stdout (pipeable)
- `--output <path>`: write to file
- Multiple reels: creates `<reel-id>.<ext>` files in current dir or specified folder

---

## Error Handling

| Error | Handling |
|-------|----------|
| Invalid URL/ID | "Invalid reel URL or ID. Expected format: https://instagram.com/p/ABC123 or ABC123" |
| Reel not found / private | "Reel not found or is private. Only public reels can be transcribed." |
| yt-dlp not installed | Auto-download prompt, or manual install instructions |
| Network failure | Retry up to 3 times with backoff, then fail with suggestion |
| Model download fails | Resume support for partial downloads, clear error with retry option |
| Transcription fails | Log error details, suggest trying different model |
| Rate limited by Instagram | "Instagram rate limit hit. Try again in X minutes." |

---

## First Run Experience

```
$ ig2insights

  Welcome to ig2insights!

  This tool requires a few dependencies to work:

  ✗ yt-dlp      not found
  ✗ whisper     no model downloaded

? Set up now? This will download:
  • yt-dlp binary (~22 MB)
  • Whisper "small" model (~462 MB)

  [Y/n]

Downloading yt-dlp... ████████████████████ 100%
Downloading whisper model (small)... ████████████████████ 100%

✓ Setup complete! Run 'ig2insights' to get started.
```

---

## Configuration

### Config File (`~/.ig2insights/config.yaml`)

```yaml
defaults:
  model: small
  format: text
  cache_ttl: 7d

paths:
  yt_dlp: ""  # Empty = use bundled or system
```

---

## Installation

```bash
go install github.com/<username>/ig2insights@latest
```
