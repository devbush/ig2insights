package cli

import (
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
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Dependency status not yet implemented")
			return nil
		},
	}

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update yt-dlp",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Updating yt-dlp...")
			return nil
		},
	}

	cmd.AddCommand(statusCmd, updateCmd)
	return cmd
}
