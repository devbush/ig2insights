# Progress Display & Download Options - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add step-based progress display with spinners, optional video/thumbnail downloads, and fix language auto-detection.

**Architecture:** Extend existing TUI components with new checkbox and progress models. Add download flags to CLI and wire through application layer to adapters.

**Tech Stack:** Go, Bubble Tea (TUI), yt-dlp (downloads), whisper.cpp (transcription)

---

## Task 1: Fix Language Auto-Detection Bug

**Files:**
- Modify: `internal/adapters/whisper/transcriber.go:202-211`
- Modify: `internal/application/transcribe.go:12-16, 70-77`

**Step 1: Update TranscribeOptions to include Language**

In `internal/application/transcribe.go`, add Language field:

```go
// TranscribeOptions configures the transcription
type TranscribeOptions struct {
	Model    string
	Format   string // text, srt, json
	NoCache  bool
	Language string // empty defaults to "auto"
}
```

**Step 2: Pass Language through to transcriber**

In `internal/application/transcribe.go`, update the Transcribe method around line 75:

```go
	language := opts.Language
	if language == "" {
		language = "auto"
	}

	transcript, err := s.transcriber.Transcribe(ctx, downloadResult.VideoPath, ports.TranscribeOpts{
		Model:    model,
		Language: language,
	})
```

**Step 3: Always pass -l flag to whisper**

In `internal/adapters/whisper/transcriber.go`, update args construction (lines 202-211):

```go
	language := opts.Language
	if language == "" {
		language = "auto"
	}

	args := []string{
		"-m", t.modelPath(model),
		"-f", videoPath,
		"-of", outputBase,
		"-oj",
		"-l", language,
	}
```

Remove the conditional `-l` block that follows (lines 209-211).

**Step 4: Build and verify**

Run: `go build ./...`
Expected: No errors

**Step 5: Commit**

```bash
git add internal/application/transcribe.go internal/adapters/whisper/transcriber.go
git commit -m "fix(whisper): default to auto language detection

Previously, omitting -l flag caused whisper to default to English,
translating non-English audio. Now always passes -l auto."
```

---

## Task 2: Add New CLI Flags

**Files:**
- Modify: `internal/adapters/cli/root.go:16-24, 40-45`

**Step 1: Add flag variables**

In `internal/adapters/cli/root.go`, add to the var block (after line 23):

```go
var (
	// Global flags
	formatFlag      string
	modelFlag       string
	cacheTTLFlag    string
	noCacheFlag     bool
	outputFlag      string
	quietFlag       bool
	languageFlag    string
	videoFlag       bool
	thumbnailFlag   bool
	downloadDirFlag string
)
```

**Step 2: Register flags in NewRootCmd**

In `internal/adapters/cli/root.go`, add after line 45:

```go
	rootCmd.PersistentFlags().StringVarP(&languageFlag, "language", "l", "auto", "Language code (auto, en, fr, es, etc.)")
	rootCmd.PersistentFlags().BoolVar(&videoFlag, "video", false, "Download the original video file")
	rootCmd.PersistentFlags().BoolVar(&thumbnailFlag, "thumbnail", false, "Download the video thumbnail")
	rootCmd.PersistentFlags().StringVar(&downloadDirFlag, "download-dir", "", "Directory for downloaded assets (default: same as output)")
```

**Step 3: Build and verify**

Run: `go build ./... && ./ig2insights --help`
Expected: New flags appear in help output

**Step 4: Commit**

```bash
git add internal/adapters/cli/root.go
git commit -m "feat(cli): add language, video, thumbnail, download-dir flags"
```

---

## Task 3: Create Checkbox TUI Component

**Files:**
- Create: `internal/adapters/cli/tui/checkbox.go`

**Step 1: Create the checkbox component**

Create `internal/adapters/cli/tui/checkbox.go`:

