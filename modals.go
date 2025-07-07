package main

import (
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InputBoxModal is a reusable modal for text input (e.g. custom chat flow)
type InputBoxModal struct {
	Prompt   string
	Value    string
	Cursor   int
	Message  string
	Width    int
	Height   int
	Quitting bool
}

func (m InputBoxModal) View() string {
	prompt := lipgloss.NewStyle().Bold(true).Render(m.Prompt)
	input := m.Value
	if m.Cursor >= 0 && m.Cursor <= len(input) {
		input = input[:m.Cursor] + "|" + input[m.Cursor:]
	}
	inputBox := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(0, 1).Width(40).Render(input)
	msg := ""
	if m.Message != "" {
		msg = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(m.Message)
	}
	content := prompt + "\n" + inputBox + msg
	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, content)
}

// ConfirmationModal is a reusable yes/no modal for confirmations and notices
type ConfirmationModal struct {
	Title    string
	Prompt   string
	Selected int // 0 = Yes, 1 = No
	Width    int
	Height   int
}

func (m ConfirmationModal) View() string {
	boxWidth := 40
	prompt := m.Prompt
	options := []string{"Yes", "No"}
	var renderedOptions []string
	for i, opt := range options {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(false).Width(8).Align(lipgloss.Center)
		if i == m.Selected {
			style = style.Bold(true).Foreground(lipgloss.Color("203")).Background(lipgloss.Color("236"))
		}
		renderedOptions = append(renderedOptions, style.Render(opt))
	}
	optionsLine := lipgloss.JoinHorizontal(lipgloss.Center, renderedOptions...)
	content := lipgloss.NewStyle().Width(boxWidth).Align(lipgloss.Center).Render(prompt + "\n\n" + optionsLine)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("203")).
		Padding(1, 2).
		Width(boxWidth + 4).
		Align(lipgloss.Center).
		Render(content)
	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, box)
}

// InformationModal is a reusable modal for help/about/info screens
// No background, white text, blue headings, white box outlines
type InformationModal struct {
	Title    string
	Content  string
	Width    int
	Height   int
	Quitting bool
}

func (m InformationModal) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(m.Title)
	lines := strings.Split(parseText(m.Content), "\n")
	var renderedLines []string
	mono := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).SetString("").Render
	normal := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "|") || strings.Contains(trim, "---") {
			renderedLines = append(renderedLines, mono(line))
		} else {
			renderedLines = append(renderedLines, normal(line))
		}
	}
	content := strings.Join(renderedLines, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(m.Width - 10).
		Render(title + "\n" + content)
	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, box)
}

// Add tea.Model interface methods for InputBoxModal
func (m InputBoxModal) Init() tea.Cmd { return nil }

func (m InputBoxModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c", "ctrl+q", "esc":
			m.Quitting = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case "backspace":
			if m.Cursor > 0 && len(m.Value) > 0 {
				m.Value = m.Value[:m.Cursor-1] + m.Value[m.Cursor:]
				m.Cursor--
			}
		case "left":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "right":
			if m.Cursor < len(m.Value) {
				m.Cursor++
			}
		case "ctrl+v":
			paste, err := clipboard.ReadAll()
			if err == nil && paste != "" {
				m.Value = m.Value[:m.Cursor] + paste + m.Value[m.Cursor:]
				m.Cursor += len(paste)
			}
		default:
			if len(keyMsg.String()) == 1 && keyMsg.Type == tea.KeyRunes {
				m.Value = m.Value[:m.Cursor] + keyMsg.String() + m.Value[m.Cursor:]
				m.Cursor++
			}
		}
	}
	return m, nil
}

// ErrorModal is a reusable modal for displaying errors
type ErrorModal struct {
	Message  string
	Width    int
	Height   int
	Quitting bool
}

func (m ErrorModal) View() string {
	errorStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("15")). // white border
		Padding(1, 2).
		Foreground(lipgloss.Color("196")). // red text
		Width(m.Width - 10)
	box := errorStyle.Render(m.Message + "\n\nESC or Enter to close")
	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, box)
}

func (m ErrorModal) Init() tea.Cmd { return nil }

func (m ErrorModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c", "esc", "enter":
			m.Quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}
