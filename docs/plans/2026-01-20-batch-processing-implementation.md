# Batch Processing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add batch processing command to process multiple reels concurrently with worker pool.

**Architecture:** New `batch` subcommand with semaphore-based worker pool. Input from CLI args and/or file. New batch progress display component. `--no-save-media` flag cleans up cache after processing.

**Tech Stack:** Go, Cobra CLI, goroutines with semaphore pattern, sync package

---

## Task 1: Input Parsing Functions

**Files:**
- Create: `internal/adapters/cli/batch_input.go`
- Create: `internal/adapters/cli/batch_input_test.go`

**Step 1: Write the failing test for ParseInputFile**

```go
// internal/adapters/cli/batch_input_test.go
package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseInputFile(t *testing.T) {
	// Create temp file
	content := `# Comment line
https://www.instagram.com/reel/ABC123/
DEF456

# Another comment
https://instagram.com/p/GHI789/
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "urls.txt")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ids, err := ParseInputFile(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"ABC123", "DEF456", "GHI789"}
	if len(ids) != len(expected) {
		t.Fatalf("got %d IDs, want %d", len(ids), len(expected))
	}
	for i, id := range ids {
		if id != expected[i] {
			t.Errorf("ids[%d] = %q, want %q", i, id, expected[i])
		}
	}
}

func TestParseInputFile_NotFound(t *testing.T) {
	_, err := ParseInputFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/cli/... -run TestParseInputFile -v`
Expected: FAIL with "undefined: ParseInputFile"

**Step 3: Write minimal implementation**

```go
// internal/adapters/cli/batch_input.go
package cli

import (
	"bufio"
	"os"
	"strings"

	"github.com/devbush/ig2insights/internal/domain"
)

// ParseInputFile reads a file with URLs/IDs, one per line.
// Blank lines and lines starting with # are ignored.
func ParseInputFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ids []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip blank lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Parse URL or ID
		reel, err := domain.ParseReelInput(line)
		if err != nil {
			continue // Skip invalid lines
		}
		ids = append(ids, reel.ID)
	}
	return ids, scanner.Err()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/adapters/cli/... -run TestParseInputFile -v`
Expected: PASS

**Step 5: Add test for CollectInputs (combines args + file)**

```go
// Add to batch_input_test.go
func TestCollectInputs(t *testing.T) {
	// Create temp file
	content := "ABC123\nDEF456"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "urls.txt")
	os.WriteFile(tmpFile, []byte(content), 0644)

	// Args + file, should dedupe
	ids, err := CollectInputs([]string{"ABC123", "GHI789"}, tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ABC123 appears in both, should only appear once
	expected := []string{"ABC123", "GHI789", "DEF456"}
	if len(ids) != len(expected) {
		t.Fatalf("got %d IDs, want %d: %v", len(ids), len(expected), ids)
	}
}

