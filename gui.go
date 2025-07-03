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
	"regexp"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// UpdateChatHistory updates the chat history display with proper formatting
func (g *ChatGUI) UpdateChatHistory() {
	// This will be handled by the Bubble Tea model
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
	switch cmd {
	case ":g":
		// Generate new title for chat using API
		g.generateTitleWithAPI()
		return true
	case ":f":
		// Toggle favorite status
		chatFile, err := loadChatWithMetadata(g.chatName)
		if err == nil {
			chatFile.Metadata.Favorite = !chatFile.Metadata.Favorite
			// Save the updated metadata by saving the entire chat
			if err := saveChat(g.chatName, chatFile.Messages); err == nil {
				// Success - status will be shown in the UI
			}
		}
		return true
	case ":q":
		// Save and quit
		if err := saveChat(g.chatName, g.messages); err == nil {
			return false // Exit
		}
		return true
	case ":h":
		// g.showHelp = true // REMOVE this line, ChatGUI does not have showHelp
		// Instead, handle help popup in ChatModel only
		return true
	default:
		return false // Not a vim command, treat as regular input
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
	chatName        string
	messages        []Message
	model           string
	inputBuffer     string
	width           int
	height          int
	status          string
	quitting        bool
	loading         bool
	spinner         int
	scrollPos       int       // Current scroll position (index of first visible message)
	autoScroll      bool      // Whether to auto-scroll to bottom
	stopChan        chan bool // Channel to signal stop request
	showConfirm     bool      // Whether to show exit confirmation
	generatingTitle bool      // Whether we are generating a title
	showError       bool      // Whether to show an error popup
	errorMsg        string    // The error message to display
	cursorPos       int       // Current cursor position in the input buffer
	showHelp        bool      // Whether to show the help popup
	blinkOn         bool      // Whether the cursor is currently visible
	lastBlink       time.Time // Last time the cursor blinked
}

func (m ChatModel) Init() tea.Cmd {
	return tea.Batch(
		blinkTick(),
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
		// Typable key wakeup logic
		if m.inputBuffer == "" && len(msg.String()) == 1 && msg.Type == tea.KeyRunes && !m.loading {
			m.inputBuffer = msg.String()
			m.cursorPos = 1
			m.blinkOn = true
			return m, blinkTick()
		}
		if len(msg.String()) == 1 && msg.Type == tea.KeyRunes && !m.loading {
			m.inputBuffer = m.inputBuffer[:m.cursorPos] + msg.String() + m.inputBuffer[m.cursorPos:]
			m.cursorPos++
			m.blinkOn = true
			return m, blinkTick()
		}
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
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
			if m.showConfirm {
				if err := saveChat(m.chatName, m.messages); err == nil {
					m.status = fmt.Sprintf("Saving %s...", m.chatName)
				}
				m.quitting = true
				return m, tea.Quit
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
							return m, tea.Batch(generateTitleCmd(m.messages, m.model), spinnerTick())
						}
						m.inputBuffer = ""
						return m, nil
					}
				}
				m.messages = append(m.messages, Message{Role: "user", Content: m.inputBuffer})
				m.inputBuffer = ""
				m.loading = true
				m.status = "Waiting for AI response..."
				m.autoScroll = true
				m.stopChan = make(chan bool)
				messagesCopy := make([]Message, len(m.messages))
				copy(messagesCopy, m.messages)
				return m, tea.Batch(getAIResponseCmd(messagesCopy, m.model, m.stopChan), spinnerTick())
			}
		case "pageup":
			if !m.loading {
				m.scrollPos = max(0, m.scrollPos-1)
				m.autoScroll = false
			}
		case "pagedown":
			if !m.loading {
				maxScroll := max(0, len(m.getVisibleMessages())-1)
				m.scrollPos = min(maxScroll, m.scrollPos+1)
				m.autoScroll = false
			}
		case "up":
			if !m.loading && m.inputBuffer == "" {
				m.scrollPos = max(0, m.scrollPos-1)
				m.autoScroll = false
			}
		case "down":
			if !m.loading && m.inputBuffer == "" {
				maxScroll := max(0, len(m.getVisibleMessages())-(m.height-6))
				m.scrollPos = min(maxScroll, m.scrollPos+1)
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
		return m, blinkTick()
	case aiResponseMsg:
		m.loading = false
		if m.stopChan != nil {
			close(m.stopChan)
			m.stopChan = nil
		}
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %v", msg.err)
		} else {
			if msg.response == "" {
				m.status = "Warning: Empty response received"
			} else {
				shouldAppend := true
				if len(m.messages) > 0 {
					lastMsg := m.messages[len(m.messages)-1]
					if lastMsg.Role == "assistant" && lastMsg.Content == msg.response {
						shouldAppend = false
					}
				}
				if shouldAppend {
					m.messages = append(m.messages, Message{Role: "assistant", Content: msg.response})
					m.status = "Ready"
					if m.autoScroll {
						m.scrollPos = max(0, len(m.getVisibleMessages())-(m.height-6))
					}
				}
			}
		}
		if err := saveChat(m.chatName, m.messages); err != nil {
			m.status = fmt.Sprintf("Save error: %v", err)
		}
	case aiTitleMsg:
		m.generatingTitle = false
		if err := setChatTitle(m.chatName, msg.title); err == nil {
			m.status = fmt.Sprintf("Title generated: %s", msg.title)
		} else {
			m.status = "Failed to set title"
		}
		return m, nil
	}
	return m, nil
}

