# Smart Asset Caching Design

## Overview

Enable independent caching for transcript, video, and thumbnail assets. When a user requests assets for a previously processed reel, the system should use cached versions where available and only download/process what's missing.

## Current Behavior

- Cache check happens at transcription level only
- If transcript is cached, early return skips video/thumbnail downloads entirely
- Thumbnail is not cached at all
- User cannot get video/thumbnail from a previously transcribed reel without re-processing

## Desired Behavior

1. **Independent caching** - Each asset (transcript, video, thumbnail) cached separately
2. **Cache all requested assets** - Even download-only requests are cached for future use
3. **Copy from cache to output** - Cached assets copied to user's output directory
4. **Progress display** - Shows all steps, marks cached ones as instant complete with "(cached)" suffix

## Design

### 1. Cache Model Changes

**File**: `internal/ports/cache.go`

Add `ThumbnailPath` to `CachedItem`:

```go
type CachedItem struct {
    Reel          *domain.Reel
    Transcript    *domain.Transcript
    VideoPath     string
    ThumbnailPath string    // NEW
    CreatedAt     time.Time
    ExpiresAt     time.Time
}
```

Backwards compatible - existing cache entries have empty `ThumbnailPath`.

### 2. TranscribeService Changes

**File**: `internal/application/transcribe.go`

Update `TranscribeResult` to track per-asset cache status:

```go
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

Refactor `Transcribe()` method:

```
1. Try to get cached item (even if partial)
2. If transcript requested AND not cached → download video, transcribe
3. If video requested AND not cached → download video
4. If thumbnail requested AND not cached → download thumbnail
5. Update cache with any new assets
6. Return result with per-asset cache flags
```

Key optimization: If transcript is needed and video isn't cached, download video for transcription. If user also requested video, reuse that download (no double download).

### 3. CLI Progress Display Changes

**File**: `internal/adapters/cli/root.go`

1. **Pre-flight cache check** - Query cache before starting progress to know what's available
2. **Build dynamic step list** - Include all requested steps, mark which will be cached
3. **Instant complete cached steps** - Mark cached steps done immediately with "(cached)" suffix
4. **Process remaining steps normally** - Download/transcribe with current spinner/progress

Example output:
```
✓ Checking dependencies
✓ Downloading video (cached)
✓ Extracting audio (cached)
✓ Transcribing (cached)
⟳ Saving video...
✓ Downloading thumbnail (cached)
```

### 4. Output Handling

**File**: `internal/adapters/cli/root.go`

When assets are cached, copy from cache to output directory:

```go
if wantVideo {
    destPath := filepath.Join(outputDir, reelID+".mp4")
    copyFile(result.VideoPath, destPath)
    // Step name includes "(cached)" if VideoFromCache is true
}
```

Same pattern for thumbnail. Skip copy if output dir equals cache dir.

## Files Changed

| Layer | File | Change |
|-------|------|--------|
| Ports | `internal/ports/cache.go` | Add `ThumbnailPath` to `CachedItem` |
| Application | `internal/application/transcribe.go` | Refactor for per-asset caching; update `TranscribeResult` |
| CLI | `internal/adapters/cli/root.go` | Pre-flight cache check; pass cache status to progress; copy cached files |
| TUI | `internal/adapters/cli/tui/progress.go` | Support "(cached)" suffix (may work already) |

## Files Not Changed

- `internal/adapters/cache/store.go` - Already handles partial JSON data
- `internal/adapters/ytdlp/` - No changes needed
- `internal/domain/` - No changes needed

## Edge Cases

1. **Expired cache** - Treat as cache miss, re-download/process
2. **Partial cache** - Use what's available, fetch the rest
3. **Cache file missing but path stored** - Treat as cache miss for that asset
4. **Output dir is cache dir** - Skip copy operation
