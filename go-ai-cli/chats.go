package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"
)

// listChats lists all saved chat filenames without extension
func listChats() ([]string, error) {
    files, err := os.ReadDir(chatsPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read chat directory: %w", err)
    }
    chats := []string{}
    for _, f := range files {
        if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
            name := strings.TrimSuffix(f.Name(), ".json")
            chats = append(chats, name)
        }
    }
    return chats, nil
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
    chatFile := ChatFile{
        Messages: messages,
    }
    
    // Generate and store summary if there are messages
    if len(messages) > 0 {
        summary := generateChatSummary(messages, DefaultModel())
        chatFile.Metadata.Summary = summary
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

        fmt.Printf("%d) %s\n", i+1, c)
        summary := chatFile.Metadata.Summary
        if summary == "" && len(chatFile.Messages) > 0 {
            // Generate summary if not stored
            summary = generateChatSummary(chatFile.Messages, DefaultModel())
            // Save the generated summary
            chatFile.Metadata.Summary = summary
            if err := saveChat(c, chatFile.Messages); err != nil {
                fmt.Printf("   (Failed to save summary: %v)\n", err)
            }
        }
        if summary == "" {
            summary = "Empty chat."
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
        fmt.Printf("%d) %s\n", i+1, c)
    }
    fmt.Print("Enter chat number to load: ")
    input, _ := reader.ReadString('\n')
    input = strings.TrimSpace(input)

    idx, err := strconv.Atoi(input)
    if err != nil || idx < 1 || idx > len(chats) {
        return fmt.Errorf("invalid chat number")
    }

    chatName := chats[idx-1]
    messages, err := loadChat(chatName)
    if err != nil {
        return fmt.Errorf("failed to load chat '%s': %w", chatName, err)
    }

    model, err := promptModelAtChatStart(reader)
    if err != nil {
        fmt.Println("Error loading models, using default.")
        model = DefaultModel()
    }
    fmt.Printf("Using model: %s\n", model)

    runChat(chatName, messages, reader, model)
    return nil
}

// generateTimestampChatName generates a timestamp-based chat name in ddmmyyhhss format
func generateTimestampChatName() string {
    now := time.Now()
    return now.Format("020106150405") // ddmmyyhhss
}

// quickChatFlow creates a new chat using default model and prompt
func quickChatFlow(reader *bufio.Reader) error {
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
        return fmt.Errorf("failed to check existing chats: %w", err)
    }
    for _, c := range chats {
        if c == chatName {
            return fmt.Errorf("chat '%s' already exists", chatName)
        }
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

    // Save the new chat
    if err := saveChat(chatName, messages); err != nil {
        return fmt.Errorf("failed to save new chat '%s': %w", chatName, err)
    }

    fmt.Printf("Starting quick chat with default model '%s' and prompt '%s'...\n", 
        defaultModel, defaultPrompt.Name)

    runChat(chatName, messages, reader, defaultModel)
    return nil
}

// customChatFlow creates a new chat with user-selected model and prompt
func customChatFlow(reader *bufio.Reader) error {
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
        return fmt.Errorf("failed to check existing chats: %w", err)
    }
    for _, c := range chats {
        if c == chatName {
            return fmt.Errorf("chat '%s' already exists", chatName)
        }
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

    // Save the new chat
    if err := saveChat(chatName, messages); err != nil {
        return fmt.Errorf("failed to save new chat '%s': %w", chatName, err)
    }

    fmt.Printf("Starting custom chat with model '%s' and prompt '%s'...\n", 
        model, promptName)

    runChat(chatName, messages, reader, model)
    return nil
}

// chatsMenu handles chat-related submenu options
func chatsMenu(reader *bufio.Reader) {
    for {
        fmt.Println("\nChats Menu:")
        fmt.Println("1) List chats")
        fmt.Println("2) Load chat")
        fmt.Println("3) Quick chat (use defaults)")
        fmt.Println("4) Custom chat")
        fmt.Println("5) Back to main menu")
        fmt.Print("Choose option: ")
        
        input, _ := reader.ReadString('\n')
        input = strings.TrimSpace(input)

        switch input {
        case "1":
            if err := listChatsAndSummarize(); err != nil {
                fmt.Println("Error:", err)
            }
        case "2":
            if err := loadAndContinueChat(reader); err != nil {
                fmt.Println("Error:", err)
            }
            return // Return to main menu after chat ends
        case "3":
            if err := quickChatFlow(reader); err != nil {
                fmt.Println("Error:", err)
            }
            return // Return to main menu after chat ends
        case "4":
            if err := customChatFlow(reader); err != nil {
                fmt.Println("Error:", err)
            }
            return // Return to main menu after chat ends
        case "5", "":
            return
        default:
            fmt.Println("Invalid chat menu option.")
        }
    }
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

	// Call your streaming chat response function
	summary, err := streamChatResponse(summaryMessages, model)
	if err != nil {
		return fmt.Sprintf("Chat with %d messages. (Summary unavailable: %v)", len(messages), err)
	}
	return summary
}