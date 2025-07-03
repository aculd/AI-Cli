# Go AI CLI 🚀

A beautiful, fast, and powerful AI chat app for your terminal, built in Go with Bubble Tea & Lipgloss.

---

## ✨ Features
- 🖥️ Modern terminal GUI (scroll, wrap, color)
- 🤖 Multiple AI models (GPT-4, Claude, Llama, etc)
- 🔑 API key manager (add, set active, clipboard)
- 📝 Custom prompts & models
- 💬 Save, load, favorite chats
- ⏳ Real-time streaming & spinner
- 🛑 Stop/cancel requests (Ctrl+S)
- 🛠️ Vim-style commands (`:g`, `:f`, `:q`)
- 📋 Clipboard integration
- 🧭 Robust error handling

---

## ⚡ Quick Start

```sh
git clone https://github.com/aculd/go-ai-cli.git
cd go-ai-cli
go build -o aichat.exe .
./aichat.exe
```

Or grab a binary from [Releases](https://github.com/aculd/go-ai-cli/releases).

---

## 🕹️ Controls Cheat Sheet

| Menu/General         | Chat Window           | Vim-style      |
|---------------------|----------------------|---------------|
| ↑↓      Navigate    | Enter   Send message | :g  AI Title  |
| Enter   Select      | Ctrl+S Stop request  | :t "Title"    |
| Esc     Back        | Ctrl+C Quit          | :f  Favorite  |
| Ctrl+C  Quit        | ↑↓      Scroll msgs  | :q  Quit      |
|                     | PgUp/Dn Scroll page  |               |
|                     | Home/End Top/Bottom  |               |
|                     |                      |               |

---

## 🗂️ Configuration

On first run, creates:
```
.util/
├── api_keys.json   # API keys
├── models.json     # Model configs
├── prompts.json    # Prompts
└── chats/          # Saved chats
```

---

## 🛠️ Usage
- **Add API Key:** Menu → API Keys → Add (paste or clipboard)
- **Start Chat:** Menu → New Chat or Custom Chat
- **Custom Chat:** Pick API key, model, prompt
- **Scroll:** Use ↑↓ PgUp/Dn Home/End
- **Stop:** Ctrl+S during response
- **Vim:** `:g` AI title, `:t "title"` set title, `:f` favorite, `:q` quit
- **Chat Names:** Auto-timestamped (YYYY-MM-DD_HH-MM-SS) or custom titles

---

## 🌐 Supported Models
- OpenAI (GPT-4, GPT-3.5)
- Anthropic (Claude)
- Meta (Llama)
- ...via [OpenRouter](https://openrouter.ai/)

---

## 🤝 Contributing
- Fork, branch, PRs welcome!
- See [Issues](https://github.com/aculd/go-ai-cli/issues) for help

## 📜 License
MIT — see [LICENSE](LICENSE)

---

> ⚠️ **API keys are stored locally in `.util/`. Keep your system secure!**