func (m ChatModel) View() string {
	if m.quitting {
		return "Chat saved. Goodbye!\n"
	}

	// Styles
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	assistantStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	loadingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)

	// Box styles with borders
	chatBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	inputBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

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
			wrappedContent := wrapText(parsedContent, m.width-15)
			lines := strings.Split(wrappedContent, "\n")
			var boxStyle lipgloss.Style
			if msg.Role == "user" {
				boxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("203")).Padding(1, 2).Margin(1, 0)
			} else {
				boxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("39")).Padding(1, 2).Margin(1, 0)
			}
			if msg.Role == "user" {
				messageText = userStyle.Render("You:") + "\n" + strings.Join(lines, "\n")
			} else {
				messageText = assistantStyle.Render("Assistant:") + "\n" + strings.Join(lines, "\n")
			}
			visible = append(visible, boxStyle.Render(messageText))
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

	// Add scroll indicators at top and bottom
	scrollIndicatorTop := ""
	scrollIndicatorBottom := ""

	if len(visibleMessages) > chatBoxHeight {
		if m.scrollPos > 0 {
			scrollIndicatorTop = "↑ More messages above ↑\n"
		}
		if m.scrollPos < len(visibleMessages)-chatBoxHeight {
			indicator := ""
			indicatorText := "↓ More messages below ↓"
			indicator = lipgloss.NewStyle().Width(m.width - 2).Align(lipgloss.Center).Foreground(lipgloss.Color("39")).Render(indicatorText)
			scrollIndicatorBottom = "\n" + indicator
		} else {
			scrollIndicatorBottom = ""
		}
	}

	// Join messages with proper spacing
	messageContent := strings.Join(visible, "\n\n")
	if messageContent == "" {
		messageContent = "No messages yet..."
	}

	chatContent := scrollIndicatorTop + messageContent + scrollIndicatorBottom
	if strings.TrimSpace(chatContent) == "" {
		chatContent = "No messages yet..."
	}

	// Create chat box with border
	chatBox := chatBoxStyle.Width(m.width - 2).Height(chatBoxHeight).Render(chatContent)

	// Show confirmation dialog if needed
	if m.showConfirm {
		confirmStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("203")).
			Padding(1, 2).
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252"))

		confirmText := fmt.Sprintf("Are you sure you want to end the chat?\n\nPress ESC to confirm, any other key to cancel")
		confirmBox := confirmStyle.Width(m.width - 10).Render(confirmText)

		// Center the confirmation box
		centeredConfirm := lipgloss.NewStyle().Margin((m.height-5)/2, (m.width-lipgloss.Width(confirmBox))/2).Render(confirmBox)

		return lipgloss.NewStyle().Width(m.width).Height(m.height).Render(centeredConfirm)
	}

	// Input area - always 3 lines tall at bottom
	inputText := "Input: "
	if m.inputBuffer == "" {
		inputText = "*waiting for input...*\n\n:h for help"
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
		centeredPopup := lipgloss.NewStyle().Margin((m.height-5)/2, (m.width-lipgloss.Width(popupBox))/2).Render(popupBox)
		return lipgloss.NewStyle().Width(m.width).Height(m.height).Render(centeredPopup)
	}

	if m.showError {
		errorStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("203")).
			Padding(1, 2).
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252"))
		errorBox := errorStyle.Width(m.width - 10).Render(m.errorMsg + "\n\nESC to close")
		centeredError := lipgloss.NewStyle().Margin((m.height-5)/2, (m.width-lipgloss.Width(errorBox))/2).Render(errorBox)
		return lipgloss.NewStyle().Width(m.width).Height(m.height).Render(centeredError)
	}

	if m.showHelp {
		helpStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(1, 2).
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252"))
		helpText := "Vim Commands:\n:g  Generate title\n:t \"title\"  Set title\n:f  Favorite\n:q  Quit\n:h  Help\n\nChat Scroll:\nShift+Up/Down, PgUp/PgDn  Scroll chat\nHome/End  Move cursor\nLeft/Right  Move cursor"
		helpBox := helpStyle.Width(m.width - 10).Render(helpText + "\n\nESC to close")
		centeredHelp := lipgloss.NewStyle().Margin((m.height-7)/2, (m.width-lipgloss.Width(helpBox))/2).Render(helpBox)
		return lipgloss.NewStyle().Width(m.width).Height(m.height).Render(centeredHelp)
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
		showConfirm: false,
	}

	// Run the program
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run program: %w", err)
	}

	return nil
}

// runChatGUI is a wrapper function to start the GUI chat
func runChatGUI(chatName string, messages []Message, reader *bufio.Reader, model string) {
	gui := NewChatGUI(chatName, messages, model, reader)
	if err := gui.Run(); err != nil {
		fmt.Printf("GUI error: %v\n", err)
	}
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
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			m.quitting = true
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
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	if m.quitting {
		return ""
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var options strings.Builder
	options.WriteString(titleStyle.Render(m.title) + "\n\n")
	for i, option := range m.options {
		if i == m.selected {
			options.WriteString(selectedStyle.Render("> "+option) + "\n")
		} else {
			options.WriteString(normalStyle.Render("  "+option) + "\n")
		}
	}

	// Enhanced help text with more detailed controls
	help := helpStyle.Render("\nControls: ↑↓ to navigate, Enter to select, Esc to go back, Ctrl+C to quit")
	return options.String() + help
}

// Main menu
func RunGUIMainMenu() error {
	for {
		mainMenuOptions := []string{"Chats", "Favorites", "Prompts", "Models", "API Key", "Help", "Exit"}
		model := MenuModel{
			title:    "Main Menu",
			options:  mainMenuOptions,
			selected: 0,
			quitting: false,
		}
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("failed to run main menu: %w", err)
		}
		menuModel := finalModel.(MenuModel)
		if menuModel.quitting || menuModel.selected == len(mainMenuOptions)-1 {
			return nil
		}
		switch mainMenuOptions[menuModel.selected] {
		case "Chats":
			if err := GUIMenuChats(); err != nil {
				return err
			}
		case "Favorites":
			if err := GUIMenuFavorites(); err != nil {
				return err
			}
		case "Prompts":
			if err := GUIMenuPrompts(); err != nil {
				return err
			}
		case "Models":
			if err := GUIMenuModels(); err != nil {
				return err
			}
		case "API Key":
			if err := GUIMenuAPIKey(); err != nil {
				return err
			}
		case "Help":
			if err := GUIShowHelp(); err != nil {
				return err
			}
		}
	}
}

// Example for GUIMenuChats (apply this pattern to all menus)
func GUIMenuChats() error {
	for {
		options := []string{"List chats", "Load chat", "New chat", "Custom chat", "Back"}
		model := MenuModel{
			title:    "Chats Menu",
			options:  options,
			selected: 0,
			quitting: false,
		}
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("failed to run chats menu: %w", err)
		}
		menuModel := finalModel.(MenuModel)
		if menuModel.quitting || menuModel.selected == len(options)-1 {
			return nil
		}
		switch options[menuModel.selected] {
		case "List chats":
			if err := GUIListChats(); err != nil {
				return err
			}
		case "Load chat":
			if err := GUILoadChat(); err != nil {
				return err
			}
		case "New chat":
			if err := GUINewChat(); err != nil {
				return err
			}
		case "Custom chat":
			if err := GUICustomChat(); err != nil {
				return err
			}
		}
	}
}