func TestCollectInputs_NoFile(t *testing.T) {
	ids, err := CollectInputs([]string{"ABC123", "DEF456"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("got %d IDs, want 2", len(ids))
	}
}
```

**Step 6: Run test to verify it fails**

Run: `go test ./internal/adapters/cli/... -run TestCollectInputs -v`
Expected: FAIL with "undefined: CollectInputs"

**Step 7: Implement CollectInputs**

```go
// Add to batch_input.go

// CollectInputs combines CLI args and file input, deduplicating.
// Args are processed first, then file entries.
func CollectInputs(args []string, filePath string) ([]string, error) {
	seen := make(map[string]bool)
	var ids []string

	// Process CLI args first
	for _, arg := range args {
		reel, err := domain.ParseReelInput(arg)
		if err != nil {
			continue
		}
		if !seen[reel.ID] {
			seen[reel.ID] = true
			ids = append(ids, reel.ID)
		}
	}

	// Process file if provided
	if filePath != "" {
		fileIDs, err := ParseInputFile(filePath)
		if err != nil {
			return nil, err
		}
		for _, id := range fileIDs {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}

	return ids, nil
}
```

**Step 8: Run all tests**

Run: `go test ./internal/adapters/cli/... -run "TestParseInputFile|TestCollectInputs" -v`
Expected: PASS

**Step 9: Commit**

```bash
git add internal/adapters/cli/batch_input.go internal/adapters/cli/batch_input_test.go
git commit -m "feat(cli): add batch input parsing functions"
```

---

## Task 2: Batch Result Types

**Files:**
- Create: `internal/adapters/cli/batch_types.go`

**Step 1: Create batch result types (no test needed - pure data)**

```go
// internal/adapters/cli/batch_types.go
package cli

import "time"

// BatchResult represents the result of processing a single reel in a batch
type BatchResult struct {
	ReelID   string
	Success  bool
	Error    string
	Duration time.Duration
	Cached   bool // true if transcript was from cache
}

// BatchSummary aggregates results from a batch run
type BatchSummary struct {
	Total     int
	Succeeded int
	Failed    int
	Results   []BatchResult
}

// FailedResults returns only the failed results
func (s *BatchSummary) FailedResults() []BatchResult {
	var failed []BatchResult
	for _, r := range s.Results {
		if !r.Success {
			failed = append(failed, r)
		}
	}
	return failed
}
```

**Step 2: Commit**

```bash
git add internal/adapters/cli/batch_types.go
git commit -m "feat(cli): add batch result types"
```

---

## Task 3: Batch Progress Display

**Files:**
- Create: `internal/adapters/cli/tui/batch_progress.go`
- Create: `internal/adapters/cli/tui/batch_progress_test.go`

**Step 1: Write failing test for progress bar rendering**

```go
// internal/adapters/cli/tui/batch_progress_test.go
package tui

import "testing"

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		current, total int
		width          int
		want           string
	}{
		{0, 10, 10, "[          ]"},
		{5, 10, 10, "[=====>    ]"},
		{10, 10, 10, "[==========]"},
		{3, 10, 10, "[==>       ]"},
	}

	for _, tt := range tests {
		got := renderProgressBar(tt.current, tt.total, tt.width)
		if got != tt.want {
			t.Errorf("renderProgressBar(%d, %d, %d) = %q, want %q",
				tt.current, tt.total, tt.width, got, tt.want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/cli/tui/... -run TestRenderProgressBar -v`
Expected: FAIL with "undefined: renderProgressBar"

**Step 3: Implement batch progress display**

```go
// internal/adapters/cli/tui/batch_progress.go
package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// BatchProgress displays batch processing progress
type BatchProgress struct {
	total       int
	completed   int
	quiet       bool
	mu          sync.Mutex
	results     []batchResultLine
	startTime   time.Time
	lastRender  time.Time
}

type batchResultLine struct {
	reelID   string
	success  bool
	error    string
	duration time.Duration
	cached   bool
}

// NewBatchProgress creates a new batch progress display
func NewBatchProgress(total int, quiet bool) *BatchProgress {
	return &BatchProgress{
		total:     total,
		quiet:     quiet,
		startTime: time.Now(),
	}
}

// AddResult adds a completed result and updates display
func (p *BatchProgress) AddResult(reelID string, success bool, errMsg string, duration time.Duration, cached bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.completed++
	p.results = append(p.results, batchResultLine{
		reelID:   reelID,
		success:  success,
		error:    errMsg,
		duration: duration,
		cached:   cached,
	})

	p.render()
}

// render displays current progress
func (p *BatchProgress) render() {
	if p.quiet {
		return
	}

	// Throttle renders
	if time.Since(p.lastRender) < 50*time.Millisecond && p.completed < p.total {
		return
	}
	p.lastRender = time.Now()

	// Clear previous output (move up and clear)
	lineCount := len(p.results) + 2 // progress line + blank + results
	if lineCount > 1 {
		fmt.Printf("\033[%dA\033[J", lineCount)
	}

	// Progress bar
	pct := float64(p.completed) / float64(p.total) * 100
	bar := renderProgressBar(p.completed, p.total, 30)
	fmt.Printf("Batch processing %d/%d reels %s %.0f%%\n\n", p.completed, p.total, bar, pct)

	// Recent results (last 10)
	start := 0
	if len(p.results) > 10 {
		start = len(p.results) - 10
	}
	for _, r := range p.results[start:] {
		if r.success {
			cached := ""
			if r.cached {
				cached = " [cached]"
			}
			fmt.Printf("✓ %s (%.1fs)%s\n", r.reelID, r.duration.Seconds(), cached)
		} else {
			fmt.Printf("✗ %s: %s\n", r.reelID, r.error)
		}
	}
}

// renderProgressBar creates a text progress bar
func renderProgressBar(current, total, width int) string {
	if total == 0 {
		return "[" + strings.Repeat(" ", width) + "]"
	}

	filled := (current * width) / total
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("=", filled)
	if filled < width && current > 0 {
		bar += ">"
		filled++
	}
	bar += strings.Repeat(" ", width-filled)

	return "[" + bar + "]"
}

// Complete prints the final summary
func (p *BatchProgress) Complete() {
	if p.quiet {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	succeeded := 0
	var failed []batchResultLine
	for _, r := range p.results {
		if r.success {
			succeeded++
		} else {
			failed = append(failed, r)
		}
	}

	fmt.Printf("\nBatch complete: %d/%d succeeded\n", succeeded, p.total)

	if len(failed) > 0 {
		fmt.Printf("\nFailed (%d):\n", len(failed))
		for _, r := range failed {
			fmt.Printf("  - %s: %s\n", r.reelID, r.error)
		}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/adapters/cli/tui/... -run TestRenderProgressBar -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/adapters/cli/tui/batch_progress.go internal/adapters/cli/tui/batch_progress_test.go
git commit -m "feat(tui): add batch progress display component"
```

---

## Task 4: Batch Command Structure

**Files:**
- Create: `internal/adapters/cli/batch.go`
- Modify: `internal/adapters/cli/root.go` (add subcommand)

**Step 1: Create batch command with flags**

```go
// internal/adapters/cli/batch.go
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/devbush/ig2insights/internal/adapters/cli/tui"
	"github.com/devbush/ig2insights/internal/application"
	"github.com/spf13/cobra"
)

var (
	batchFileFlag        string
	batchNoSaveMediaFlag bool
	batchConcurrencyFlag int
)

// NewBatchCmd creates the batch subcommand
func NewBatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch [url|id]...",
		Short: "Process multiple reels concurrently",
		Long: `Process multiple Instagram reels in parallel.

Provide URLs or IDs as arguments, or use --file to read from a file.
The file should have one URL or ID per line. Blank lines and lines
starting with # are ignored.

Examples:
  ig2insights batch ABC123 DEF456 GHI789
  ig2insights batch --file urls.txt
  ig2insights batch ABC123 --file more-urls.txt --concurrency 5`,
		RunE: runBatch,
	}

	cmd.Flags().StringVarP(&batchFileFlag, "file", "f", "", "File with URLs/IDs (one per line)")
	cmd.Flags().BoolVar(&batchNoSaveMediaFlag, "no-save-media", false, "Don't save audio/video to cache after processing")
	cmd.Flags().IntVarP(&batchConcurrencyFlag, "concurrency", "c", 10, "Max concurrent workers")

	return cmd
}

func runBatch(cmd *cobra.Command, args []string) error {
	// Collect all inputs
	reelIDs, err := CollectInputs(args, batchFileFlag)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	if len(reelIDs) == 0 {
		return fmt.Errorf("no reel URLs or IDs provided")
	}

	app, err := GetApp()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	ctx := context.Background()

	// Create output directory
	outputDir := dirFlag
	if outputDir == "" {
		outputDir = "."
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Process batch
	return processBatch(ctx, app, reelIDs, outputDir)
}

func processBatch(ctx context.Context, app *App, reelIDs []string, outputDir string) error {
	total := len(reelIDs)
	progress := tui.NewBatchProgress(total, quietFlag)

	// Semaphore for concurrency control
	concurrency := batchConcurrencyFlag
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > 50 {
		concurrency = 50 // Cap at 50
	}
	sem := make(chan struct{}, concurrency)

	// Results collection
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []BatchResult

	for _, reelID := range reelIDs {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(id string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			start := time.Now()
			result := processOneReel(ctx, app, id, outputDir)
			result.Duration = time.Since(start)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()

			progress.AddResult(id, result.Success, result.Error, result.Duration, result.Cached)
		}(reelID)
	}

	wg.Wait()
	progress.Complete()

	// Return error if any failed
	for _, r := range results {
		if !r.Success {
			return fmt.Errorf("batch completed with %d failures", countFailed(results))
		}
	}
	return nil
}

func processOneReel(ctx context.Context, app *App, reelID string, outputDir string) BatchResult {
	result := BatchResult{ReelID: reelID}

	opts := application.TranscribeOptions{
		Model:         modelFlag,
		NoCache:       noCacheFlag,
		Language:      languageFlag,
		SaveAudio:     audioFlag,
		SaveVideo:     videoFlag,
		SaveThumbnail: thumbnailFlag,
	}

	transcribeResult, err := app.TranscribeSvc.Transcribe(ctx, reelID, opts)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	result.Success = true
	result.Cached = transcribeResult.TranscriptFromCache

	// Write transcript to output
	format := formatFlag
	if format == "" {
		format = "text"
	}

	var output string
	var ext string
	switch format {
	case "text":
		output = transcribeResult.Transcript.ToText()
		ext = "txt"
	case "srt":
		output = transcribeResult.Transcript.ToSRT()
		ext = "srt"
	case "json":
		output = fmt.Sprintf(`{"reel_id":"%s","text":"%s"}`, reelID, transcribeResult.Transcript.Text)
		ext = "json"
	}

	outPath := filepath.Join(outputDir, reelID+"."+ext)
	if err := os.WriteFile(outPath, []byte(output), 0644); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to write output: %v", err)
		return result
	}

	// Copy additional assets if requested
	if audioFlag && transcribeResult.AudioPath != "" {
		audioOut := filepath.Join(outputDir, reelID+".wav")
		copyFile(transcribeResult.AudioPath, audioOut)
	}
	if videoFlag && transcribeResult.VideoPath != "" {
		videoOut := filepath.Join(outputDir, reelID+".mp4")
		copyFile(transcribeResult.VideoPath, videoOut)
	}
	if thumbnailFlag && transcribeResult.ThumbnailPath != "" {
		thumbOut := filepath.Join(outputDir, reelID+".jpg")
		copyFile(transcribeResult.ThumbnailPath, thumbOut)
	}

	// Clean up media from cache if --no-save-media
	if batchNoSaveMediaFlag {
		cleanupCacheMedia(app, reelID, transcribeResult)
	}

	return result
}

func cleanupCacheMedia(app *App, reelID string, result *application.TranscribeResult) {
	// Delete audio file from cache
	if result.AudioPath != "" {
		os.Remove(result.AudioPath)
	}
	// Delete video file from cache
	if result.VideoPath != "" {
		os.Remove(result.VideoPath)
	}
	// Delete thumbnail from cache
	if result.ThumbnailPath != "" {
		os.Remove(result.ThumbnailPath)
	}

	// Update cache entry to clear paths
	ctx := context.Background()
	cached, err := app.Cache.Get(ctx, reelID)
	if err == nil && cached != nil {
		cached.AudioPath = ""
		cached.VideoPath = ""
		cached.ThumbnailPath = ""
		app.Cache.Set(ctx, reelID, cached)
	}
}

func countFailed(results []BatchResult) int {
	count := 0
	for _, r := range results {
		if !r.Success {
			count++
		}
	}
	return count
}
```

**Step 2: Add batch command to root**

In `internal/adapters/cli/root.go`, add to `NewRootCmd()`:

```go
// Add subcommands
rootCmd.AddCommand(NewAccountCmd())
rootCmd.AddCommand(NewBatchCmd())  // Add this line
rootCmd.AddCommand(NewCacheCmd())
rootCmd.AddCommand(NewModelCmd())
rootCmd.AddCommand(NewDepsCmd())
```

**Step 3: Build and verify**

Run: `go build ./cmd/ig2insights`
Expected: Build succeeds

Run: `./ig2insights batch --help`
Expected: Shows batch command help with flags

**Step 4: Commit**

```bash
git add internal/adapters/cli/batch.go internal/adapters/cli/root.go
git commit -m "feat(cli): add batch command with concurrent processing"
```

---

## Task 5: Interactive Menu Option

**Files:**
- Modify: `internal/adapters/cli/root.go`

**Step 1: Add batch option to interactive menu**

In `runInteractiveMenu()`, add the batch option:

```go
func runInteractiveMenu() error {
	options := []tui.MenuOption{
		{Label: "Transcribe a single reel", Value: "transcribe"},
		{Label: "Batch process multiple reels", Value: "batch"},  // Add this
		// Note: "Browse an account's reels" hidden - Instagram blocking yt-dlp user page scraping
		{Label: "Manage cache", Value: "cache"},
		{Label: "Quit", Value: "quit"},
	}

	selected, err := tui.RunMenu(options)
	if err != nil {
		return err
	}

	switch selected {
	case "transcribe":
		return runTranscribeInteractive()
	case "batch":
		return runBatchInteractive()  // Add this
	case "account":
		// ... existing code
	}
	// ...
}
```

**Step 2: Implement runBatchInteractive**

Add to `root.go` or `batch.go`:

```go
func runBatchInteractive() error {
	fmt.Println("Enter reel URLs or IDs (one per line, empty line when done):")
	fmt.Println("Or enter a file path starting with @")
	fmt.Println()

	var inputs []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "@") {
			// File path
			batchFileFlag = strings.TrimPrefix(line, "@")
		} else {
			inputs = append(inputs, line)
		}
	}

	if len(inputs) == 0 && batchFileFlag == "" {
		fmt.Println("No inputs provided")
		return nil
	}

	// Collect and process
	reelIDs, err := CollectInputs(inputs, batchFileFlag)
	if err != nil {
		return err
	}

	if len(reelIDs) == 0 {
		fmt.Println("No valid reel URLs or IDs found")
		return nil
	}

	fmt.Printf("Found %d reels to process\n", len(reelIDs))

	app, err := GetApp()
	if err != nil {
		return err
	}

	outputDir := dirFlag
	if outputDir == "" {
		outputDir = "."
	}

	return processBatch(context.Background(), app, reelIDs, outputDir)
}
```

Add import at top of file:
```go
import (
	"bufio"
	// ... other imports
)
```

**Step 3: Build and test interactively**

Run: `go build ./cmd/ig2insights && ./ig2insights`
Expected: Menu shows "Batch process multiple reels" option

**Step 4: Commit**

```bash
git add internal/adapters/cli/root.go internal/adapters/cli/batch.go
git commit -m "feat(cli): add batch option to interactive menu"
```

---

## Task 6: End-to-End Testing

**Files:** None (manual testing)

**Step 1: Create test input file**

```bash
cat > test-urls.txt << 'EOF'
# Test batch processing
# Add 2-3 real reel IDs here for testing
EOF
```

**Step 2: Test CLI batch command**

Run: `./ig2insights batch --file test-urls.txt --dir ./test-output`
Expected:
- Progress bar shows
- Each reel completion shows with checkmark or X
- Final summary shows success/failure count
- Transcripts appear in ./test-output/

**Step 3: Test with --no-save-media**

Run: `./ig2insights batch --file test-urls.txt --no-save-media --dir ./test-output2`
Expected:
- Same output as above
- Cache directory should NOT have audio/video files for processed reels

**Step 4: Test concurrency flag**

Run: `./ig2insights batch --file test-urls.txt -c 2 --dir ./test-output3`
Expected: Works with reduced concurrency

**Step 5: Clean up test files**

```bash
rm -rf test-urls.txt test-output test-output2 test-output3
```

**Step 6: Final commit if any fixes needed**

```bash
git add -A
git commit -m "fix(batch): address issues found in testing"
```

---

## Task 7: Run All Tests

**Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: All tests pass

**Step 2: Build final binary**

Run: `go build ./cmd/ig2insights`
Expected: Build succeeds

**Step 3: Push changes**

```bash
git push
```
