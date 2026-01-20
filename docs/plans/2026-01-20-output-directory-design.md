# Output Directory Design

## Overview

Change the default output behavior to create a folder named after the reel ID and place all outputs there. Add flexibility with `--dir` and `--name` flags.

## Current Behavior

- `--output` / `-o`: Transcript file path
- `--download-dir`: Directory for video/thumbnail
- Default: outputs to current directory with `{reelID}.ext` filenames
- Transcript printed to stdout OR written to file (not both)

## New Behavior

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--dir` | `-d` | Output directory | `./{reelID}/` |
| `--name` | `-n` | Base filename for all outputs | `{reelID}` |

**Removed flags:**
- `--output` / `-o` (replaced by `--dir` and `--name`)
- `--download-dir` (redundant)

### Output Files

| Asset | Filename |
|-------|----------|
| Transcript | `{name}.txt` / `{name}.srt` / `{name}.json` (based on `--format`) |
| Video | `{name}.mp4` |
| Thumbnail | `{name}.jpg` |

### Transcript Output

- Always written to file in output directory
- Also printed to stdout (unless `--quiet`)

### Directory Creation

- Output directory created automatically if it doesn't exist

## Examples

```bash
# Default - creates ./ABC123/ folder with reel ID as filename
ig2insights https://instagram.com/reel/ABC123
# → ./ABC123/ABC123.txt (also printed to stdout)

# With video and thumbnail
ig2insights --video --thumbnail https://instagram.com/reel/ABC123
# → ./ABC123/ABC123.txt
# → ./ABC123/ABC123.mp4
# → ./ABC123/ABC123.jpg

# Custom directory
ig2insights -d ./transcripts https://instagram.com/reel/ABC123
# → ./transcripts/ABC123.txt

# Custom directory and filename
ig2insights -d ./out -n interview https://instagram.com/reel/ABC123
# → ./out/interview.txt

# Full example with all options
ig2insights -d ./out -n interview --format srt --video --thumbnail https://instagram.com/reel/ABC123
# → ./out/interview.srt
# → ./out/interview.mp4
# → ./out/interview.jpg

# Quiet mode (no stdout)
ig2insights -q https://instagram.com/reel/ABC123
# → ./ABC123/ABC123.txt (no stdout output)
```

## Code Changes

### Files to Modify

- `internal/adapters/cli/root.go`

### Changes

1. **Remove flags:**
   - `outputFlag` / `--output` / `-o`
   - `downloadDirFlag` / `--download-dir`

2. **Add flags:**
   - `dirFlag` / `--dir` / `-d`: Output directory
   - `nameFlag` / `--name` / `-n`: Base filename

3. **Update `runTranscribe`:**
   - Compute output directory (default: `./{reelID}/`)
   - Create directory with `os.MkdirAll`
   - Compute base filename (default: `{reelID}`)
   - Update all file paths to use `{dir}/{name}.{ext}`

4. **Update `outputResult`:**
   - Accept `outputDir` and `baseName` parameters
   - Always write transcript to file
   - Print to stdout unless `--quiet`

5. **Update `runDownloadOnly`:**
   - Same output directory logic

6. **Update flag help text in `NewRootCmd`**