// Example for GUILoadChat (apply this pattern to all list/select menus)
func GUILoadChat() error {
	chats, err := listChats()
	if err != nil {
		return err
	}
	if len(chats) == 0 {
		showMessage("No saved chats.", "Load Chat")
		return nil
	}
	var formattedChats []string
	for _, chat := range chats {
		chatFile, err := loadChatWithMetadata(chat)
		favoriteMark := " "
		if err == nil && chatFile.Metadata.Favorite {
			favoriteMark = "★"
		}
		formattedChats = append(formattedChats, fmt.Sprintf("%s %s", chat, favoriteMark))
	}
	model := MenuModel{
		title:    "Select Chat to Load",
		options:  formattedChats,
		selected: 0,
		quitting: false,
	}
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run load chat: %w", err)
	}
	menuModel := finalModel.(MenuModel)
	if menuModel.quitting {
		return nil
	}
	if menuModel.selected < len(chats) {
		chatName := chats[menuModel.selected]
		chatFile, err := loadChatWithMetadata(chatName)
		if err != nil {
			showMessage("Failed to load chat: "+err.Error(), "Error")
			return nil
		}
		model := chatFile.Metadata.Model
		if model == "" {
			model = DefaultModel()
		}
		runChatGUI(chatName, chatFile.Messages, nil, model)
	}
	return nil
}

// GUIMenuFavorites displays the Favorites menu
func GUIMenuFavorites() error {
	for {
		options := []string{"List favorites", "Load favorite", "Back"}
		model := MenuModel{
			title:    "Favorites Menu",
			options:  options,
			selected: 0,
			quitting: false,
		}
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("failed to run favorites menu: %w", err)
		}
		menuModel := finalModel.(MenuModel)
		if menuModel.quitting || menuModel.selected == len(options)-1 {
			return nil
		}
		switch options[menuModel.selected] {
		case "List favorites":
			if err := GUIListFavorites(); err != nil {
				return err
			}
		case "Load favorite":
			if err := GUILoadFavorite(); err != nil {
				return err
			}
		}
	}
}

// GUIListFavorites displays a list of favorite chats
func GUIListFavorites() error {
	favorites, err := listFavoriteChats()
	if err != nil {
		return err
	}
	if len(favorites) == 0 {
		showMessage("No favorite chats.", "Favorites List")
		return nil
	}

	model := MenuModel{
		title:    "Favorite Chats",
		options:  favorites,
		selected: 0,
		quitting: false,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run favorites list: %w", err)
	}
	menuModel := finalModel.(MenuModel)
	if menuModel.quitting {
		return nil
	}
	if menuModel.selected < len(favorites) {
		chatName := favorites[menuModel.selected]
		chatFile, err := loadChatWithMetadata(chatName)
		if err == nil {
			chatFile.Metadata.Favorite = !chatFile.Metadata.Favorite
			if err := saveChat(chatName, chatFile.Messages); err == nil {
				// Refresh the list
				return GUIListFavorites()
			}
		}
	}
	return nil
}

// GUILoadFavorite lets the user select and open a favorite chat
func GUILoadFavorite() error {
	favorites, err := listFavoriteChats()
	if err != nil {
		return err
	}
	if len(favorites) == 0 {
		showMessage("No favorite chats.", "Load Favorite")
		return nil
	}

	model := MenuModel{
		title:    "Select Favorite to Load",
		options:  favorites,
		selected: 0,
		quitting: false,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run load favorite: %w", err)
	}
	menuModel := finalModel.(MenuModel)
	if menuModel.quitting {
		return nil
	}
	if menuModel.selected < len(favorites) {
		chatName := favorites[menuModel.selected]
		chatFile, err := loadChatWithMetadata(chatName)
		if err != nil {
			showMessage("Failed to load chat: "+err.Error(), "Error")
			return nil
		}
		model := chatFile.Metadata.Model
		if model == "" {
			model = DefaultModel()
		}
		runChatGUI(chatName, chatFile.Messages, nil, model)
	}
	return nil
}

// GUIMenuPrompts displays the Prompts menu
func GUIMenuPrompts() error {
	for {
		options := []string{"List prompts", "Add prompt", "Set default", "Remove prompt", "Back"}
		model := MenuModel{
			title:    "Prompts Menu",
			options:  options,
			selected: 0,
			quitting: false,
		}
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("failed to run prompts menu: %w", err)
		}
		menuModel := finalModel.(MenuModel)
		if menuModel.quitting || menuModel.selected == len(options)-1 {
			return nil
		}
		switch options[menuModel.selected] {
		case "List prompts":
			if err := GUIListPrompts(); err != nil {
				return err
			}
		case "Add prompt":
			if err := GUIAddPrompt(); err != nil {
				return err
			}
		case "Set default":
			if err := GUISetDefaultPrompt(); err != nil {
				return err
			}
		case "Remove prompt":
			if err := GUIRemovePrompt(); err != nil {
				return err
			}
		}
	}
}

// GUIListPrompts displays a list of prompts
func GUIListPrompts() error {
	prompts, err := loadPrompts()
	if err != nil {
		showMessage("Failed to load prompts: "+err.Error(), "Error")
		return nil
	}

	var formattedPrompts []string
	for _, prompt := range prompts {
		mark := " "
		if prompt.Default {
			mark = "*"
		}
		formattedPrompts = append(formattedPrompts, fmt.Sprintf("%s %s", prompt.Name, mark))
	}

	model := MenuModel{
		title:    "Prompts",
		options:  formattedPrompts,
		selected: 0,
		quitting: false,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run prompts list: %w", err)
	}
	menuModel := finalModel.(MenuModel)
	if menuModel.quitting {
		return nil
	}
	if menuModel.selected < len(prompts) {
		prompt := prompts[menuModel.selected]
		details := fmt.Sprintf("Name: %s\n\nContent:\n%s", prompt.Name, prompt.Content)
		showMessage(details, "Prompt Details")
	}
	return nil
}

