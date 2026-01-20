# Batch Processing Design

## Overview

Add the ability to process multiple reels concurrently with a worker pool, supporting both CLI arguments and file input.

## User Interface

### CLI Usage

```bash
# Multiple arguments
ig2insights batch url1 url2 url3 ...

# File input (one URL/ID per line)
ig2insights batch --file urls.txt

# Combined
ig2insights batch url1 url2 --file more-urls.txt

# With options
ig2insights batch --file urls.txt --dir ./output --no-save-media --format srt
```

### New Flags (batch command)

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--file` | `-f` | Text file with URLs/IDs (one per line, blank lines and `#` comments ignored) | - |
| `--no-save-media` | - | Delete audio/video from cache after processing (keeps transcript) | false |
| `--concurrency` | `-c` | Max concurrent workers | 10 |

### Inherited Flags

All existing flags work with batch: `--dir`, `--format`, `--model`, `--language`, `--quiet`, `--audio`, `--video`, `--thumbnail`

### Interactive Menu

New option: "Batch process multiple reels" prompts for file path or paste URLs.

## Concurrency Model

Worker pool with sliding window using a semaphore:

```
Queue: [url1, url2, url3, url4, url5, url6, ...]
        |
   +-------------------------------------+
   |  Worker Pool (max 10 concurrent)    |
   |  [1] [2] [3] [4] [5] [6] [7] ...    |
   +-------------------------------------+
        |
   Results channel -> Progress display
```

### Implementation

```go
sem := make(chan struct{}, concurrency)
results := make(chan BatchResult, len(reelIDs))

for _, id := range reelIDs {
    sem <- struct{}{} // blocks if at capacity
    go func(id string) {
        defer func() { <-sem }()
        result := processReel(id)
        results <- result
    }(id)
}
```

### Why Semaphore

- Simpler than managing fixed worker goroutines
- Each job runs in its own goroutine
- Semaphore naturally blocks when at capacity
- As one finishes, another starts immediately

## Progress Display

### Live Display (non-quiet mode)

```
Batch processing 47/150 reels [=====>          ] 31%

✓ ABC123def (3.2s)
✓ XYZ789ghi (2.8s) [cached]
✗ BAD456url: reel not found or is private
✓ QRS147jkl (4.1s)
```

### Final Summary

```
Batch complete: 145/150 succeeded

Output directory: ./output
  - 145 transcripts (.txt)

Failed (5):
  - BAD456url: reel not found or is private
  - ERR789xyz: rate limited by Instagram
  - ...
```

### Quiet Mode

Only final summary with counts and failures.

## Output Files

- All files in single `--dir` directory (default: current directory)
- Named by reel ID: `{reelID}.txt` (or `.srt`/`.json` per format flag)
- Optional `--audio`, `--video`, `--thumbnail` flags add those files

## Cache Behavior

### Default (no flag)

- Audio/video saved to cache as usual
- Transcript saved to cache
- Same behavior as single-reel processing

### With `--no-save-media`

- Transcript saved to cache
- Audio deleted from cache after transcription
- Video/thumbnail never downloaded unless explicitly requested
- If `--video`/`--thumbnail` used: download, copy to output, delete from cache

### Rationale

- 150 reels x 5MB audio = 750MB cache bloat
- Transcripts are tiny (~2KB each)
- Large batch users typically don't need cached audio

## Error Handling

- Continue on error (don't stop batch for one failure)
- Log each failure with reason
- Final summary shows all failures
- Non-zero exit code if any failures

## File Format for `--file`

```
# Comments start with #
# Blank lines are ignored

https://www.instagram.com/reel/ABC123/
DEF456
https://instagram.com/p/GHI789/

# Can mix URLs and IDs
JKL012
```
