package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// NewModelCmd creates the model subcommand
func NewModelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Manage Whisper models",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available models",
		RunE:  runModelList,
	}

	downloadCmd := &cobra.Command{
		Use:   "download <model>",
		Short: "Download a model",
		Args:  cobra.ExactArgs(1),
		RunE:  runModelDownload,
	}

	removeCmd := &cobra.Command{
		Use:   "remove <model>",
		Short: "Remove a downloaded model",
		Args:  cobra.ExactArgs(1),
		RunE:  runModelRemove,
	}

	cmd.AddCommand(listCmd, downloadCmd, removeCmd)
	return cmd
}

func runModelList(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	models := app.Transcriber.AvailableModels()

	fmt.Println()
	fmt.Printf("  %-10s %-12s %s\n", "Model", "Size", "Status")
	fmt.Println("  " + strings.Repeat("-", 40))

	for _, m := range models {
		status := "not downloaded"
		if m.Downloaded {
			status = "downloaded"
		}
		if m.Name == app.Config.Defaults.Model {
			status += " (default)"
		}

		size := formatSize(m.Size)
		fmt.Printf("  %-10s %-12s %s\n", m.Name, size, status)
	}
	fmt.Println()

	return nil
}

func runModelDownload(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	model := args[0]

	if app.Transcriber.IsModelDownloaded(model) {
		fmt.Printf("Model '%s' is already downloaded\n", model)
		return nil
	}

	fmt.Printf("Downloading model '%s'...\n", model)

	err = app.Transcriber.DownloadModel(context.Background(), model, func(downloaded, total int64) {
		if total > 0 {
			pct := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rProgress: %.1f%% (%s / %s)", pct, formatSize(downloaded), formatSize(total))
		}
	})

	if err != nil {
		return err
	}

	fmt.Println("\nModel downloaded successfully")
	return nil
}

func runModelRemove(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	model := args[0]

	if !app.Transcriber.IsModelDownloaded(model) {
		fmt.Printf("Model '%s' is not downloaded\n", model)
		return nil
	}

	if err := app.Transcriber.DeleteModel(model); err != nil {
		return err
	}

	fmt.Printf("Model '%s' removed\n", model)
	return nil
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.0f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.0f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