// GUIAddPrompt adds a new prompt with clipboard support
func GUIAddPrompt() error {
	// First, prompt for the prompt name
	nameModel := InputModel{
		title:     "Add Prompt",
		prompt:    "Enter a name for this prompt:",
		input:     "",
		multiline: false,
	}

	p := tea.NewProgram(nameModel, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run name input: %w", err)
	}

	nameInputModel := finalModel.(InputModel)
	if nameInputModel.quitting || !nameInputModel.submitted {
		return nil
	}

	name := strings.TrimSpace(nameInputModel.input)
	if name == "" {
		showMessage("Prompt name cannot be empty.", "Error")
		return nil
	}

	// Check if prompt name already exists
	prompts, err := loadPrompts()
	if err != nil {
		showMessage("Failed to load prompts: "+err.Error(), "Error")
		return nil
	}

	for _, prompt := range prompts {
		if prompt.Name == name {
			showMessage("A prompt with this name already exists.", "Error")
			return nil
		}
	}

	// Now prompt for the prompt content with clipboard option
	contentModel := InputModel{
		title:     "Add Prompt Content",
		prompt:    "Enter the prompt content (or press Enter to read from clipboard):",
		input:     "",
		multiline: true,
	}

	p = tea.NewProgram(contentModel, tea.WithAltScreen())
	finalModel, err = p.Run()
	if err != nil {
		return fmt.Errorf("failed to run content input: %w", err)
	}

	contentInputModel := finalModel.(InputModel)
	if contentInputModel.quitting {
		return nil
	}

	content := strings.TrimSpace(contentInputModel.input)

	// If content is empty, try to read from clipboard
	if content == "" {
		clipCmd := "powershell Get-Clipboard"
		clipOut, err := execCommand(clipCmd)
		if err != nil {
			showMessage("Failed to read clipboard. Please enter the prompt content manually.", "Error")
			return nil
		}

		content = strings.TrimSpace(clipOut)
		if content == "" {
			showMessage("Clipboard is empty. Please enter the prompt content manually.", "Error")
			return nil
		}

		showMessage("Content read from clipboard successfully.", "Info")
	}

	// Add the new prompt
	newPrompt := Prompt{
		Name:    name,
		Content: content,
		Default: false,
	}

	prompts = append(prompts, newPrompt)

	if err := savePrompts(prompts); err != nil {
		showMessage("Failed to save prompt: "+err.Error(), "Error")
		return nil
	}

	showMessage(fmt.Sprintf("Prompt '%s' added successfully.", name), "Success")
	return nil
}

// GUISetDefaultPrompt sets a prompt as default
func GUISetDefaultPrompt() error {
	prompts, err := loadPrompts()
	if err != nil {
		showMessage("Failed to load prompts: "+err.Error(), "Error")
		return nil
	}

	var formattedPrompts []string
	for _, prompt := range prompts {
		mark := " "
		if prompt.Default {
			mark = "*"
		}
		formattedPrompts = append(formattedPrompts, fmt.Sprintf("%s %s", prompt.Name, mark))
	}

	model := MenuModel{
		title:    "Select Prompt to Set as Default",
		options:  formattedPrompts,
		selected: 0,
		quitting: false,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run set default prompt: %w", err)
	}
	menuModel := finalModel.(MenuModel)
	if menuModel.quitting {
		return nil
	}
	if menuModel.selected < len(prompts) {
		prompt := prompts[menuModel.selected]
		if err := setPromptAsDefault(prompt.Name); err == nil {
			showMessage(fmt.Sprintf("Set '%s' as default prompt.", prompt.Name), "Success")
		} else {
			showMessage("Failed to set default prompt: "+err.Error(), "Error")
		}
	}
	return nil
}

// GUIRemovePrompt removes a prompt
func GUIRemovePrompt() error {
	prompts, err := loadPrompts()
	if err != nil {
		showMessage("Failed to load prompts: "+err.Error(), "Error")
		return nil
	}

	if len(prompts) == 0 {
		showMessage("No prompts to remove.", "Remove Prompt")
		return nil
	}

	var formattedPrompts []string
	for _, prompt := range prompts {
		mark := " "
		if prompt.Default {
			mark = "*"
		}
		formattedPrompts = append(formattedPrompts, fmt.Sprintf("%s %s", prompt.Name, mark))
	}

	model := MenuModel{
		title:    "Select Prompt to Remove",
		options:  formattedPrompts,
		selected: 0,
		quitting: false,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run remove prompt: %w", err)
	}
	menuModel := finalModel.(MenuModel)
	if menuModel.quitting {
		return nil
	}
	if menuModel.selected < len(prompts) {
		prompt := prompts[menuModel.selected]
		if prompt.Default {
			showMessage("Cannot remove the default prompt. Please set another prompt as default first.", "Error")
			return nil
		}

		// Remove the prompt
		newPrompts := make([]Prompt, 0, len(prompts)-1)
		for _, p := range prompts {
			if p.Name != prompt.Name {
				newPrompts = append(newPrompts, p)
			}
		}

		if err := savePrompts(newPrompts); err == nil {
			showMessage(fmt.Sprintf("Removed prompt '%s'.", prompt.Name), "Success")
		} else {
			showMessage("Failed to remove prompt: "+err.Error(), "Error")
		}
	}
	return nil
}

// GUIMenuModels displays the Models menu
func GUIMenuModels() error {
	for {
		options := []string{"List models", "Add model", "Set default", "Remove model", "Back"}
		model := MenuModel{
			title:    "Models Menu",
			options:  options,
			selected: 0,
			quitting: false,
		}
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("failed to run models menu: %w", err)
		}
		menuModel := finalModel.(MenuModel)
		if menuModel.quitting || menuModel.selected == len(options)-1 {
			return nil
		}
		switch options[menuModel.selected] {
		case "List models":
			if err := GUIListModels(); err != nil {
				return err
			}
		case "Add model":
			if err := GUIAddModel(); err != nil {
				return err
			}
		case "Set default":
			if err := GUISetDefaultModel(); err != nil {
				return err
			}
		case "Remove model":
			if err := GUIRemoveModel(); err != nil {
				return err
			}
		}
	}
}

// GUIListModels displays a list of models
func GUIListModels() error {
	models, defaultModel, err := loadModelsWithMostRecent()
	if err != nil {
		showMessage("Failed to load models: "+err.Error(), "Error")
		return nil
	}

	var formattedModels []string
	for _, model := range models {
		mark := " "
		if model == defaultModel {
			mark = "*"
		}
		formattedModels = append(formattedModels, fmt.Sprintf("%s %s", model, mark))
	}

	model := MenuModel{
		title:    "Models",
		options:  formattedModels,
		selected: 0,
		quitting: false,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run models list: %w", err)
	}
	menuModel := finalModel.(MenuModel)
	if menuModel.quitting {
		return nil
	}
	if menuModel.selected < len(models) {
		modelName := models[menuModel.selected]
		details := fmt.Sprintf("Model: %s\nDefault: %t", modelName, modelName == defaultModel)
		showMessage(details, "Model Details")
	}
	return nil
}

// GUIAddModel adds a new model
func GUIAddModel() error {
	showMessage("Add model functionality not yet implemented in GUI.\nUse CLI version for now.", "Add Model")
	return nil
}

