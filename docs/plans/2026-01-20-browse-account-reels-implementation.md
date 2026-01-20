# Browse Account Reels Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the "Browse an account's reels" feature with pagination, sorting, and multi-select processing.

**Architecture:** Extend the existing TUI components to support paginated reel selection with "Load more" and "Change sort" actions. Use existing BrowseService for fetching, TranscribeService for processing.

**Tech Stack:** Go, bubbletea TUI framework, existing BrowseService/TranscribeService

**Note:** Share count (repost_count) is not available from Instagram via yt-dlp. We'll display: views, likes, comments.

---

## Task 1: Extend Reel Domain Model

**Files:**
- Modify: `internal/domain/reel.go`

**Step 1: Add new fields to Reel struct**

In `internal/domain/reel.go`, add LikeCount, CommentCount, and UploadedAt fields:

```go
// Reel represents an Instagram Reel
type Reel struct {
	ID              string
	URL             string
	Author          string
	Title           string
	DurationSeconds int
	ViewCount       int64
	LikeCount       int64     // NEW
	CommentCount    int64     // NEW
	UploadedAt      time.Time // NEW - when the reel was posted
	FetchedAt       time.Time
}
```

**Step 2: Run tests to verify no breakage**

Run: `go test ./internal/domain/...`
Expected: PASS (existing tests should still pass)

**Step 3: Commit**

```bash
git add internal/domain/reel.go
git commit -m "feat(domain): add LikeCount, CommentCount, UploadedAt to Reel"
```

---

## Task 2: Update yt-dlp Adapter to Populate New Fields

**Files:**
- Modify: `internal/adapters/ytdlp/downloader.go:419-444`

**Step 1: Update the JSON struct in ListReels**

In `ListReels()` function around line 424, update the anonymous struct to capture more fields:

```go
var info struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Uploader     string  `json:"uploader"`
	Duration     float64 `json:"duration"`
	ViewCount    int64   `json:"view_count"`
	LikeCount    int64   `json:"like_count"`
	CommentCount int64   `json:"comment_count"`
	UploadDate   string  `json:"upload_date"` // YYYYMMDD format
	Timestamp    int64   `json:"timestamp"`   // Unix timestamp
}
```

**Step 2: Update the Reel creation to use new fields**

Update the reel creation (around line 436) to populate the new fields:

```go
var uploadedAt time.Time
if info.Timestamp > 0 {
	uploadedAt = time.Unix(info.Timestamp, 0)
} else if info.UploadDate != "" {
	// Parse YYYYMMDD format
	if t, err := time.Parse("20060102", info.UploadDate); err == nil {
		uploadedAt = t
	}
}

reels = append(reels, &domain.Reel{
	ID:              info.ID,
	Author:          info.Uploader,
	Title:           info.Title,
	DurationSeconds: int(info.Duration),
	ViewCount:       info.ViewCount,
	LikeCount:       info.LikeCount,
	CommentCount:    info.CommentCount,
	UploadedAt:      uploadedAt,
	FetchedAt:       time.Now(),
})
```

**Step 3: Run tests**

Run: `go test ./internal/adapters/ytdlp/...`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/adapters/ytdlp/downloader.go
git commit -m "feat(ytdlp): extract like count, comment count, upload date from reels"
```

---

## Task 3: Create Format Helper Functions

**Files:**
- Create: `internal/adapters/cli/tui/format.go`

**Step 1: Create the format.go file**

Create `internal/adapters/cli/tui/format.go`:

```go
package tui

import (
	"fmt"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
)

// FormatCount formats a number with K/M suffix
// Examples: 892 -> "892", 1234 -> "1.2K", 1500000 -> "1.5M"
func FormatCount(count int64) string {
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	}
	if count >= 1000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%d", count)
}

// FormatDate formats a date as "Jan 15" style
func FormatDate(t time.Time) string {
	if t.IsZero() {
		return "---"
	}
	return t.Format("Jan 2")
}

