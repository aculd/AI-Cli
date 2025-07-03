# Go AI CLI

A powerful, feature-rich AI chat application built in Go with a beautiful terminal GUI using Bubble Tea and Lipgloss.

## Features

### 🚀 Core Features
- **Beautiful Terminal GUI** - Modern, responsive interface with scrolling, keyboard navigation, and visual indicators
- **Multiple AI Models** - Support for various AI models including GPT-4, Claude, and others
- **API Key Management** - Secure storage and management of multiple API keys
- **Custom Prompts** - Create and manage custom system prompts for different use cases
- **Chat Management** - Save, load, favorite, and organize your conversations
- **Real-time Streaming** - See AI responses as they're generated with a spinner indicator
- **Stop Functionality** - Cancel ongoing requests with Ctrl+S

### 🎯 Advanced Features
- **Custom Chat Creation** - Specify API key, model, and prompt before starting a chat
- **Scroll Controls** - Navigate through long conversations with Page Up/Down, Home/End, and arrow keys
- **Auto-scroll** - Automatically scroll to new messages
- **Vim-style Commands** - Use `:g` to generate titles, `:f` to favorite chats, `:q` to quit
- **Clipboard Integration** - Add API keys and prompts directly from clipboard
- **Error Handling** - Comprehensive error handling with user-friendly messages

## Installation

### Prerequisites
- Go 1.22 or later
- Windows, macOS, or Linux

### Build from Source
```bash
git clone https://github.com/aculd/go-ai-cli.git
cd go-ai-cli
go build -o aichat.exe .
```

### Download Pre-built Binary
Download the latest release from the [Releases page](https://github.com/aculd/go-ai-cli/releases).

## Quick Start

1. **Run the application:**
   ```bash
   ./aichat.exe
   ```

2. **Add your first API key:**
   - Navigate to "API Keys" → "Add API Key"
   - Enter a name for your key
   - Paste your API key (or press Enter to read from clipboard)

3. **Start chatting:**
   - Choose "New Chat" for a quick start
   - Or use "Custom Chat" to specify model and prompt

## Configuration

The application automatically creates the following directory structure on first run:

```
.util/
├── api_keys.json      # API key storage
├── models.json        # AI model configurations
├── prompts.json       # Custom prompts
└── chats/            # Saved chat conversations
```

### API Keys
- Store multiple API keys with descriptive names
- Set an active key for current sessions
- Secure storage with proper file permissions

### Models
- Pre-configured with popular AI models
- Add custom models as needed
- Set default model for new chats

### Prompts
- Create custom system prompts
- Set default prompt for new chats
- Organize prompts by use case

## Usage

### Main Menu Navigation
- Use arrow keys (↑↓) to navigate
- Press Enter to select
- Press ESC to go back
- Press 'q' to quit

### Chat Interface
- **Type your message** and press Enter to send
- **Ctrl+S** to stop/cancel ongoing requests
- **Ctrl+C** to quit the application
- **Page Up/Down** to scroll through messages
- **Home/End** to jump to top/bottom
- **Arrow keys** to scroll when not typing

### Vim-style Commands
- `:g` - Generate chat title from last user message
- `:f` - Toggle favorite status
- `:q` - Save and quit

### Custom Chat Creation
1. Select "Custom Chat" from the Chats menu
2. Choose your API key
3. Select the AI model
4. Pick a system prompt
5. Start chatting with your custom configuration

## API Support

The application supports various AI providers through OpenRouter:
- OpenAI (GPT-4, GPT-3.5)
- Anthropic (Claude)
- Meta (Llama)
- And many more

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Development Setup
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling
- [OpenRouter](https://openrouter.ai/) for AI model access

## Support

If you encounter any issues or have questions:
1. Check the [Issues](https://github.com/aculd/go-ai-cli/issues) page
2. Create a new issue with detailed information
3. Include your operating system and Go version

---

**Note:** This application stores sensitive information (API keys) locally. Ensure your system is secure and never share your `.util` directory. 