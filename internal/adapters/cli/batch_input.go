package cli

import (
	"bufio"
	"os"
	"strings"

	"github.com/devbush/ig2insights/internal/domain"
)

// ParseInputFile reads a file containing URLs or IDs, one per line.
// Blank lines and lines starting with # are ignored.
// Returns a slice of reel IDs (extracted from URLs if needed).
func ParseInputFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ids []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip blank lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse the input to extract the reel ID
		reel, err := domain.ParseReelInput(line)
		if err != nil {
			// Skip invalid lines
			continue
		}

		ids = append(ids, reel.ID)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}

// CollectInputs combines CLI arguments and file input, deduplicating.
// Args are processed first, then file entries.
// Returns a slice of unique reel IDs in order of first appearance.
func CollectInputs(args []string, filePath string) ([]string, error) {
	seen := make(map[string]bool)
	var ids []string

	// Process CLI args first
	for _, arg := range args {
		reel, err := domain.ParseReelInput(arg)
		if err != nil {
			continue
		}
		if !seen[reel.ID] {
			seen[reel.ID] = true
			ids = append(ids, reel.ID)
		}
	}

	// Process file if provided
	if filePath != "" {
		fileIDs, err := ParseInputFile(filePath)
		if err != nil {
			return nil, err
		}
		for _, id := range fileIDs {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}

	return ids, nil
}
