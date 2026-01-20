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
		Short: "Manage dependencies (yt-dlp)",
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
		fmt.Printf("  yt-dlp:   installed (%s)\n", path)
	} else {
		fmt.Println("  yt-dlp:   not found")
	}

	// Whisper models
	models := app.Transcriber.AvailableModels()
	downloaded := 0
	for _, m := range models {
		if m.Downloaded {
			downloaded++
		}
	}
	fmt.Printf("  whisper:  %d/%d models downloaded\n", downloaded, len(models))
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

	if app.Downloader.IsAvailable() {
		fmt.Println("yt-dlp is already installed")
		return nil
	}

	fmt.Println("Installing yt-dlp...")

	ctx := context.Background()
	err = app.Downloader.Install(ctx, func(downloaded, total int64) {
		if total > 0 {
			pct := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rProgress: %.1f%%", pct)
		}
	})

	if err != nil {
		return err
	}

	fmt.Println("\nyt-dlp installed")
	return nil
}
