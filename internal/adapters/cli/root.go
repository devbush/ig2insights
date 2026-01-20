package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/devbush/ig2insights/internal/adapters/cli/tui"
	"github.com/devbush/ig2insights/internal/application"
	"github.com/devbush/ig2insights/internal/domain"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	formatFlag   string
	modelFlag    string
	cacheTTLFlag string
	noCacheFlag  bool
	outputFlag   string
	quietFlag    bool
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
		fmt.Print("Enter reel URL or ID: ")
		var input string
		fmt.Scanln(&input)
		return runTranscribe(input)
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

	// Parse input
	reel, err := domain.ParseReelInput(input)
	if err != nil {
		return err
	}

	// Check dependencies
	if !app.Downloader.IsAvailable() {
		fmt.Println("yt-dlp not found. Installing...")
		if err := app.Downloader.Install(context.Background(), printProgress); err != nil {
			return fmt.Errorf("failed to install yt-dlp: %w", err)
		}
		fmt.Println("\n✓ yt-dlp installed")
	}

	if !app.Transcriber.IsAvailable() {
		instructions := app.Transcriber.InstallationInstructions()
		if instructions != "" {
			return errors.New(instructions)
		}
		fmt.Println("whisper.cpp not found. Installing...")
		if err := app.Transcriber.Install(context.Background(), printProgress); err != nil {
			return fmt.Errorf("failed to install whisper.cpp: %w", err)
		}
		fmt.Println("\n✓ whisper.cpp installed")
	}

	model := modelFlag
	if model == "" {
		model = app.Config.Defaults.Model
	}

	if !app.Transcriber.IsModelDownloaded(model) {
		fmt.Printf("Model '%s' not found. Downloading...\n", model)
		if err := app.Transcriber.DownloadModel(context.Background(), model, printProgress); err != nil {
			return fmt.Errorf("failed to download model: %w", err)
		}
		fmt.Println("\n✓ Model downloaded")
	}

	// Transcribe
	if !quietFlag {
		fmt.Printf("Transcribing %s...\n", reel.ID)
	}

	ctx := context.Background()
	result, err := app.TranscribeSvc.Transcribe(ctx, reel.ID, application.TranscribeOptions{
		Model:   model,
		NoCache: noCacheFlag,
	})
	if err != nil {
		return err
	}

	if result.FromCache && !quietFlag {
		fmt.Println("(from cache)")
	}

	// Output
	return outputResult(result)
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
