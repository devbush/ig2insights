package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

// MenuOption represents a menu choice
type MenuOption struct {
	Label string
	Value string
}

// MenuModel is the bubbletea model for the main menu
type MenuModel struct {
	options  []MenuOption
	cursor   int
	selected string
}

// NewMenuModel creates a new menu
func NewMenuModel(options []MenuOption) MenuModel {
	return MenuModel{
		options: options,
	}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "enter":
			m.selected = m.options[m.cursor].Value
			return m, tea.Quit
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	s := "? What would you like to do?\n\n"

	for i, opt := range m.options {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}
		s += fmt.Sprintf("%s%s\n", cursor, style.Render(opt.Label))
	}

	s += "\n(up/down to navigate, enter to select, q to quit)\n"
	return s
}

// Selected returns the selected value
func (m MenuModel) Selected() string {
	return m.selected
}

// RunMenu displays the menu and returns the selection
func RunMenu(options []MenuOption) (string, error) {
	model := NewMenuModel(options)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	return finalModel.(MenuModel).Selected(), nil
}