// GUISetDefaultModel sets a model as default
func GUISetDefaultModel() error {
	models, defaultModel, err := loadModelsWithMostRecent()
	if err != nil {
		showMessage("Failed to load models: "+err.Error(), "Error")
		return nil
	}

	var formattedModels []string
	for _, model := range models {
		mark := " "
		if model == defaultModel {
			mark = "*"
		}
		formattedModels = append(formattedModels, fmt.Sprintf("%s %s", model, mark))
	}

	model := MenuModel{
		title:    "Select Model to Set as Default",
		options:  formattedModels,
		selected: 0,
		quitting: false,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run set default model: %w", err)
	}
	menuModel := finalModel.(MenuModel)
	if menuModel.quitting {
		return nil
	}
	if menuModel.selected < len(models) {
		modelName := models[menuModel.selected]
		if err := saveModelsWithMostRecent(modelName, models); err == nil {
			showMessage(fmt.Sprintf("Set '%s' as default model.", modelName), "Success")
		} else {
			showMessage("Failed to set default model: "+err.Error(), "Error")
		}
	}
	return nil
}

// GUIRemoveModel removes a model
func GUIRemoveModel() error {
	showMessage("Remove model functionality not yet implemented in GUI.\nUse CLI version for now.", "Remove Model")
	return nil
}

// GUIMenuAPIKey displays the API Key menu
func GUIMenuAPIKey() error {
	for {
		options := []string{"List API keys", "Add API key", "Set active", "Remove API key", "Back"}
		model := MenuModel{
			title:    "API Key Menu",
			options:  options,
			selected: 0,
			quitting: false,
		}
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("failed to run API key menu: %w", err)
		}
		menuModel := finalModel.(MenuModel)
		if menuModel.quitting || menuModel.selected == len(options)-1 {
			return nil
		}
		switch options[menuModel.selected] {
		case "List API keys":
			if err := GUIListAPIKeys(); err != nil {
				return err
			}
		case "Add API key":
			if err := GUIAddAPIKey(); err != nil {
				return err
			}
		case "Set active":
			if err := GUISetActiveAPIKey(); err != nil {
				return err
			}
		case "Remove API key":
			if err := GUIRemoveAPIKey(); err != nil {
				return err
			}
		}
	}
}

// GUIListAPIKeys displays a list of API keys
func GUIListAPIKeys() error {
	keys, activeKey, err := listAPIKeys()
	if err != nil {
		showMessage("Failed to load API keys: "+err.Error(), "Error")
		return nil
	}

	var formattedKeys []string
	for _, key := range keys {
		mark := " "
		if key.Title == activeKey {
			mark = "*"
		}
		formattedKeys = append(formattedKeys, fmt.Sprintf("%s %s", key.Title, mark))
	}

	model := apiKeyMenuModel{
		MenuModel: MenuModel{
			title:    "API Keys",
			options:  formattedKeys,
			selected: 0,
			quitting: false,
		},
		keys:      keys,
		activeKey: activeKey,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run API keys list: %w", err)
	}
	menuModel := finalModel.(apiKeyMenuModel)
	if menuModel.quitting {
		return nil
	}
	if menuModel.selected < len(keys) {
		key := keys[menuModel.selected]
		details := fmt.Sprintf("Title: %s\nActive: %t", key.Title, key.Title == activeKey)
		showMessage(details, "API Key Details")
	}
	return nil
}

// InputModel represents a simple input model for getting user input
type InputModel struct {
	title     string
	prompt    string
	input     string
	quitting  bool
	submitted bool
	multiline bool
}

func (m InputModel) Init() tea.Cmd {
	return nil
}

func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if m.multiline {
				// For multiline input, Enter adds a newline
				m.input += "\n"
			} else {
				// For single line input, Enter submits
				if m.input != "" {
					m.submitted = true
					return m, tea.Quit
				}
			}
		case "ctrl+s":
			// Ctrl+S submits multiline input
			if m.multiline && m.input != "" {
				m.submitted = true
				return m, tea.Quit
			}
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(msg.String()) == 1 {
				char := msg.String()[0]
				if char >= 32 && char <= 126 {
					m.input += msg.String()
				}
			}
		}
	}
	return m, nil
}

func (m InputModel) View() string {
	if m.quitting {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	inputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	helpText := "Press Enter to submit, Esc to cancel"
	if m.multiline {
		helpText = "Press Ctrl+S to submit, Esc to cancel"
	}

	content := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
		titleStyle.Render(m.title),
		promptStyle.Render(m.prompt),
		inputStyle.Render("> "+m.input),
		helpStyle.Render(helpText))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60).
		Align(lipgloss.Center)

	return boxStyle.Render(content)
}

