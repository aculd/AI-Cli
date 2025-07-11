package main

import (
	"regexp"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InputBoxModal is a reusable modal for text input (e.g. custom chat flow)
type InputBoxModal struct {
	Prompt      string
	Value       string
	Cursor      int
	SelectStart int // -1 if no selection
	SelectEnd   int // -1 if no selection
	Message     string
	Width       int
	Height      int
	Quitting    bool
}

func (m InputBoxModal) View() string {
	input := m.Value
	// Render selection if present
	if m.SelectStart >= 0 && m.SelectEnd > m.SelectStart {
		start, end := m.SelectStart, m.SelectEnd
		if start > end {
			start, end = end, start
		}
		selected := lipgloss.NewStyle().Reverse(true).Render(input[start:end])
		input = input[:start] + selected + input[end:]
	}
	if m.Cursor >= 0 && m.Cursor <= len(input) {
		input = input[:m.Cursor] + "|" + input[m.Cursor:]
	}
	prompt := lipgloss.NewStyle().Bold(true).Render(m.Prompt)
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

// parseText parses markdown and special characters for display in the UI
func parseText(content string) string {
	// Basic markdown to plain text: remove **, __, *, _, ``, etc.
	// Preserve newlines and code blocks
	// (For more advanced markdown, use a library, but keep it simple here)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	// Remove bold/italic markers
	re := regexp.MustCompile(`([*_]{1,2}|` + "`" + `)`)
	content = re.ReplaceAllString(content, "")
	// Replace multiple newlines with a single newline
	reNL := regexp.MustCompile(`\n{3,}`)
	content = reNL.ReplaceAllString(content, "\n\n")
	return content
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
		shift := false
		ctrl := false
		key := keyMsg.String()
		if strings.HasPrefix(key, "shift+") {
			shift = true
			key = strings.TrimPrefix(key, "shift+")
		}
		if strings.HasPrefix(key, "ctrl+") {
			ctrl = true
			key = strings.TrimPrefix(key, "ctrl+")
		}
		switch key {
		case "x":
			if ctrl {
				// Cut selection or all
				if m.SelectStart >= 0 && m.SelectEnd > m.SelectStart {
					_ = clipboard.WriteAll(m.Value[m.SelectStart:m.SelectEnd])
					m.Value = m.Value[:m.SelectStart] + m.Value[m.SelectEnd:]
					m.Cursor = m.SelectStart
					m.SelectStart, m.SelectEnd = -1, -1
				} else {
					_ = clipboard.WriteAll(m.Value)
					m.Value = ""
					m.Cursor = 0
				}
				return m, nil
			}
		case "c":
			if ctrl {
				// Copy selection or all
				if m.SelectStart >= 0 && m.SelectEnd > m.SelectStart {
					_ = clipboard.WriteAll(m.Value[m.SelectStart:m.SelectEnd])
				} else {
					_ = clipboard.WriteAll(m.Value)
				}
				return m, nil
			}
		case "q", "esc":
			if ctrl || key == "esc" {
				m.Quitting = true
				return m, tea.Quit
			}
		case "v":
			if ctrl {
				paste, err := clipboard.ReadAll()
				if err == nil && paste != "" {
					if m.SelectStart >= 0 && m.SelectEnd > m.SelectStart {
						m.Value = m.Value[:m.SelectStart] + paste + m.Value[m.SelectEnd:]
						m.Cursor = m.SelectStart + len(paste)
						m.SelectStart, m.SelectEnd = -1, -1
					} else {
						m.Value = m.Value[:m.Cursor] + paste + m.Value[m.Cursor:]
						m.Cursor += len(paste)
					}
				}
				return m, nil
			}
		case "left":
			if shift && ctrl {
				// Ctrl+Shift+Left: select word left
				if m.SelectStart == -1 {
					m.SelectStart = m.Cursor
				}
				for m.Cursor > 0 && m.Value[m.Cursor-1] != ' ' {
					m.Cursor--
				}
				m.SelectEnd = m.Cursor
				return m, nil
			} else if shift {
				if m.SelectStart == -1 {
					m.SelectStart = m.Cursor
				}
				if m.Cursor > 0 {
					m.Cursor--
				}
				m.SelectEnd = m.Cursor
				return m, nil
			} else {
				if m.Cursor > 0 {
					m.Cursor--
				}
				m.SelectStart, m.SelectEnd = -1, -1
				return m, nil
			}
		case "right":
			if shift && ctrl {
				// Ctrl+Shift+Right: select word right
				if m.SelectStart == -1 {
					m.SelectStart = m.Cursor
				}
				for m.Cursor < len(m.Value) && m.Value[m.Cursor] != ' ' {
					m.Cursor++
				}
				m.SelectEnd = m.Cursor
				return m, nil
			} else if shift {
				if m.SelectStart == -1 {
					m.SelectStart = m.Cursor
				}
				if m.Cursor < len(m.Value) {
					m.Cursor++
				}
				m.SelectEnd = m.Cursor
				return m, nil
			} else {
				if m.Cursor < len(m.Value) {
					m.Cursor++
				}
				m.SelectStart, m.SelectEnd = -1, -1
				return m, nil
			}
		default:
			if len(keyMsg.String()) == 1 && keyMsg.Type == tea.KeyRunes {
				if m.SelectStart >= 0 && m.SelectEnd > m.SelectStart {
					m.Value = m.Value[:m.SelectStart] + keyMsg.String() + m.Value[m.SelectEnd:]
					m.Cursor = m.SelectStart + 1
					m.SelectStart, m.SelectEnd = -1, -1
				} else {
					m.Value = m.Value[:m.Cursor] + keyMsg.String() + m.Value[m.Cursor:]
					m.Cursor++
				}
				return m, nil
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
