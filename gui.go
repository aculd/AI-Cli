package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
)

// ChatGUI represents the GUI interface for the chat application
type ChatGUI struct {
	chatName      string
	messages      []Message
	model         string
	reader        *bufio.Reader
	inputBuffer   string
	initialized   bool
	interruptChan chan os.Signal
	width         int
	height        int
}

// NewChatGUI creates a new GUI instance
func NewChatGUI(chatName string, messages []Message, model string, reader *bufio.Reader) *ChatGUI {
	return &ChatGUI{
		chatName:      chatName,
		messages:      messages,
		model:         model,
		reader:        reader,
		inputBuffer:   "",
		initialized:   false,
		interruptChan: make(chan os.Signal, 1),
		width:         80,
		height:        24,
	}
}

// generateTitleWithAPI generates a title using the AI API
func (g *ChatGUI) generateTitleWithAPI() {
	if len(g.messages) == 0 {
		return
	}

	// Create a summary of the conversation for title generation
	var conversationSummary strings.Builder
	conversationSummary.WriteString("Conversation summary:\n")

	// Include last few messages for context
	startIdx := len(g.messages) - 3
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(g.messages); i++ {
		msg := g.messages[i]
		if msg.Role != "system" {
			conversationSummary.WriteString(fmt.Sprintf("%s: %s\n", strings.Title(msg.Role), msg.Content))
		}
	}

	// Create title generation prompt
	titlePrompt := fmt.Sprintf("%s\n\nDevise a short title for this chat, no longer than 5 words so that it can be easily picked and recognized from a list of chats. Return only the title, nothing else.", conversationSummary.String())

	// Create messages for title generation
	titleMessages := []Message{
		{Role: "system", Content: "You are a helpful assistant that generates concise, descriptive titles for chat conversations."},
		{Role: "user", Content: titlePrompt},
	}

	// Generate title in background
	go func() {
		title, err := streamChatResponse(titleMessages, g.model)
		if err != nil {
			return
		}

		// Clean up the title
		title = strings.TrimSpace(title)
		title = strings.ReplaceAll(title, "\"", "")
		title = strings.ReplaceAll(title, "'", "")

		// Limit to 5 words
		words := strings.Fields(title)
		if len(words) > 5 {
			words = words[:5]
		}
		cleanTitle := strings.Join(words, "-")

		// Rename the chat file
		oldPath := filepath.Join(chatsPath, g.chatName+".json")
		newPath := filepath.Join(chatsPath, cleanTitle+".json")

		if err := os.Rename(oldPath, newPath); err == nil {
			g.chatName = cleanTitle
		}
	}()
}

// handleVimCommand processes vim-like commands
func (g *ChatGUI) handleVimCommand(cmd string) bool {
	switch {
	case cmd == ":g":
		g.generateTitleWithAPI()
		return true
	case cmd == ":f":
		chatFile, err := loadChatWithMetadata(g.chatName)
		if err == nil {
			chatFile.Metadata.Favorite = !chatFile.Metadata.Favorite
			if err := saveChat(g.chatName, chatFile.Messages); err == nil {
				// Success
			}
		}
		return true
	case cmd == ":q":
		if err := saveChat(g.chatName, g.messages); err == nil {
			return false // Exit
		}
		return true
	case cmd == ":h":
		// Show help popup in ChatModel only
		return true
	case strings.HasPrefix(cmd, ":t "):
		title := strings.TrimSpace(strings.TrimPrefix(cmd, ":t "))
		if title != "" {
			_ = setChatTitle(g.chatName, title)
		}
		return true
	default:
		return false
	}
}

// --- ChatModel and async AI response refactor ---

type aiResponseMsg struct {
	response string
	err      error
}

type spinnerTickMsg struct{}

type stopRequestMsg struct{}

type blinkTickMsg struct{}

func getAIResponseCmd(messages []Message, model string, stopChan chan bool) tea.Cmd {
	return func() tea.Msg {
		reply, err := streamChatResponseGUI(messages, model, stopChan)
		return aiResponseMsg{reply, err}
	}
}

func spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