// GUIAddAPIKey adds a new API key by reading from clipboard and prompting for name
func GUIAddAPIKey() error {
	// Confirmation prompt
	confirmModel := InputModel{
		title:     "Add API Key",
		prompt:    "Press Enter to read API key from clipboard, or Esc to cancel.",
		input:     "",
		multiline: false,
	}
	p := tea.NewProgram(confirmModel, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	inputModel := finalModel.(InputModel)
	if inputModel.quitting {
		return nil
	}

	// Read from clipboard
	clipCmd := "powershell Get-Clipboard"
	clipOut, err := execCommand(clipCmd)
	if err != nil {
		showMessage("Failed to read clipboard. Please copy your API key first.", "Error")
		return nil
	}
	key := strings.TrimSpace(clipOut)
	if key == "" {
		showMessage("Clipboard is empty. Please copy your API key first.", "Error")
		return nil
	}

	// Prompt for key name
	nameModel := InputModel{
		title:  "Add API Key",
		prompt: "Enter a title for this API key:",
		input:  "",
	}
	p = tea.NewProgram(nameModel, tea.WithAltScreen())
	finalModel, err = p.Run()
	if err != nil {
		return err
	}
	nameInputModel := finalModel.(InputModel)
	if nameInputModel.quitting || !nameInputModel.submitted {
		return nil
	}
	title := strings.TrimSpace(nameInputModel.input)
	if title == "" {
		title = "Default"
	}
	if err := addAPIKey(title, key); err != nil {
		showMessage("Failed to add API key: "+err.Error(), "Error")
		return nil
	}
	showMessage(fmt.Sprintf("API key '%s' added successfully.", title), "Success")
	return nil
}

// GUISetActiveAPIKey sets an API key as active
func GUISetActiveAPIKey() error {
	keys, activeKey, err := listAPIKeys()
	if err != nil {
		showMessage("Failed to load API keys: "+err.Error(), "Error")
		return nil
	}

	var formattedKeys []string
	for _, key := range keys {
		mark := " "
		if key.Title == activeKey {
			mark = "*"
		}
		formattedKeys = append(formattedKeys, fmt.Sprintf("%s %s", key.Title, mark))
	}

	model := MenuModel{
		title:    "Select API Key to Set as Active",
		options:  formattedKeys,
		selected: 0,
		quitting: false,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run set active API key: %w", err)
	}
	menuModel := finalModel.(MenuModel)
	if menuModel.quitting {
		return nil
	}
	if menuModel.selected < len(keys) {
		key := keys[menuModel.selected]
		if err := setActiveAPIKey(key.Title); err == nil {
			showMessage(fmt.Sprintf("Set '%s' as active API key.", key.Title), "Success")
		} else {
			showMessage("Failed to set active API key: "+err.Error(), "Error")
		}
	}
	return nil
}

// GUIRemoveAPIKey removes an API key
func GUIRemoveAPIKey() error {
	keys, activeKey, err := listAPIKeys()
	if err != nil {
		showMessage("Failed to load API keys: "+err.Error(), "Error")
		return nil
	}

	if len(keys) == 0 {
		showMessage("No API keys to remove.", "Remove API Key")
		return nil
	}

	var formattedKeys []string
	for _, key := range keys {
		mark := " "
		if key.Title == activeKey {
			mark = "*"
		}
		formattedKeys = append(formattedKeys, fmt.Sprintf("%s %s", key.Title, mark))
	}

	model := MenuModel{
		title:    "Select API Key to Remove",
		options:  formattedKeys,
		selected: 0,
		quitting: false,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run remove API key: %w", err)
	}
	menuModel := finalModel.(MenuModel)
	if menuModel.quitting {
		return nil
	}
	if menuModel.selected < len(keys) {
		key := keys[menuModel.selected]
		if key.Title == activeKey {
			showMessage("Cannot remove the active API key. Please set another key as active first.", "Error")
			return nil
		}
		if err := removeAPIKey(key.Title); err == nil {
			showMessage(fmt.Sprintf("Removed API key '%s'.", key.Title), "Success")
		} else {
			showMessage("Failed to remove API key: "+err.Error(), "Error")
		}
	}
	return nil
}

// GUIListChats displays a list of recent chats using Bubble Tea
func GUIListChats() error {
	chats, err := listChats()
	if err != nil {
		showMessage("Failed to list chats: "+err.Error(), "Chats List")
		return nil
	}
	if len(chats) == 0 {
		showMessage("No saved chats.", "Chats List")
		return nil
	}
	var formattedChats []string
	for _, chat := range chats {
		chatFile, err := loadChatWithMetadata(chat)
		favoriteMark := " "
		if err == nil && chatFile.Metadata.Favorite {
			favoriteMark = "★"
		}
		formattedChats = append(formattedChats, fmt.Sprintf("%s %s", chat, favoriteMark))
	}
	model := MenuModel{
		title:    "Recent Chats",
		options:  formattedChats,
		selected: 0,
		quitting: false,
	}
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run chat list: %w", err)
	}
	menuModel := finalModel.(MenuModel)
	if menuModel.quitting {
		return nil
	}
	if menuModel.selected < len(chats) {
		// Toggle favorite on selection
		chatName := chats[menuModel.selected]
		chatFile, err := loadChatWithMetadata(chatName)
		if err == nil {
			chatFile.Metadata.Favorite = !chatFile.Metadata.Favorite
			if err := saveChat(chatName, chatFile.Messages); err == nil {
				return GUIListChats()
			}
		}
	}
	return nil
}

// GUINewChat creates a new chat and opens it
func GUINewChat() error {
	chatName := fmt.Sprintf("chat-%d", time.Now().Unix())
	model := DefaultModel()
	prompt := "You are a helpful AI assistant."
	messages := []Message{{Role: "system", Content: prompt}}
	// Save the new chat
	var chatFile ChatFile
	chatFile.Messages = messages
	chatFile.Metadata.Model = model
	chatFile.Metadata.CreatedAt = time.Now()
	data, err := json.MarshalIndent(chatFile, "", "  ")
	if err != nil {
		showMessage("Failed to create chat: "+err.Error(), "Error")
		return nil
	}
	err = os.WriteFile(filepath.Join(chatsPath, chatName+".json"), data, 0644)
	if err != nil {
		showMessage("Failed to save chat: "+err.Error(), "Error")
		return nil
	}
	runChatGUI(chatName, messages, nil, model)
	return nil
}

// GUICustomChat creates a new chat with custom API key, model, and prompt selection
func GUICustomChat() error {
	// Step 1: Select API Key
	apiKeys, activeKey, err := listAPIKeys()
	if err != nil {
		showMessage("Failed to load API keys: "+err.Error(), "Error")
		return nil
	}

	if len(apiKeys) == 0 {
		showMessage("No API keys found. Please add an API key first.", "No API Keys")
		return nil
	}

	var apiKeyOptions []string
	for _, key := range apiKeys {
		mark := " "
		if key.Title == activeKey {
			mark = "*"
		}
		apiKeyOptions = append(apiKeyOptions, fmt.Sprintf("%s %s", key.Title, mark))
	}

	apiKeyModel := MenuModel{
		title:    "Select API Key",
		options:  apiKeyOptions,
		selected: 0,
		quitting: false,
	}

	p := tea.NewProgram(apiKeyModel, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run API key selection: %w", err)
	}

	menuModel := finalModel.(MenuModel)
	if menuModel.quitting {
		return nil
	}

	selectedAPIKey := apiKeys[menuModel.selected].Title

	// Step 2: Select Model
	models, defaultModel, err := loadModelsWithMostRecent()
	if err != nil {
		showMessage("Failed to load models: "+err.Error(), "Error")
		return nil
	}

	var modelOptions []string
	for _, model := range models {
		mark := " "
		if model == defaultModel {
			mark = "*"
		}
		modelOptions = append(modelOptions, fmt.Sprintf("%s %s", model, mark))
	}

	modelMenuModel := MenuModel{
		title:    "Select Model",
		options:  modelOptions,
		selected: 0,
		quitting: false,
	}

	p = tea.NewProgram(modelMenuModel, tea.WithAltScreen())
	finalModel, err = p.Run()
	if err != nil {
		return fmt.Errorf("failed to run model selection: %w", err)
	}

	menuModel = finalModel.(MenuModel)
	if menuModel.quitting {
		return nil
	}

	selectedModel := models[menuModel.selected]

	// Step 3: Select Prompt
	prompts, err := loadPrompts()
	if err != nil {
		showMessage("Failed to load prompts: "+err.Error(), "Error")
		return nil
	}

	var promptOptions []string
	for _, prompt := range prompts {
		mark := " "
		if prompt.Default {
			mark = "*"
		}
		promptOptions = append(promptOptions, fmt.Sprintf("%s %s", prompt.Name, mark))
	}

	promptMenuModel := MenuModel{
		title:    "Select Prompt",
		options:  promptOptions,
		selected: 0,
		quitting: false,
	}

	p = tea.NewProgram(promptMenuModel, tea.WithAltScreen())
	finalModel, err = p.Run()
	if err != nil {
		return fmt.Errorf("failed to run prompt selection: %w", err)
	}

	menuModel = finalModel.(MenuModel)
	if menuModel.quitting {
		return nil
	}

	selectedPrompt := prompts[menuModel.selected]

	// Step 4: Set the selected API key as active for this session
	if err := setActiveAPIKey(selectedAPIKey); err != nil {
		showMessage("Failed to set active API key: "+err.Error(), "Error")
		return nil
	}

	// Step 5: Create and start the chat
	chatName := fmt.Sprintf("chat-%d", time.Now().Unix())
	messages := []Message{{Role: "system", Content: selectedPrompt.Content}}

	// Save the new chat
	var chatFile ChatFile
	chatFile.Messages = messages
	chatFile.Metadata.Model = selectedModel
	chatFile.Metadata.CreatedAt = time.Now()
	data, err := json.MarshalIndent(chatFile, "", "  ")
	if err != nil {
		showMessage("Failed to create chat: "+err.Error(), "Error")
		return nil
	}
	err = os.WriteFile(filepath.Join(chatsPath, chatName+".json"), data, 0644)
	if err != nil {
		showMessage("Failed to save chat: "+err.Error(), "Error")
		return nil
	}

	runChatGUI(chatName, messages, nil, selectedModel)
	return nil
}

