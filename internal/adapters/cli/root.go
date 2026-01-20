package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/devbush/ig2insights/internal/adapters/cli/tui"
	"github.com/devbush/ig2insights/internal/application"
	"github.com/devbush/ig2insights/internal/domain"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	formatFlag      string
	modelFlag       string
	cacheTTLFlag    string
	noCacheFlag     bool
	outputFlag      string
	quietFlag       bool
	languageFlag    string
	videoFlag       bool
	thumbnailFlag   bool
	downloadDirFlag string
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
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "", "Output file path")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress progress output")
	rootCmd.PersistentFlags().StringVarP(&languageFlag, "language", "l", "auto", "Language code (auto, en, fr, es, etc.)")
	rootCmd.PersistentFlags().BoolVar(&videoFlag, "video", false, "Download the original video file")
	rootCmd.PersistentFlags().BoolVar(&thumbnailFlag, "thumbnail", false, "Download the video thumbnail")
	rootCmd.PersistentFlags().StringVar(&downloadDirFlag, "download-dir", "", "Directory for downloaded assets (default: same as output)")

	// Add subcommands
	rootCmd.AddCommand(NewAccountCmd())
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
		{Label: "Browse an account's reels", Value: "account"},
		{Label: "Manage cache", Value: "cache"},
		{Label: "Settings", Value: "settings"},
	}

	selected, err := tui.RunMenu(options)
	if err != nil {
		return err
	}

	switch selected {
	case "transcribe":
		return runTranscribeInteractive()
	case "account":
		fmt.Print("Enter username: ")
		var username string
		fmt.Scanln(&username)
		return runAccountInteractive(username)
	case "cache":
		return runCacheInteractive()
	case "settings":
		fmt.Println("Settings not yet implemented")
	case "":
		fmt.Println("Cancelled")
	}

	return nil
}

func runTranscribeInteractive() error {
	// Show output options
	checkboxOpts := []tui.CheckboxOption{
		{Label: "Transcript", Value: "transcript", Checked: true},
		{Label: "Download video", Value: "video", Checked: false},
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
	wantVideo := false
	wantThumbnail := false
	for _, s := range selected {
		switch s {
		case "transcript":
			wantTranscript = true
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
	videoFlag = wantVideo
	thumbnailFlag = wantThumbnail

	if wantTranscript {
		return runTranscribe(input)
	}

	// Download only (no transcription)
	return runDownloadOnly(input, wantVideo, wantThumbnail)
}

func runDownloadOnly(input string, video, thumbnail bool) error {
	app, err := GetApp()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	reel, err := domain.ParseReelInput(input)
	if err != nil {
		return err
	}

	ctx := context.Background()
	outputDir := downloadDirFlag
	if outputDir == "" {
		outputDir = "."
	}

	if video {
		destPath := filepath.Join(outputDir, reel.ID+".mp4")
		fmt.Printf("Downloading video to %s...\n", destPath)
		if err := app.Downloader.DownloadVideo(ctx, reel.ID, destPath); err != nil {
			return fmt.Errorf("video download failed: %w", err)
		}
		fmt.Println("✓ Video downloaded")
	}

	if thumbnail {
		destPath := filepath.Join(outputDir, reel.ID+".jpg")
		fmt.Printf("Downloading thumbnail to %s...\n", destPath)
		if err := app.Downloader.DownloadThumbnail(ctx, reel.ID, destPath); err != nil {
			return fmt.Errorf("thumbnail download failed: %w", err)
		}
		fmt.Println("✓ Thumbnail downloaded")
	}

	return nil
}

func runAccountInteractive(username string) error {
	// TODO: Implement with BrowseService
	fmt.Printf("Browsing %s...\n", username)
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

	// Build step list based on what we're doing
	steps := []string{"Checking dependencies", "Downloading video", "Extracting audio", "Transcribing"}

	videoStepIdx := -1
	thumbStepIdx := -1

	if videoFlag {
		videoStepIdx = len(steps)
		steps = append(steps, "Saving video")
	}
	if thumbnailFlag {
		thumbStepIdx = len(steps)
		steps = append(steps, "Downloading thumbnail")
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

	// Step 2: Download video (starts at step index 1)
	progress.StartStep(1)

	ctx := context.Background()
	result, err := app.TranscribeSvc.Transcribe(ctx, reel.ID, application.TranscribeOptions{
		Model:    model,
		NoCache:  noCacheFlag,
		Language: languageFlag,
	})

	if err != nil {
		close(spinnerDone)
		progress.FailStep(1, err.Error())
		return err
	}

	// Mark transcription steps complete
	progress.CompleteStep(1) // Download
	progress.CompleteStep(2) // Extract
	progress.CompleteStep(3) // Transcribe

	// Determine output directory
	outputDir := downloadDirFlag
	if outputDir == "" {
		if outputFlag != "" {
			outputDir = filepath.Dir(outputFlag)
		} else {
			outputDir = "."
		}
	}

	outputs := make(map[string]string)

	// Step: Save video (if requested)
	if videoFlag && videoStepIdx >= 0 {
		progress.StartStep(videoStepIdx)
		videoPath := filepath.Join(outputDir, reel.ID+".mp4")
		if err := app.Downloader.DownloadVideo(ctx, reel.ID, videoPath); err != nil {
			progress.FailStep(videoStepIdx, err.Error())
			// Non-fatal, continue
		} else {
			progress.CompleteStep(videoStepIdx)
			outputs["Video"] = videoPath
		}
	}

	// Step: Download thumbnail (if requested)
	if thumbnailFlag && thumbStepIdx >= 0 {
		progress.StartStep(thumbStepIdx)
		thumbPath := filepath.Join(outputDir, reel.ID+".jpg")
		if err := app.Downloader.DownloadThumbnail(ctx, reel.ID, thumbPath); err != nil {
			progress.FailStep(thumbStepIdx, err.Error())
			// Non-fatal, continue
		} else {
			progress.CompleteStep(thumbStepIdx)
			outputs["Thumbnail"] = thumbPath
		}
	}

	// Stop spinner
	close(spinnerDone)

	// Output transcript
	if err := outputResult(result); err != nil {
		return err
	}

	if outputFlag != "" {
		outputs["Transcript"] = outputFlag
	}

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

func outputResult(result *application.TranscribeResult) error {
	format := formatFlag
	if format == "" {
		format = "text"
	}

	var output string
	switch format {
	case "text":
		output = result.Transcript.ToText()
	case "srt":
		output = result.Transcript.ToSRT()
	case "json":
		data := map[string]interface{}{
			"reel":       result.Reel,
			"transcript": result.Transcript,
		}
		jsonBytes, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		output = string(jsonBytes)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	if outputFlag != "" {
		return os.WriteFile(outputFlag, []byte(output), 0644)
	}

	fmt.Println(output)
	return nil
}

// Execute runs the CLI
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
