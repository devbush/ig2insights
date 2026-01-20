package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
