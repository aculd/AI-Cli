package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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
	files, err := os.ReadDir(chatsPath)
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

// listFavoriteChats lists all favorite chats
func listFavoriteChats() ([]string, error) {
	files, err := os.ReadDir(chatsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read chat directory: %w", err)
	}

	var favoriteChats []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
			name := strings.TrimSuffix(f.Name(), ".json")
			chatFile, err := loadChatWithMetadata(name)
			if err == nil && chatFile.Metadata.Favorite {
				favoriteChats = append(favoriteChats, name)
			}
		}
	}

	// Sort by creation date (newest first)
	sort.Slice(favoriteChats, func(i, j int) bool {
		chatI, _ := loadChatWithMetadata(favoriteChats[i])
		chatJ, _ := loadChatWithMetadata(favoriteChats[j])
		if chatI == nil || chatJ == nil {
			return false
		}
		return chatI.Metadata.CreatedAt.After(chatJ.Metadata.CreatedAt)
	})

	return favoriteChats, nil
}

// loadChat loads chat file with messages and metadata
func loadChat(name string) ([]Message, error) {
	data, err := os.ReadFile(filepath.Join(chatsPath, name+".json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read chat file '%s': %w", name, err)
	}
	var chatFile ChatFile
	if err := json.Unmarshal(data, &chatFile); err != nil {
		// Try loading legacy format (just messages array)
		var messages []Message
		if err2 := json.Unmarshal(data, &messages); err2 != nil {
			return nil, fmt.Errorf("failed to unmarshal chat file '%s': %w", name, err)
		}
		return messages, nil
	}
	return chatFile.Messages, nil
}

