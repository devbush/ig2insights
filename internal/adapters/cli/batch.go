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
	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
	"github.com/spf13/cobra"
)

var (
	batchFileFlag      string
	batchNoSaveMedia   bool
	batchConcurrency   int
)

// NewBatchCmd creates the batch command
func NewBatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch [urls/ids...]",
		Short: "Batch process multiple reels",
		Long: `Batch process multiple Instagram Reels concurrently.

Provide reel URLs or IDs as arguments and/or via a file with --file.
Each reel will be transcribed and saved to the output directory.

Example:
  ig2insights batch reel1 reel2 reel3
  ig2insights batch --file reels.txt
  ig2insights batch reel1 --file more-reels.txt --concurrency 5`,
		RunE: runBatch,
	}

	// Batch-specific flags
	cmd.Flags().StringVarP(&batchFileFlag, "file", "f", "", "File with URLs/IDs (one per line)")
	cmd.Flags().BoolVar(&batchNoSaveMedia, "no-save-media", false, "Don't save audio/video to cache after processing")
	cmd.Flags().IntVarP(&batchConcurrency, "concurrency", "c", 10, "Max concurrent workers (max 50)")

	return cmd
}

func runBatch(cmd *cobra.Command, args []string) error {
	// Validate concurrency
	if batchConcurrency < 1 {
		batchConcurrency = 1
	}
	if batchConcurrency > 50 {
		batchConcurrency = 50
	}

	// Collect all reel IDs from args and file
	reelIDs, err := CollectInputs(args, batchFileFlag)
	if err != nil {
		return fmt.Errorf("failed to collect inputs: %w", err)
	}

	if len(reelIDs) == 0 {
		return fmt.Errorf("no valid reel URLs or IDs provided")
	}

	// Initialize app
	app, err := GetApp()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Determine output directory
	outputDir := dirFlag
	if outputDir == "" {
		outputDir = "." // Default to current directory for batch
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	ctx := context.Background()

	// Process batch
	return processBatch(ctx, app, reelIDs, outputDir)
}

func processBatch(ctx context.Context, app *App, reelIDs []string, outputDir string) error {
	total := len(reelIDs)
	progress := tui.NewBatchProgress(total, quietFlag)

	// Results collection with mutex
	var results []BatchResult
	var resultsMu sync.Mutex

	// Worker pool using semaphore pattern
	sem := make(chan struct{}, batchConcurrency)
	var wg sync.WaitGroup

	for _, reelID := range reelIDs {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(id string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			result := processOneReel(ctx, app, id, outputDir)

			// Thread-safe result collection
			resultsMu.Lock()
			results = append(results, result)
			resultsMu.Unlock()

			// Update progress display
			progress.AddResult(id, result.Success, result.Error, result.Duration, result.Cached)
		}(reelID)
	}

	wg.Wait()

	// Print completion summary
	progress.Complete()

	// Return error if any failed
	failCount := countFailed(results)
	if failCount > 0 {
		return fmt.Errorf("%d of %d reels failed", failCount, total)
	}

	return nil
}

func processOneReel(ctx context.Context, app *App, reelID string, outputDir string) BatchResult {
	start := time.Now()

	makeResult := func(success bool, errMsg string, cached bool) BatchResult {
		return BatchResult{
			ReelID:   reelID,
			Success:  success,
			Error:    errMsg,
			Duration: time.Since(start),
			Cached:   cached,
		}
	}

	opts := application.TranscribeOptions{
		Model:         modelFlag,
		NoCache:       noCacheFlag,
		Language:      languageFlag,
		SaveAudio:     audioFlag,
		SaveVideo:     videoFlag,
		SaveThumbnail: thumbnailFlag,
	}

	result, err := app.TranscribeSvc.Transcribe(ctx, reelID, opts)
	if err != nil {
		return makeResult(false, err.Error(), false)
	}

	transcriptContent, ext := formatTranscript(result.Transcript)
	transcriptPath := filepath.Join(outputDir, reelID+"."+ext)
	if err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0644); err != nil {
		return makeResult(false, fmt.Sprintf("failed to write transcript: %v", err), result.TranscriptFromCache)
	}

	// Copy requested media files
	mediaFiles := []struct {
		enabled bool
		srcPath string
		dstName string
		label   string
	}{
		{audioFlag, result.AudioPath, reelID + ".wav", "audio"},
		{videoFlag, result.VideoPath, reelID + ".mp4", "video"},
		{thumbnailFlag, result.ThumbnailPath, reelID + ".jpg", "thumbnail"},
	}

	for _, media := range mediaFiles {
		if !media.enabled || media.srcPath == "" {
			continue
		}
		dstPath := filepath.Join(outputDir, media.dstName)
		if err := copyFile(media.srcPath, dstPath); err != nil {
			return makeResult(false, fmt.Sprintf("failed to copy %s: %v", media.label, err), result.TranscriptFromCache)
		}
	}

	if batchNoSaveMedia {
		cleanupCacheMedia(ctx, app, reelID, result)
	}

	return makeResult(true, "", result.TranscriptFromCache)
}

// cleanupCacheMedia deletes audio/video/thumbnail from cache and updates cache entry
func cleanupCacheMedia(ctx context.Context, app *App, reelID string, result *application.TranscribeResult) {
	// Get current cache entry
	cached, err := app.Cache.Get(ctx, reelID)
	if err != nil || cached == nil {
		return
	}

	// Delete audio file from cache
	if cached.AudioPath != "" {
		os.Remove(cached.AudioPath)
	}

	// Delete video file from cache
	if cached.VideoPath != "" {
		os.Remove(cached.VideoPath)
	}

	// Delete thumbnail file from cache
	if cached.ThumbnailPath != "" {
		os.Remove(cached.ThumbnailPath)
	}

	// Update cache entry to remove media paths (keep transcript)
	updatedItem := &ports.CachedItem{
		Reel:          cached.Reel,
		Transcript:    cached.Transcript,
		AudioPath:     "", // Cleared
		VideoPath:     "", // Cleared
		ThumbnailPath: "", // Cleared
		CreatedAt:     cached.CreatedAt,
		ExpiresAt:     cached.ExpiresAt,
	}

	_ = app.Cache.Set(ctx, reelID, updatedItem)
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

// formatTranscript returns the transcript content and file extension based on formatFlag
func formatTranscript(transcript *domain.Transcript) (content, ext string) {
	if formatFlag == "srt" {
		return transcript.ToSRT(), "srt"
	}
	return transcript.ToText(), "txt"
}