// streamChatResponseGUI is a version of streamChatResponse that doesn't print to stdout
func streamChatResponseGUI(messages []Message, model string, stopChan chan bool) (string, error) {
	reqBody := StreamRequestBody{
		Model:       model,
		Messages:    messages,
		Stream:      true,
		MaxTokens:   2048,
		Temperature: 0.7,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	apiKey, err := getActiveAPIKey()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/go-ai-cli")
	req.Header.Set("X-Title", "Go AI CLI")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var errorResp struct {
			Error ErrorResponse `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			return "", fmt.Errorf("API error %d: %s", errorResp.Error.Code, errorResp.Error.Message)
		}
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	var fullReply strings.Builder
	var buffer string

	for {
		// Check for stop request
		select {
		case <-stopChan:
			return "", fmt.Errorf("request cancelled by user")
		default:
			// Continue with normal processing
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fullReply.String(), err
		}

		line = strings.TrimSpace(line)

		// Handle server-sent events comments
		if strings.HasPrefix(line, ":") {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := line[len("data: "):]
		if data == "[DONE]" {
			break
		}

		// Append new chunk to buffer
		buffer += data

		// Process complete JSON objects from buffer
		for {
			openBrace := strings.Index(buffer, "{")
			if openBrace == -1 {
				break
			}

			// Find matching closing brace
			depth := 1
			closeBrace := -1
			for i := openBrace + 1; i < len(buffer); i++ {
				if buffer[i] == '{' {
					depth++
				} else if buffer[i] == '}' {
					depth--
					if depth == 0 {
						closeBrace = i
						break
					}
				}
			}

			if closeBrace == -1 {
				break
			}

			jsonStr := buffer[openBrace : closeBrace+1]
			buffer = buffer[closeBrace+1:]

			var streamResp StreamResponse
			if err := json.Unmarshal([]byte(jsonStr), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) > 0 {
				content := streamResp.Choices[0].Delta.Content
				if content != "" {
					fullReply.WriteString(content)
				}
			}
		}
	}

	result := fullReply.String()
	return result, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getSpinnerChar returns the spinner character for the given index
func getSpinnerChar(index int) string {
	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return spinnerChars[index%len(spinnerChars)]
}

// wrapText wraps text to the specified width
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var wrapped []string

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			wrapped = append(wrapped, "")
			continue
		}
		words := strings.Fields(line)
		if len(words) == 0 {
			wrapped = append(wrapped, "")
			continue
		}
		currentLine := words[0]
		for i := 1; i < len(words); i++ {
			word := words[i]
			if len(currentLine)+1+len(word) <= width {
				currentLine += " " + word
			} else {
				wrapped = append(wrapped, currentLine)
				currentLine = word
			}
		}
		if currentLine != "" {
			wrapped = append(wrapped, currentLine)
		}
	}
	return strings.Join(wrapped, "\n")
}

// ChatModel represents the Bubble Tea model for the chat interface
type ChatModel struct {
	chatName           string
	messages           []Message
	model              string
	inputBuffer        string
	width              int
	height             int
	status             string
	quitting           bool
	loading            bool
	spinner            int
	scrollPos          int       // Current scroll position (index of first visible message)
	autoScroll         bool      // Whether to auto-scroll to bottom
	stopChan           chan bool // Channel to signal stop request
	showConfirm        bool      // Whether to show exit confirmation
	generatingTitle    bool      // Whether we are generating a title
	showError          bool      // Whether to show an error popup
	errorMsg           string    // The error message to display
	cursorPos          int       // Current cursor position in the input buffer
	showHelp           bool      // Whether to show the help popup
	blinkOn            bool      // Whether the cursor is currently visible
	lastBlink          time.Time // Last time the cursor blinked
	confirmingExit     bool      // Whether we are currently showing the yes/no dialog
	confirmResult      *bool     // Pointer to store the result of confirmation
	pendingUserMessage string
	confirmModel       YesNoModel
	selectedMessageIdx int  // Index of selected message in visible window, -1 if none
	showGoodbye        bool // Show goodbye modal after chat is saved
	showMessageModal   bool // Show modal for viewing a message
	modalMessageIdx    int  // Index of message being viewed in modal (absolute index in visibleMessages)
	showExitGoodbye    bool // Show goodbye modal for ctrl+q exit
	exitAfterGoodbye   bool // Track if goodbye modal is for ctrl+q exit
}

func (m ChatModel) Init() tea.Cmd {
	return tea.Batch(
		nil,
		spinnerTick(),
	)
}

func (m ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showError {
			if msg.String() == "esc" {
				m.showError = false
			}
			return m, nil
		}
		if m.generatingTitle {
			if msg.String() == "esc" {
				m.generatingTitle = false
				m.status = "Title generation cancelled"
			}
			return m, nil
		}
		if m.showHelp {
			if msg.String() == "esc" {
				m.showHelp = false
			}
			return m, nil
		}
		if m.showConfirm {
			model, _ := m.confirmModel.Update(msg)
			cm := model.(YesNoModel)
			if msg.String() == "enter" && cm.selected == 0 {
				m.showConfirm = false
				m.showGoodbye = true
				return m, nil
			} else if msg.String() == "enter" && cm.selected == 1 {
				m.showConfirm = false
				return m, nil
			} else if msg.String() == "esc" {
				m.showConfirm = false
				return m, nil
			}
			m.confirmModel = cm
			return m, nil
		}
		// Typable key wakeup logic
		if m.inputBuffer == "" && len(msg.String()) == 1 && msg.Type == tea.KeyRunes && !m.loading {
			m.inputBuffer = msg.String()
			m.cursorPos = 1
			m.blinkOn = true
			return m, nil
		}
		if len(msg.String()) == 1 && msg.Type == tea.KeyRunes && !m.loading {
			m.inputBuffer = m.inputBuffer[:m.cursorPos] + msg.String() + m.inputBuffer[m.cursorPos:]
			m.cursorPos++
			m.blinkOn = true
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c":
			if m.selectedMessageIdx >= 0 && m.inputBuffer == "" && !m.loading {
				visibleMessages := m.getVisibleMessages()
				chatBoxHeight := m.height - 1 - 1 - 3 - 2
				if chatBoxHeight < 1 {
					chatBoxHeight = 1
				}
				startIdx := m.scrollPos
				endIdx := min(startIdx+chatBoxHeight, len(visibleMessages))
				idx := m.selectedMessageIdx
				if idx >= 0 && idx < endIdx-startIdx {
					msg := visibleMessages[startIdx+idx]
					safe := strings.ReplaceAll(msg.Content, "\x00", "")
					clipboard.WriteAll(safe)
					m.status = "Copied message to clipboard"
				}
			}
			return m, nil
		case "ctrl+s":
			if m.loading {
				if m.stopChan != nil {
					close(m.stopChan)
				}
				if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "user" {
					m.messages = m.messages[:len(m.messages)-1]
				}
				m.loading = false
				m.status = "Request cancelled"
				return m, nil
			}
		case "q":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			if m.loading {
				if m.stopChan != nil {
					close(m.stopChan)
				}
				// Restore pending user message to input
				if m.pendingUserMessage != "" {
					m.inputBuffer = m.pendingUserMessage
					m.cursorPos = len(m.inputBuffer)
					m.pendingUserMessage = ""
				}
				m.loading = false
				m.status = "Request cancelled"
				return m, nil
			}
			if m.showConfirm && !m.confirmingExit {
				// Start confirmation dialog
				m.confirmModel = YesNoModel{
					title:    "Confirm End Chat",
					prompt:   "Are you sure you want to quit?",
					selected: 1, // default No
				}
				m.showConfirm = true
				return m, nil
			} else {
				m.showConfirm = true
				return m, nil
			}
		case "left":
			if m.cursorPos > 0 {
				m.cursorPos--
			}
		case "right":
			if m.cursorPos < len(m.inputBuffer) {
				m.cursorPos++
			}
		case "home":
			if !m.loading && m.inputBuffer == "" {
				m.scrollPos = 0
				m.autoScroll = false
			} else {
				m.cursorPos = 0
			}
		case "end":
			if !m.loading && m.inputBuffer == "" {
				m.scrollPos = max(0, len(m.getVisibleMessages())-(m.height-6))
				m.autoScroll = true
			} else {
				m.cursorPos = len(m.inputBuffer)
			}
		case "backspace":
			if m.cursorPos > 0 && len(m.inputBuffer) > 0 {
				m.inputBuffer = m.inputBuffer[:m.cursorPos-1] + m.inputBuffer[m.cursorPos:]
				m.cursorPos--
			}
		case "delete":
			if m.cursorPos < len(m.inputBuffer) && len(m.inputBuffer) > 0 {
				m.inputBuffer = m.inputBuffer[:m.cursorPos] + m.inputBuffer[m.cursorPos+1:]
			}
		case "enter":
			if m.inputBuffer != "" && !m.loading {
				if strings.HasPrefix(m.inputBuffer, ":") {
					if m.handleVimCommand(m.inputBuffer) {
						if m.generatingTitle {
							m.inputBuffer = ""
							return m, spinnerTick()
						}
						m.inputBuffer = ""
						return m, nil
					}
				}
				m.pendingUserMessage = m.inputBuffer
				m.inputBuffer = ""
				m.loading = true
				m.status = "Waiting for AI response..."
				m.autoScroll = true
				m.stopChan = make(chan bool)
				// Do not append to m.messages yet; only after response
				return m, tea.Batch(getAIResponseCmd(append(m.messages, Message{Role: "user", Content: m.pendingUserMessage}), m.model, m.stopChan), spinnerTick())
			}
		case "pageup":
			if !m.loading {
				chatBoxHeight := m.height - 1 - 1 - 3 - 2
				if chatBoxHeight < 1 {
					chatBoxHeight = 1
				}
				m.scrollPos = max(0, m.scrollPos-chatBoxHeight)
				m.autoScroll = false
			}
		case "pagedown":
			if !m.loading {
				chatBoxHeight := m.height - 1 - 1 - 3 - 2
				if chatBoxHeight < 1 {
					chatBoxHeight = 1
				}
				maxScroll := max(0, len(m.getVisibleMessages())-chatBoxHeight)
				m.scrollPos = min(maxScroll, m.scrollPos+chatBoxHeight)
				m.autoScroll = false
			}
		case "up":
			if !m.loading && m.inputBuffer == "" {
				visibleMessages := m.getVisibleMessages()
				chatBoxHeight := m.height - 1 - 1 - 3 - 2
				if chatBoxHeight < 1 {
					chatBoxHeight = 1
				}
				startIdx := m.scrollPos
				endIdx := min(startIdx+chatBoxHeight, len(visibleMessages))
				if m.selectedMessageIdx == -1 && endIdx-startIdx > 0 {
					// Highlight the top message
					m.selectedMessageIdx = 0
				} else if m.selectedMessageIdx > 0 {
					m.selectedMessageIdx--
				} else if m.selectedMessageIdx == 0 && m.scrollPos > 0 {
					m.scrollPos--
					// keep selection at top
				}
				m.autoScroll = false
			}
		case "down":
			if !m.loading && m.inputBuffer == "" {
				visibleMessages := m.getVisibleMessages()
				chatBoxHeight := m.height - 1 - 1 - 3 - 2
				if chatBoxHeight < 1 {
					chatBoxHeight = 1
				}
				startIdx := m.scrollPos
				endIdx := min(startIdx+chatBoxHeight, len(visibleMessages))
				if m.selectedMessageIdx == -1 && endIdx-startIdx > 0 {
					// Highlight the bottom message
					m.selectedMessageIdx = endIdx - startIdx - 1
				} else if m.selectedMessageIdx < endIdx-startIdx-1 {
					m.selectedMessageIdx++
				} else if m.selectedMessageIdx == endIdx-startIdx-1 && m.scrollPos < len(visibleMessages)-chatBoxHeight {
					m.scrollPos++
					// keep selection at bottom
				}
				m.autoScroll = false
			}
		case "shift+up", "pgup":
			if !m.loading {
				m.scrollPos = max(0, m.scrollPos-1)
				m.autoScroll = false
			}
		case "shift+down", "pgdn":
			if !m.loading {
				maxScroll := max(0, len(m.getVisibleMessages())-1)
				m.scrollPos = min(maxScroll, m.scrollPos+1)
				m.autoScroll = false
			}
		case "ctrl+up":
			if !m.loading {
				m.scrollPos = max(0, m.scrollPos-1)
				m.autoScroll = false
			}
		case "ctrl+down":
			if !m.loading {
				maxScroll := max(0, len(m.getVisibleMessages())-(m.height-6))
				m.scrollPos = min(maxScroll, m.scrollPos+1)
				m.autoScroll = false
			}
		case "ctrl+q":
			m.showConfirm = true
			m.exitAfterGoodbye = true
			return m, nil
		case "ctrl+x":
			if m.inputBuffer != "" && m.cursorPos > 0 {
				safe := strings.ReplaceAll(m.inputBuffer, "\x00", "")
				clipboard.WriteAll(safe)
				m.inputBuffer = ""
				m.cursorPos = 0
			}
		case "ctrl+v":
			paste, err := clipboard.ReadAll()
			if err == nil {
				m.inputBuffer = m.inputBuffer[:m.cursorPos] + paste + m.inputBuffer[m.cursorPos:]
				m.cursorPos += len(paste)
			}
		case "ctrl+o":
			if m.selectedMessageIdx >= 0 {
				m.showMessageModal = true
				m.modalMessageIdx = m.selectedMessageIdx + m.scrollPos
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case spinnerTickMsg:
		if m.loading {
			m.spinner = (m.spinner + 1) % 4
			return m, spinnerTick()
		}
	case blinkTickMsg:
		m.blinkOn = !m.blinkOn
		m.lastBlink = time.Now()
		return m, nil
	case aiResponseMsg:
		m.loading = false
		if m.stopChan != nil {
			close(m.stopChan)
			m.stopChan = nil
		}
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %v", msg.err)
			m.pendingUserMessage = ""
		} else {
			if msg.response == "" {
				m.status = "Warning: Empty response received"
				m.pendingUserMessage = ""
			} else {
				if m.pendingUserMessage != "" {
					m.messages = append(m.messages, Message{Role: "user", Content: m.pendingUserMessage})
					m.pendingUserMessage = ""
				}
				m.messages = append(m.messages, Message{Role: "assistant", Content: msg.response})
				m.status = "Ready"
				if m.autoScroll {
					m.scrollPos = max(0, len(m.getVisibleMessages())-(m.height-6))
				}
				if err := saveChat(m.chatName, m.messages); err != nil {
					m.status = fmt.Sprintf("Save error: %v", err)
				}
			}
		}
	case aiTitleMsg:
		m.generatingTitle = false
		if err := setChatTitle(m.chatName, msg.title); err == nil {
			m.status = fmt.Sprintf("Title generated: %s", msg.title)
		} else {
			m.status = "Failed to set title"
		}
		return m, nil
	case tea.MouseMsg:
		if !m.loading && m.inputBuffer == "" {
			chatBoxHeight := m.height - 1 - 1 - 3 - 2 // header, status, input, spacing
			if chatBoxHeight < 1 {
				chatBoxHeight = 1
			}
			scrolled := false
			switch msg.Type {
			case tea.MouseWheelUp:
				m.scrollPos = max(0, m.scrollPos-chatBoxHeight)
				m.autoScroll = false
				scrolled = true
			case tea.MouseWheelDown:
				maxScroll := max(0, len(m.getVisibleMessages())-chatBoxHeight)
				m.scrollPos = min(maxScroll, m.scrollPos+chatBoxHeight)
				m.autoScroll = false
				scrolled = true
			}
			if scrolled {
				// Trigger a re-render
				tea.Println("")
			}
		}
		return m, nil
	}
	if m.inputBuffer != "" && m.selectedMessageIdx != -1 {
		m.selectedMessageIdx = -1
	}
	if m.showGoodbye {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "enter" {
				m.quitting = true
				return m, tea.Quit
			}
			// Ignore all other keys, including esc
			return m, nil
		}
		return m, nil
	}
	if m.showExitGoodbye {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "enter" {
				m.quitting = true
				return m, tea.Quit
			}
			// Ignore all other keys, including esc
			return m, nil
		}
		return m, nil
	}
	return m, nil
}

func (m ChatModel) View() string {
	if m.quitting {
		return "Chat saved. Goodbye!\n"
	}
	if m.showConfirm {
		box := m.confirmModel.View()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}

	// Styles
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	loadingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)

	// Box styles with borders
	chatBoxStyle := lipgloss.NewStyle().
		Padding(2, 8).
		Margin(1, 0)

	inputBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Background(lipgloss.Color("235")).
		Padding(1, 2).
		Margin(1, 0)

	// Calculate layout dimensions
	headerHeight := 1                                                         // Title line
	statusHeight := 1                                                         // Status line
	inputHeight := 3                                                          // Fixed 3-line input box
	chatBoxHeight := m.height - headerHeight - statusHeight - inputHeight - 2 // -2 for spacing

	if chatBoxHeight < 1 {
		chatBoxHeight = 1
	}

	// Header
	scrollIndicator := ""
	if len(m.getVisibleMessages()) > chatBoxHeight {
		scrollIndicator = fmt.Sprintf(" [Scroll: %d/%d]", m.scrollPos+1, len(m.getVisibleMessages()))
	}
	header := titleStyle.Render(fmt.Sprintf("Chat: %s | Model: %s | Messages: %d%s", m.chatName, m.model, len(m.messages), scrollIndicator))

	// Status
	statusText := m.status
	if m.loading {
		statusText = loadingStyle.Render(getSpinnerChar(m.spinner) + " " + m.status)
	}

	// Add comprehensive control hints
	controlHints := []string{}

	// Always show basic controls
	controlHints = append(controlHints, "Ctrl+S to stop", "Ctrl+C to quit")

	// Add scroll controls if there are many messages
	if len(m.getVisibleMessages()) > chatBoxHeight {
		controlHints = append(controlHints, "PageUp/Down, Home/End, Ctrl+↑↓, ↑↓ to scroll")
	}

	// Add vim commands hint
	controlHints = append(controlHints, ":g to generate title, :t \"title\" to set title, :f to favorite, :q to quit, :h for help")

	if len(controlHints) > 0 {
		statusText += " | " + strings.Join(controlHints, ", ")
	}
	status := statusStyle.Render(statusText)

	// Prepare chat history content with enhanced scrolling
	var visible []string
	visibleMessages := m.getVisibleMessages()

	// Apply scroll position with bounds checking
	startIdx := m.scrollPos
	if startIdx < 0 {
		startIdx = 0
		m.scrollPos = 0
	}

	maxScroll := max(0, len(visibleMessages)-chatBoxHeight)
	if startIdx > maxScroll {
		startIdx = maxScroll
		m.scrollPos = maxScroll
	}

	endIdx := min(startIdx+chatBoxHeight, len(visibleMessages))

	if startIdx < len(visibleMessages) {
		for i := startIdx; i < endIdx; i++ {
			msg := visibleMessages[i]
			var messageText string
			parsedContent := parseMarkdown(msg.Content)
			wrappedContent := wrapText(parsedContent, m.width-24) // narrower bubbles
			lines := strings.Split(wrappedContent, "\n")
			var bubbleStyle lipgloss.Style
			var label string
			isSelected := (m.selectedMessageIdx == i-startIdx)
			bubbleWidth := m.width / 2
			if msg.Role == "user" {
				bubbleStyle = lipgloss.NewStyle().
					Background(lipgloss.Color("238")).
					Foreground(lipgloss.Color("252")).
					Padding(1, 3).
					Margin(0, 0, 1, 0).
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("240"))
				labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
				if isSelected {
					labelStyle = labelStyle.Background(lipgloss.Color("238")).Underline(true)
				}
				label = fmt.Sprintf("#%d %s", msg.MessageNumber, "User")
				label = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, labelStyle.Render(label))
				bubble := bubbleStyle.Width(bubbleWidth).Render(strings.Join(lines, "\n"))
				messageText = label + "\n" + lipgloss.PlaceHorizontal(m.width, lipgloss.Center, bubble)
			} else {
				bubbleStyle = lipgloss.NewStyle().
					Background(lipgloss.Color("236")).
					Foreground(lipgloss.Color("252")).
					Padding(1, 3).
					Margin(0, 10, 1, 0).
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("240"))
				labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
				if isSelected {
					labelStyle = labelStyle.Background(lipgloss.Color("236")).Underline(true)
				}
				label = fmt.Sprintf("#%d %s", msg.MessageNumber, "Assistant")
				label = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, labelStyle.Render(label))
				bubble := bubbleStyle.Width(bubbleWidth).Render(strings.Join(lines, "\n"))
				messageText = label + "\n" + lipgloss.PlaceHorizontal(m.width, lipgloss.Center, bubble)
			}
			visible = append(visible, messageText)
		}
	}

	// Pad the top with empty lines if not enough messages to fill the chatbox
	if len(visible) < chatBoxHeight {
		padLines := make([]string, chatBoxHeight-len(visible))
		for i := range padLines {
			padLines[i] = ""
		}
		visible = append(padLines, visible...)
	}

	// Join messages with vertical spacing
	messageContent := strings.Join(visible, "\n\n")
	if messageContent == "" {
		messageContent = "No messages yet..."
	}
	chatBox := chatBoxStyle.Width(m.width - 1).Height(chatBoxHeight).Render(messageContent)

	// Input area - always 3 lines tall at bottom
	inputText := "Input: "
	if m.inputBuffer == "" {
		inputText = "*waiting for input...*\n(Esc to cancel sending, Enter :h for help)"
	} else {
		inputRunes := []rune(m.inputBuffer)
		cursor := "|"
		renderedInput := ""
		for i := 0; i <= len(inputRunes); i++ {
			if i == m.cursorPos && m.blinkOn {
				renderedInput += cursor
			}
			if i < len(inputRunes) {
				renderedInput += string(inputRunes[i])
			}
		}
		inputText += renderedInput
	}
	// Pad input text to fill 3 lines
	lines := strings.Split(inputText, "\n")
	for len(lines) < 3 {
		lines = append(lines, "")
	}
	inputText = strings.Join(lines[:3], "\n")

	// If loading, show spinner and 'Waiting for response...' in input box
	if m.loading {
		inputText = getSpinnerChar(m.spinner) + " Waiting for response...\n(Esc to cancel sending, Enter :h for help)\n"
	}

	// Create input box with border
	inputBox := inputBoxStyle.Width(m.width - 2).Height(3).Render(inputText)

	// Compose final layout
	layout := fmt.Sprintf("%s\n%s\n\n%s\n\n%s", header, status, chatBox, inputBox)

	if m.generatingTitle {
		popupStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(1, 2).
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252"))
		popupText := fmt.Sprintf("Generating title %s\n\nESC to cancel", getSpinnerChar(m.spinner))
		popupBox := popupStyle.Width(m.width - 10).Render(popupText)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, popupBox)
	}

	if m.showError {
		errorStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("203")).
			Padding(1, 2).
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252"))
		errorBox := errorStyle.Width(m.width - 10).Render(m.errorMsg + "\n\nESC to close")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, errorBox)
	}

	if m.showHelp {
		helpText := "Vim Commands:\n:g  Generate title\n:t \"title\"  Set title\n:f  Favorite\n:q  Quit\n:h  Help\n\nChat Scroll:\nShift+Up/Down, PgUp/PgDn  Scroll chat\nHome/End  Move cursor\nLeft/Right  Move cursor"
		helpBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(1, 2).
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Width(m.width - 10).
			Render(helpText + "\n\nESC to close")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, helpBox)
	}

	if m.showGoodbye {
		boxWidth := 40
		goodbyeText := "Chat Saved.\n\nPress enter to continue."
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("203")).
			Padding(1, 2).
			Width(boxWidth + 4).
			Align(lipgloss.Center).
			Render(lipgloss.NewStyle().Width(boxWidth).Align(lipgloss.Center).Render(goodbyeText))
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}

	if m.showMessageModal {
		modalContent := m.getVisibleMessages()[m.modalMessageIdx].Content
		modalBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("203")).
			Padding(1, 2).
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Width(m.width - 10).
			Render(modalContent + "\n\nESC to close")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalBox)
	}

	if m.showExitGoodbye {
		boxWidth := 40
		goodbyeText := "Goodbye.\n\nPress enter to exit."
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("203")).
			Padding(1, 2).
			Width(boxWidth + 4).
			Align(lipgloss.Center).
			Render(lipgloss.NewStyle().Width(boxWidth).Align(lipgloss.Center).Render(goodbyeText))
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}

	return lipgloss.NewStyle().Width(m.width).Height(m.height).Render(layout)
}

// Run starts the GUI interface using Bubble Tea
func (g *ChatGUI) Run() error {
	// Set up interrupt handling
	signal.Notify(g.interruptChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(g.interruptChan)

	// Set active chat for interrupt handling
	activeChatName = g.chatName
	defer func() {
		activeChatName = ""
	}()

	// Create the model
	model := ChatModel{
		chatName:    g.chatName,
		messages:    g.messages,
		model:       g.model,
		inputBuffer: "",
		width:       80,
		height:      24,
		status:      "Ready",
		quitting:    false,
	}
	// Set scroll position to bottom
	visibleMsgs := model.getVisibleMessages()
	chatBoxHeight := model.height - 1 - 1 - 3 - 2 // header, status, input, spacing
	if chatBoxHeight < 1 {
		chatBoxHeight = 1
	}
	model.scrollPos = max(0, len(visibleMsgs)-chatBoxHeight)
	model.autoScroll = true

	// Run the program
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run program: %w", err)
	}
	if chatModel, ok := finalModel.(ChatModel); ok && chatModel.quitting {
		if chatModel.showGoodbye || chatModel.showExitGoodbye {
			return ErrMenuBack
		}
	}
	return nil
}

// Refactored MenuModel without callback
// All menu functions now use the returned model from tea.NewProgram for selection
// ESC/back returns to the previous menu instead of quitting the app
// All menu navigation is robust and functional

type MenuModel struct {
	title    string
	options  []string
	selected int
	quitting bool
	width    int
	height   int
}

type apiKeyMenuModel struct {
	MenuModel
	keys      []APIKey
	activeKey string
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			m.selected = -1 // signal exit app
			return m, tea.Quit
		case "esc":
			m.quitting = true
			m.selected = -2 // signal back
			return m, tea.Quit
		case "q":
			m.quitting = true
			m.selected = -2 // treat 'q' as back
			return m, tea.Quit
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.options)-1 {
				m.selected++
			}
		case "enter":
			return m, tea.Quit
		case "ctrl+q":
			m.quitting = true
			m.selected = -1
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m MenuModel) View() string {
	if m.quitting {
		return ""
	}

	asciiArt := figure.NewFigure("AI CHAT", "", true).String()
	w := m.width
	h := m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	parentBoxWidth := int(float64(w) * 0.618)
	if parentBoxWidth < 30 {
		parentBoxWidth = 30
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")).Align(lipgloss.Center)
	customSelected := func(option string) string {
		borderWidth := parentBoxWidth - 4 // 2 for each side border
		if borderWidth < 10 {
			borderWidth = 10
		}
		content := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true).Width(borderWidth).Align(lipgloss.Center).Render(option)
		borderColor := lipgloss.Color("203")
		top := lipgloss.NewStyle().Foreground(borderColor).Render("╭" + strings.Repeat("─", borderWidth) + "╮")
		bottom := lipgloss.NewStyle().Foreground(borderColor).Render("╰" + strings.Repeat("─", borderWidth) + "╯")
		// No extra blank lines, just top, content, bottom
		box := top + "\n" + content + "\n" + bottom
		return lipgloss.PlaceHorizontal(parentBoxWidth, lipgloss.Center, box)
	}
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(false).Width(parentBoxWidth).Align(lipgloss.Center)
	menuBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Width(parentBoxWidth)

	var options []string
	for i, option := range m.options {
		var rendered string
		if i == m.selected {
			rendered = customSelected(option)
			options = append(options, rendered)
		} else {
			rendered = normalStyle.Render(option)
			options = append(options, rendered)
		}
	}
	// Vertically center the options block in the parent box
	optionsBlock := strings.Join(options, "\n")
	maxOptions := len(m.options) * 3 // Each highlighted option is 3 lines, normal is 1, but this is safe upper bound
	optionsBlock = lipgloss.PlaceVertical(maxOptions, lipgloss.Center, optionsBlock)

	menuBox := menuBoxStyle.Render(optionsBlock)
	title := titleStyle.Width(parentBoxWidth).Render(m.title)

	controlsText := "Controls: ↑↓ to navigate, Enter to select, Esc to go back, Ctrl+C to quit"
	controls := lipgloss.NewStyle().Align(lipgloss.Center).Render(controlsText)

	menuBlock := lipgloss.JoinVertical(lipgloss.Center, asciiArt, title, menuBox, controls)
	centeredMenu := lipgloss.NewStyle().Width(w).Height(h).Align(lipgloss.Center, lipgloss.Center).Render(menuBlock)
	return centeredMenu
}

// Add missing methods and functions for ChatModel
func (m ChatModel) getVisibleMessages() []Message {
	msgs := m.messages
	if len(msgs) == 0 {
		return nil
	}
	var visible []Message
	for _, msg := range msgs {
		if msg.Role != "system" {
			visible = append(visible, msg)
		}
	}
	return visible
}

func (m *ChatModel) handleVimCommand(cmd string) bool {
	switch {
	case cmd == ":g":
		m.generatingTitle = true
		return true
	case cmd == ":f":
		_ = toggleChatFavorite(m.chatName)
		return true
	case cmd == ":q":
		m.showGoodbye = true
		m.quitting = true
		return false // Exit
	case cmd == ":h":
		m.showHelp = true
		return true
	case strings.HasPrefix(cmd, ":t "):
		title := strings.TrimSpace(strings.TrimPrefix(cmd, ":t "))
		if title != "" {
			_ = setChatTitle(m.chatName, title)
			m.status = "Title set: " + title
		}
		return true
	default:
		return false
	}
}

// --- Submenu implementations ---

// Helper: Simple selection menu for a list of strings, returns index or -1 if cancelled
func selectFromList(title string, items []string) (int, error) {
	if len(items) == 0 {
		return -1, nil
	}
	model := MenuModel{
		title:    title,
		options:  items,
		selected: 0,
		quitting: false,
		width:    80,
		height:   24,
	}
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return -1, err
	}
	menuModel := finalModel.(MenuModel)
	if menuModel.quitting {
		return -1, nil
	}
	return menuModel.selected, nil
}

func GUIMenuChats() error {
	chatMenuOptions := []string{"List Chats", "Add New Chat", "Custom Chat", "Load Chat", "Back"}
	for {
		model := MenuModel{
			title:    "Chats",
			options:  chatMenuOptions,
			selected: 0,
			quitting: false,
			width:    80,
			height:   24,
		}
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return err
		}
		menuModel := finalModel.(MenuModel)
		if menuModel.selected == -1 {
			return ErrMenuBack
		}
		if menuModel.selected == -2 || menuModel.selected == 4 {
			return ErrMenuBack
		}
		switch menuModel.selected {
		case 0: // List Chats
			chats, err := listChats()
			if err != nil {
				return err
			}
			if len(chats) == 0 {
				return nil
			}
			// Add star to favorites
			for i, c := range chats {
				chatFile, err := loadChatWithMetadata(c)
				if err == nil && chatFile.Metadata.Favorite {
					chats[i] = "★ " + c
				}
			}
			idx, err := selectFromList("Select Chat to View/Continue", chats)
			if err != nil || idx < 0 || idx >= len(chats) {
				continue
			}
			// Remove star for loading
			chatName := strings.TrimPrefix(chats[idx], "★ ")
			chatFile, err := loadChatWithMetadata(chatName)
			if err != nil {
				return err
			}
			reader := bufio.NewReader(os.Stdin)
			gui := NewChatGUI(chatName, chatFile.Messages, chatFile.Metadata.Model, reader)
			if err := gui.Run(); err != nil {
				if err == ErrMenuBack {
					return ErrMenuBack
				}
				return err
			}
		case 1: // Add New Chat
			// Use InputBoxModal for entering chat title
			inputModal := InputBoxModal{
				Prompt: "Enter chat name (leave blank for timestamp):",
				Value:  "",
				Cursor: 0,
				Width:  80,
				Height: 24,
			}
			p := tea.NewProgram(inputModal, tea.WithAltScreen())
			finalModel, err := p.Run()
			if err != nil {
				return err
			}
			inputResult := finalModel.(InputBoxModal)
			if inputResult.Quitting {
				return ErrMenuBack
			}
			chatName := strings.TrimSpace(inputResult.Value)
			if chatName == "" {
				chatName = generateTimestampChatName()
			}
			// Check for duplicate
			chats, err := listChats()
			if err != nil {
				return err
			}
			for _, c := range chats {
				if c == chatName {
					return fmt.Errorf("chat '%s' already exists", chatName)
				}
			}

			// 2. Model selection
			models, _, err := loadModelsWithMostRecent()
			if err != nil || len(models) == 0 {
				return fmt.Errorf("no models available")
			}
			modelIdx, err := selectFromList("Select Model", models)
			if err != nil || modelIdx < 0 || modelIdx >= len(models) {
				return ErrMenuBack
			}
			model := models[modelIdx]

			// 3. Prompt selection
			prompts, err := loadPrompts()
			if err != nil || len(prompts) == 0 {
				// Try to initialize default prompts if missing
				prompts, err = initializeDefaultPrompts()
				if err != nil || len(prompts) == 0 {
					return fmt.Errorf("no prompts available")
				}
			}
			promptNames := make([]string, len(prompts))
			for i, p := range prompts {
				promptNames[i] = p.Name
			}
			promptIdx, err := selectFromList("Select Prompt", promptNames)
			if err != nil || promptIdx < 0 || promptIdx >= len(prompts) {
				return ErrMenuBack
			}
			promptContent := prompts[promptIdx].Content

			// 4. Create chat and launch
			messages := []Message{{Role: "system", Content: promptContent}}
			chatFile := ChatFile{
				Messages: messages,
				Metadata: ChatMetadata{
					Model:     model,
					CreatedAt: time.Now(),
				},
			}
			data, err := json.MarshalIndent(chatFile, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal chat: %w", err)
			}
			err = os.WriteFile(filepath.Join(chatsPath, chatName+".json"), data, 0644)
			if err != nil {
				return fmt.Errorf("failed to write chat file '%s': %w", chatName, err)
			}
			gui := NewChatGUI(chatName, messages, model, bufio.NewReader(os.Stdin))
			return gui.Run()
		case 2: // Custom Chat
			if err := GUICustomChatFlow(); err != nil {
				return err
			}
		case 3: // Load Chat
			chats, err := listChats()
			if err != nil {
				return err
			}
			if len(chats) == 0 {
				return nil
			}
			// Add star to favorites
			for i, c := range chats {
				chatFile, err := loadChatWithMetadata(c)
				if err == nil && chatFile.Metadata.Favorite {
					chats[i] = "★ " + c
				}
			}
			idx, err := selectFromList("Select Chat to Load", chats)
			if err != nil || idx < 0 || idx >= len(chats) {
				continue
			}
			chatName := strings.TrimPrefix(chats[idx], "★ ")
			chatFile, err := loadChatWithMetadata(chatName)
			if err != nil {
				return err
			}
			reader := bufio.NewReader(os.Stdin)
			gui := NewChatGUI(chatName, chatFile.Messages, chatFile.Metadata.Model, reader)
			if err := gui.Run(); err != nil {
				if err == ErrMenuBack {
					return ErrMenuBack
				}
				return err
			}
		case 4: // Back
			return ErrMenuBack
		}
	}
}

func GUIMenuFavorites() error {
	favMenuOptions := []string{"List Favorites", "Add Favorite", "Remove Favorite", "Back"}
	for {
		model := MenuModel{
			title:    "Favorites",
			options:  favMenuOptions,
			selected: 0,
			quitting: false,
			width:    80,
			height:   24,
		}
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return err
		}
		menuModel := finalModel.(MenuModel)
		if menuModel.selected == -1 {
			return ErrMenuBack
		}
		if menuModel.selected == -2 || menuModel.selected == 3 {
			return ErrMenuBack
		}
		switch menuModel.selected {
		case 0: // List Favorites
			chats, err := listChats()
			if err != nil {
				return err
			}
			var favorites []string
			for _, c := range chats {
				chatFile, err := loadChatWithMetadata(c)
				if err == nil && chatFile.Metadata.Favorite {
					favorites = append(favorites, "★ "+c)
				}
			}
			if len(favorites) == 0 {
				continue
			}
			idx, err := selectFromList("Select Favorite Chat to Load", favorites)
			if err != nil || idx < 0 || idx >= len(favorites) {
				continue
			}
			chatName := strings.TrimPrefix(favorites[idx], "★ ")
			chatFile, err := loadChatWithMetadata(chatName)
			if err != nil {
				return err
			}
			reader := bufio.NewReader(os.Stdin)
			gui := NewChatGUI(chatName, chatFile.Messages, chatFile.Metadata.Model, reader)
			if err := gui.Run(); err != nil {
				if err == ErrMenuBack {
					return ErrMenuBack
				}
				return err
			}
		case 1: // Add Favorite
			chats, err := listChats()
			if err != nil {
				return err
			}
			idx, err := selectFromList("Select Chat to Mark as Favorite", chats)
			if err != nil || idx < 0 || idx >= len(chats) {
				continue
			}
			if err := toggleChatFavorite(chats[idx]); err != nil {
				return err
			}
		case 2: // Remove Favorite
			chats, err := listChats()
			if err != nil {
				return err
			}
			var favorites []string
			for _, c := range chats {
				chatFile, err := loadChatWithMetadata(c)
				if err == nil && chatFile.Metadata.Favorite {
					favorites = append(favorites, "★ "+c)
				}
			}
			if len(favorites) == 0 {
				continue
			}
			idx, err := selectFromList("Select Favorite to Unmark", favorites)
			if err != nil || idx < 0 || idx >= len(favorites) {
				continue
			}
			chatName := strings.TrimPrefix(favorites[idx], "★ ")
			if err := toggleChatFavorite(chatName); err != nil {
				return err
			}
		case 3: // Back
			return ErrMenuBack
		}
	}
}

func GUIMenuPrompts() error {
	promptMenuOptions := []string{"List Prompts", "Add Prompt", "Remove Prompt", "Set Default Prompt", "Back"}
	for {
		model := MenuModel{
			title:    "Prompts",
			options:  promptMenuOptions,
			selected: 0,
			quitting: false,
			width:    80,
			height:   24,
		}
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return err
		}
		menuModel := finalModel.(MenuModel)
		if menuModel.selected == -1 {
			return ErrMenuBack
		}
		if menuModel.selected == -2 || menuModel.selected == 4 {
			return ErrMenuBack
		}
		switch menuModel.selected {
		case 0: // List Prompts
			prompts, err := loadPrompts()
			if err != nil {
				return err
			}
			var names []string
			for _, p := range prompts {
				name := p.Name
				if p.Default {
					name += " (default)"
				}
				names = append(names, name)
			}
			_, _ = selectFromList("Prompts", names)
		case 1: // Add Prompt
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Prompt name: ")
			name, _ := reader.ReadString('\n')
			name = strings.TrimSpace(name)
			fmt.Print("Prompt content: ")
			content, _ := reader.ReadString('\n')
			content = strings.TrimSpace(content)
			prompts, err := loadPrompts()
			if err != nil {
				return err
			}
			prompts = append(prompts, Prompt{Name: name, Content: content})
			if err := savePrompts(prompts); err != nil {
				return err
			}
		case 2: // Remove Prompt
			prompts, err := loadPrompts()
			if err != nil {
				return err
			}
			var names []string
			for _, p := range prompts {
				names = append(names, p.Name)
			}
			idx, err := selectFromList("Remove Prompt", names)
			if err != nil || idx < 0 || idx >= len(prompts) {
				continue
			}
			prompts = append(prompts[:idx], prompts[idx+1:]...)
			if err := savePrompts(prompts); err != nil {
				return err
			}
		case 3: // Set Default Prompt
			prompts, err := loadPrompts()
			if err != nil {
				return err
			}
			var names []string
			for _, p := range prompts {
				names = append(names, p.Name)
			}
			idx, err := selectFromList("Set Default Prompt", names)
			if err != nil || idx < 0 || idx >= len(prompts) {
				continue
			}
			for i := range prompts {
				prompts[i].Default = (i == idx)
			}
			if err := savePrompts(prompts); err != nil {
				return err
			}
			fmt.Println("Default prompt set to:", prompts[idx].Name)
		case 4: // Back
			return ErrMenuBack
		}
	}
}

func GUIMenuModels() error {
	modelMenuOptions := []string{"List Models", "Add Model", "Remove Model", "Set Default Model", "Back"}
	for {
		model := MenuModel{
			title:    "Models",
			options:  modelMenuOptions,
			selected: 0,
			quitting: false,
			width:    80,
			height:   24,
		}
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return err
		}
		menuModel := finalModel.(MenuModel)
		if menuModel.selected == -1 {
			return ErrMenuBack
		}
		if menuModel.selected == -2 || menuModel.selected == 4 {
			return ErrMenuBack
		}
		switch menuModel.selected {
		case 0: // List Models
			models, defaultModel, err := loadModelsWithMostRecent()
			if err != nil {
				return err
			}
			var names []string
			for _, m := range models {
				name := m
				if m == defaultModel {
					name += " (default)"
				}
				names = append(names, name)
			}
			_, _ = selectFromList("Models", names)
		case 1: // Add Model
			// Prompt for model name (text input)
			inputModel := TextInputModel{prompt: "Enter model name (leave blank to paste from clipboard):", value: "", cursor: 0, width: 40}
			p := tea.NewProgram(inputModel, tea.WithAltScreen())
			finalInput, err := p.Run()
			if err != nil {
				return err
			}
			input := finalInput.(TextInputModel)
			if input.quitting {
				continue
			}
			modelName := strings.TrimSpace(input.value)
			if modelName == "" {
				clipText, err := clipboard.ReadAll()
				if err != nil {
					fmt.Println("Failed to read from clipboard:", err)
					continue
				}
				modelName = strings.TrimSpace(clipText)
			}
			if modelName == "" {
				fmt.Println("Model name cannot be empty.")
				continue
			}
			// Load models config
			path := filepath.Join(utilPath, "models.json")
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var config struct {
				Models []struct {
					Name      string `json:"name"`
					IsDefault bool   `json:"is_default"`
				} `json:"models"`
			}
			if err := json.Unmarshal(data, &config); err != nil {
				return err
			}
			// Check for duplicate
			duplicate := false
			for _, m := range config.Models {
				if m.Name == modelName {
					duplicate = true
					break
				}
			}
			if duplicate {
				fmt.Println("Model already exists.")
				continue
			}
			config.Models = append(config.Models, struct {
				Name      string `json:"name"`
				IsDefault bool   `json:"is_default"`
			}{Name: modelName, IsDefault: false})
			newData, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				return err
			}
			if err := os.WriteFile(path, newData, 0644); err != nil {
				return err
			}
			fmt.Println("Model added:", modelName)
		case 2: // Remove Model
			models, _, err := loadModelsWithMostRecent()
			if err != nil {
				return err
			}
			idx, err := selectFromList("Remove Model", models)
			if err != nil || idx < 0 || idx >= len(models) {
				continue
			}
			// TODO: Actually remove model from config and save
			fmt.Println("Model removed (not yet implemented)")
		case 3: // Set Default Model
			models, _, err := loadModelsWithMostRecent()
			if err != nil {
				return err
			}
			idx, err := selectFromList("Set Default Model", models)
			if err != nil || idx < 0 || idx >= len(models) {
				continue
			}
			// Save default model to config (implement saveDefaultModel)
			if err := saveDefaultModel(models[idx]); err != nil {
				return err
			}
			fmt.Println("Default model set to:", models[idx])
		case 4: // Back
			return ErrMenuBack
		}
	}
}

func GUIMenuAPIKey() error {
	apiKeyMenuOptions := []string{"List API Keys", "Add API Key", "Remove API Key", "Set Active API Key", "Back"}
	for {
		model := MenuModel{
			title:    "API Key",
			options:  apiKeyMenuOptions,
			selected: 0,
			quitting: false,
			width:    80,
			height:   24,
		}
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return err
		}
		menuModel := finalModel.(MenuModel)
		switch menuModel.selected {
		case 0: // List API Keys
			config, err := loadAPIKeys()
			if err != nil {
				return err
			}
			var names []string
			for _, k := range config.Keys {
				name := k.Title
				if k.Title == config.ActiveKey {
					name += " (active)"
				}
				names = append(names, name)
			}
			_, _ = selectFromList("API Keys", names)
		case 1: // Add API Key
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("API Key title: ")
			title, _ := reader.ReadString('\n')
			title = strings.TrimSpace(title)
			fmt.Print("API Key: ")
			key, _ := reader.ReadString('\n')
			key = strings.TrimSpace(key)
			if err := addAPIKey(title, key); err != nil {
				return err
			}
		case 2: // Remove API Key
			config, err := loadAPIKeys()
			if err != nil {
				return err
			}
			var names []string
			for _, k := range config.Keys {
				names = append(names, k.Title)
			}
			idx, err := selectFromList("Remove API Key", names)
			if err != nil || idx < 0 || idx >= len(config.Keys) {
				continue
			}
			// Remove key
			config.Keys = append(config.Keys[:idx], config.Keys[idx+1:]...)
			if err := saveAPIKeys(config); err != nil {
				return err
			}
		case 3: // Set Active API Key
			config, err := loadAPIKeys()
			if err != nil {
				return err
			}
			var names []string
			for _, k := range config.Keys {
				names = append(names, k.Title)
			}
			idx, err := selectFromList("Set Active API Key", names)
			if err != nil || idx < 0 || idx >= len(config.Keys) {
				continue
			}
			config.ActiveKey = config.Keys[idx].Title
			if err := saveAPIKeys(config); err != nil {
				return err
			}
		case 4: // Back
			return ErrMenuBack
		}
	}
}

func GUIShowHelp() error {
	helpMenuOptions := []string{"Show Controls", "Show About", "Back"}
	for {
		model := MenuModel{
			title:    "Help",
			options:  helpMenuOptions,
			selected: 0,
			quitting: false,
			width:    80,
			height:   24,
		}
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return err
		}
		menuModel := finalModel.(MenuModel)
		switch menuModel.selected {
		case 0: // Show Controls
			fmt.Println("\nControls Cheat Sheet:\n")
			fmt.Println("| Menu/General         | Chat Window           | Vim-style      |\n|---------------------|----------------------|---------------|")
			fmt.Println("| ↑↓      Navigate    | Enter   Send message | :g  AI Title  |\n| Enter   Select      | Ctrl+S Stop request  | :t \"Title\"    |\n| Esc     Back        | Ctrl+C Quit          | :f  Favorite  |\n| Ctrl+C  Quit        | ↑↓      Scroll msgs  | :q  Quit      |\n|                     | PgUp/Dn Scroll page  |               |\n|                     | Home/End Top/Bottom  |               |\n|                     |                      |               |\n")
			fmt.Println("Press Enter to continue...")
			bufio.NewReader(os.Stdin).ReadBytes('\n')
		case 1: // Show About
			fmt.Println("\nGo AI CLI - Terminal AI Chat\nBuilt with Go, Bubble Tea, Lipgloss\nhttps://github.com/aculd/go-ai-cli\n")
			fmt.Println("Press Enter to continue...")
			bufio.NewReader(os.Stdin).ReadBytes('\n')
		case 2: // Back
			return ErrMenuBack
		}
	}
}

// Minimal stubs for missing types and functions to fix build errors
// YesNoModel is a placeholder for confirmation dialogs
// aiTitleMsg is a placeholder for AI title messages
// parseMarkdown is a placeholder for markdown parsing

type YesNoModel struct {
	title    string
	prompt   string
	selected int
	result   bool
}

type aiTitleMsg struct {
	title string
}

func parseMarkdown(content string) string {
	return content
}

// YesNoModel minimal tea.Model implementation
func (m YesNoModel) Init() tea.Cmd { return nil }

// Update YesNoModel to support red border and default No
// Update YesNoModel.View to render the prompt with red border and highlight selected option
func (m YesNoModel) View() string {
	boxWidth := 40
	prompt := "Are you sure you want to quit?"
	options := []string{"Yes", "No"}
	var renderedOptions []string
	for i, opt := range options {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(false).Width(8).Align(lipgloss.Center)
		if i == m.selected {
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
	return box
}

// Update YesNoModel tea.Model logic to support left/right/enter
func (m YesNoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h", "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "right", "l", "down", "j":
			if m.selected < 1 {
				m.selected++
			}
		case "tab":
			m.selected = 1 - m.selected
		case "enter":
			m.result = (m.selected == 0)
			return m, tea.Quit
		case "esc":
			m.result = false
			return m, tea.Quit
		}
	}
	return m, nil
}

// --- Custom Chat GUI Flow ---

// TextInputModel for chat name input
type TextInputModel struct {
	prompt   string
	value    string
	cursor   int
	quitting bool
	width    int
	height   int
	message  string
}

func (m TextInputModel) Init() tea.Cmd { return nil }

func (m TextInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case "backspace":
			if m.cursor > 0 && len(m.value) > 0 {
				m.value = m.value[:m.cursor-1] + m.value[m.cursor:]
				m.cursor--
			}
		case "left":
			if m.cursor > 0 {
				m.cursor--
			}
		case "right":
			if m.cursor < len(m.value) {
				m.cursor++
			}
		default:
			if len(msg.String()) == 1 && msg.Type == tea.KeyRunes {
				m.value = m.value[:m.cursor] + msg.String() + m.value[m.cursor:]
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m TextInputModel) View() string {
	// Center vertically and horizontally in the window
	prompt := lipgloss.NewStyle().Bold(true).Render(m.prompt)
	input := m.value
	if m.cursor >= 0 && m.cursor <= len(input) {
		input = input[:m.cursor] + "|" + input[m.cursor:]
	}
	inputBox := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(0, 1).Width(40).Render(input)
	msg := ""
	if m.message != "" {
		msg = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(m.message)
	}
	content := prompt + "\n" + inputBox + msg
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func GUICustomChatFlow() error {
	// 1. Chat name input
	nameModel := TextInputModel{prompt: "Enter chat name (leave blank for timestamp):", value: "", cursor: 0, width: 40, height: 24}
	p := tea.NewProgram(nameModel, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	nameInput := finalModel.(TextInputModel)
	if nameInput.quitting {
		return ErrMenuBack
	}
	chatName := strings.TrimSpace(nameInput.value)
	if chatName == "" {
		chatName = generateTimestampChatName()
	}
	// Check for duplicate
	chats, err := listChats()
	if err != nil {
		return err
	}
	for _, c := range chats {
		if c == chatName {
			return fmt.Errorf("chat '%s' already exists", chatName)
		}
	}

	// 2. Model selection
	models, _, err := loadModelsWithMostRecent()
	if err != nil || len(models) == 0 {
		return fmt.Errorf("no models available")
	}
	modelIdx, err := selectFromList("Select Model", models)
	if err != nil || modelIdx < 0 || modelIdx >= len(models) {
		return ErrMenuBack
	}
	model := models[modelIdx]

	// 3. Prompt selection
	prompts, err := loadPrompts()
	if err != nil || len(prompts) == 0 {
		// Try to initialize default prompts if missing
		prompts, err = initializeDefaultPrompts()
		if err != nil || len(prompts) == 0 {
			return fmt.Errorf("no prompts available")
		}
	}
	promptNames := make([]string, len(prompts))
	for i, p := range prompts {
		promptNames[i] = p.Name
	}
	promptIdx, err := selectFromList("Select Prompt", promptNames)
	if err != nil || promptIdx < 0 || promptIdx >= len(prompts) {
		return ErrMenuBack
	}
	promptContent := prompts[promptIdx].Content

	// 4. Create chat and launch
	messages := []Message{{Role: "system", Content: promptContent}}
	chatFile := ChatFile{
		Messages: messages,
		Metadata: ChatMetadata{
			Model:     model,
			CreatedAt: time.Now(),
		},
	}
	data, err := json.MarshalIndent(chatFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal chat: %w", err)
	}
	err = os.WriteFile(filepath.Join(chatsPath, chatName+".json"), data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write chat file '%s': %w", chatName, err)
	}
	gui := NewChatGUI(chatName, messages, model, bufio.NewReader(os.Stdin))
	return gui.Run()
}

// Implement saveDefaultModel
func saveDefaultModel(modelName string) error {
	// Load models config
	path := filepath.Join(utilPath, "models.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var config struct {
		Models []struct {
			Name      string `json:"name"`
			IsDefault bool   `json:"is_default"`
		} `json:"models"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	found := false
	for i := range config.Models {
		if config.Models[i].Name == modelName {
			config.Models[i].IsDefault = true
			found = true
		} else {
			config.Models[i].IsDefault = false
		}
	}
	if !found {
		// Add model if not present
		config.Models = append(config.Models, struct {
			Name      string `json:"name"`
			IsDefault bool   `json:"is_default"`
		}{Name: modelName, IsDefault: true})
	}
	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, newData, 0644)
}

// RunGUIMainMenu launches the Bubble Tea GUI main menu and routes to submenus
func RunGUIMainMenu() error {
	var width, height int
	width, height = 0, 0
	mainMenuOptions := []string{"Chats", "Favorites", "Prompts", "Models", "API Key", "Help", "Exit"}
	for {
		model := MenuModel{
			title:    "Main Menu",
			options:  mainMenuOptions,
			selected: 0,
			quitting: false,
			width:    width,
			height:   height,
		}
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("failed to run main menu: %w", err)
		}
		menuModel := finalModel.(MenuModel)
		width = menuModel.width
		height = menuModel.height
		if menuModel.selected == -1 || menuModel.selected == len(mainMenuOptions)-1 {
			return nil
		}
		if menuModel.selected == -2 {
			return ErrMenuBack
		}
		switch mainMenuOptions[menuModel.selected] {
		case "Chats":
			if err := GUIMenuChats(); err != nil {
				if err == ErrMenuBack {
					continue
				}
				return err
			}
		case "Favorites":
			if err := GUIMenuFavorites(); err != nil {
				if err == ErrMenuBack {
					continue
				}
				return err
			}
		case "Prompts":
			if err := GUIMenuPrompts(); err != nil {
				if err == ErrMenuBack {
					continue
				}
				return err
			}
		case "Models":
			if err := GUIMenuModels(); err != nil {
				if err == ErrMenuBack {
					continue
				}
				return err
			}
		case "API Key":
			if err := GUIMenuAPIKey(); err != nil {
				if err == ErrMenuBack {
					continue
				}
				return err
			}
		case "Help":
			if err := GUIShowHelp(); err != nil {
				if err == ErrMenuBack {
					continue
				}
				return err
			}
		}
	}
}
