package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// NewDepsCmd creates the deps subcommand
func NewDepsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deps",
		Short: "Manage dependencies (yt-dlp, whisper.cpp, ffmpeg)",
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show dependency status",
		RunE:  runDepsStatus,
	}

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update yt-dlp to latest version",
		RunE:  runDepsUpdate,
	}

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install yt-dlp",
		RunE:  runDepsInstall,
	}

	cmd.AddCommand(statusCmd, updateCmd, installCmd)
	return cmd
}

func runDepsStatus(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Dependency Status:")
	fmt.Println()

	// yt-dlp
	if app.Downloader.IsAvailable() {
		path := app.Downloader.GetBinaryPath()
		fmt.Printf("  yt-dlp:        installed (%s)\n", path)
	} else {
		fmt.Println("  yt-dlp:        not found")
	}

	// Whisper binary
	if app.Transcriber.IsAvailable() {
		path := app.Transcriber.GetBinaryPath()
		fmt.Printf("  whisper.cpp:   installed (%s)\n", path)
	} else {
		fmt.Println("  whisper.cpp:   not found")
	}

	// ffmpeg
	if app.Downloader.IsFFmpegAvailable() {
		path := app.Downloader.GetFFmpegPath()
		fmt.Printf("  ffmpeg:        installed (%s)\n", path)
	} else {
		fmt.Println("  ffmpeg:        not found")
	}

	// Whisper models
	models := app.Transcriber.AvailableModels()
	downloaded := 0
	for _, m := range models {
		if m.Downloaded {
			downloaded++
		}
	}
	fmt.Printf("  whisper models: %d/%d downloaded\n", downloaded, len(models))
	fmt.Println()

	return nil
}

func runDepsUpdate(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	if !app.Downloader.IsAvailable() {
		return fmt.Errorf("yt-dlp is not installed. Run 'ig2insights deps install' first")
	}

	fmt.Println("Updating yt-dlp...")

	ctx := context.Background()
	if err := app.Downloader.Update(ctx); err != nil {
		return err
	}

	fmt.Println("yt-dlp updated")
	return nil
}

func runDepsInstall(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	ctx := context.Background()
	progress := func(downloaded, total int64) {
		if total > 0 {
			pct := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rProgress: %.1f%%", pct)
		}
	}

	// Install yt-dlp
	if app.Downloader.IsAvailable() {
		fmt.Println("yt-dlp is already installed")
	} else {
		fmt.Println("Installing yt-dlp...")
		if err := app.Downloader.Install(ctx, progress); err != nil {
			return fmt.Errorf("failed to install yt-dlp: %w", err)
		}
		fmt.Println("\nyt-dlp installed")
	}

	// Install whisper.cpp
	if app.Transcriber.IsAvailable() {
		fmt.Println("whisper.cpp is already installed")
	} else {
		fmt.Println("Installing whisper.cpp...")
		if err := app.Transcriber.Install(ctx, progress); err != nil {
			return fmt.Errorf("failed to install whisper.cpp: %w", err)
		}
		fmt.Println("\nwhisper.cpp installed")
	}

	// Install ffmpeg
	if app.Downloader.IsFFmpegAvailable() {
		fmt.Println("ffmpeg is already installed")
	} else {
		instructions := app.Downloader.FFmpegInstructions()
		if instructions != "" {
			fmt.Println(instructions)
		} else {
			fmt.Println("Installing ffmpeg...")
			if err := app.Downloader.InstallFFmpeg(ctx, progress); err != nil {
				return fmt.Errorf("failed to install ffmpeg: %w", err)
			}
			fmt.Println("\nffmpeg installed")
		}
	}

	return nil
}
