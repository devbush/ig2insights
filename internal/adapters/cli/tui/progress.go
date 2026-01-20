package tui

import (
	"fmt"
	"sync"
	"time"
)

// StepStatus represents the state of a progress step
type StepStatus int

const (
	StepPending StepStatus = iota
	StepRunning
	StepComplete
	StepError
)

// ProgressStep represents a single step in the progress
type ProgressStep struct {
	Name     string
	Status   StepStatus
	Progress float64 // 0-100, only used for download steps
	Total    int64   // Total bytes for download
	Current  int64   // Current bytes for download
	Error    string
}

// ProgressDisplay manages multi-step progress output
type ProgressDisplay struct {
	steps       []ProgressStep
	currentStep int
	spinnerIdx  int
	quiet       bool
	mu          sync.Mutex
	lastRender  time.Time
	rendered    bool
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewProgressDisplay creates a new progress display
func NewProgressDisplay(steps []string, quiet bool) *ProgressDisplay {
	pd := &ProgressDisplay{
		steps: make([]ProgressStep, len(steps)),
		quiet: quiet,
	}
	for i, name := range steps {
		pd.steps[i] = ProgressStep{Name: name, Status: StepPending}
	}
	return pd
}

// StartStep marks a step as running
func (p *ProgressDisplay) StartStep(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if index >= 0 && index < len(p.steps) {
		p.currentStep = index
		p.steps[index].Status = StepRunning
		p.render()
	}
}

// CompleteStep marks a step as complete
func (p *ProgressDisplay) CompleteStep(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if index >= 0 && index < len(p.steps) {
		p.steps[index].Status = StepComplete
		p.render()
	}
}

// FailStep marks a step as failed
func (p *ProgressDisplay) FailStep(index int, err string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if index >= 0 && index < len(p.steps) {
		p.steps[index].Status = StepError
		p.steps[index].Error = err
		p.render()
	}
}

// UpdateProgress updates download progress for a step
func (p *ProgressDisplay) UpdateProgress(index int, current, total int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if index >= 0 && index < len(p.steps) {
		p.steps[index].Current = current
		p.steps[index].Total = total
		if total > 0 {
			p.steps[index].Progress = float64(current) / float64(total) * 100
		}
		// Throttle renders to avoid flickering
		if time.Since(p.lastRender) > 100*time.Millisecond {
			p.render()
		}
	}
}

// Tick advances the spinner animation
func (p *ProgressDisplay) Tick() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.spinnerIdx = (p.spinnerIdx + 1) % len(spinnerFrames)
	p.render()
}

func (p *ProgressDisplay) render() {
	if p.quiet {
		return
	}

	p.lastRender = time.Now()

	// Clear previous lines and redraw
	// Move cursor up by number of steps, clear each line
	if p.rendered {
		fmt.Print("\033[" + fmt.Sprintf("%d", len(p.steps)) + "A") // Move up
		fmt.Print("\033[J")                                        // Clear from cursor to end
	}

	total := len(p.steps)
	for i, step := range p.steps {
		stepNum := fmt.Sprintf("[%d/%d]", i+1, total)

		var status string
		switch step.Status {
		case StepPending:
			status = " "
		case StepRunning:
			if step.Total > 0 {
				// Download progress
				status = fmt.Sprintf("%.1f%% (%s / %s)",
					step.Progress,
					formatBytes(step.Current),
					formatBytes(step.Total))
			} else {
				// Spinner
				status = spinnerFrames[p.spinnerIdx]
			}
		case StepComplete:
			status = "✓"
		case StepError:
			status = "✗"
		}

		fmt.Printf("%s %s... %s\n", stepNum, step.Name, status)
	}

	p.rendered = true
}

// Complete prints the final success message
func (p *ProgressDisplay) Complete(outputs map[string]string) {
	if p.quiet {
		return
	}

	fmt.Println()
	fmt.Println("✓ Complete!")
	for label, path := range outputs {
		fmt.Printf("  %s: %s\n", label, path)
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// StartSpinner starts a goroutine that ticks the spinner
func (p *ProgressDisplay) StartSpinner() chan struct{} {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				p.Tick()
			}
		}
	}()
	return done
}
