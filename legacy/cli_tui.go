package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TUIChatModel is a simplified Bubble Tea model for terminal chat
// No padding, minimal borders, colorized text

type TUIChatModel struct {
	chatName    string
	messages    []Message
	model       string
	inputBuffer string
	loading     bool
	status      string
	scrollPos   int
	selectedIdx int
	pendingUser string
	stopChan    chan bool
	width       int
	height      int
	quitting    bool
	showHelp    bool
	showError   bool
	errorMsg    string
}

func NewTUIChatModel(chatName string, messages []Message, model string) *TUIChatModel {
	return &TUIChatModel{
		chatName:    chatName,
		messages:    messages,
		model:       model,
		inputBuffer: "",
		loading:     false,
		status:      "",
		scrollPos:   0,
		selectedIdx: -1,
		pendingUser: "",
		stopChan:    nil,
		width:       80,
		height:      24,
		quitting:    false,
		showHelp:    false,
		showError:   false,
		errorMsg:    "",
	}
}

func (m *TUIChatModel) Init() tea.Cmd {
	return nil
}

func (m *TUIChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "up":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
			return m, nil
		case "down":
			if m.selectedIdx < len(m.getVisibleMessages())-1 {
				m.selectedIdx++
			}
			return m, nil
		case "pageup":
			m.scrollPos = max(0, m.scrollPos-10)
			return m, nil
		case "pagedown":
			m.scrollPos = min(len(m.getVisibleMessages())-1, m.scrollPos+10)
			return m, nil
		case "home":
			m.selectedIdx = 0
			return m, nil
		case "end":
			m.selectedIdx = len(m.getVisibleMessages()) - 1
			return m, nil
		case "enter":
			trimmed := strings.TrimSpace(m.inputBuffer)
			if trimmed == ":h" || trimmed == ":help" {
				m.showHelp = true
				m.inputBuffer = ""
				return m, nil
			}
			if trimmed == ":q" || trimmed == ":quit" {
				// Save chat before quitting
				_ = saveChat(m.chatName, m.messages)
				m.quitting = true
				return m, tea.Quit
			}
			if m.inputBuffer != "" && !m.loading {
				m.pendingUser = m.inputBuffer
				m.inputBuffer = ""
				m.loading = true
				return m, m.sendUserMessage()
			}
			return m, nil
		case "backspace":
			if len(m.inputBuffer) > 0 {
				m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
			}
			return m, nil
		case "h":
			// Only show help if explicitly sent as a message
			return m, nil
		default:
			if !m.loading && len(msg.String()) == 1 && msg.Type == tea.KeyRunes {
				m.inputBuffer += msg.String()
			}
			return m, nil
		}
	case aiResponseMsg:
		m.loading = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %v", msg.err)
			m.showError = true
			m.errorMsg = msg.err.Error()
		} else {
			if m.pendingUser != "" {
				m.messages = append(m.messages, Message{Role: "user", Content: m.pendingUser, MessageNumber: len(m.messages)})
				m.pendingUser = ""
			}
			m.messages = append(m.messages, Message{Role: "assistant", Content: msg.response, MessageNumber: len(m.messages)})
			m.status = ""
			m.selectedIdx = len(m.getVisibleMessages()) - 1 // select most recent
		}
		return m, nil
	}
	return m, nil
}

func (m *TUIChatModel) sendUserMessage() tea.Cmd {
	return getAIResponseCmd(append(m.messages, Message{Role: "user", Content: m.pendingUser}), m.model, m.stopChan)
}

func (m *TUIChatModel) getVisibleMessages() []Message {
	return filterNonSystemMessages(m.messages)
}

func (m *TUIChatModel) View() string {
	if m.quitting {
		return "Exiting chat..."
	}
	if m.showHelp {
		return "TUI Chat Help:\nUp/Down: Navigate  Enter: Send  :h/:help: Help  :q/:quit: Quit and save\n"
	}
	if m.showError {
		return fmt.Sprintf("Error: %s\nPress any key to continue...", m.errorMsg)
	}
	visible := m.getVisibleMessages()
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Render(fmt.Sprintf("Chat: %s | Model: %s\n", m.chatName, m.model)))
	for i, msg := range visible {
		var label, line string
		if msg.Role == "user" {
			labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			if i == m.selectedIdx {
				labelStyle = labelStyle.Underline(true)
			}
			label = labelStyle.Render("User:")
			line = label + "\n\t" + msg.Content
		} else {
			labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
			if i == m.selectedIdx {
				labelStyle = labelStyle.Underline(true)
			}
			label = labelStyle.Render("Assistant:")
			line = label + "\n\t" + msg.Content
		}
		b.WriteString(line + "\n\n")
	}
	if m.loading {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("[Waiting for response...]\n"))
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Input: "))
	b.WriteString(m.inputBuffer)
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(m.status))
	return b.String()
}

// Entry point for TUI chat menu
func RunTUIChatMenu() error {
	chats, err := listChats()
	if err != nil {
		return err
	}
	if len(chats) == 0 {
		fmt.Println("No chats available.")
		return nil
	}
	fmt.Println("Select a chat to load (TUI):")
	for i, c := range chats {
		fmt.Printf("%d) %s\n", i+1, c)
	}
	fmt.Print("Enter chat number: ")
	var idx int
	_, err = fmt.Scanf("%d", &idx)
	if err != nil || idx < 1 || idx > len(chats) {
		fmt.Println("Invalid selection.")
		return nil
	}
	chatName := chats[idx-1]
	chatFile, err := loadChatWithMetadata(chatName)
	if err != nil {
		return err
	}
	model := chatFile.Metadata.Model
	p := tea.NewProgram(NewTUIChatModel(chatName, chatFile.Messages, model), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
