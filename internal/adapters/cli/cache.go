package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var clearAllFlag bool

// NewCacheCmd creates the cache subcommand
func NewCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage cached transcripts",
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

func runCacheClear(cmd *cobra.Command, args []string) error {
	if clearAllFlag {
		fmt.Println("Clearing all cache...")
	} else {
		fmt.Println("Clearing expired cache entries...")
	}
	return nil
}