// loadChatWithMetadata loads the complete chat file including metadata
func loadChatWithMetadata(name string) (*ChatFile, error) {
	data, err := os.ReadFile(filepath.Join(chatsPath, name+".json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read chat file '%s': %w", name, err)
	}
	var chatFile ChatFile
	if err := json.Unmarshal(data, &chatFile); err != nil {
		// Try loading legacy format (just messages array)
		var messages []Message
		if err2 := json.Unmarshal(data, &messages); err2 != nil {
			return nil, fmt.Errorf("failed to unmarshal chat file '%s': %w", name, err)
		}
		chatFile.Messages = messages
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
	err = os.WriteFile(filepath.Join(chatsPath, name+".json"), data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write chat file '%s': %w", name, err)
	}
	return nil
}

// listChatsAndSummarize lists chats and prints their stored summaries
func listChatsAndSummarize() error {
	chats, err := listChats()
	if err != nil {
		return err
	}
	if len(chats) == 0 {
		fmt.Println("No saved chats.")
		return nil
	}

	for i, c := range chats {
		chatFile, err := loadChatWithMetadata(c)
		if err != nil {
			fmt.Printf("%d) %s\n   (Failed to load chat: %v)\n\n", i+1, c, err)
			continue
		}

		favoriteMark := " "
		if chatFile.Metadata.Favorite {
			favoriteMark = "★"
		}

		// Show title if available, otherwise show timestamp
		displayName := c
		if chatFile.Metadata.Title != "" {
			displayName = chatFile.Metadata.Title
		}

		fmt.Printf("%d) %s %s\n", i+1, displayName, favoriteMark)

		// Show timestamp and summary
		if chatFile.Metadata.Title != "" {
			fmt.Printf("   ID: %s\n", c)
		}

		summary := chatFile.Metadata.Summary
		if summary == "" {
			summary = "No summary available."
		}
		fmt.Printf("   Summary: %s\n\n", summary)
	}
	return nil
}

// loadAndContinueChat loads a chat by user choice and continues it
func loadAndContinueChat(reader *bufio.Reader) error {
	chats, err := listChats()
	if err != nil {
		return err
	}
	if len(chats) == 0 {
		fmt.Println("No saved chats.")
		return nil
	}

	fmt.Println("Available chats:")
	for i, c := range chats {
		chatFile, err := loadChatWithMetadata(c)
		favoriteMark := " "
		if err == nil && chatFile.Metadata.Favorite {
			favoriteMark = "★"
		}
		fmt.Printf("%d) %s %s\n", i+1, c, favoriteMark)
	}
	fmt.Print("Enter chat number to load (or 'f' + number to toggle favorite): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	// Check if user wants to toggle favorite
	if strings.HasPrefix(input, "f") {
		favInput := strings.TrimSpace(strings.TrimPrefix(input, "f"))
		idx, err := strconv.Atoi(favInput)
		if err != nil || idx < 1 || idx > len(chats) {
			return fmt.Errorf("invalid chat number")
		}
		chatName := chats[idx-1]
		return toggleChatFavorite(chatName)
	}

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(chats) {
		return fmt.Errorf("invalid chat number")
	}

	chatName := chats[idx-1]
	chatFile, err := loadChatWithMetadata(chatName)
	if err != nil {
		return fmt.Errorf("failed to load chat '%s': %w", chatName, err)
	}
	model := chatFile.Metadata.Model
	if model == "" {
		model = DefaultModel()
	}
	fmt.Printf("Using model: %s\n", model)

	// Print last message if any (not system prompt)
	if len(chatFile.Messages) > 1 {
		lastMsg := chatFile.Messages[len(chatFile.Messages)-1]
		fmt.Printf("\nLast message (%s): %s\n\n", strings.Title(lastMsg.Role), lastMsg.Content)
	}

	runChat(chatName, chatFile.Messages, reader, model)
	return nil
}

// generateTimestampChatName generates a timestamp-based chat name in ddmmyyhhss format
func generateTimestampChatName() string {
	now := time.Now()
	return now.Format("2006-01-02_15-04-05") // YYYY-MM-DD_HH-MM-SS
}

// quickChatFlow creates a new chat using default model and prompt
func quickChatFlow(reader *bufio.Reader) error {
	chatName, err := setupNewChat(reader)
	err = os.Setenv("OPENAI_API_KEY", "sk-")
	if err != nil {
		return fmt.Errorf("failed to set API key: %w", err)
	}

	// Get default model
	_, defaultModel, err := loadModelsWithMostRecent()
	if err != nil {
		fmt.Println("Error loading models, using fallback default.")
		defaultModel = DefaultModel()
	}

	// Get default prompt
	defaultPrompt, err := getDefaultPrompt()
	if err != nil {
		return fmt.Errorf("failed to get default prompt: %w", err)
	}

	// Create initial message slice with system role
	messages := []Message{
		{Role: "system", Content: defaultPrompt.Content},
	}

	// Save the new chat with model in metadata
	var chatFile ChatFile
	chatFile.Messages = messages
	chatFile.Metadata.Model = defaultModel
	chatFile.Metadata.CreatedAt = time.Now()
	data, err := json.MarshalIndent(chatFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal chat: %w", err)
	}
	err = os.WriteFile(filepath.Join(chatsPath, chatName+".json"), data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write chat file '%s': %w", chatName, err)
	}

	fmt.Printf("Starting quick chat with default model '%s' and prompt '%s'...\n\n",
		defaultModel, defaultPrompt.Name)

	runChat(chatName, messages, reader, defaultModel)
	return nil
}

// customChatFlow creates a new chat with user-selected model and prompt
func customChatFlow(reader *bufio.Reader) error {
	chatName, err := setupNewChat(reader)
	if err != nil {
		return err
	}

	// Let user select model
	model, err := promptModelAtChatStart(reader)
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
	err = os.WriteFile(filepath.Join(chatsPath, chatName+".json"), data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write chat file '%s': %w", chatName, err)
	}

	fmt.Printf("Starting custom chat with model '%s' and prompt '%s'...\n\n",
		model, promptName)

	runChat(chatName, messages, reader, model)
	return nil
}

// chatsMenu handles chat-related submenu options
func chatsMenu(reader *bufio.Reader) error {
	menu := Menu{
		Title: "Chats Menu",
		Items: []MenuItem{
			{Label: "List chats", Handler: func(r *bufio.Reader) error {
				return listChatsAndSummarize()
			}},
			{Label: "Load chat", Handler: loadAndContinueChat},
			{Label: "Quick chat (use defaults)", Handler: quickChatFlow},
			{Label: "Custom chat", Handler: customChatFlow},
			{Label: "Back to main menu", ExitItem: true},
		},
	}
	RunMenu(menu, reader)
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
						oldPath := filepath.Join(chatsPath, chatName+".json")
						newPath := filepath.Join(chatsPath, finalName+".json")
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

// handleChatError wraps error handling for chat operations
func handleChatError(err error, operation string) {
	if err != nil {
		fmt.Printf("\033[31mError during %s: %v\033[0m\n", operation, err)
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

		messages = append(messages, Message{Role: "user", Content: userInput})
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

		messages = append(messages, Message{Role: "assistant", Content: reply})
		chatFile.Messages = messages

		// Auto-save without regenerating summary
		data, err := json.MarshalIndent(chatFile, "", "  ")
		if err != nil {
			handleError(err, "auto-saving chat")
		} else {
			if err := os.WriteFile(filepath.Join(chatsPath, chatName+".json"), data, 0644); err != nil {
				handleError(err, "auto-saving chat")
			}
		}
	}
}

// generateSummariesForActiveChats generates summary only for the active chat
func generateSummariesForActiveChats() error {
	if activeChatName == "" {
		// No active chat, nothing to do
		return nil
	}

	chatFile, err := loadChatWithMetadata(activeChatName)
	if err != nil {
		return fmt.Errorf("failed to load active chat '%s': %w", activeChatName, err)
	}

	// Generate summary if not present and chat has messages
	if chatFile.Metadata.Summary == "" && len(chatFile.Messages) > 1 {
		fmt.Printf("Generating summary for active chat '%s'...\n", activeChatName)
		model := chatFile.Metadata.Model
		if model == "" {
			model = DefaultModel()
		}
		summary := generateChatSummary(chatFile.Messages, model)
		chatFile.Metadata.Summary = summary

		// Save the updated chat file
		data, err := json.MarshalIndent(chatFile, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal active chat '%s': %w", activeChatName, err)
		}
		if err := os.WriteFile(filepath.Join(chatsPath, activeChatName+".json"), data, 0644); err != nil {
			return fmt.Errorf("failed to save active chat '%s': %w", activeChatName, err)
		}
		fmt.Printf("Summary generated and saved for active chat '%s'\n", activeChatName)
	}
	return nil
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

	if err := os.WriteFile(filepath.Join(chatsPath, chatName+".json"), data, 0644); err != nil {
		return fmt.Errorf("failed to save chat '%s': %w", chatName, err)
	}

	status := "favorited"
	if !chatFile.Metadata.Favorite {
		status = "unfavorited"
	}
	fmt.Printf("Chat '%s' %s.\n", chatName, status)
	return nil
}

// loadFavoriteChat loads a favorite chat by user choice
func loadFavoriteChat(reader *bufio.Reader) error {
	favoriteChats, err := listFavoriteChats()
	if err != nil {
		return err
	}
	if len(favoriteChats) == 0 {
		fmt.Println("No favorite chats found.")
		return nil
	}

	fmt.Println("Favorite chats:")
	for i, c := range favoriteChats {
		fmt.Printf("%d) %s\n", i+1, c)
	}
	fmt.Print("Enter chat number to load: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(favoriteChats) {
		return fmt.Errorf("invalid chat number")
	}

	chatName := favoriteChats[idx-1]
	chatFile, err := loadChatWithMetadata(chatName)
	if err != nil {
		return fmt.Errorf("failed to load chat '%s': %w", chatName, err)
	}
	model := chatFile.Metadata.Model
	if model == "" {
		model = DefaultModel()
	}
	fmt.Printf("Using model: %s\n", model)

	// Print last message if any (not system prompt)
	if len(chatFile.Messages) > 1 {
		lastMsg := chatFile.Messages[len(chatFile.Messages)-1]
		fmt.Printf("\nLast message (%s): %s\n\n", strings.Title(lastMsg.Role), lastMsg.Content)
	}

	runChat(chatName, chatFile.Messages, reader, model)
	return nil
}

// favoritesMenu handles favorite chat-related submenu options
func favoritesMenu(reader *bufio.Reader) error {
	menu := Menu{
		Title: "Favorites Menu",
		Items: []MenuItem{
			{Label: "List favorite chats", Handler: func(r *bufio.Reader) error {
				favoriteChats, err := listFavoriteChats()
				if err != nil {
					return err
				}
				if len(favoriteChats) == 0 {
					fmt.Println("No favorite chats found.")
					return nil
				}

				fmt.Println("Favorite chats:")
				for i, c := range favoriteChats {
					chatFile, err := loadChatWithMetadata(c)
					if err != nil {
						fmt.Printf("%d) %s (Failed to load: %v)\n", i+1, c, err)
						continue
					}
					summary := chatFile.Metadata.Summary
					if summary == "" {
						summary = "No summary available."
					}
					fmt.Printf("%d) %s\n   Summary: %s\n\n", i+1, c, summary)
				}
				return nil
			}},
			{Label: "Load favorite chat", Handler: loadFavoriteChat},
			{Label: "Load favorite chat in GUI", Handler: func(r *bufio.Reader) error {
				favoriteChats, err := listFavoriteChats()
				if err != nil {
					return err
				}
				if len(favoriteChats) == 0 {
					fmt.Println("No favorite chats found.")
					return nil
				}

				fmt.Println("Favorite chats:")
				for i, c := range favoriteChats {
					fmt.Printf("%d) %s\n", i+1, c)
				}
				fmt.Print("Enter chat number to load in GUI: ")
				input, _ := r.ReadString('\n')
				input = strings.TrimSpace(input)

				idx, err := strconv.Atoi(input)
				if err != nil || idx < 1 || idx > len(favoriteChats) {
					return fmt.Errorf("invalid chat number")
				}

				chatName := favoriteChats[idx-1]
				chatFile, err := loadChatWithMetadata(chatName)
				if err != nil {
					return fmt.Errorf("failed to load chat '%s': %w", chatName, err)
				}
				model := chatFile.Metadata.Model
				if model == "" {
					model = DefaultModel()
				}
				fmt.Printf("Loading favorite chat '%s' with model '%s' in GUI...\n", chatName, model)

				runChatGUI(chatName, chatFile.Messages, r, model)
				return nil
			}},
			{Label: "Add favorite", Handler: func(r *bufio.Reader) error {
				allChats, err := listChats()
				if err != nil {
					return err
				}
				var nonFavChats []string
				for _, c := range allChats {
					chatFile, err := loadChatWithMetadata(c)
					if err == nil && !chatFile.Metadata.Favorite {
						nonFavChats = append(nonFavChats, c)
					}
				}
				if len(nonFavChats) == 0 {
					fmt.Println("No non-favorite chats available to add.")
					return nil
				}
				fmt.Println("Non-favorite chats:")
				for i, c := range nonFavChats {
					fmt.Printf("%d) %s\n", i+1, c)
				}
				fmt.Print("Enter number to add to favorites, or 0 to return: ")
				input, _ := r.ReadString('\n')
				input = strings.TrimSpace(input)
				idx, err := strconv.Atoi(input)
				if err != nil || idx < 0 || idx > len(nonFavChats) {
					fmt.Println("Invalid selection.")
					return nil
				}
				if idx == 0 {
					return nil
				}
				chatName := nonFavChats[idx-1]
				chatFile, err := loadChatWithMetadata(chatName)
				if err != nil {
					fmt.Printf("Failed to load chat '%s': %v\n", chatName, err)
					return nil
				}
				chatFile.Metadata.Favorite = true
				data, err := json.MarshalIndent(chatFile, "", "  ")
				if err != nil {
					fmt.Printf("Failed to marshal chat '%s': %v\n", chatName, err)
					return nil
				}
				if err := os.WriteFile(filepath.Join(chatsPath, chatName+".json"), data, 0644); err != nil {
					fmt.Printf("Failed to save chat '%s': %v\n", chatName, err)
					return nil
				}
				fmt.Printf("Chat '%s' added to favorites.\n", chatName)
				return nil
			}},
			{Label: "Back to main menu", ExitItem: true},
		},
	}
	RunMenu(menu, reader)
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

// guiChatMenu handles GUI chat-related submenu options
func guiChatMenu(reader *bufio.Reader) error {
	menu := Menu{
		Title: "GUI Chat Menu",
		Items: []MenuItem{
			{Label: "Load chat in GUI", Handler: func(r *bufio.Reader) error {
				chats, err := listChats()
				if err != nil {
					return err
				}
				if len(chats) == 0 {
					fmt.Println("No saved chats.")
					return nil
				}

				fmt.Println("Available chats:")
				for i, c := range chats {
					chatFile, err := loadChatWithMetadata(c)
					favoriteMark := " "
					if err == nil && chatFile.Metadata.Favorite {
						favoriteMark = "★"
					}
					fmt.Printf("%d) %s %s\n", i+1, c, favoriteMark)
				}
				fmt.Print("Enter chat number to load in GUI: ")
				input, _ := r.ReadString('\n')
				input = strings.TrimSpace(input)

				idx, err := strconv.Atoi(input)
				if err != nil || idx < 1 || idx > len(chats) {
					return fmt.Errorf("invalid chat number")
				}

				chatName := chats[idx-1]
				chatFile, err := loadChatWithMetadata(chatName)
				if err != nil {
					return fmt.Errorf("failed to load chat '%s': %w", chatName, err)
				}
				model := chatFile.Metadata.Model
				if model == "" {
					model = DefaultModel()
				}
				fmt.Printf("Loading chat '%s' with model '%s' in GUI...\n", chatName, model)

				runChatGUI(chatName, chatFile.Messages, r, model)
				return nil
			}},
			{Label: "New GUI chat", Handler: func(r *bufio.Reader) error {
				chatName, err := setupNewChat(r)
				if err != nil {
					return err
				}

				// Let user select model
				model, err := promptModelAtChatStart(r)
				if err != nil {
					return fmt.Errorf("failed to select model: %w", err)
				}

				// Let user select prompt
				promptName, promptContent, err := promptPromptSelection(r)
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
				err = os.WriteFile(filepath.Join(chatsPath, chatName+".json"), data, 0644)
				if err != nil {
					return fmt.Errorf("failed to write chat file '%s': %w", chatName, err)
				}

				fmt.Printf("Starting new GUI chat with model '%s' and prompt '%s'...\n",
					model, promptName)

				runChatGUI(chatName, messages, r, model)
				return nil
			}},
			{Label: "Back to main menu", ExitItem: true},
		},
	}
	RunMenu(menu, reader)
	return nil
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

	err = os.WriteFile(filepath.Join(chatsPath, chatName+".json"), data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write chat file: %w", err)
	}

	return nil
}

// getChatTitle gets the title for a chat, returns empty string if no title
func getChatTitle(chatName string) string {
	chatFile, err := loadChatWithMetadata(chatName)
	if err != nil {
		return ""
	}
	return chatFile.Metadata.Title
}

// generateChatTitle generates a title for the chat using AI
func generateChatTitle(messages []Message, model string) string {
	if len(messages) == 0 {
		return "Empty chat"
	}

	// Create prompt for title generation
	titlePrompt := Message{
		Role:    "user",
		Content: "Please provide an accurate title for this chat so that it can be easily recognized from a list of archived chats. Keep it concise (under 50 characters) and descriptive.",
	}
	titleMessages := append(messages, titlePrompt)

	// Temporarily redirect stdout to /dev/null during title generation
	savedStdout := os.Stdout
	os.Stdout = nil

	title, err := streamChatResponse(titleMessages, model)

	// Restore stdout
	os.Stdout = savedStdout

	if err != nil {
		return fmt.Sprintf("Chat with %d messages", len(messages))
	}

	// Clean up the title (remove quotes, extra whitespace, etc.)
	title = strings.TrimSpace(title)
	title = strings.Trim(title, `"'`)

	return title
}
