package cli

import (
	"fmt"
	"os"

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
	// TODO: Implement with bubbletea
	fmt.Println("Interactive menu not yet implemented")
	fmt.Println("Usage: ig2insights <reel-url|reel-id>")
	return nil
}

func runTranscribe(input string) error {
	// TODO: Implement transcription flow
	fmt.Printf("Transcribing: %s\n", input)
	return nil
}

// Execute runs the CLI
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
