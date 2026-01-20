package cli

import "time"

// BatchResult represents the result of processing a single reel in a batch
type BatchResult struct {
	ReelID   string
	Success  bool
	Error    string
	Duration time.Duration
	Cached   bool // true if transcript was from cache
}

// BatchSummary aggregates results from a batch run
type BatchSummary struct {
	Total     int
	Succeeded int
	Failed    int
	Results   []BatchResult
}

// FailedResults returns only the failed results
func (s *BatchSummary) FailedResults() []BatchResult {
	var failed []BatchResult
	for _, r := range s.Results {
		if !r.Success {
			failed = append(failed, r)
		}
	}
	return failed
}
