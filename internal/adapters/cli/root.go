package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/devbush/ig2insights/internal/adapters/cli/tui"
	"github.com/devbush/ig2insights/internal/application"
	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	formatFlag    string
	modelFlag     string
	cacheTTLFlag  string
	noCacheFlag   bool
	dirFlag       string
	nameFlag      string
	quietFlag     bool
	languageFlag  string
	audioFlag     bool
	videoFlag     bool
	thumbnailFlag bool
)

// NewRootCmd creates the root command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ig2insights [reel-url|reel-id]",
		Short: "Transcribe Instagram Reels",
		Long: `ig2insights is a CLI tool that transcribes Instagram Reels.

Provide a reel URL or ID to transcribe it, or run without arguments
for an interactive menu.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runRoot,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&formatFlag, "format", "", "Output format: text, srt, json")
	rootCmd.PersistentFlags().StringVar(&modelFlag, "model", "small", "Whisper model: tiny, base, small, medium, large")
	rootCmd.PersistentFlags().StringVar(&cacheTTLFlag, "cache-ttl", "7d", "Cache lifetime (e.g., 24h, 7d)")
	rootCmd.PersistentFlags().BoolVar(&noCacheFlag, "no-cache", false, "Skip cache")
	rootCmd.PersistentFlags().StringVarP(&dirFlag, "dir", "d", "", "Output directory (default: ./{reelID})")
	rootCmd.PersistentFlags().StringVarP(&nameFlag, "name", "n", "", "Base filename for outputs (default: {reelID})")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress progress output")
	rootCmd.PersistentFlags().StringVarP(&languageFlag, "language", "l", "auto", "Language code (auto, en, fr, es, etc.)")
	rootCmd.PersistentFlags().BoolVar(&audioFlag, "audio", false, "Download the audio file (WAV)")
	rootCmd.PersistentFlags().BoolVar(&videoFlag, "video", false, "Download the original video file")
	rootCmd.PersistentFlags().BoolVar(&thumbnailFlag, "thumbnail", false, "Download the video thumbnail")

	// Add subcommands
	rootCmd.AddCommand(NewAccountCmd())
	rootCmd.AddCommand(NewBatchCmd())
	rootCmd.AddCommand(NewCacheCmd())
	rootCmd.AddCommand(NewModelCmd())
	rootCmd.AddCommand(NewDepsCmd())

	return rootCmd
}

func runRoot(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// No arguments - show interactive menu
		return runInteractiveMenu()
	}

	// Transcribe the provided reel
	return runTranscribe(args[0])
}

func runInteractiveMenu() error {
	options := []tui.MenuOption{
		{Label: "Transcribe a single reel", Value: "transcribe"},
		{Label: "Batch process multiple reels", Value: "batch"},
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
		return runBatchInteractive()
	case "account":
		fmt.Print("Enter username: ")
		var username string
		fmt.Scanln(&username)
		return runAccountInteractive(username)
	case "cache":
		return runCacheInteractive()
	case "quit", "":
		// User selected quit or pressed Esc
	}

	return nil
}

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

func runTranscribeInteractive() error {
	// Show output options
	checkboxOpts := []tui.CheckboxOption{
		{Label: "Transcript", Value: "transcript", Checked: true},
		{Label: "Download audio (WAV)", Value: "audio", Checked: false},
		{Label: "Download video (MP4)", Value: "video", Checked: false},
		{Label: "Download thumbnail", Value: "thumbnail", Checked: false},
	}

	selected, err := tui.RunCheckbox("What would you like to get?", checkboxOpts)
	if err != nil {
		return err
	}
	if selected == nil {
		fmt.Println("Cancelled")
		return nil
	}

	// Parse selections
	wantTranscript := false
	wantAudio := false
	wantVideo := false
	wantThumbnail := false
	for _, s := range selected {
		switch s {
		case "transcript":
			wantTranscript = true
		case "audio":
			wantAudio = true
		case "video":
			wantVideo = true
		case "thumbnail":
			wantThumbnail = true
		}
	}

	// Get reel URL
	fmt.Print("Enter reel URL or ID: ")
	var input string
	fmt.Scanln(&input)

	// Set flags based on selections
	audioFlag = wantAudio
	videoFlag = wantVideo
	thumbnailFlag = wantThumbnail

	if wantTranscript {
		return runTranscribe(input)
	}

	// Download only (no transcription)
	return runDownloadOnly(input, wantAudio, wantVideo, wantThumbnail)
}

// assetDownloadConfig holds configuration for downloading a single asset type
type assetDownloadConfig struct {
	enabled     bool
	cachedPath  string
	destPath    string
	assetType   string
	downloadFn  func() (string, error)
}

func runDownloadOnly(input string, wantAudio, wantVideo, wantThumbnail bool) error {
	app, err := GetApp()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	reel, err := domain.ParseReelInput(input)
	if err != nil {
		return err
	}

	ctx := context.Background()
	outputDir, baseName := resolveOutputPaths(reel.ID)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	cached, _ := app.Cache.Get(ctx, reel.ID)
	cacheDir := app.Cache.GetCacheDir(reel.ID)

	// Build asset download configurations
	assets := []assetDownloadConfig{
		{
			enabled:    wantAudio,
			cachedPath: getCachedPath(cached, "audio"),
			destPath:   filepath.Join(outputDir, baseName+".wav"),
			assetType:  "audio",
			downloadFn: func() (string, error) {
				if err := os.MkdirAll(cacheDir, 0755); err != nil {
					return "", err
				}
				result, err := app.Downloader.DownloadAudio(ctx, reel.ID, cacheDir)
				if err != nil {
					return "", err
				}
				return result.AudioPath, nil
			},
		},
		{
			enabled:    wantVideo,
			cachedPath: getCachedPath(cached, "video"),
			destPath:   filepath.Join(outputDir, baseName+".mp4"),
			assetType:  "video",
			downloadFn: func() (string, error) {
				cachePath := filepath.Join(cacheDir, "video.mp4")
				if err := os.MkdirAll(cacheDir, 0755); err != nil {
					return "", err
				}
				if err := app.Downloader.DownloadVideo(ctx, reel.ID, cachePath); err != nil {
					return "", err
				}
				return cachePath, nil
			},
		},
		{
			enabled:    wantThumbnail,
			cachedPath: getCachedPath(cached, "thumbnail"),
			destPath:   filepath.Join(outputDir, baseName+".jpg"),
			assetType:  "thumbnail",
			downloadFn: func() (string, error) {
				cachePath := filepath.Join(cacheDir, "thumbnail.jpg")
				if err := os.MkdirAll(cacheDir, 0755); err != nil {
					return "", err
				}
				if err := app.Downloader.DownloadThumbnail(ctx, reel.ID, cachePath); err != nil {
					return "", err
				}
				return cachePath, nil
			},
		},
	}

	// Process each asset and collect new cache paths
	cacheUpdated := false
	newCachePaths := make(map[string]string)

	for _, asset := range assets {
		if !asset.enabled {
			continue
		}

		cachePath, wasDownloaded, err := downloadAsset(asset)
		if err != nil {
			return err
		}

		newCachePaths[asset.assetType] = cachePath
		if wasDownloaded {
			cacheUpdated = true
		}
	}

	if cacheUpdated {
		updateAssetCache(ctx, app, reel.ID, cached, newCachePaths)
	}

	return nil
}

// getCachedPath returns the cached path for an asset if it exists
func getCachedPath(cached *ports.CachedItem, assetType string) string {
	if cached == nil {
		return ""
	}
	var path string
	switch assetType {
	case "audio":
		path = cached.AudioPath
	case "video":
		path = cached.VideoPath
	case "thumbnail":
		path = cached.ThumbnailPath
	}
	if path != "" && fileExists(path) {
		return path
	}
	return ""
}

// downloadAsset handles downloading or copying a single asset.
// Returns the cache path, whether a download occurred, and any error.
func downloadAsset(cfg assetDownloadConfig) (cachePath string, wasDownloaded bool, err error) {
	if cfg.cachedPath != "" {
		if !quietFlag {
			fmt.Printf("Copying %s from cache to %s...\n", cfg.assetType, cfg.destPath)
		}
		if err := copyFile(cfg.cachedPath, cfg.destPath); err != nil {
			return "", false, fmt.Errorf("failed to copy %s: %w", cfg.assetType, err)
		}
		if !quietFlag {
			fmt.Printf("%s copied\n", capitalizeFirst(cfg.assetType))
		}
		return cfg.cachedPath, false, nil
	}

	if !quietFlag {
		fmt.Printf("Downloading %s to %s...\n", cfg.assetType, cfg.destPath)
	}
	downloadedPath, err := cfg.downloadFn()
	if err != nil {
		return "", false, fmt.Errorf("%s download failed: %w", cfg.assetType, err)
	}
	if err := copyFile(downloadedPath, cfg.destPath); err != nil {
		return "", false, fmt.Errorf("failed to copy %s: %w", cfg.assetType, err)
	}
	if !quietFlag {
		fmt.Printf("%s downloaded\n", capitalizeFirst(cfg.assetType))
	}
	return downloadedPath, true, nil
}

// capitalizeFirst returns the string with its first letter capitalized
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// updateAssetCache updates the cache with newly downloaded asset paths
func updateAssetCache(ctx context.Context, app *App, reelID string, cached *ports.CachedItem, newPaths map[string]string) {
	now := time.Now()
	ttl, _ := time.ParseDuration(cacheTTLFlag)
	if ttl == 0 {
		ttl = 7 * 24 * time.Hour
	}

	cacheItem := &ports.CachedItem{
		AudioPath:     newPaths["audio"],
		VideoPath:     newPaths["video"],
		ThumbnailPath: newPaths["thumbnail"],
		CreatedAt:     now,
		ExpiresAt:     now.Add(ttl),
	}

	if cached != nil {
		cacheItem.Reel = cached.Reel
		cacheItem.Transcript = cached.Transcript
		if cacheItem.AudioPath == "" {
			cacheItem.AudioPath = cached.AudioPath
		}
		if cacheItem.VideoPath == "" {
			cacheItem.VideoPath = cached.VideoPath
		}
		if cacheItem.ThumbnailPath == "" {
			cacheItem.ThumbnailPath = cached.ThumbnailPath
		}
		cacheItem.CreatedAt = cached.CreatedAt
	}

	_ = app.Cache.Set(ctx, reelID, cacheItem)
}

// resolveOutputPaths returns the output directory and base filename
func resolveOutputPaths(reelID string) (outputDir, baseName string) {
	outputDir = dirFlag
	if outputDir == "" {
		outputDir = reelID
	}
	baseName = nameFlag
	if baseName == "" {
		baseName = reelID
	}
	return outputDir, baseName
}

// stepName returns the step name with "(cached)" suffix if cached
func stepName(name string, cached bool) string {
	if cached {
		return name + " (cached)"
	}
	return name
}

func runAccountInteractive(username string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Step 1: Ask for sort order
	sortOptions := []tui.MenuOption{
		{Label: "Latest", Value: "latest"},
		{Label: "Top (most viewed)", Value: "top"},
	}
	sortChoice, err := tui.RunMenuWithTitle("Sort by:", sortOptions)
	if err != nil {
		return err
	}
	if sortChoice == "" {
		return nil // Cancelled
	}

	currentSort := domain.SortLatest
	if sortChoice == "top" {
		currentSort = domain.SortMostViewed
	}

	// Step 2: Fetch initial reels
	fmt.Printf("Fetching reels from @%s...\n", username)
	const pageSize = 10
	reels, err := app.BrowseSvc.ListReels(ctx, username, currentSort, pageSize)
	if err != nil {
		if errors.Is(err, domain.ErrInstagramScrapingBlocked) {
			fmt.Println("\nInstagram is currently blocking profile access.")
			fmt.Println("This is a yt-dlp limitation - Instagram has restricted scraping of user pages.")
			fmt.Println("\nWorkaround: Use the 'Transcribe a single reel' option with direct reel URLs instead.")
			return nil
		}
		return fmt.Errorf("failed to fetch reels: %w", err)
	}

	if len(reels) == 0 {
		fmt.Printf("No reels found for @%s\n", username)
		return nil
	}

	hasMore := len(reels) == pageSize

	// Step 3: Paginated selection loop
	model := tui.NewReelSelectorModel(reels, currentSort, hasMore)

	for {
		p := tea.NewProgram(model)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}
		model = finalModel.(tui.ReelSelectorModel)

		switch model.Action() {
		case tui.ActionCancel:
			return nil

		case tui.ActionLoadMore:
			fmt.Println("Loading more...")
			currentCount := len(reels)
			moreReels, err := app.BrowseSvc.ListReels(ctx, username, currentSort, pageSize+currentCount)
			if err != nil {
				fmt.Printf("Error loading more: %v\n", err)
				continue
			}
			// Get only the new ones
			if len(moreReels) > currentCount {
				newReels := moreReels[currentCount:]
				hasMore = len(moreReels) == pageSize+currentCount
				model.AddReels(newReels, hasMore)
				reels = moreReels
			} else {
				hasMore = false
				model.AddReels(nil, false)
			}

		case tui.ActionChangeSort:
			// Toggle sort
			if currentSort == domain.SortLatest {
				currentSort = domain.SortMostViewed
			} else {
				currentSort = domain.SortLatest
			}
			fmt.Printf("Fetching reels sorted by %s...\n", currentSort)
			reels, err = app.BrowseSvc.ListReels(ctx, username, currentSort, pageSize)
			if err != nil {
				return fmt.Errorf("failed to fetch reels: %w", err)
			}
			hasMore = len(reels) == pageSize
			model.ClearAndSetReels(reels, currentSort, hasMore)

		case tui.ActionContinue:
			selectedReels := model.SelectedReels()
			if len(selectedReels) == 0 {
				fmt.Println("No reels selected.")
				return nil
			}

			// Step 4: Get output options
			outputOpts, err := tui.RunOutputSelector(len(selectedReels))
			if err != nil {
				return err
			}
			if outputOpts == nil {
				return nil // Cancelled
			}

			// Step 5: Process selected reels
			return processSelectedReels(ctx, app, selectedReels, outputOpts)
		}
	}
}

func processSelectedReels(ctx context.Context, app *App, reels []*domain.Reel, opts *tui.OutputOptions) error {
	total := len(reels)
	var failed []string

	for i, reel := range reels {
		fmt.Printf("Processing %d/%d: %s...\n", i+1, total, reel.ID)

		transcribeOpts := application.TranscribeOptions{
			SaveAudio:     opts.Audio,
			SaveVideo:     opts.Video,
			SaveThumbnail: opts.Thumbnail,
		}

		result, err := app.TranscribeSvc.Transcribe(ctx, reel.ID, transcribeOpts)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", reel.ID, err))
			continue
		}

		// Copy outputs to specified directory (or current if not set)
		outputDir := dirFlag
		if outputDir == "" {
			outputDir = "."
		}
		baseName := reel.ID

		if opts.Transcript && result.Transcript != nil {
			outPath := filepath.Join(outputDir, baseName+".txt")
			if err := os.WriteFile(outPath, []byte(result.Transcript.Text), 0644); err != nil {
				failed = append(failed, fmt.Sprintf("%s (transcript): %v", reel.ID, err))
			}
		}

		if opts.Audio && result.AudioPath != "" {
			outPath := filepath.Join(outputDir, baseName+".wav")
			if err := copyFile(result.AudioPath, outPath); err != nil {
				failed = append(failed, fmt.Sprintf("%s (audio): %v", reel.ID, err))
			}
		}

		if opts.Video && result.VideoPath != "" {
			outPath := filepath.Join(outputDir, baseName+".mp4")
			if err := copyFile(result.VideoPath, outPath); err != nil {
				failed = append(failed, fmt.Sprintf("%s (video): %v", reel.ID, err))
			}
		}

		if opts.Thumbnail && result.ThumbnailPath != "" {
			outPath := filepath.Join(outputDir, baseName+".jpg")
			if err := copyFile(result.ThumbnailPath, outPath); err != nil {
				failed = append(failed, fmt.Sprintf("%s (thumbnail): %v", reel.ID, err))
			}
		}
	}

	// Summary
	succeeded := total - len(failed)
	fmt.Printf("\nCompleted %d/%d reels.\n", succeeded, total)
	if len(failed) > 0 {
		fmt.Println("Failed:")
		for _, f := range failed {
			fmt.Printf("  - %s\n", f)
		}
	}

	return nil
}

func runCacheInteractive() error {
	// TODO: Implement cache management
	fmt.Println("Cache management not yet implemented")
	return nil
}

func runTranscribe(input string) error {
	app, err := GetApp()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	reel, err := domain.ParseReelInput(input)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Pre-flight cache check to determine what's cached
	var cached *ports.CachedItem
	if !noCacheFlag {
		cached, _ = app.Cache.Get(ctx, reel.ID)
	}

	hasTranscript := cached != nil && cached.Transcript != nil
	hasAudio := cached != nil && cached.AudioPath != "" && fileExists(cached.AudioPath)
	hasVideo := cached != nil && cached.VideoPath != "" && fileExists(cached.VideoPath)
	hasThumbnail := cached != nil && cached.ThumbnailPath != "" && fileExists(cached.ThumbnailPath)

	// Build step list based on what we're doing and what's cached
	steps := []string{"Checking dependencies"}
	steps = append(steps, stepName("Downloading video", hasTranscript))
	steps = append(steps, stepName("Extracting audio", hasTranscript))
	steps = append(steps, stepName("Transcribing", hasTranscript))

	audioStepIdx := -1
	videoStepIdx := -1
	thumbStepIdx := -1

	if audioFlag {
		audioStepIdx = len(steps)
		steps = append(steps, stepName("Saving audio", hasAudio))
	}
	if videoFlag {
		videoStepIdx = len(steps)
		steps = append(steps, stepName("Saving video", hasVideo))
	}
	if thumbnailFlag {
		thumbStepIdx = len(steps)
		steps = append(steps, stepName("Downloading thumbnail", hasThumbnail))
	}

	progress := tui.NewProgressDisplay(steps, quietFlag)

	// Step 1: Check dependencies
	progress.StartStep(0)

	if !app.Downloader.IsAvailable() {
		if err := app.Downloader.Install(context.Background(), func(d, t int64) {
			progress.UpdateProgress(0, d, t)
		}); err != nil {
			progress.FailStep(0, err.Error())
			return fmt.Errorf("failed to install yt-dlp: %w", err)
		}
	}

	if !app.Transcriber.IsAvailable() {
		instructions := app.Transcriber.InstallationInstructions()
		if instructions != "" {
			progress.FailStep(0, "whisper.cpp not found")
			return errors.New(instructions)
		}
		if err := app.Transcriber.Install(context.Background(), func(d, t int64) {
			progress.UpdateProgress(0, d, t)
		}); err != nil {
			progress.FailStep(0, err.Error())
			return fmt.Errorf("failed to install whisper.cpp: %w", err)
		}
	}

	if !app.Downloader.IsFFmpegAvailable() {
		instructions := app.Downloader.FFmpegInstructions()
		if instructions != "" {
			progress.FailStep(0, "ffmpeg not found")
			return errors.New(instructions)
		}
		if err := app.Downloader.InstallFFmpeg(context.Background(), func(d, t int64) {
			progress.UpdateProgress(0, d, t)
		}); err != nil {
			progress.FailStep(0, err.Error())
			return fmt.Errorf("failed to install ffmpeg: %w", err)
		}
	}

	model := modelFlag
	if model == "" {
		model = app.Config.Defaults.Model
	}

	if !app.Transcriber.IsModelDownloaded(model) {
		if err := app.Transcriber.DownloadModel(context.Background(), model, func(d, t int64) {
			progress.UpdateProgress(0, d, t)
		}); err != nil {
			progress.FailStep(0, err.Error())
			return fmt.Errorf("failed to download model: %w", err)
		}
	}
	progress.CompleteStep(0)

	// Start spinner for indeterminate steps
	spinnerDone := progress.StartSpinner()

	// If transcript is cached, immediately complete those steps
	if hasTranscript {
		progress.CompleteStep(1) // Download (cached)
		progress.CompleteStep(2) // Extract (cached)
		progress.CompleteStep(3) // Transcribe (cached)
	} else {
		progress.StartStep(1)
	}

	result, err := app.TranscribeSvc.Transcribe(ctx, reel.ID, application.TranscribeOptions{
		Model:         model,
		NoCache:       noCacheFlag,
		Language:      languageFlag,
		SaveAudio:     audioFlag,
		SaveVideo:     videoFlag,
		SaveThumbnail: thumbnailFlag,
	})

	if err != nil {
		close(spinnerDone)
		progress.FailStep(1, err.Error())
		return err
	}

	// Mark transcription steps complete if not already
	if !hasTranscript {
		progress.CompleteStep(1) // Download
		progress.CompleteStep(2) // Extract
		progress.CompleteStep(3) // Transcribe
	}

	outputDir, baseName := resolveOutputPaths(reel.ID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		close(spinnerDone)
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputs := make(map[string]string)

	// Step: Save audio (if requested)
	if audioFlag && audioStepIdx >= 0 {
		if result.AudioFromCache {
			progress.CompleteStep(audioStepIdx)
		} else {
			progress.StartStep(audioStepIdx)
		}

		audioPath := filepath.Join(outputDir, baseName+".wav")
		if result.AudioPath != "" {
			if err := copyFile(result.AudioPath, audioPath); err != nil {
				progress.FailStep(audioStepIdx, err.Error())
			} else {
				progress.CompleteStep(audioStepIdx)
				outputs["Audio"] = audioPath
			}
		} else {
			progress.FailStep(audioStepIdx, "no audio available")
		}
	}

	// Step: Save video (if requested)
	if videoFlag && videoStepIdx >= 0 {
		if result.VideoFromCache {
			// Already marked as cached in step name, just complete it
			progress.CompleteStep(videoStepIdx)
		} else {
			progress.StartStep(videoStepIdx)
		}

		videoPath := filepath.Join(outputDir, baseName+".mp4")
		if result.VideoPath != "" {
			if err := copyFile(result.VideoPath, videoPath); err != nil {
				progress.FailStep(videoStepIdx, err.Error())
			} else {
				progress.CompleteStep(videoStepIdx)
				outputs["Video"] = videoPath
			}
		} else {
			progress.FailStep(videoStepIdx, "no video available")
		}
	}

	// Step: Download thumbnail (if requested)
	if thumbnailFlag && thumbStepIdx >= 0 {
		if result.ThumbnailFromCache {
			progress.CompleteStep(thumbStepIdx)
		} else {
			progress.StartStep(thumbStepIdx)
		}

		thumbPath := filepath.Join(outputDir, baseName+".jpg")
		if result.ThumbnailPath != "" {
			if err := copyFile(result.ThumbnailPath, thumbPath); err != nil {
				progress.FailStep(thumbStepIdx, err.Error())
			} else {
				progress.CompleteStep(thumbStepIdx)
				outputs["Thumbnail"] = thumbPath
			}
		} else {
			// Thumbnail wasn't cached, try downloading directly
			if err := app.Downloader.DownloadThumbnail(ctx, reel.ID, thumbPath); err != nil {
				progress.FailStep(thumbStepIdx, err.Error())
			} else {
				progress.CompleteStep(thumbStepIdx)
				outputs["Thumbnail"] = thumbPath
			}
		}
	}

	// Stop spinner
	close(spinnerDone)

	// Output transcript
	transcriptPath, err := outputResult(result, outputDir, baseName)
	if err != nil {
		return err
	}
	outputs["Transcript"] = transcriptPath

	if !quietFlag && len(outputs) > 0 {
		progress.Complete(outputs)
	}

	return nil
}

func printProgress(downloaded, total int64) {
	if quietFlag {
		return
	}
	if total > 0 {
		pct := float64(downloaded) / float64(total) * 100
		fmt.Printf("\rDownloading... %.1f%%", pct)
	}
}

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

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Execute runs the CLI
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
