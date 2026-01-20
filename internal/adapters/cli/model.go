package cli

import (
	"fmt"

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
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Model list not yet implemented")
			return nil
		},
	}

	downloadCmd := &cobra.Command{
		Use:   "download <model>",
		Short: "Download a model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Downloading model: %s\n", args[0])
			return nil
		},
	}

	removeCmd := &cobra.Command{
		Use:   "remove <model>",
		Short: "Remove a downloaded model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Removing model: %s\n", args[0])
			return nil
		},
	}

	cmd.AddCommand(listCmd, downloadCmd, removeCmd)
	return cmd
}