// MessageModel represents a simple message display model
type MessageModel struct {
	content  string
	quitting bool
}

func (m MessageModel) Init() tea.Cmd {
	return nil
}

func (m MessageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m MessageModel) View() string {
	return m.content
}

// showMessage displays a simple message
func showMessage(msg, title string) {
	messageStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(50).
		Align(lipgloss.Center)
	content := messageStyle.Render(fmt.Sprintf("%s\n\n%s\n\nPress any key to continue", title, msg))
	msgModel := MessageModel{
		content:  content,
		quitting: false,
	}
	p := tea.NewProgram(msgModel, tea.WithAltScreen())
	_, _ = p.Run()
}

// Patch MenuModel's Update for apiKeyMenuModel to handle 's' key
func (m apiKeyMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			m.quitting = true
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
		case "s":
			if m.selected < len(m.keys) {
				key := m.keys[m.selected]
				showMessage(fmt.Sprintf("Title: %s\n\nAPI Key (Sensitive!):\n%s", key.Title, key.Key), "Show API Key")
			}
		}
	}
	return m, nil
}

// Custom View function for apiKeyMenuModel with enhanced help text
func (m apiKeyMenuModel) View() string {
	if m.quitting {
		return ""
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var options strings.Builder
	options.WriteString(titleStyle.Render(m.title) + "\n\n")
	for i, option := range m.options {
		if i == m.selected {
			options.WriteString(selectedStyle.Render("> "+option) + "\n")
		} else {
			options.WriteString(normalStyle.Render("  "+option) + "\n")
		}
	}

	// Enhanced help text with 's' key hint for showing API keys
	help := helpStyle.Render("\nControls: ↑↓ to navigate, Enter to select, S to show key, Esc to go back, Ctrl+C to quit")
	return options.String() + help
}

// getVisibleMessages returns the list of visible messages (excluding system messages)
func (m ChatModel) getVisibleMessages() []Message {
	var visible []Message
	for _, msg := range m.messages {
		if msg.Role != "system" {
			visible = append(visible, msg)
		}
	}
	return visible
}

func (m *ChatModel) handleVimCommand(cmd string) bool {
	switch {
	case cmd == ":g":
		// Generate title using AI
		if len(m.messages) > 0 {
			title := generateChatTitle(m.messages, m.model)
			if err := setChatTitle(m.chatName, title); err == nil {
				m.status = fmt.Sprintf("Title generated: %s", title)
			} else {
				m.status = "Failed to set title"
			}
		}
		return true
	case strings.HasPrefix(cmd, ":t "):
		// Set custom title
		title := strings.TrimSpace(strings.TrimPrefix(cmd, ":t "))
		title = strings.Trim(title, `"'`)
		if title == "" {
			m.showError = true
			m.errorMsg = "Please enter a title\nExample: :t \"My Project Chat\""
			return true
		}
		if err := setChatTitle(m.chatName, title); err == nil {
			m.status = fmt.Sprintf("Title set: %s", title)
		} else {
			m.status = "Failed to set title"
		}
		return true
	case cmd == ":f":
		chatFile, err := loadChatWithMetadata(m.chatName)
		if err == nil {
			chatFile.Metadata.Favorite = !chatFile.Metadata.Favorite
			if err := saveChat(m.chatName, chatFile.Messages); err == nil {
				status := "unfavorited"
				if chatFile.Metadata.Favorite {
					status = "favorited"
				}
				m.status = fmt.Sprintf("Chat %s", status)
			}
		}
		return true
	case cmd == ":q":
		if err := saveChat(m.chatName, m.messages); err == nil {
			m.quitting = true
			return true
		}
		return true
	case cmd == ":h":
		// g.showHelp = true // REMOVE this line, ChatGUI does not have showHelp
		// Instead, handle help popup in ChatModel only
		return true
	default:
		return false
	}
}

// parseMarkdown parses markdown content and returns styled text
func parseMarkdown(content string) string {
	if content == "" {
		return content
	}

	// Split content into lines for processing
	lines := strings.Split(content, "\n")
	var result []string

	for i, line := range lines {
		styledLine := line

		// Headers (h1-h6)
		if strings.HasPrefix(line, "# ") {
			styledLine = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")).Render(strings.TrimPrefix(line, "# "))
		} else if strings.HasPrefix(line, "## ") {
			styledLine = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(strings.TrimPrefix(line, "## "))
		} else if strings.HasPrefix(line, "### ") {
			styledLine = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203")).Render(strings.TrimPrefix(line, "### "))
		} else if strings.HasPrefix(line, "#### ") {
			styledLine = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render(strings.TrimPrefix(line, "#### "))
		} else if strings.HasPrefix(line, "##### ") {
			styledLine = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("142")).Render(strings.TrimPrefix(line, "##### "))
		} else if strings.HasPrefix(line, "###### ") {
			styledLine = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240")).Render(strings.TrimPrefix(line, "###### "))
		} else {
			// Bold text (**text** or __bold__)
			styledLine = parseBoldText(styledLine)

			// Italic text (*text* or _text_)
			styledLine = parseItalicText(styledLine)

			// Code blocks (```code```)
			styledLine = parseCodeBlocks(styledLine)

			// Inline code (`code`)
			styledLine = parseInlineCode(styledLine)

			// Links [text](url)
			styledLine = parseLinks(styledLine)

			// Lists
			styledLine = parseLists(styledLine)
		}

		result = append(result, styledLine)

		// Add newline after each line except the last one
		if i < len(lines)-1 {
			result = append(result, "")
		}
	}

	return strings.Join(result, "\n")
}

