package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// renderProgressBar creates a text progress bar like [=====>    ]
func renderProgressBar(current, total, width int) string {
	if total <= 0 || current == 0 {
		return "[" + strings.Repeat(" ", width) + "]"
	}

	if current >= total {
		return "[" + strings.Repeat("=", width) + "]"
	}

	// Calculate filled portion: for ratio r, filled = floor(r * width)
	// This gives: 3/10 * 10 = 3 -> 2 equals + arrow, 5/10 * 10 = 5 -> 5 equals + arrow
	filled := (current * width) / total
	if filled < 1 {
		filled = 1
	}

	equals := filled - 1
	if current*2 >= total {
		// At 50% or more, show filled equals before arrow
		equals = filled
	}

	spaces := width - equals - 1
	if spaces < 0 {
		spaces = 0
	}

	return "[" + strings.Repeat("=", equals) + ">" + strings.Repeat(" ", spaces) + "]"
}

// BatchResult represents the result of processing a single reel
type BatchResult struct {
	ReelID   string
	Success  bool
	ErrMsg   string
	Duration time.Duration
	Cached   bool
}

// BatchProgress manages batch processing progress display
type BatchProgress struct {
	total     int
	completed int
	results   []BatchResult
	failures  []BatchResult
	quiet     bool
	mu        sync.Mutex
	rendered  bool
}

// NewBatchProgress creates a new batch progress display
func NewBatchProgress(total int, quiet bool) *BatchProgress {
	if total < 0 {
		total = 0
	}
	return &BatchProgress{
		total:    total,
		results:  make([]BatchResult, 0),
		failures: make([]BatchResult, 0),
		quiet:    quiet,
	}
}

// AddResult adds a result and updates the display
func (bp *BatchProgress) AddResult(reelID string, success bool, errMsg string, duration time.Duration, cached bool) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	result := BatchResult{
		ReelID:   reelID,
		Success:  success,
		ErrMsg:   errMsg,
		Duration: duration,
		Cached:   cached,
	}

	bp.results = append(bp.results, result)
	bp.completed++

	if !success {
		bp.failures = append(bp.failures, result)
	}

	bp.render()
}

func (bp *BatchProgress) render() {
	if bp.quiet {
		return
	}

	// Calculate how many lines to clear (progress line + up to 10 results)
	linesToClear := 1 + min(len(bp.results), 10)
	if bp.rendered && linesToClear > 0 {
		// Move cursor up and clear
		fmt.Printf("\033[%dA", linesToClear)
		fmt.Print("\033[J")
	}

	// Render progress line
	percent := 0
	if bp.total > 0 {
		percent = (bp.completed * 100) / bp.total
	}
	progressBar := renderProgressBar(bp.completed, bp.total, 20)
	fmt.Printf("Batch processing %d/%d reels %s %d%%\n", bp.completed, bp.total, progressBar, percent)

	// Render last 10 results
	startIdx := 0
	if len(bp.results) > 10 {
		startIdx = len(bp.results) - 10
	}

	for i := startIdx; i < len(bp.results); i++ {
		result := bp.results[i]
		if result.Success {
			cached := ""
			if result.Cached {
				cached = " [cached]"
			}
			fmt.Printf("✓ %s (%.1fs)%s\n", result.ReelID, result.Duration.Seconds(), cached)
		} else {
			fmt.Printf("✗ %s: %s\n", result.ReelID, result.ErrMsg)
		}
	}

	bp.rendered = true
}

// Complete prints the final summary
func (bp *BatchProgress) Complete() {
	if bp.quiet {
		return
	}

	bp.mu.Lock()
	completed := bp.completed
	total := bp.total
	failures := make([]BatchResult, len(bp.failures))
	copy(failures, bp.failures)
	bp.mu.Unlock()

	succeeded := completed - len(failures)

	fmt.Println()
	fmt.Printf("Batch complete: %d/%d succeeded\n", succeeded, total)

	if len(failures) > 0 {
		fmt.Println("\nFailures:")
		for _, f := range failures {
			fmt.Printf("  ✗ %s: %s\n", f.ReelID, f.ErrMsg)
		}
	}
}

// GetSuccessCount returns the number of successful results
func (bp *BatchProgress) GetSuccessCount() int {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.completed - len(bp.failures)
}

// GetFailureCount returns the number of failed results
func (bp *BatchProgress) GetFailureCount() int {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return len(bp.failures)
}