// FormatReelLine formats a reel as a single line for display
// Example: "Had an amazing day at..."  Jan 15  👁 12.3K  ❤️ 1.2K  💬 45
func FormatReelLine(reel *domain.Reel, maxCaptionLen int) string {
	caption := reel.Title
	if len(caption) > maxCaptionLen {
		caption = caption[:maxCaptionLen-3] + "..."
	}

	// Pad caption to fixed width
	captionFmt := fmt.Sprintf("%%-%ds", maxCaptionLen)
	paddedCaption := fmt.Sprintf(captionFmt, caption)

	date := FormatDate(reel.UploadedAt)
	views := FormatCount(reel.ViewCount)
	likes := FormatCount(reel.LikeCount)
	comments := FormatCount(reel.CommentCount)

	return fmt.Sprintf("%s  %s  👁 %6s  ❤️ %6s  💬 %5s",
		paddedCaption, date, views, likes, comments)
}
```

**Step 2: Write tests for format functions**

Create `internal/adapters/cli/tui/format_test.go`:

```go
package tui

import (
	"testing"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
)

func TestFormatCount(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{892, "892"},
		{999, "999"},
		{1000, "1.0K"},
		{1234, "1.2K"},
		{12345, "12.3K"},
		{999999, "1000.0K"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{12345678, "12.3M"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatCount(tt.input)
			if result != tt.expected {
				t.Errorf("FormatCount(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	tests := []struct {
		input    time.Time
		expected string
	}{
		{time.Time{}, "---"},
		{time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), "Jan 15"},
		{time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC), "Dec 25"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatDate(tt.input)
			if result != tt.expected {
				t.Errorf("FormatDate(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatReelLine(t *testing.T) {
	reel := &domain.Reel{
		Title:        "This is a test caption",
		UploadedAt:   time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		ViewCount:    12345,
		LikeCount:    1234,
		CommentCount: 45,
	}

	result := FormatReelLine(reel, 25)

	// Should contain formatted values
	if len(result) == 0 {
		t.Error("FormatReelLine returned empty string")
	}
	// Check it contains the emoji stats
	if !contains(result, "👁") || !contains(result, "❤️") || !contains(result, "💬") {
		t.Errorf("FormatReelLine missing stat emojis: %s", result)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
```

**Step 3: Run tests**

Run: `go test ./internal/adapters/cli/tui/...`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/adapters/cli/tui/format.go internal/adapters/cli/tui/format_test.go
git commit -m "feat(tui): add format helpers for counts, dates, reel lines"
```

---

## Task 4: Create Paginated Reel Selector Component

**Files:**
- Create: `internal/adapters/cli/tui/reel_selector.go`

**Step 1: Create the reel_selector.go file**

Create `internal/adapters/cli/tui/reel_selector.go`:

```go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/devbush/ig2insights/internal/domain"
)

// ReelSelectorAction represents user actions in the selector
type ReelSelectorAction string

const (
	ActionNone       ReelSelectorAction = ""
	ActionLoadMore   ReelSelectorAction = "load_more"
	ActionChangeSort ReelSelectorAction = "change_sort"
	ActionContinue   ReelSelectorAction = "continue"
	ActionCancel     ReelSelectorAction = "cancel"
)

// ReelSelectorModel is the bubbletea model for paginated reel selection
type ReelSelectorModel struct {
	reels       []*domain.Reel
	selected    map[string]bool // keyed by reel ID
	cursor      int
	currentSort domain.SortOrder
	hasMore     bool
	action      ReelSelectorAction

	// Menu items are after reels: Load more, Change sort, Continue
	menuStart int
}

// NewReelSelectorModel creates a new paginated reel selector
func NewReelSelectorModel(reels []*domain.Reel, currentSort domain.SortOrder, hasMore bool) ReelSelectorModel {
	return ReelSelectorModel{
		reels:       reels,
		selected:    make(map[string]bool),
		currentSort: currentSort,
		hasMore:     hasMore,
		menuStart:   len(reels),
	}
}

// AddReels appends more reels (for pagination)
func (m *ReelSelectorModel) AddReels(reels []*domain.Reel, hasMore bool) {
	m.reels = append(m.reels, reels...)
	m.hasMore = hasMore
	m.menuStart = len(m.reels)
}

// ClearAndSetReels replaces all reels (for sort change)
func (m *ReelSelectorModel) ClearAndSetReels(reels []*domain.Reel, sort domain.SortOrder, hasMore bool) {
	m.reels = reels
	m.selected = make(map[string]bool)
	m.currentSort = sort
	m.hasMore = hasMore
	m.cursor = 0
	m.menuStart = len(reels)
}

func (m ReelSelectorModel) Init() tea.Cmd {
	return nil
}

func (m ReelSelectorModel) menuItemCount() int {
	count := 2 // Change sort + Continue always visible
	if m.hasMore {
		count++ // Load more
	}
	return count
}

func (m ReelSelectorModel) totalItems() int {
	return len(m.reels) + m.menuItemCount()
}

func (m ReelSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < m.totalItems()-1 {
				m.cursor++
			}
		case " ", "x":
			// Toggle selection (only for reel items)
			if m.cursor < len(m.reels) {
				id := m.reels[m.cursor].ID
				m.selected[id] = !m.selected[id]
			}
		case "enter":
			if m.cursor >= m.menuStart {
				// Menu item selected
				menuIdx := m.cursor - m.menuStart
				if m.hasMore {
					switch menuIdx {
					case 0:
						m.action = ActionLoadMore
					case 1:
						m.action = ActionChangeSort
					case 2:
						m.action = ActionContinue
					}
				} else {
					switch menuIdx {
					case 0:
						m.action = ActionChangeSort
					case 1:
						m.action = ActionContinue
					}
				}
				return m, tea.Quit
			} else {
				// Reel item - toggle selection
				id := m.reels[m.cursor].ID
				m.selected[id] = !m.selected[id]
			}
		case "a":
			// Select all visible
			for _, reel := range m.reels {
				m.selected[reel.ID] = true
			}
		case "n":
			// Select none
			m.selected = make(map[string]bool)
		case "q", "ctrl+c", "esc":
			m.action = ActionCancel
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ReelSelectorModel) View() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Select reels:"))
	sb.WriteString("\n\n")

	// Reel items
	for i, reel := range m.reels {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		checkbox := "[ ]"
		style := uncheckedStyle
		if m.selected[reel.ID] {
			checkbox = "[x]"
			style = checkedStyle
		}

		line := fmt.Sprintf("%s%s %s", cursor, checkbox, FormatReelLine(reel, 30))
		sb.WriteString(style.Render(line))
		sb.WriteString("\n")
	}

	// Separator
	sb.WriteString("────────────────────────────────────────────────────────────────\n")

	// Menu items
	menuIdx := 0

	if m.hasMore {
		cursor := "  "
		if m.cursor == m.menuStart+menuIdx {
			cursor = "> "
		}
		sb.WriteString(fmt.Sprintf("%s[Load more]\n", cursor))
		menuIdx++
	}

	// Change sort
	cursor := "  "
	if m.cursor == m.menuStart+menuIdx {
		cursor = "> "
	}
	sortLabel := "Latest"
	if m.currentSort == domain.SortMostViewed {
		sortLabel = "Top"
	}
	sb.WriteString(fmt.Sprintf("%s[Change sort (%s)]\n", cursor, sortLabel))
	menuIdx++

	// Continue
	cursor = "  "
	if m.cursor == m.menuStart+menuIdx {
		cursor = "> "
	}
	selectedCount := len(m.selected)
	sb.WriteString(fmt.Sprintf("%s[Continue with %d selected]\n", cursor, selectedCount))

	sb.WriteString("\n(space=toggle, a=all, n=none, enter=select, q=cancel)\n")

	return sb.String()
}

// Action returns what action the user took
func (m ReelSelectorModel) Action() ReelSelectorAction {
	return m.action
}

// SelectedIDs returns the IDs of selected reels
func (m ReelSelectorModel) SelectedIDs() []string {
	var ids []string
	for id, sel := range m.selected {
		if sel {
			ids = append(ids, id)
		}
	}
	return ids
}

// SelectedReels returns the selected reel objects
func (m ReelSelectorModel) SelectedReels() []*domain.Reel {
	var result []*domain.Reel
	for _, reel := range m.reels {
		if m.selected[reel.ID] {
			result = append(result, reel)
		}
	}
	return result
}

// CurrentSort returns the current sort order
func (m ReelSelectorModel) CurrentSort() domain.SortOrder {
	return m.currentSort
}
```

**Step 2: Run build to verify syntax**

Run: `go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/adapters/cli/tui/reel_selector.go
git commit -m "feat(tui): add paginated reel selector component"
```

---

## Task 5: Create Output Options Selector

**Files:**
- Create: `internal/adapters/cli/tui/output_selector.go`

**Step 1: Create the output_selector.go file**

Create `internal/adapters/cli/tui/output_selector.go`:

```go
package tui

// OutputOptions represents what the user wants to download
type OutputOptions struct {
	Transcript bool
	Audio      bool
	Video      bool
	Thumbnail  bool
}

// RunOutputSelector displays output options and returns selections
func RunOutputSelector(reelCount int) (*OutputOptions, error) {
	title := "What to download for selected reels?"
	if reelCount == 1 {
		title = "What to download?"
	}

	options := []CheckboxOption{
		{Label: "Transcript (text)", Value: "transcript", Checked: true},
		{Label: "Audio (WAV)", Value: "audio", Checked: false},
		{Label: "Video (MP4)", Value: "video", Checked: false},
		{Label: "Thumbnail (JPG)", Value: "thumbnail", Checked: false},
	}

	selected, err := RunCheckbox(title, options)
	if err != nil {
		return nil, err
	}
	if selected == nil {
		return nil, nil // Cancelled
	}

	opts := &OutputOptions{}
	for _, v := range selected {
		switch v {
		case "transcript":
			opts.Transcript = true
		case "audio":
			opts.Audio = true
		case "video":
			opts.Video = true
		case "thumbnail":
			opts.Thumbnail = true
		}
	}

	return opts, nil
}
```

**Step 2: Run build to verify**

Run: `go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/adapters/cli/tui/output_selector.go
git commit -m "feat(tui): add output options selector component"
```

---

## Task 6: Implement runAccountInteractive Function

**Files:**
- Modify: `internal/adapters/cli/root.go:338-342`

**Step 1: Add necessary imports**

At the top of `internal/adapters/cli/root.go`, ensure these imports exist:

```go
import (
	// ... existing imports ...
	"github.com/devbush/ig2insights/internal/domain"
	tea "github.com/charmbracelet/bubbletea"
)
```

**Step 2: Implement the runAccountInteractive function**

Replace the stub function (around line 338) with the full implementation:

```go
func runAccountInteractive(username string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Step 1: Ask for sort order
	sortOptions := []tui.MenuOption{
		{Label: "Latest", Value: "latest"},
		{Label: "Top (most viewed)", Value: "top"},
	}
	sortChoice, err := tui.RunMenu(sortOptions)
	if err != nil {
		return err
	}
	if sortChoice == "" {
		return nil // Cancelled
	}

	currentSort := domain.SortLatest
	if sortChoice == "top" {
		currentSort = domain.SortMostViewed
	}

	// Step 2: Fetch initial reels
	fmt.Printf("Fetching reels from @%s...\n", username)
	const pageSize = 10
	reels, err := app.BrowseSvc.ListReels(ctx, username, currentSort, pageSize)
	if err != nil {
		return fmt.Errorf("failed to fetch reels: %w", err)
	}

	if len(reels) == 0 {
		fmt.Printf("No reels found for @%s\n", username)
		return nil
	}

	hasMore := len(reels) == pageSize

	// Step 3: Paginated selection loop
	model := tui.NewReelSelectorModel(reels, currentSort, hasMore)

	for {
		p := tea.NewProgram(model)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}
		model = finalModel.(tui.ReelSelectorModel)

		switch model.Action() {
		case tui.ActionCancel:
			return nil

		case tui.ActionLoadMore:
			fmt.Println("Loading more...")
			offset := len(model.SelectedIDs()) // This is wrong, need total loaded
			// Actually we need to track total loaded reels
			currentCount := len(reels)
			moreReels, err := app.BrowseSvc.ListReels(ctx, username, currentSort, pageSize+currentCount)
			if err != nil {
				fmt.Printf("Error loading more: %v\n", err)
				continue
			}
			// Get only the new ones
			if len(moreReels) > currentCount {
				newReels := moreReels[currentCount:]
				hasMore = len(moreReels) == pageSize+currentCount
				model.AddReels(newReels, hasMore)
				reels = moreReels
			} else {
				hasMore = false
				model.AddReels(nil, false)
			}

		case tui.ActionChangeSort:
			// Toggle sort
			if currentSort == domain.SortLatest {
				currentSort = domain.SortMostViewed
			} else {
				currentSort = domain.SortLatest
			}
			fmt.Printf("Fetching reels sorted by %s...\n", currentSort)
			reels, err = app.BrowseSvc.ListReels(ctx, username, currentSort, pageSize)
			if err != nil {
				return fmt.Errorf("failed to fetch reels: %w", err)
			}
			hasMore = len(reels) == pageSize
			model.ClearAndSetReels(reels, currentSort, hasMore)

		case tui.ActionContinue:
			selectedReels := model.SelectedReels()
			if len(selectedReels) == 0 {
				fmt.Println("No reels selected.")
				return nil
			}

			// Step 4: Get output options
			outputOpts, err := tui.RunOutputSelector(len(selectedReels))
			if err != nil {
				return err
			}
			if outputOpts == nil {
				return nil // Cancelled
			}

			// Step 5: Process selected reels
			return processSelectedReels(ctx, app, selectedReels, outputOpts)
		}
	}
}

func processSelectedReels(ctx context.Context, app *App, reels []*domain.Reel, opts *tui.OutputOptions) error {
	total := len(reels)
	var failed []string

	for i, reel := range reels {
		fmt.Printf("Processing %d/%d: %s...\n", i+1, total, reel.ID)

		transcribeOpts := application.TranscribeOptions{
			SaveAudio:     opts.Audio,
			SaveVideo:     opts.Video,
			SaveThumbnail: opts.Thumbnail,
		}

		result, err := app.TranscribeSvc.Transcribe(ctx, reel.ID, transcribeOpts)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", reel.ID, err))
			continue
		}

		// Copy outputs to current directory
		outputDir := "."
		baseName := reel.ID

		if opts.Transcript && result.Transcript != nil {
			outPath := filepath.Join(outputDir, baseName+".txt")
			if err := os.WriteFile(outPath, []byte(result.Transcript.Text), 0644); err != nil {
				failed = append(failed, fmt.Sprintf("%s (transcript): %v", reel.ID, err))
			}
		}

		if opts.Audio && result.AudioPath != "" {
			outPath := filepath.Join(outputDir, baseName+".wav")
			if err := copyFile(result.AudioPath, outPath); err != nil {
				failed = append(failed, fmt.Sprintf("%s (audio): %v", reel.ID, err))
			}
		}

		if opts.Video && result.VideoPath != "" {
			outPath := filepath.Join(outputDir, baseName+".mp4")
			if err := copyFile(result.VideoPath, outPath); err != nil {
				failed = append(failed, fmt.Sprintf("%s (video): %v", reel.ID, err))
			}
		}

		if opts.Thumbnail && result.ThumbnailPath != "" {
			outPath := filepath.Join(outputDir, baseName+".jpg")
			if err := copyFile(result.ThumbnailPath, outPath); err != nil {
				failed = append(failed, fmt.Sprintf("%s (thumbnail): %v", reel.ID, err))
			}
		}
	}

	// Summary
	succeeded := total - len(failed)
	fmt.Printf("\nCompleted %d/%d reels.\n", succeeded, total)
	if len(failed) > 0 {
		fmt.Println("Failed:")
		for _, f := range failed {
			fmt.Printf("  - %s\n", f)
		}
	}

	return nil
}
```

**Step 3: Ensure copyFile function exists**

Check if `copyFile` already exists in root.go. If not, add it:

```go
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
```

**Step 4: Add missing imports**

Ensure these imports are at the top of root.go:
```go
"context"
"path/filepath"
"github.com/devbush/ig2insights/internal/application"
```

**Step 5: Run build**

Run: `go build ./...`
Expected: Build succeeds

**Step 6: Commit**

```bash
git add internal/adapters/cli/root.go
git commit -m "feat(cli): implement browse account reels with pagination"
```

---

## Task 7: Fix Menu Component Title

**Files:**
- Modify: `internal/adapters/cli/tui/menu.go`

The menu currently hardcodes "What would you like to do?". We need to make it customizable for the sort selection menu.

**Step 1: Add title field to MenuModel**

In `internal/adapters/cli/tui/menu.go`, update the struct and constructor:

```go
// MenuModel is the bubbletea model for the main menu
type MenuModel struct {
	title    string
	options  []MenuOption
	cursor   int
	selected string
}

// NewMenuModel creates a new menu
func NewMenuModel(options []MenuOption) MenuModel {
	return MenuModel{
		title:   "What would you like to do?",
		options: options,
	}
}

// NewMenuModelWithTitle creates a new menu with custom title
func NewMenuModelWithTitle(title string, options []MenuOption) MenuModel {
	return MenuModel{
		title:   title,
		options: options,
	}
}
```

**Step 2: Update View to use title field**

Update the `View()` function:

```go
func (m MenuModel) View() string {
	s := fmt.Sprintf("? %s\n\n", m.title)
	// ... rest unchanged
}
```

**Step 3: Add RunMenuWithTitle function**

```go
// RunMenuWithTitle displays a menu with custom title
func RunMenuWithTitle(title string, options []MenuOption) (string, error) {
	model := NewMenuModelWithTitle(title, options)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	return finalModel.(MenuModel).Selected(), nil
}
```

**Step 4: Update runAccountInteractive to use new function**

In `root.go`, change the sort selection to:

```go
sortChoice, err := tui.RunMenuWithTitle("Sort by:", sortOptions)
```

**Step 5: Run build and tests**

Run: `go build ./... && go test ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/adapters/cli/tui/menu.go internal/adapters/cli/root.go
git commit -m "feat(tui): add customizable menu title"
```

---

## Task 8: Manual Testing and Bug Fixes

**Step 1: Build the application**

Run: `go build -o ig2insights ./cmd/ig2insights`
Expected: Build succeeds

**Step 2: Test the browse flow**

Run: `./ig2insights`

1. Select "Browse an account's reels"
2. Enter a username (e.g., "npcfaizan")
3. Verify sort selection appears
4. Select "Latest"
5. Verify reels list appears with stats (👁 ❤️ 💬)
6. Select a few reels with space
7. Test "Load more" - verify more reels appear
8. Test "Change sort" - verify list refreshes
9. Select "Continue with X selected"
10. Verify output options appear
11. Select options and verify processing works

**Step 3: Fix any bugs found**

Document and fix issues as they arise.

**Step 4: Run full test suite**

Run: `go test ./...`
Expected: All tests pass

**Step 5: Final commit**

```bash
git add -A
git commit -m "fix: address issues found in manual testing"
```

---

## Verification Checklist

- [ ] Build succeeds: `go build ./...`
- [ ] All tests pass: `go test ./...`
- [ ] Can browse account reels
- [ ] Sort selection works (Latest/Top)
- [ ] Reel list shows: caption, date, 👁 views, ❤️ likes, 💬 comments
- [ ] Multi-select works (space to toggle)
- [ ] "Load more" fetches additional reels
- [ ] "Change sort" refreshes with new sort order
- [ ] "Continue" shows output options
- [ ] Processing creates expected output files
- [ ] Progress shown during processing
- [ ] Summary shows success/failure count
