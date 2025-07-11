package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Message represents a chat message
type Message struct {
	Role          string `json:"role"`
	Content       string `json:"content"`
	MessageNumber int    `json:"message_number"`
}

// ChatMetadata stores additional information about the chat
// Add Model string to store the model used for the chat
type ChatMetadata struct {
	Summary   string    `json:"summary,omitempty"`
	Title     string    `json:"title,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	Model     string    `json:"model,omitempty"`
	Favorite  bool      `json:"favorite,omitempty"`
}

// ChatFile represents the complete chat file structure
type ChatFile struct {
	Metadata ChatMetadata `json:"metadata"`
	Messages []Message    `json:"messages"`
}

// ChatCommand represents a chat command
type ChatCommand struct {
	Command     string
	Description string
	Handler     func(messages []Message, chatName string, model string) (bool, error)
}

// Default system prompt for chat initialization
var systemPrompt Message

var commands []ChatCommand

// Global variable to track the currently active chat
var activeChatName string

// listChats lists the 10 most recent saved chat filenames without extension, sorted by creation date (newest to oldest)
func listChats() ([]string, error) {
	files, err := os.ReadDir(chatsPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read chat directory: %w", err)
	}
	type chatInfo struct {
		Name       string
		CreatedAt  time.Time
		ModifiedAt time.Time
	}
	var chatInfos []chatInfo
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
			name := strings.TrimSuffix(f.Name(), ".json")
			chatFile, err := loadChatWithMetadata(name)
			created := time.Time{}
			modified := time.Time{}

			// Get file modification time
			if fileInfo, err := f.Info(); err == nil {
				modified = fileInfo.ModTime()
			}

			if err == nil {
				created = chatFile.Metadata.CreatedAt
			}
			chatInfos = append(chatInfos, chatInfo{Name: name, CreatedAt: created, ModifiedAt: modified})
		}
	}
	// Sort by ModifiedAt (newest first, nil/zero times last)
	sort.Slice(chatInfos, func(i, j int) bool {
		if chatInfos[i].ModifiedAt.IsZero() && !chatInfos[j].ModifiedAt.IsZero() {
			return false
		}
		if !chatInfos[i].ModifiedAt.IsZero() && chatInfos[j].ModifiedAt.IsZero() {
			return true
		}
		return chatInfos[i].ModifiedAt.After(chatInfos[j].ModifiedAt)
	})

	// Return only the 10 most recent chats
	maxChats := 10
	if len(chatInfos) > maxChats {
		chatInfos = chatInfos[:maxChats]
	}

	chats := make([]string, len(chatInfos))
	for i, ci := range chatInfos {
		chats[i] = ci.Name
	}
	return chats, nil
}

// loadChatWithMetadata loads the complete chat file including metadata
func loadChatWithMetadata(name string) (*ChatFile, error) {
	data, err := os.ReadFile(filepath.Join(chatsPath(), name+".json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read chat file '%s': %w", name, err)
	}
	var chatFile ChatFile
	if err := json.Unmarshal(data, &chatFile); err != nil {
		// Try loading legacy format (just messages array)
		var messages []Message
		if err2 := json.Unmarshal(data, &messages); err2 != nil {
			if serr, ok := err2.(*json.SyntaxError); ok {
				// Find line number from offset
				lineNum := 1
				for _, b := range data[:serr.Offset] {
					if b == '\n' {
						lineNum++
					}
				}
				return nil, fmt.Errorf("failed to unmarshal chat file '%s': %v (line %d)", name, err2, lineNum)
			}
			return nil, fmt.Errorf("failed to unmarshal chat file '%s': %w", name, err2)
		}
		chatFile.Messages = messages
	}

	// After loading messages from JSON, for each message, replace all occurrences of '\n' with '\n' (actual newline) in msg.Content.
	for i := range chatFile.Messages {
		chatFile.Messages[i].Content = strings.ReplaceAll(chatFile.Messages[i].Content, "\\n", "\n")
		// Assign message number if missing or zero (except for system message at index 0)
		if chatFile.Messages[i].MessageNumber == 0 && i != 0 {
			chatFile.Messages[i].MessageNumber = i
		}
	}

	return &chatFile, nil
}

// saveChat saves chat messages and metadata to a file
func saveChat(name string, messages []Message) error {
	// Try to load existing metadata
	var chatFile ChatFile
	if existingChat, err := loadChatWithMetadata(name); err == nil {
		chatFile = *existingChat
	}
	chatFile.Messages = messages
	// Set CreatedAt if not already set
	if chatFile.Metadata.CreatedAt.IsZero() {
		chatFile.Metadata.CreatedAt = time.Now()
	}
	data, err := json.MarshalIndent(chatFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal chat: %w", err)
	}
	err = os.WriteFile(filepath.Join(chatsPath(), name+".json"), data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write chat file '%s': %w", name, err)
	}
	return nil
}

// generateTimestampChatName generates a timestamp-based chat name in ddmmyyhhss format
func generateTimestampChatName() string {
	now := time.Now()
	return now.Format("2006-01-02_15-04-05") // YYYY-MM-DD_HH-MM-SS
}

// customChatFlow creates a new chat with user-selected model and prompt
func customChatFlow(reader *bufio.Reader) error {
	chatName, err := setupNewChat(reader)
	if err != nil {
		return err
	}

	// Let user select model
	models, defaultModel, err := loadModelsWithMostRecent()
	if err != nil {
		fmt.Println("Error loading models, using fallback default.")
		defaultModel = DefaultModel()
	}
	model, err := promptModelSelection(reader, models, defaultModel)
	if err != nil {
		return fmt.Errorf("failed to select model: %w", err)
	}

	// Let user select prompt
	promptName, promptContent, err := promptPromptSelection(reader)
	if err != nil {
		return fmt.Errorf("failed to select prompt: %w", err)
	}

	// Create initial message slice with system role
	messages := []Message{
		{Role: "system", Content: promptContent},
	}

	// Save the new chat with model in metadata
	var chatFile ChatFile
	chatFile.Messages = messages
	chatFile.Metadata.Model = model
	chatFile.Metadata.CreatedAt = time.Now()
	data, err := json.MarshalIndent(chatFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal chat: %w", err)
	}
	err = os.WriteFile(filepath.Join(chatsPath(), chatName+".json"), data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write chat file '%s': %w", chatName, err)
	}

	fmt.Printf("Starting custom chat with model '%s' and prompt '%s'...\n\n",
		model, promptName)

	runChat(chatName, messages, reader, model)
	return nil
}

// generateChatSummary generates a short summary of the chat
func generateChatSummary(messages []Message, model string) string {
	if len(messages) == 0 {
		return "Empty chat."
	}

	// Append user prompt requesting short summary
	summaryPrompt := Message{
		Role:    "user",
		Content: "Please provide a short summary of the chat, no longer than 2 sentences.",
	}
	summaryMessages := append(messages, summaryPrompt)

	// Temporarily redirect stdout to /dev/null during summary generation
	savedStdout := os.Stdout
	os.Stdout = nil

	summary, err := streamChatResponse(summaryMessages, model)

	// Restore stdout
	os.Stdout = savedStdout

	if err != nil {
		return fmt.Sprintf("Chat with %d messages. (Summary unavailable: %v)", len(messages), err)
	}
	return summary
}

// setupNewChat handles common chat creation logic
func setupNewChat(reader *bufio.Reader) (string, error) {
	fmt.Print("Enter chat name (press Enter for timestamp): ")
	chatName, _ := reader.ReadString('\n')
	chatName = strings.TrimSpace(chatName)
	if chatName == "" {
		chatName = generateTimestampChatName()
		fmt.Printf("Using timestamp as chat name: %s\n", chatName)
	}

	// Check if chat already exists
	chats, err := listChats()
	if err != nil {
		return "", fmt.Errorf("failed to check existing chats: %w", err)
	}
	for _, c := range chats {
		if c == chatName {
			return "", fmt.Errorf("chat '%s' already exists", chatName)
		}
	}

	return chatName, nil
}

func init() {
	systemPrompt = Message{
		Role:    "system",
		Content: "You are a helpful AI assistant.",
	}

	commands = []ChatCommand{
		{
			Command:     "!q, !quit, !exit, !e",
			Description: "Exit the chat",
			Handler: func(messages []Message, chatName string, model string) (bool, error) {
				if len(messages) > 1 {
					// Always generate summary when exiting
					fmt.Println("Generating summary for chat...")
					summary := generateChatSummary(messages, model)

					// Load existing chat file to preserve metadata
					var chatFile ChatFile
					if existingChat, err := loadChatWithMetadata(chatName); err == nil {
						chatFile = *existingChat
					}
					chatFile.Messages = messages
					chatFile.Metadata.Summary = summary

					// Save with summary
					if err := saveChat(chatName, messages); err != nil {
						return true, fmt.Errorf("saving chat on exit: %w", err)
					}
					fmt.Println("Chat saved as:", chatName)

					// Prompt for new file name
					reader := bufio.NewReader(os.Stdin)
					fmt.Print("Enter a new chat file name, !g to generate a title, or leave blank to use the timestamp: ")
					newName, _ := reader.ReadString('\n')
					newName = strings.TrimSpace(newName)
					finalName := chatName

					if newName == "!g" {
						// Use the generated summary to create a title
						titlePrompt := Message{
							Role:    "user",
							Content: "Please come up with a title for a chat based on this information. No longer than 5 words.\n" + summary,
						}
						titleMessages := append(messages, titlePrompt)
						generatedTitle, err := streamChatResponse(titleMessages, model)
						if err != nil {
							fmt.Println("Failed to generate title, using timestamp.")
							finalName = chatName
						} else {
							// Clean up the generated title for filename use
							generatedTitle = strings.TrimSpace(generatedTitle)
							generatedTitle = strings.ReplaceAll(generatedTitle, " ", "_")
							generatedTitle = strings.ReplaceAll(generatedTitle, "/", "-")
							generatedTitle = strings.ReplaceAll(generatedTitle, "\\", "-")
							generatedTitle = strings.ReplaceAll(generatedTitle, ":", "-")
							generatedTitle = strings.ReplaceAll(generatedTitle, "*", "-")
							generatedTitle = strings.ReplaceAll(generatedTitle, "?", "-")
							generatedTitle = strings.ReplaceAll(generatedTitle, "\"", "-")
							generatedTitle = strings.ReplaceAll(generatedTitle, "<", "-")
							generatedTitle = strings.ReplaceAll(generatedTitle, ">", "-")
							generatedTitle = strings.ReplaceAll(generatedTitle, "|", "-")
							if generatedTitle == "" {
								finalName = chatName
							} else {
								finalName = generatedTitle
							}
						}
					} else if newName != "" {
						finalName = newName
					}

					// If the name changed, rename the file
					if finalName != chatName {
						oldPath := filepath.Join(chatsPath(), chatName+".json")
						newPath := filepath.Join(chatsPath(), finalName+".json")
						if err := os.Rename(oldPath, newPath); err != nil {
							fmt.Printf("Failed to rename chat file: %v\n", err)
						} else {
							fmt.Printf("Chat file renamed to: %s\n", finalName)
						}
					}
				}
				fmt.Println("Exiting chat.")
				return true, nil
			},
		},
		{
			Command:     "!save",
			Description: "Save the current chat",
			Handler: func(messages []Message, chatName string, _ string) (bool, error) {
				if len(messages) > 1 {
					if err := saveChat(chatName, messages); err != nil {
						return false, fmt.Errorf("manual chat save: %w", err)
					}
					fmt.Println("Chat saved as:", chatName)
				} else {
					fmt.Println("No messages to save.")
				}
				return false, nil
			},
		},
		{
			Command:     "!help",
			Description: "Show available commands",
			Handler: func(messages []Message, chatName string, _ string) (bool, error) {
				fmt.Println("\nAvailable commands:")
				for _, cmd := range commands {
					fmt.Printf("%s - %s\n", cmd.Command, cmd.Description)
				}
				return false, nil
			},
		},
		{
			Command:     "!clear",
			Description: "Clear the chat history but keep the system prompt",
			Handler: func(messages []Message, chatName string, _ string) (bool, error) {
				if len(messages) <= 1 {
					fmt.Println("Chat is already empty.")
					return false, nil
				}
				systemMsg := messages[0]
				messages = []Message{systemMsg}
				fmt.Println("Chat history cleared.")
				return false, nil
			},
		},
		{
			Command:     "!summary",
			Description: "Generate a summary of the current chat",
			Handler: func(messages []Message, chatName string, model string) (bool, error) {
				if len(messages) <= 1 {
					fmt.Println("Not enough messages to generate a summary.")
					return false, nil
				}
				summary := generateChatSummary(messages, model)
				fmt.Printf("\nChat summary: %s\n", summary)
				return false, nil
			},
		},
	}
}

// runChat handles the chat interaction loop
func runChat(chatName string, messages []Message, reader *bufio.Reader, model string) {
	// Set this as the active chat
	activeChatName = chatName
	defer func() {
		// Clear active chat when function exits
		activeChatName = ""
	}()

	messages = prependSystemPrompt(messages, systemPrompt)

	// Load existing chat file to preserve metadata
	var chatFile ChatFile
	if existingChat, err := loadChatWithMetadata(chatName); err == nil {
		chatFile = *existingChat
	}
	chatFile.Messages = messages

	if len(messages) == 1 {
		fmt.Println("Sending initial system prompt to AI...")
		resp, err := streamChatResponse(messages, model)
		if err != nil {
			handleError(err, "getting initial AI response")
			if strings.Contains(err.Error(), "API returned status 400") {
				return
			}
		} else {
			messages = append(messages, Message{Role: "assistant", Content: resp})
			chatFile.Messages = messages
		}
	}

	for {
		userInput := readMultiLineInput(reader)
		if userInput == "" {
			continue
		}

		foundCommand := false
		for _, cmd := range commands {
			cmdParts := strings.Split(cmd.Command, ", ")
			for _, part := range cmdParts {
				if userInput == part {
					foundCommand = true
					exit, err := cmd.Handler(messages, chatName, model)
					if err != nil {
						handleError(err, "executing command")
					}
					if exit {
						return
					}
					break
				}
			}
			if foundCommand {
				break
			}
		}
		if foundCommand {
			continue
		}

		messages = append(messages, Message{Role: "user", Content: userInput, MessageNumber: len(messages)})
		chatFile.Messages = messages

		reply, err := streamChatResponse(messages, model)
		if err != nil {
			handleError(err, "getting AI response")
			messages = messages[:len(messages)-1]
			chatFile.Messages = messages
			if strings.Contains(err.Error(), "API returned status 400") {
				return
			}
			continue
		}

		messages = append(messages, Message{Role: "assistant", Content: reply, MessageNumber: len(messages)})
		chatFile.Messages = messages

		// Auto-save without regenerating summary
		data, err := json.MarshalIndent(chatFile, "", "  ")
		if err != nil {
			handleError(err, "auto-saving chat")
		} else {
			if err := os.WriteFile(filepath.Join(chatsPath(), chatName+".json"), data, 0644); err != nil {
				handleError(err, "auto-saving chat")
			}
		}
	}
}

// toggleChatFavorite toggles the favorite status of a chat
func toggleChatFavorite(chatName string) error {
	chatFile, err := loadChatWithMetadata(chatName)
	if err != nil {
		return fmt.Errorf("failed to load chat '%s': %w", chatName, err)
	}

	chatFile.Metadata.Favorite = !chatFile.Metadata.Favorite

	data, err := json.MarshalIndent(chatFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal chat '%s': %w", chatName, err)
	}

	if err := os.WriteFile(filepath.Join(chatsPath(), chatName+".json"), data, 0644); err != nil {
		return fmt.Errorf("failed to save chat '%s': %w", chatName, err)
	}

	status := "favorited"
	if !chatFile.Metadata.Favorite {
		status = "unfavorited"
	}
	fmt.Printf("Chat '%s' %s.\n", chatName, status)
	return nil
}

// readMultiLineInput reads input from the user, supporting Shift+Enter for new lines
func readMultiLineInput(reader *bufio.Reader) string {
	var lines []string
	fmt.Print("\033[31mYou:\033[0m ")

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		// Remove the newline character
		line = strings.TrimSuffix(line, "\n")

		// Check if this line ends with a backslash (Shift+Enter equivalent)
		if strings.HasSuffix(line, "\\") {
			// Remove the backslash and add the line (without newline)
			line = strings.TrimSuffix(line, "\\")
			lines = append(lines, line)
			fmt.Print("  ") // Indent for continuation
			continue
		}

		// Add the final line and break
		lines = append(lines, line)
		break
	}

	// Join all lines with actual newlines
	result := strings.Join(lines, "\n")

	// Show hint about Shift+Enter on first use (you can remove this after users get familiar)
	if len(lines) > 1 {
		fmt.Println("\033[36m(Tip: Use \\ at the end of a line for multi-line input)\033[0m")
	}

	return result
}

// setChatTitle sets the title for a chat
func setChatTitle(chatName string, title string) error {
	chatFile, err := loadChatWithMetadata(chatName)
	if err != nil {
		return fmt.Errorf("failed to load chat: %w", err)
	}

	chatFile.Metadata.Title = title

	data, err := json.MarshalIndent(chatFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal chat: %w", err)
	}

	err = os.WriteFile(filepath.Join(chatsPath(), chatName+".json"), data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write chat file: %w", err)
	}

	return nil
}

// promptModelSelection prompts the user to select a model from the list, defaulting to defaultModel if input is empty or invalid.
func promptModelSelection(reader *bufio.Reader, models []string, defaultModel string) (string, error) {
	fmt.Println("\nSelect model for this chat:")
	for i, model := range models {
		mark := " "
		if model == defaultModel {
			mark = "*"
		}
		fmt.Printf("%d) %s %s\n", i+1, model, mark)
	}
	fmt.Printf("Enter model number (or press Enter for default '%s'): ", defaultModel)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultModel, nil
	}

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(models) {
		fmt.Println("Invalid input; using default model.")
		return defaultModel, nil
	}

	return models[choice-1], nil
}

// Helper to filter out system messages
func filterNonSystemMessages(messages []Message) []Message {
	var filtered []Message
	for _, msg := range messages {
		if msg.Role != "system" {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// MenuEntry represents a single menu item and its associated callback.
type MenuEntry struct {
	Label    string
	OnSelect func()
}

// MenuModalModel is a Bubble Tea model for displaying a menu modal.
type MenuModalModel struct {
	title    string
	entries  []MenuEntry
	index    int
	quitting bool
}

func (m MenuModalModel) Init() tea.Cmd { return nil }

func (m MenuModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if m.index > 0 {
				m.index--
			} else {
				m.index = len(m.entries) - 1
			}
		case "down":
			if m.index < len(m.entries)-1 {
				m.index++
			} else {
				m.index = 0
			}
		case "enter":
			m.entries[m.index].OnSelect()
			m.quitting = true
			return m, tea.Quit
		case "esc":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m MenuModalModel) View() string {
	var out string
	out += lipgloss.NewStyle().Bold(true).Render(m.title) + "\n\n"
	for i, entry := range m.entries {
		style := lipgloss.NewStyle()
		if i == m.index {
			style = style.Bold(true).Foreground(lipgloss.Color("33")).Background(lipgloss.Color("236"))
		}
		out += style.Render(fmt.Sprintf("  %s", entry.Label)) + "\n"
	}
	out += "\n[Up/Down] Navigate  [Enter] Select  [Esc] Cancel"
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).Render(out)
}

// CreateMenuModal displays a menu modal with the given entries and calls the associated function on selection.
func CreateMenuModal(entries []MenuEntry, title string) {
	model := MenuModalModel{title: title, entries: entries, index: 0}
	p := tea.NewProgram(model)
	_ = p.Start()
}

// ShowMainMenu displays the main menu and handles user selection.
func ShowMainMenu() {
	entries := []MenuEntry{
		{Label: "Start New Chat", OnSelect: func() { /* Start new chat logic */ }},
		{Label: "List Chats", OnSelect: func() { /* List chats logic */ }},
		{Label: "Settings", OnSelect: func() { /* Settings logic */ }},
		{Label: "Exit", OnSelect: func() { /* Exit logic */ }},
	}
	CreateMenuModal(entries, "Main Menu")
}
