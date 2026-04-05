# lazychat 🦥

A fast, terminal-based user interface (TUI) for interacting with LLMs (Groq, Gemini). Built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- **Multi-Provider Support**: Switch between Groq, Gemini API, and the `gemini` CLI.
- **Dynamic Skills**: Configure custom prompt templates for specific tasks (translation, code review, summarization).
- **Keyboard-First**: Optimized for speed with Vim-like navigation and global shortcuts.
- **Session Persistence**: All conversations are automatically saved to your local data directory.
- **Streaming**: Real-time response streaming for a smooth chat experience.

## Installation

```bash
# Clone the repository
git clone https://github.com/user/lazychat
cd lazychat

# Build the binary
go build -o lazychat .

# Move to your path (optional)
mv lazychat /usr/local/bin/
```

## Configuration

### API Keys
Set your API keys as environment variables:
```bash
export GROQ_API_KEY="your-key-here"
export GEMINI_API_KEY="your-key-here"
```

### Skills (`config.json`)
You can define custom "skills" that appear at the top of your conversation list. Create a configuration file at `~/.local/share/lazychat/config.json`:

```json
{
  "skills": [
    {
      "mode": "translate-only",
      "title": "[to english]",
      "prompt": "Translate the following to English and make the result all lower case, but only keep the 'I' uppercase (if any):\n\n{{.Input}}"
    },
    {
      "mode": "review",
      "title": "[code review]",
      "prompt": "Act as a Senior Engineer. Review this code for performance and safety:\n\n{{.Input}}"
    }
  ]
}
```

## Key Bindings

| Key | Action |
|-----|--------|
| `tab` | Toggle focus between Sidebar and Chat |
| `ctrl+n` | Start a new chat (Global) |
| `ctrl+k` | Open Skill Picker menu (Global) |
| `m` | Open Model Picker menu |
| `k` / `j` | Vim-style up/down navigation |
| `enter` | Select conversation / Send message |
| `d` | Delete conversation (Sidebar) or Message (Chat) |
| `r` | Refresh rate limits (Usage view) |
| `esc` | Close menus / Cancel delete |
| `ctrl+c` | Quit |

## Data Directory

- **Linux**: `~/.local/share/lazychat`
- **macOS/Windows**: Standard local data paths used by Go's `os.UserConfigDir`.

---
*Built with ❤️ for the terminal.*
