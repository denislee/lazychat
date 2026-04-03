# lazychat — Design Document

## Overview

A lightweight Go TUI chat client for Groq's LLM API. Multi-conversation support with a split-panel layout, streaming responses, and JSON-based persistence.

## Decisions

- **LLM backend:** Groq (free tier, `llama-3.3-70b-versatile`)
- **API key:** Environment variable `GROQ_API_KEY` only
- **TUI framework:** Bubble Tea + Lipgloss + Bubbles (Charmbracelet)
- **Storage:** JSON files at `~/.local/share/lazychat/<id>.json`
- **No external SDK:** Call Groq's OpenAI-compatible REST API directly with `net/http`

## Layout

```
┌─────────────────────────────────────────────────┐
│                   lazychat                       │
├──────────────┬──────────────────────────────────┤
│ Conversations│  Chat View                       │
│              │                                  │
│ > General    │  You: How do I reverse a list?   │
│   Work stuff │                                  │
│   Ideas      │  AI: You can use slices.Reverse  │
│              │  from the slices package...       │
│              │                                  │
│              │                                  │
│              │                                  │
│              │                                  │
│  [n]ew [d]el │──────────────────────────────────│
│              │  > Type a message...        [⏎]  │
└──────────────┴──────────────────────────────────┘
```

### Components

- **Sidebar (left):** Conversation list. Navigate `↑/↓`, select `Enter`, `n` new, `d` delete.
- **Chat viewport (right top):** Scrollable message history with streaming AI responses.
- **Input area (right bottom):** Text input, `Enter` to send.
- **Focus:** `Tab` switches between sidebar and chat. `Ctrl+C` or `q` (sidebar focused) to quit.

## Data Flow

1. User types message and presses Enter
2. Message appended to conversation's message list
3. POST to `https://api.groq.com/openai/v1/chat/completions` with `stream: true`
4. SSE tokens arrive, appended to assistant message in real-time
5. Viewport auto-scrolls during streaming
6. On stream complete, conversation saved to JSON

## Groq API

OpenAI-compatible endpoint. Full conversation history sent each request. Streaming via Server-Sent Events.

Request:
```json
{
  "model": "llama-3.3-70b-versatile",
  "messages": [{"role": "user", "content": "..."}],
  "stream": true
}
```

Headers: `Authorization: Bearer $GROQ_API_KEY`, `Content-Type: application/json`

## Storage Format

One JSON file per conversation at `~/.local/share/lazychat/<id>.json`:

```json
{
  "id": "2026-04-02T21-39-00_general",
  "title": "General",
  "created_at": "2026-04-02T21:39:00Z",
  "messages": [
    {"role": "user", "content": "Hello"},
    {"role": "assistant", "content": "Hi there!"}
  ]
}
```

## Project Structure

```
lazychat/
├── main.go              # Entry point, env var check, launch TUI
├── tui/
│   ├── model.go         # Main Bubble Tea model, layout, key bindings
│   ├── sidebar.go       # Conversation list component
│   ├── chat.go          # Chat viewport + input component
│   └── styles.go        # Lipgloss styles
├── groq/
│   └── client.go        # Groq API client, streaming
├── store/
│   └── store.go         # Load/save conversations as JSON
└── conversation/
    └── conversation.go  # Conversation and message types
```

## Key Bindings

| Key | Context | Action |
|-----|---------|--------|
| `Tab` | Global | Switch focus sidebar/chat |
| `↑/↓` | Sidebar | Navigate conversations |
| `Enter` | Sidebar | Select conversation |
| `n` | Sidebar | New conversation |
| `d` | Sidebar | Delete conversation |
| `Enter` | Chat input | Send message |
| `Ctrl+C` | Global | Quit |
| `q` | Sidebar | Quit |
