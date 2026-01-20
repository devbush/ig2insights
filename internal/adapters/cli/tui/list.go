package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/devbush/ig2insights/internal/domain"
)

var (
	checkedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	uncheckedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	titleStyle     = lipgloss.NewStyle().Bold(true)
)

// ReelListModel is the bubbletea model for reel selection
type ReelListModel struct {
	reels    []*domain.Reel
	cursor   int
	selected map[int]bool
	done     bool
}

// NewReelListModel creates a new reel list
func NewReelListModel(reels []*domain.Reel) ReelListModel {
	return ReelListModel{
		reels:    reels,
		selected: make(map[int]bool),
	}
}

func (m ReelListModel) Init() tea.Cmd {
	return nil
}

func (m ReelListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.reels)-1 {
				m.cursor++
			}
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "enter":
			m.done = true
			return m, tea.Quit
		case "q", "ctrl+c":
			m.selected = make(map[int]bool) // Clear selection
			return m, tea.Quit
		case "a":
			// Select all
			for i := range m.reels {
				m.selected[i] = true
			}
		case "n":
			// Select none
			m.selected = make(map[int]bool)
		}
	}
	return m, nil
}

func (m ReelListModel) View() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Select reels to transcribe:"))
	sb.WriteString("\n\n")

	for i, reel := range m.reels {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		checkbox := "[ ]"
		style := uncheckedStyle
		if m.selected[i] {
			checkbox = "[x]"
			style = checkedStyle
		}

		title := reel.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}

		views := formatViews(reel.ViewCount)
		line := fmt.Sprintf("%s %s %-42s %8s views", cursor, checkbox, title, views)
		sb.WriteString(style.Render(line))
		sb.WriteString("\n")
	}

	count := len(m.selected)
	sb.WriteString(fmt.Sprintf("\n%d selected | space=toggle, a=all, n=none, enter=confirm, q=cancel\n", count))

	return sb.String()
}

// SelectedReels returns the selected reels
func (m ReelListModel) SelectedReels() []*domain.Reel {
	var result []*domain.Reel
	for i, selected := range m.selected {
		if selected && i < len(m.reels) {
			result = append(result, m.reels[i])
		}
	}
	return result
}

// RunReelList displays the list and returns selected reels
func RunReelList(reels []*domain.Reel) ([]*domain.Reel, error) {
	if len(reels) == 0 {
		return nil, nil
	}

	model := NewReelListModel(reels)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	return finalModel.(ReelListModel).SelectedReels(), nil
}

func formatViews(count int64) string {
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	}
	if count >= 1000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%d", count)
}