// parseBoldText handles **bold** and __bold__ text
func parseBoldText(text string) string {
	// Handle **bold** text
	text = regexp.MustCompile(`\*\*(.*?)\*\*`).ReplaceAllStringFunc(text, func(match string) string {
		content := match[2 : len(match)-2] // Remove **
		return lipgloss.NewStyle().Bold(true).Render(content)
	})

	// Handle __bold__ text
	text = regexp.MustCompile(`__(.*?)__`).ReplaceAllStringFunc(text, func(match string) string {
		content := match[2 : len(match)-2] // Remove __
		return lipgloss.NewStyle().Bold(true).Render(content)
	})

	return text
}

// parseItalicText handles *italic* and _italic_ text
func parseItalicText(text string) string {
	// Handle *italic* text
	text = regexp.MustCompile(`\*(.*?)\*`).ReplaceAllStringFunc(text, func(match string) string {
		content := match[1 : len(match)-1] // Remove *
		return lipgloss.NewStyle().Italic(true).Render(content)
	})

	// Handle _italic_ text
	text = regexp.MustCompile(`_(.*?)_`).ReplaceAllStringFunc(text, func(match string) string {
		content := match[1 : len(match)-1] // Remove _
		return lipgloss.NewStyle().Italic(true).Render(content)
	})

	return text
}

// parseCodeBlocks handles ```code``` blocks
func parseCodeBlocks(text string) string {
	return regexp.MustCompile("```(.*?)```").ReplaceAllStringFunc(text, func(match string) string {
		// Extract content between ```
		start := strings.Index(match, "```")
		end := strings.LastIndex(match, "```")
		if start != -1 && end != -1 && end > start {
			content := match[start+3 : end]
			return lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("252")).
				Padding(0, 1).
				Render(content)
		}
		return match
	})
}

// parseInlineCode handles `code` inline code
func parseInlineCode(text string) string {
	return regexp.MustCompile("`([^`]+)`").ReplaceAllStringFunc(text, func(match string) string {
		// Extract content between backticks
		content := match[1 : len(match)-1] // Remove backticks
		return lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1).
			Render(content)
	})
}

// parseLinks handles [text](url) links
func parseLinks(text string) string {
	return regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`).ReplaceAllStringFunc(text, func(match string) string {
		// Extract link text and URL
		re := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
		matches := re.FindStringSubmatch(match)
		if len(matches) >= 3 {
			linkText := matches[1]
			url := matches[2]
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Underline(true).
				Render(linkText + " (" + url + ")")
		}
		return match
	})
}

// parseLists handles markdown lists
func parseLists(text string) string {
	// Handle numbered lists (1. item)
	text = regexp.MustCompile(`^(\d+\.\s+)(.*)$`).ReplaceAllStringFunc(text, func(match string) string {
		re := regexp.MustCompile(`^(\d+\.\s+)(.*)$`)
		matches := re.FindStringSubmatch(match)
		if len(matches) >= 3 {
			number := matches[1]
			content := matches[2]
			return lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(number) + content
		}
		return match
	})

	// Handle bullet lists (- item, * item, + item)
	text = regexp.MustCompile(`^([-*+]\s+)(.*)$`).ReplaceAllStringFunc(text, func(match string) string {
		re := regexp.MustCompile(`^([-*+]\s+)(.*)$`)
		matches := re.FindStringSubmatch(match)
		if len(matches) >= 3 {
			bullet := matches[1]
			content := matches[2]
			return lipgloss.NewStyle().Foreground(lipgloss.Color("142")).Render(bullet) + content
		}
		return match
	})

	return text
}

// Add generateTitleCmd
func generateTitleCmd(messages []Message, model string) tea.Cmd {
	return func() tea.Msg {
		title := generateChatTitle(messages, model)
		return aiTitleMsg{title: title}
	}
}

type aiTitleMsg struct {
	title string
}

// Move helpModel and its methods to top-level:
type helpModel struct{ quitting bool }

var helpText = `
Go AI CLI - Help

Controls:
  Arrow keys: Move cursor in input
  Home/End: Move to start/end of input (or scroll if input is empty)
  Shift+Up/Down, PgUp/PgDn: Scroll chat
  Ctrl+S: Stop/cancel AI response
  Ctrl+C: Quit
  :g - Generate chat title
  :t "title" - Set chat title
  :f - Toggle favorite
  :q - Save and quit chat
  :h - Show this help

Paths:
  .util path: .util
  Chats folder: .util/chats

Functionality:
  - Markdown rendering for chat messages
  - Scrollable chat history
  - Input box with navigation and cursor
  - Popups for errors, help, and title generation
  - Multi-chat management
`

func (m helpModel) Init() tea.Cmd { return nil }
func (m helpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		return helpModel{quitting: true}, tea.Quit
	}
	return m, nil
}
func (m helpModel) View() string {
	helpStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(2, 4).Width(80).Align(lipgloss.Center)
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	box := helpStyle.Render(helpText + "\n\n" + redStyle.Render("Press ESC to return to main menu"))
	padV := (24 - lipgloss.Height(box)) / 2
	padH := (100 - lipgloss.Width(box)) / 2
	return lipgloss.NewStyle().Margin(padV, padH).Render(box)
}

// Update GUIShowHelp to just run the program:
func GUIShowHelp() error {
	_, err := tea.NewProgram(helpModel{}, tea.WithAltScreen()).Run()
	return err
}

func blinkTick() tea.Cmd {
	return tea.Tick(530*time.Millisecond, func(t time.Time) tea.Msg {
		return blinkTickMsg{}
	})
}