```go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CheckboxOption represents a checkbox choice
type CheckboxOption struct {
	Label   string
	Value   string
	Checked bool
}

// CheckboxModel is the bubbletea model for checkbox selection
type CheckboxModel struct {
	title    string
	options  []CheckboxOption
	cursor   int
	done     bool
	minSelect int
}

// NewCheckboxModel creates a new checkbox selector
func NewCheckboxModel(title string, options []CheckboxOption) CheckboxModel {
	return CheckboxModel{
		title:     title,
		options:   options,
		minSelect: 1,
	}
}

func (m CheckboxModel) Init() tea.Cmd {
	return nil
}

func (m CheckboxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case " ", "x":
			m.options[m.cursor].Checked = !m.options[m.cursor].Checked
		case "enter":
			if m.countSelected() >= m.minSelect {
				m.done = true
				return m, tea.Quit
			}
		case "q", "ctrl+c", "esc":
			m.done = false
			for i := range m.options {
				m.options[i].Checked = false
			}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m CheckboxModel) countSelected() int {
	count := 0
	for _, opt := range m.options {
		if opt.Checked {
			count++
		}
	}
	return count
}

func (m CheckboxModel) View() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render(m.title))
	sb.WriteString("\n\n")

	for i, opt := range m.options {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		checkbox := "[ ]"
		style := uncheckedStyle
		if opt.Checked {
			checkbox = "[x]"
			style = checkedStyle
		}

		line := fmt.Sprintf("%s%s %s", cursor, checkbox, opt.Label)
		sb.WriteString(style.Render(line))
		sb.WriteString("\n")
	}

	selected := m.countSelected()
	hint := "\n"
	if selected < m.minSelect {
		hint = fmt.Sprintf("\n(select at least %d)\n", m.minSelect)
	}
	sb.WriteString(hint)
	sb.WriteString("(space=toggle, enter=confirm, q=cancel)\n")

	return sb.String()
}

// Selected returns the selected option values
func (m CheckboxModel) Selected() []string {
	var result []string
	for _, opt := range m.options {
		if opt.Checked {
			result = append(result, opt.Value)
		}
	}
	return result
}

// Cancelled returns true if the user cancelled
func (m CheckboxModel) Cancelled() bool {
	return !m.done
}

// RunCheckbox displays checkboxes and returns selected values
func RunCheckbox(title string, options []CheckboxOption) ([]string, error) {
	model := NewCheckboxModel(title, options)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	result := finalModel.(CheckboxModel)
	if result.Cancelled() {
		return nil, nil
	}
	return result.Selected(), nil
}
```

**Step 2: Build and verify**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/adapters/cli/tui/checkbox.go
git commit -m "feat(tui): add checkbox selection component"
```

---

## Task 4: Create Progress Display Component

**Files:**
- Create: `internal/adapters/cli/tui/progress.go`

**Step 1: Create the progress component**

Create `internal/adapters/cli/tui/progress.go`:

```go
package tui

import (
	"fmt"
	"strings"
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

	if index < len(p.steps) {
		p.currentStep = index
		p.steps[index].Status = StepRunning
		p.render()
	}
}

// CompleteStep marks a step as complete
func (p *ProgressDisplay) CompleteStep(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if index < len(p.steps) {
		p.steps[index].Status = StepComplete
		p.render()
	}
}

// FailStep marks a step as failed
func (p *ProgressDisplay) FailStep(index int, err string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if index < len(p.steps) {
		p.steps[index].Status = StepError
		p.steps[index].Error = err
		p.render()
	}
}

