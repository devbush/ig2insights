# Output Directory Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Change output behavior to use `--dir` and `--name` flags, defaulting to `./{reelID}/` directory with automatic creation.

**Architecture:** Update flag definitions, refactor `runTranscribe` and `runDownloadOnly` to use new output logic, update `outputResult` to write file AND print to stdout.

**Tech Stack:** Go, Cobra CLI

---

## Task 1: Update Flag Definitions

**Files:**
- Modify: `internal/adapters/cli/root.go:19-31` (flag variables)
- Modify: `internal/adapters/cli/root.go:46-56` (flag definitions)

**Step 1: Update flag variables**

Replace lines 19-31:

```go
var (
	// Global flags
	formatFlag   string
	modelFlag    string
	cacheTTLFlag string
	noCacheFlag  bool
	dirFlag      string
	nameFlag     string
	quietFlag    bool
	languageFlag string
	videoFlag    bool
	thumbnailFlag bool
)
```

**Step 2: Update flag definitions in NewRootCmd**

Replace lines 46-56:

```go
	// Global flags
	rootCmd.PersistentFlags().StringVar(&formatFlag, "format", "", "Output format: text, srt, json")
	rootCmd.PersistentFlags().StringVar(&modelFlag, "model", "small", "Whisper model: tiny, base, small, medium, large")
	rootCmd.PersistentFlags().StringVar(&cacheTTLFlag, "cache-ttl", "7d", "Cache lifetime (e.g., 24h, 7d)")
	rootCmd.PersistentFlags().BoolVar(&noCacheFlag, "no-cache", false, "Skip cache")
	rootCmd.PersistentFlags().StringVarP(&dirFlag, "dir", "d", "", "Output directory (default: ./{reelID})")
	rootCmd.PersistentFlags().StringVarP(&nameFlag, "name", "n", "", "Base filename for outputs (default: {reelID})")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress progress output")
	rootCmd.PersistentFlags().StringVarP(&languageFlag, "language", "l", "auto", "Language code (auto, en, fr, es, etc.)")
	rootCmd.PersistentFlags().BoolVar(&videoFlag, "video", false, "Download the original video file")
	rootCmd.PersistentFlags().BoolVar(&thumbnailFlag, "thumbnail", false, "Download the video thumbnail")
```

**Step 3: Build to verify**

Run: `go build ./cmd/ig2insights`
Expected: Build fails (references to removed flags)

**Step 4: Commit**

Do not commit yet - we need to fix the references first.

---

## Task 2: Update runTranscribe Output Logic

**Files:**
- Modify: `internal/adapters/cli/root.go:432-440` (output directory logic)
- Modify: `internal/adapters/cli/root.go:453` (video path)
- Modify: `internal/adapters/cli/root.go:474` (thumbnail path)
- Modify: `internal/adapters/cli/root.go:496-507` (transcript output)

**Step 1: Update output directory logic**

Replace lines 432-440 with:

```go
	// Determine output directory and base filename
	outputDir := dirFlag
	if outputDir == "" {
		outputDir = reel.ID // Default to ./{reelID}/
	}
	baseName := nameFlag
	if baseName == "" {
		baseName = reel.ID // Default to {reelID}
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		close(spinnerDone)
		return fmt.Errorf("failed to create output directory: %w", err)
	}
```

**Step 2: Update video path**

Replace line 453:

```go
		videoPath := filepath.Join(outputDir, baseName+".mp4")
```

**Step 3: Update thumbnail path**

Replace line 474:

```go
		thumbPath := filepath.Join(outputDir, baseName+".jpg")
```

**Step 4: Update transcript output section**

Replace lines 496-507 with:

```go
	// Output transcript
	transcriptPath, err := outputResult(result, outputDir, baseName)
	if err != nil {
		return err
	}
	outputs["Transcript"] = transcriptPath

	if !quietFlag && len(outputs) > 0 {
		progress.Complete(outputs)
	}
```

**Step 5: Build to verify**

Run: `go build ./cmd/ig2insights`
Expected: Build fails (outputResult signature changed)

---

## Task 3: Update outputResult Function

**Files:**
- Modify: `internal/adapters/cli/root.go:522-554` (outputResult function)

**Step 1: Replace outputResult function**

Replace lines 522-554:

```go
func outputResult(result *application.TranscribeResult, outputDir, baseName string) (string, error) {
	format := formatFlag
	if format == "" {
		format = "text"
	}

	var output string
	var ext string
	switch format {
	case "text":
		output = result.Transcript.ToText()
		ext = "txt"
	case "srt":
		output = result.Transcript.ToSRT()
		ext = "srt"
	case "json":
		data := map[string]interface{}{
			"reel":       result.Reel,
			"transcript": result.Transcript,
		}
		jsonBytes, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return "", err
		}
		output = string(jsonBytes)
		ext = "json"
	default:
		return "", fmt.Errorf("unknown format: %s", format)
	}

	// Write to file
	filePath := filepath.Join(outputDir, baseName+"."+ext)
	if err := os.WriteFile(filePath, []byte(output), 0644); err != nil {
		return "", err
	}

	// Also print to stdout (unless quiet)
	if !quietFlag {
		fmt.Println(output)
	}

	return filePath, nil
}
```

**Step 2: Build to verify**

Run: `go build ./cmd/ig2insights`
Expected: Build succeeds

**Step 3: Run tests**

Run: `go test ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/adapters/cli/root.go
git commit -m "feat(cli): update runTranscribe with new output directory logic"
```

---

## Task 4: Update runDownloadOnly Output Logic

**Files:**
- Modify: `internal/adapters/cli/root.go:170-173` (output directory logic)
- Modify: `internal/adapters/cli/root.go:185` (video dest path)
- Modify: `internal/adapters/cli/root.go:212` (thumbnail dest path)

**Step 1: Update output directory logic**

Replace lines 170-173:

```go
	// Determine output directory and base filename
	outputDir := dirFlag
	if outputDir == "" {
		outputDir = reel.ID // Default to ./{reelID}/
	}
	baseName := nameFlag
	if baseName == "" {
		baseName = reel.ID // Default to {reelID}
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
```

**Step 2: Update video dest path**

Replace line 185:

```go
		destPath := filepath.Join(outputDir, baseName+".mp4")
```

**Step 3: Update thumbnail dest path**

Replace line 212:

```go
		destPath := filepath.Join(outputDir, baseName+".jpg")
```

**Step 4: Build and test**

Run: `go build ./cmd/ig2insights && go test ./...`
Expected: Build succeeds, tests pass

**Step 5: Commit**

```bash
git add internal/adapters/cli/root.go
git commit -m "feat(cli): update runDownloadOnly with new output directory logic"
```

---

## Task 5: Final Build and Manual Test

**Step 1: Build the binary**

Run: `go build -o ig2insights ./cmd/ig2insights`
Expected: Build succeeds

**Step 2: Run all tests**

Run: `go test ./...`
Expected: All tests pass

**Step 3: Verify help text**

Run: `./ig2insights --help`
Expected: Shows `-d, --dir` and `-n, --name` flags, no `--output` or `--download-dir`

**Step 4: Commit final state**

```bash
git add -A
git commit -m "feat(cli): complete output directory refactor"
```
