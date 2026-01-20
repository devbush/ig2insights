package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	latestFlag int
	topFlag    int
)

// NewAccountCmd creates the account subcommand
// Note: Hidden because Instagram is blocking yt-dlp user page scraping
func NewAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "account <username|url>",
		Short:  "Browse and transcribe reels from an account",
		Args:   cobra.MaximumNArgs(1),
		RunE:   runAccount,
		Hidden: true, // Hidden until yt-dlp fixes Instagram user page scraping
	}

	cmd.Flags().IntVar(&latestFlag, "latest", 0, "Transcribe N most recent reels")
	cmd.Flags().IntVar(&topFlag, "top", 0, "Transcribe N most viewed reels")

	return cmd
}

func runAccount(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// Interactive: prompt for username
		fmt.Println("Account browsing not yet implemented")
		return nil
	}

	username := args[0]
	fmt.Printf("Browsing account: %s\n", username)

	if latestFlag > 0 {
		fmt.Printf("Fetching %d latest reels\n", latestFlag)
	} else if topFlag > 0 {
		fmt.Printf("Fetching %d top reels\n", topFlag)
	} else {
		// Interactive: show scrollable list
		fmt.Println("Interactive reel selection not yet implemented")
	}

	return nil
}
