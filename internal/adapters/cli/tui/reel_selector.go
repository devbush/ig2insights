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
	selectedCount := len(m.SelectedIDs())
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