// UpdateProgress updates download progress for a step
func (p *ProgressDisplay) UpdateProgress(index int, current, total int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if index < len(p.steps) {
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
	if p.currentStep > 0 || p.steps[0].Status != StepPending {
		fmt.Print("\033[" + fmt.Sprintf("%d", len(p.steps)) + "A") // Move up
		fmt.Print("\033[J") // Clear from cursor to end
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
```

**Step 2: Build and verify**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/adapters/cli/tui/progress.go
git commit -m "feat(tui): add step-based progress display with spinners"
```

---

## Task 5: Add Thumbnail Download to Downloader Interface

**Files:**
- Modify: `internal/ports/downloader.go:16-43`

**Step 1: Add new methods to VideoDownloader interface**

In `internal/ports/downloader.go`, add to the interface:

```go
// VideoDownloader handles video download from Instagram
type VideoDownloader interface {
	// Download fetches a video by reel ID, returns path to downloaded file
	Download(ctx context.Context, reelID string, destDir string) (*DownloadResult, error)

	// DownloadVideo downloads the full video file (not just audio)
	DownloadVideo(ctx context.Context, reelID string, destPath string) error

	// DownloadThumbnail downloads the video thumbnail
	DownloadThumbnail(ctx context.Context, reelID string, destPath string) error

	// IsAvailable checks if the downloader is ready (yt-dlp installed)
	IsAvailable() bool

	// GetBinaryPath returns path to yt-dlp binary
	GetBinaryPath() string

	// Install downloads and installs yt-dlp
	Install(ctx context.Context, progress func(downloaded, total int64)) error

	// Update updates yt-dlp to latest version
	Update(ctx context.Context) error

	// IsFFmpegAvailable checks if ffmpeg is installed
	IsFFmpegAvailable() bool

	// GetFFmpegPath returns path to ffmpeg binary
	GetFFmpegPath() string

	// InstallFFmpeg downloads and installs ffmpeg (Windows only)
	InstallFFmpeg(ctx context.Context, progress func(downloaded, total int64)) error

	// FFmpegInstructions returns platform-specific install instructions
	FFmpegInstructions() string
}
```

**Step 2: Build (expect failure - interface not satisfied)**

Run: `go build ./...`
Expected: Compile error - Downloader doesn't implement VideoDownloader

**Step 3: Commit interface change**

```bash
git add internal/ports/downloader.go
git commit -m "feat(ports): add DownloadVideo and DownloadThumbnail to interface"
```

---

## Task 6: Implement Thumbnail and Video Download in yt-dlp Adapter

**Files:**
- Modify: `internal/adapters/ytdlp/downloader.go`

**Step 1: Add DownloadThumbnail method**

Add to `internal/adapters/ytdlp/downloader.go` (before the `var _ ports.VideoDownloader` line):

```go
func (d *Downloader) DownloadThumbnail(ctx context.Context, reelID string, destPath string) error {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return fmt.Errorf("yt-dlp not found")
	}

	url := buildReelURL(reelID)

	// Download thumbnail only
	args := []string{
		"--no-warnings",
		"--skip-download",
		"--write-thumbnail",
		"--convert-thumbnails", "jpg",
		"-o", strings.TrimSuffix(destPath, filepath.Ext(destPath)),
		url,
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			return fmt.Errorf("thumbnail download failed: %s", stderr)
		}
		return fmt.Errorf("thumbnail download failed: %w", err)
	}

	// yt-dlp adds extension, verify file exists
	expectedPath := strings.TrimSuffix(destPath, filepath.Ext(destPath)) + ".jpg"
	if _, err := os.Stat(expectedPath); err != nil {
		// Try webp as fallback
		webpPath := strings.TrimSuffix(destPath, filepath.Ext(destPath)) + ".webp"
		if _, err := os.Stat(webpPath); err == nil {
			return os.Rename(webpPath, destPath)
		}
		return fmt.Errorf("thumbnail file not found after download")
	}

	if expectedPath != destPath {
		return os.Rename(expectedPath, destPath)
	}
	return nil
}
```

**Step 2: Add DownloadVideo method**

Add to `internal/adapters/ytdlp/downloader.go`:

```go
func (d *Downloader) DownloadVideo(ctx context.Context, reelID string, destPath string) error {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return fmt.Errorf("yt-dlp not found")
	}

	url := buildReelURL(reelID)

	// Download best quality video
	args := []string{
		"--no-warnings",
		"-f", "best",
		"-o", destPath,
		url,
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "Private video") || strings.Contains(stderr, "Video unavailable") {
				return domain.ErrReelNotFound
			}
			return fmt.Errorf("video download failed: %s", stderr)
		}
		return fmt.Errorf("video download failed: %w", err)
	}

	return nil
}
```

**Step 3: Build and verify**

Run: `go build ./...`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/adapters/ytdlp/downloader.go
git commit -m "feat(ytdlp): implement DownloadVideo and DownloadThumbnail"
```

---

## Task 7: Update Application Layer TranscribeOptions

**Files:**
- Modify: `internal/application/transcribe.go`

**Step 1: Extend TranscribeOptions and TranscribeResult**

Update the structs in `internal/application/transcribe.go`:

```go
// TranscribeOptions configures the transcription
type TranscribeOptions struct {
	Model         string
	Format        string // text, srt, json
	NoCache       bool
	Language      string // empty defaults to "auto"
	SaveVideo     bool
	SaveThumbnail bool
	OutputDir     string // directory for outputs
}

// TranscribeResult contains the transcription result
type TranscribeResult struct {
	Reel          *domain.Reel
	Transcript    *domain.Transcript
	FromCache     bool
	VideoPath     string // populated if SaveVideo was true
	ThumbnailPath string // populated if SaveThumbnail was true
}
```

**Step 2: Build and verify**

Run: `go build ./...`
Expected: No errors (result fields are optional)

**Step 3: Commit**

```bash
git add internal/application/transcribe.go
git commit -m "feat(application): extend TranscribeOptions with download flags"
```

---

## Task 8: Update Mock Downloader for Tests

**Files:**
- Modify: `internal/application/transcribe_test.go`

**Step 1: Find and update mock**

Check if there's a mock downloader in tests. Add the new methods to satisfy the interface:

```go
func (m *mockDownloader) DownloadVideo(ctx context.Context, reelID string, destPath string) error {
	return nil
}

func (m *mockDownloader) DownloadThumbnail(ctx context.Context, reelID string, destPath string) error {
	return nil
}
```

**Step 2: Run tests**

Run: `go test ./internal/application/...`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/application/transcribe_test.go
git commit -m "test(application): add new interface methods to mock downloader"
```

---

## Task 9: Wire Checkbox into Interactive Menu

**Files:**
- Modify: `internal/adapters/cli/root.go:66-99`

**Step 1: Update runInteractiveMenu to use checkbox**

Replace the "transcribe" case in `runInteractiveMenu`:

```go
func runInteractiveMenu() error {
	options := []tui.MenuOption{
		{Label: "Transcribe a single reel", Value: "transcribe"},
		{Label: "Browse an account's reels", Value: "account"},
		{Label: "Manage cache", Value: "cache"},
		{Label: "Settings", Value: "settings"},
	}

	selected, err := tui.RunMenu(options)
	if err != nil {
		return err
	}

	switch selected {
	case "transcribe":
		return runTranscribeInteractive()
	case "account":
		fmt.Print("Enter username: ")
		var username string
		fmt.Scanln(&username)
		return runAccountInteractive(username)
	case "cache":
		return runCacheInteractive()
	case "settings":
		fmt.Println("Settings not yet implemented")
	case "":
		fmt.Println("Cancelled")
	}

	return nil
}

func runTranscribeInteractive() error {
	// Show output options
	checkboxOpts := []tui.CheckboxOption{
		{Label: "Transcript", Value: "transcript", Checked: true},
		{Label: "Download video", Value: "video", Checked: false},
		{Label: "Download thumbnail", Value: "thumbnail", Checked: false},
	}

	selected, err := tui.RunCheckbox("What would you like to get?", checkboxOpts)
	if err != nil {
		return err
	}
	if selected == nil {
		fmt.Println("Cancelled")
		return nil
	}

	// Parse selections
	wantTranscript := false
	wantVideo := false
	wantThumbnail := false
	for _, s := range selected {
		switch s {
		case "transcript":
			wantTranscript = true
		case "video":
			wantVideo = true
		case "thumbnail":
			wantThumbnail = true
		}
	}

	// Get reel URL
	fmt.Print("Enter reel URL or ID: ")
	var input string
	fmt.Scanln(&input)

	// Set flags based on selections
	videoFlag = wantVideo
	thumbnailFlag = wantThumbnail

	if wantTranscript {
		return runTranscribe(input)
	}

	// Download only (no transcription)
	return runDownloadOnly(input, wantVideo, wantThumbnail)
}

func runDownloadOnly(input string, video, thumbnail bool) error {
	app, err := GetApp()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	reel, err := domain.ParseReelInput(input)
	if err != nil {
		return err
	}

	ctx := context.Background()
	outputDir := downloadDirFlag
	if outputDir == "" {
		outputDir = "."
	}

	if video {
		destPath := filepath.Join(outputDir, reel.ID+".mp4")
		fmt.Printf("Downloading video to %s...\n", destPath)
		if err := app.Downloader.DownloadVideo(ctx, reel.ID, destPath); err != nil {
			return fmt.Errorf("video download failed: %w", err)
		}
		fmt.Println("✓ Video downloaded")
	}

	if thumbnail {
		destPath := filepath.Join(outputDir, reel.ID+".jpg")
		fmt.Printf("Downloading thumbnail to %s...\n", destPath)
		if err := app.Downloader.DownloadThumbnail(ctx, reel.ID, destPath); err != nil {
			return fmt.Errorf("thumbnail download failed: %w", err)
		}
		fmt.Println("✓ Thumbnail downloaded")
	}

	return nil
}
```

**Step 2: Add filepath import**

Add `"path/filepath"` to imports if not present.

**Step 3: Build and verify**

Run: `go build ./...`
Expected: No errors

**Step 4: Test interactively**

Run: `./ig2insights`
Expected: Menu appears, selecting "Transcribe a single reel" shows checkboxes

**Step 5: Commit**

```bash
git add internal/adapters/cli/root.go
git commit -m "feat(cli): add checkbox selection for interactive transcribe"
```

---

## Task 10: Integrate Progress Display into Transcription Flow

**Files:**
- Modify: `internal/adapters/cli/root.go:113-192`

**Step 1: Refactor runTranscribe to use progress display**

Replace `runTranscribe` function:

```go
func runTranscribe(input string) error {
	app, err := GetApp()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	reel, err := domain.ParseReelInput(input)
	if err != nil {
		return err
	}

	// Build step list based on what we're doing
	steps := []string{"Checking dependencies"}
	downloadStepIdx := 1
	extractStepIdx := 2
	transcribeStepIdx := 3
	videoStepIdx := -1
	thumbStepIdx := -1

	steps = append(steps, "Downloading video", "Extracting audio", "Transcribing")

	if videoFlag {
		videoStepIdx = len(steps)
		steps = append(steps, "Saving video")
	}
	if thumbnailFlag {
		thumbStepIdx = len(steps)
		steps = append(steps, "Downloading thumbnail")
	}

	progress := tui.NewProgressDisplay(steps, quietFlag)

	// Step 1: Check dependencies
	progress.StartStep(0)
	if !app.Downloader.IsAvailable() {
		if !quietFlag {
			fmt.Println("yt-dlp not found. Installing...")
		}
		if err := app.Downloader.Install(context.Background(), func(d, t int64) {
			progress.UpdateProgress(0, d, t)
		}); err != nil {
			progress.FailStep(0, err.Error())
			return fmt.Errorf("failed to install yt-dlp: %w", err)
		}
	}

	if !app.Transcriber.IsAvailable() {
		instructions := app.Transcriber.InstallationInstructions()
		if instructions != "" {
			progress.FailStep(0, "whisper.cpp not found")
			return errors.New(instructions)
		}
		if err := app.Transcriber.Install(context.Background(), func(d, t int64) {
			progress.UpdateProgress(0, d, t)
		}); err != nil {
			progress.FailStep(0, err.Error())
			return fmt.Errorf("failed to install whisper.cpp: %w", err)
		}
	}

	if !app.Downloader.IsFFmpegAvailable() {
		instructions := app.Downloader.FFmpegInstructions()
		if instructions != "" {
			progress.FailStep(0, "ffmpeg not found")
			return errors.New(instructions)
		}
		if err := app.Downloader.InstallFFmpeg(context.Background(), func(d, t int64) {
			progress.UpdateProgress(0, d, t)
		}); err != nil {
			progress.FailStep(0, err.Error())
			return fmt.Errorf("failed to install ffmpeg: %w", err)
		}
	}

	model := modelFlag
	if model == "" {
		model = app.Config.Defaults.Model
	}

	if !app.Transcriber.IsModelDownloaded(model) {
		if err := app.Transcriber.DownloadModel(context.Background(), model, func(d, t int64) {
			progress.UpdateProgress(0, d, t)
		}); err != nil {
			progress.FailStep(0, err.Error())
			return fmt.Errorf("failed to download model: %w", err)
		}
	}
	progress.CompleteStep(0)

	// Start spinner for indeterminate steps
	spinnerDone := progress.StartSpinner()
	defer close(spinnerDone)

	// Step 2-4: Transcribe (includes download + extract + transcribe internally)
	progress.StartStep(downloadStepIdx)

	ctx := context.Background()
	result, err := app.TranscribeSvc.Transcribe(ctx, reel.ID, application.TranscribeOptions{
		Model:    model,
		NoCache:  noCacheFlag,
		Language: languageFlag,
	})

	// Mark intermediate steps complete
	progress.CompleteStep(downloadStepIdx)
	progress.StartStep(extractStepIdx)
	progress.CompleteStep(extractStepIdx)
	progress.StartStep(transcribeStepIdx)

	if err != nil {
		progress.FailStep(transcribeStepIdx, err.Error())
		return err
	}
	progress.CompleteStep(transcribeStepIdx)

	// Determine output paths
	outputDir := downloadDirFlag
	if outputDir == "" {
		if outputFlag != "" {
			outputDir = filepath.Dir(outputFlag)
		} else {
			outputDir = "."
		}
	}

	outputs := make(map[string]string)

	// Step 5: Save video (if requested)
	if videoFlag && videoStepIdx >= 0 {
		progress.StartStep(videoStepIdx)
		videoPath := filepath.Join(outputDir, reel.ID+".mp4")
		if err := app.Downloader.DownloadVideo(ctx, reel.ID, videoPath); err != nil {
			progress.FailStep(videoStepIdx, err.Error())
			// Non-fatal, continue
		} else {
			progress.CompleteStep(videoStepIdx)
			outputs["Video"] = videoPath
		}
	}

	// Step 6: Download thumbnail (if requested)
	if thumbnailFlag && thumbStepIdx >= 0 {
		progress.StartStep(thumbStepIdx)
		thumbPath := filepath.Join(outputDir, reel.ID+".jpg")
		if err := app.Downloader.DownloadThumbnail(ctx, reel.ID, thumbPath); err != nil {
			progress.FailStep(thumbStepIdx, err.Error())
			// Non-fatal, continue
		} else {
			progress.CompleteStep(thumbStepIdx)
			outputs["Thumbnail"] = thumbPath
		}
	}

	// Stop spinner
	close(spinnerDone)
	spinnerDone = nil

	// Output transcript
	if err := outputResult(result); err != nil {
		return err
	}

	if outputFlag != "" {
		outputs["Transcript"] = outputFlag
	}

	if !quietFlag && len(outputs) > 0 {
		progress.Complete(outputs)
	}

	return nil
}
```

**Step 2: Build and verify**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/adapters/cli/root.go
git commit -m "feat(cli): integrate progress display into transcription flow"
```

---

## Task 11: Wire Language Flag Through CLI

**Files:**
- Modify: `internal/adapters/cli/root.go`

**Step 1: Verify language flag is passed**

The language flag should already be wired in Task 10. Verify by checking the TranscribeOptions in runTranscribe includes `Language: languageFlag`.

**Step 2: Build and test**

Run: `go build ./... && ./ig2insights --language fr <reel-url>`
Expected: Transcription uses French language detection

**Step 3: Commit (if any changes needed)**

```bash
git add internal/adapters/cli/root.go
git commit -m "feat(cli): wire language flag through transcription"
```

---

## Task 12: Final Integration Test

**Step 1: Test full flow with all options**

Run: `./ig2insights --video --thumbnail --language auto <real-reel-url>`
Expected:
- Progress display shows all steps with spinners
- Video downloaded
- Thumbnail downloaded
- Transcript output

**Step 2: Test interactive mode**

Run: `./ig2insights`
- Select "Transcribe a single reel"
- Check all three options
- Enter a reel URL
Expected: All outputs generated with progress display

**Step 3: Test download-only mode**

Run interactive, uncheck "Transcript", check "Download video"
Expected: Only video downloaded, no transcription

**Step 4: Run all tests**

Run: `go test ./...`
Expected: All tests pass

**Step 5: Final commit**

```bash
git add -A
git commit -m "feat: complete progress display and download options implementation"
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Fix language auto-detection | whisper/transcriber.go, application/transcribe.go |
| 2 | Add CLI flags | cli/root.go |
| 3 | Create checkbox TUI | tui/checkbox.go (new) |
| 4 | Create progress display | tui/progress.go (new) |
| 5 | Update downloader interface | ports/downloader.go |
| 6 | Implement download methods | ytdlp/downloader.go |
| 7 | Update TranscribeOptions | application/transcribe.go |
| 8 | Update mock for tests | application/transcribe_test.go |
| 9 | Wire checkbox to menu | cli/root.go |
| 10 | Integrate progress display | cli/root.go |
| 11 | Wire language flag | cli/root.go |
| 12 | Final integration test | - |
