package cli

import (
	"context"
	"fmt"

	"github.com/devbush/ig2insights/internal/adapters/cli/tui"
	"github.com/spf13/cobra"
)

var clearAllFlag bool

// NewCacheCmd creates the cache subcommand
func NewCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage cached transcripts",
		RunE:  runCacheStatus,
	}

	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear cache entries",
		RunE:  runCacheClear,
	}
	clearCmd.Flags().BoolVar(&clearAllFlag, "all", false, "Clear all cache entries")

	cmd.AddCommand(clearCmd)

	return cmd
}

func runCacheStatus(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	ctx := context.Background()
	stats, err := app.CacheSvc.Stats(ctx)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Cache Statistics:")
	fmt.Printf("  Items: %d\n", stats.ItemCount)
	fmt.Printf("  Size:  %s\n", tui.FormatSize(stats.TotalSize))
	fmt.Printf("  TTL:   %s\n", app.Config.Defaults.CacheTTL)
	fmt.Println()

	return nil
}

func runCacheClear(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	ctx := context.Background()

	if clearAllFlag {
		if err := app.CacheSvc.Clear(ctx); err != nil {
			return err
		}
		fmt.Println("All cache entries cleared")
	} else {
		cleaned, err := app.CacheSvc.CleanExpired(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("Removed %d expired entries\n", cleaned)
	}

	return nil
}
