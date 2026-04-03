# lazychat Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go TUI chat client for Groq with multi-conversation support, split-panel layout, and streaming responses.

**Architecture:** Bubble Tea (Elm architecture) with custom sidebar component and bubbles viewport/textinput. Groq streaming via SSE over net/http with channel-based Bubble Tea integration. JSON file persistence.

**Tech Stack:** Go, Bubble Tea v1, Bubbles v1 (viewport, textinput), Lipgloss v1, net/http

---

### Task 1: Initialize Go Module and Dependencies

**Files:**
- Create: `go.mod`

**Step 1: Initialize Go module**

Run: `go mod init lazychat`

**Step 2: Install dependencies**

Run:
```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
```

**Step 3: Verify**

Run: `cat go.mod`
Expected: Module `lazychat` with three dependencies listed.

**Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "feat: initialize go module with bubble tea dependencies"
```

---

### Task 2: Conversation Types

**Files:**
- Create: `conversation/conversation.go`
- Test: `conversation/conversation_test.go`

**Step 1: Write the failing test**

```go
// conversation/conversation_test.go
package conversation

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	before := time.Now()
	conv := New("My Chat")
	after := time.Now()

	if conv.Title != "My Chat" {
		t.Errorf("expected title 'My Chat', got '%s'", conv.Title)
	}
	if conv.ID == "" {
		t.Error("expected non-empty ID")
	}
	if len(conv.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(conv.Messages))
	}
	if conv.CreatedAt.Before(before) || conv.CreatedAt.After(after) {
		t.Error("created_at not within expected time range")
	}
}

func TestNewSanitizesTitle(t *testing.T) {
	conv := New("Hello World!")
	if conv.ID == "" {
		t.Error("expected non-empty ID")
	}
	// ID should not contain spaces
	for _, c := range conv.ID {
		if c == ' ' {
			t.Error("ID should not contain spaces")
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./conversation/ -v`
Expected: FAIL — package/files don't exist yet.

**Step 3: Write implementation**

```go
// conversation/conversation.go
package conversation

import (
	"regexp"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Conversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Messages  []Message `json:"messages"`
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9-]`)

func New(title string) Conversation {
	now := time.Now()
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = nonAlphanumeric.ReplaceAllString(slug, "")
	id := now.Format("2006-01-02T15-04-05") + "_" + slug
	return Conversation{
		ID:        id,
		Title:     title,
		CreatedAt: now,
		Messages:  []Message{},
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./conversation/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add conversation/
git commit -m "feat: add conversation and message types"
```

---

### Task 3: JSON Store

**Files:**
- Create: `store/store.go`
- Test: `store/store_test.go`

**Step 1: Write the failing tests**

```go
// store/store_test.go
package store

import (
	"os"
	"testing"

	"lazychat/conversation"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return s
}

func TestSaveAndLoad(t *testing.T) {
	s := testStore(t)
	conv := conversation.New("Test Chat")
	conv.Messages = append(conv.Messages, conversation.Message{
		Role: "user", Content: "hello",
	})

	if err := s.Save(conv); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := s.Load(conv.ID)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.Title != "Test Chat" {
		t.Errorf("expected title 'Test Chat', got '%s'", loaded.Title)
	}
	if len(loaded.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(loaded.Messages))
	}
}

func TestList(t *testing.T) {
	s := testStore(t)

	convs, err := s.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(convs) != 0 {
		t.Errorf("expected 0 conversations, got %d", len(convs))
	}

	s.Save(conversation.New("Chat 1"))
	s.Save(conversation.New("Chat 2"))

	convs, err = s.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(convs) != 2 {
		t.Errorf("expected 2 conversations, got %d", len(convs))
	}
}

func TestDelete(t *testing.T) {
	s := testStore(t)
	conv := conversation.New("To Delete")
	s.Save(conv)

	if err := s.Delete(conv.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err := s.Load(conv.ID)
	if !os.IsNotExist(err) {
		t.Errorf("expected file not found error, got: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./store/ -v`
Expected: FAIL — package doesn't exist yet.

**Step 3: Write implementation**

```go
// store/store.go
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"lazychat/conversation"
)

type Store struct {
	dir string
}

func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Save(conv conversation.Conversation) error {
	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, conv.ID+".json"), data, 0644)
}

func (s *Store) Load(id string) (conversation.Conversation, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, id+".json"))
	if err != nil {
		return conversation.Conversation{}, err
	}
	var conv conversation.Conversation
	return conv, json.Unmarshal(data, &conv)
}

func (s *Store) List() ([]conversation.Conversation, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var convs []conversation.Conversation
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		conv, err := s.Load(id)
		if err != nil {
			continue
		}
		convs = append(convs, conv)
	}
	sort.Slice(convs, func(i, j int) bool {
		return convs[i].CreatedAt.After(convs[j].CreatedAt)
	})
	return convs, nil
}

func (s *Store) Delete(id string) error {
	return os.Remove(filepath.Join(s.dir, id+".json"))
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./store/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add store/
git commit -m "feat: add JSON file store for conversations"
```

---

### Task 4: Groq Streaming Client

**Files:**
- Create: `groq/client.go`
- Test: `groq/client_test.go`

**Step 1: Write the failing test for SSE parsing**

```go
// groq/client_test.go
package groq

import (
	"testing"
)

func TestParseSSELine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    string
		wantDone bool
		wantSkip bool
	}{
		{"token", `data: {"choices":[{"delta":{"content":"hello"}}]}`, "hello", false, false},
		{"done", "data: [DONE]", "", true, false},
		{"empty line", "", "", false, true},
		{"comment", ": comment", "", false, true},
		{"no prefix", "something", "", false, true},
		{"empty content", `data: {"choices":[{"delta":{"content":""}}]}`, "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, done, skip := parseSSELine(tt.line)
			if token != tt.want {
				t.Errorf("token = %q, want %q", token, tt.want)
			}
			if done != tt.wantDone {
				t.Errorf("done = %v, want %v", done, tt.wantDone)
			}
			if skip != tt.wantSkip {
				t.Errorf("skip = %v, want %v", skip, tt.wantSkip)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./groq/ -v`
Expected: FAIL

**Step 3: Write implementation**

```go
// groq/client.go
package groq

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"lazychat/conversation"
)

const (
	apiURL = "https://api.groq.com/openai/v1/chat/completions"
	Model  = "llama-3.3-70b-versatile"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

type chatRequest struct {
	Model    string                 `json:"model"`
	Messages []conversation.Message `json:"messages"`
	Stream   bool                   `json:"stream"`
}

type streamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// StreamEvent represents a single event from the SSE stream.
type StreamEvent struct {
	Token string
	Err   error
	Done  bool
}

// parseSSELine parses a single SSE line and returns (token, done, skip).
func parseSSELine(line string) (string, bool, bool) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false, true
	}
	data := strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return "", true, false
	}
	var sr streamResponse
	if err := json.Unmarshal([]byte(data), &sr); err != nil {
		return "", false, true
	}
	if len(sr.Choices) == 0 || sr.Choices[0].Delta.Content == "" {
		return "", false, true
	}
	return sr.Choices[0].Delta.Content, false, false
}

// StreamChat sends messages to Groq and returns a channel of streaming events.
func (c *Client) StreamChat(messages []conversation.Message) <-chan StreamEvent {
	ch := make(chan StreamEvent)
	go func() {
		defer close(ch)

		reqBody := chatRequest{
			Model:    Model,
			Messages: messages,
			Stream:   true,
		}
		body, err := json.Marshal(reqBody)
		if err != nil {
			ch <- StreamEvent{Err: err}
			return
		}

		req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
		if err != nil {
			ch <- StreamEvent{Err: err}
			return
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			ch <- StreamEvent{Err: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			ch <- StreamEvent{Err: fmt.Errorf("groq API error: %s", resp.Status)}
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			token, done, skip := parseSSELine(scanner.Text())
			if done {
				ch <- StreamEvent{Done: true}
				return
			}
			if skip {
				continue
			}
			ch <- StreamEvent{Token: token}
		}
		if err := scanner.Err(); err != nil {
			ch <- StreamEvent{Err: err}
		}
	}()
	return ch
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./groq/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add groq/
git commit -m "feat: add Groq streaming client with SSE parsing"
```

---

### Task 5: TUI Styles

**Files:**
- Create: `tui/styles.go`

**Step 1: Write styles**

```go
// tui/styles.go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	sidebarStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 1)

	chatStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170"))

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("170"))

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	userStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("36"))

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	focusedBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("170"))

	blurredBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
)
```

**Step 2: Verify it compiles**

Run: `go build ./tui/...` (will fail until model.go exists — that's fine, we just check syntax)
Run: `go vet ./tui/...` (same — expected to fail, move on)

**Step 3: Commit**

```bash
git add tui/styles.go
git commit -m "feat: add TUI lipgloss styles"
```

---

### Task 6: TUI Sidebar Component

**Files:**
- Create: `tui/sidebar.go`

**Step 1: Write sidebar**

```go
// tui/sidebar.go
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"lazychat/conversation"
)

type selectConvMsg int
type newConvMsg struct{}
type deleteConvMsg int

type sidebar struct {
	conversations []conversation.Conversation
	selected      int
	focused       bool
	width         int
	height        int
}

func newSidebar() sidebar {
	return sidebar{focused: true}
}

func (s sidebar) Update(msg tea.Msg) (sidebar, tea.Cmd) {
	if !s.focused {
		return s, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.selected > 0 {
				s.selected--
			}
		case "down", "j":
			if s.selected < len(s.conversations)-1 {
				s.selected++
			}
		case "enter":
			if len(s.conversations) > 0 {
				idx := s.selected
				return s, func() tea.Msg { return selectConvMsg(idx) }
			}
		case "n":
			return s, func() tea.Msg { return newConvMsg{} }
		case "d":
			if len(s.conversations) > 0 {
				idx := s.selected
				return s, func() tea.Msg { return deleteConvMsg(idx) }
			}
		}
	}
	return s, nil
}

func (s sidebar) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Conversations"))
	b.WriteString("\n\n")

	if len(s.conversations) == 0 {
		b.WriteString(dimStyle.Render("No conversations yet"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Press 'n' to start"))
	} else {
		for i, conv := range s.conversations {
			cursor := "  "
			style := normalItemStyle
			if i == s.selected {
				cursor = "> "
				style = selectedItemStyle
			}
			b.WriteString(style.Render(cursor + conv.Title))
			if i < len(s.conversations)-1 {
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}
```

**Step 2: Commit**

```bash
git add tui/sidebar.go
git commit -m "feat: add TUI sidebar component"
```

---

### Task 7: TUI Chat Component

**Files:**
- Create: `tui/chat.go`

**Step 1: Write chat component**

```go
// tui/chat.go
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"lazychat/conversation"
	"lazychat/groq"
)

type tokenMsg string
type streamDoneMsg struct{}
type streamErrMsg struct{ err error }
type sendMsg string

type chat struct {
	viewport  viewport.Model
	input     textinput.Model
	messages  []conversation.Message
	streaming bool
	streamCh  <-chan groq.StreamEvent
	focused   bool
	width     int
	height    int
	err       string
}

func newChat() chat {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.CharLimit = 0

	vp := viewport.New(0, 0)

	return chat{
		viewport: vp,
		input:    ti,
	}
}

func waitForStream(ch <-chan groq.StreamEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return streamDoneMsg{}
		}
		if event.Err != nil {
			return streamErrMsg{err: event.Err}
		}
		if event.Done {
			return streamDoneMsg{}
		}
		return tokenMsg(event.Token)
	}
}

func (c chat) Update(msg tea.Msg) (chat, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.focused && !c.streaming {
			switch msg.String() {
			case "enter":
				val := strings.TrimSpace(c.input.Value())
				if val != "" {
					c.input.SetValue("")
					return c, func() tea.Msg { return sendMsg(val) }
				}
			}
		}

	case tokenMsg:
		if len(c.messages) > 0 {
			c.messages[len(c.messages)-1].Content += string(msg)
			c.refreshViewport()
			c.viewport.GotoBottom()
		}
		return c, waitForStream(c.streamCh)

	case streamDoneMsg:
		c.streaming = false
		c.streamCh = nil

	case streamErrMsg:
		c.streaming = false
		c.streamCh = nil
		c.err = msg.err.Error()
	}

	if c.focused {
		var cmd tea.Cmd
		c.input, cmd = c.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd
	c.viewport, cmd = c.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return c, tea.Batch(cmds...)
}

func (c *chat) refreshViewport() {
	var b strings.Builder
	for _, msg := range c.messages {
		switch msg.Role {
		case "user":
			b.WriteString(userStyle.Render("You: "))
			b.WriteString(msg.Content)
		case "assistant":
			b.WriteString(assistantStyle.Render("AI: "))
			b.WriteString(msg.Content)
		}
		b.WriteString("\n\n")
	}
	if c.err != "" {
		b.WriteString(dimStyle.Render("Error: " + c.err))
		b.WriteString("\n")
	}
	c.viewport.SetContent(b.String())
}

func (c *chat) setSize(w, h int) {
	c.width = w
	c.height = h
	c.viewport.Width = w
	c.viewport.Height = h - 3
	c.input.Width = w - 2
}

func (c chat) View() string {
	var b strings.Builder
	b.WriteString(c.viewport.View())
	b.WriteString("\n")
	if c.streaming {
		b.WriteString(dimStyle.Render("  AI is typing..."))
	} else {
		b.WriteString(c.input.View())
	}
	return b.String()
}
```

**Step 2: Commit**

```bash
git add tui/chat.go
git commit -m "feat: add TUI chat component with streaming support"
```

---

### Task 8: TUI Main Model

**Files:**
- Create: `tui/model.go`

**Step 1: Write main model**

```go
// tui/model.go
package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazychat/conversation"
	"lazychat/groq"
	"lazychat/store"
)

const sidebarWidth = 25

type focus int

const (
	focusSidebar focus = iota
	focusChat
)

type Model struct {
	sidebar    sidebar
	chat       chat
	store      *store.Store
	groq       *groq.Client
	focus      focus
	width      int
	height     int
	activeConv *conversation.Conversation
	ready      bool
}

func NewModel(s *store.Store, g *groq.Client) Model {
	sb := newSidebar()
	ch := newChat()

	convs, _ := s.List()
	sb.conversations = convs

	return Model{
		sidebar: sb,
		chat:    ch,
		store:   s,
		groq:    g,
		focus:   focusSidebar,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.updateSizes()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			if m.activeConv != nil {
				m.toggleFocus()
			}
			return m, nil
		case "q":
			if m.focus == focusSidebar {
				return m, tea.Quit
			}
		}

	case selectConvMsg:
		idx := int(msg)
		if idx >= 0 && idx < len(m.sidebar.conversations) {
			conv := m.sidebar.conversations[idx]
			m.activeConv = &conv
			m.chat.messages = conv.Messages
			m.chat.err = ""
			m.chat.refreshViewport()
			m.chat.viewport.GotoBottom()
			m.toggleFocus()
		}
		return m, nil

	case newConvMsg:
		conv := conversation.New("New Chat")
		m.store.Save(conv)
		m.sidebar.conversations = append([]conversation.Conversation{conv}, m.sidebar.conversations...)
		m.sidebar.selected = 0
		m.activeConv = &m.sidebar.conversations[0]
		m.chat.messages = nil
		m.chat.err = ""
		m.chat.refreshViewport()
		m.toggleFocus()
		return m, nil

	case deleteConvMsg:
		idx := int(msg)
		if idx >= 0 && idx < len(m.sidebar.conversations) {
			conv := m.sidebar.conversations[idx]
			m.store.Delete(conv.ID)
			m.sidebar.conversations = append(
				m.sidebar.conversations[:idx],
				m.sidebar.conversations[idx+1:]...,
			)
			if m.activeConv != nil && m.activeConv.ID == conv.ID {
				m.activeConv = nil
				m.chat.messages = nil
				m.chat.refreshViewport()
			}
			if m.sidebar.selected >= len(m.sidebar.conversations) && m.sidebar.selected > 0 {
				m.sidebar.selected--
			}
		}
		return m, nil

	case sendMsg:
		if m.activeConv == nil {
			return m, nil
		}
		userMsg := conversation.Message{Role: "user", Content: string(msg)}
		m.activeConv.Messages = append(m.activeConv.Messages, userMsg)

		// Auto-title from first message
		if len(m.activeConv.Messages) == 1 && m.activeConv.Title == "New Chat" {
			title := string(msg)
			if len(title) > 30 {
				title = title[:30] + "..."
			}
			m.activeConv.Title = title
			for i := range m.sidebar.conversations {
				if m.sidebar.conversations[i].ID == m.activeConv.ID {
					m.sidebar.conversations[i].Title = title
					break
				}
			}
		}

		assistantMsg := conversation.Message{Role: "assistant", Content: ""}
		m.activeConv.Messages = append(m.activeConv.Messages, assistantMsg)

		m.chat.messages = m.activeConv.Messages
		m.chat.refreshViewport()
		m.chat.viewport.GotoBottom()

		// Start streaming (send only messages up to the user message, not the empty assistant one)
		ch := m.groq.StreamChat(m.activeConv.Messages[:len(m.activeConv.Messages)-1])
		m.chat.streamCh = ch
		m.chat.streaming = true

		return m, waitForStream(ch)

	case tokenMsg:
		// Update active conversation with streamed tokens
		if m.activeConv != nil && len(m.activeConv.Messages) > 0 {
			last := len(m.activeConv.Messages) - 1
			m.activeConv.Messages[last].Content += string(msg)
		}

	case streamDoneMsg:
		if m.activeConv != nil {
			m.store.Save(*m.activeConv)
		}

	case streamErrMsg:
		if m.activeConv != nil {
			m.store.Save(*m.activeConv)
		}
	}

	var cmd tea.Cmd
	m.sidebar, cmd = m.sidebar.Update(msg)
	cmds = append(cmds, cmd)

	m.chat, cmd = m.chat.Update(msg)
	cmds = append(cmds, cmd)

	// Sync chat messages back to active conversation
	if m.activeConv != nil {
		m.activeConv.Messages = m.chat.messages
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) toggleFocus() {
	if m.focus == focusSidebar {
		m.focus = focusChat
		m.sidebar.focused = false
		m.chat.focused = true
		m.chat.input.Focus()
	} else {
		m.focus = focusSidebar
		m.sidebar.focused = true
		m.chat.focused = false
		m.chat.input.Blur()
	}
}

func (m *Model) updateSizes() {
	chatWidth := m.width - sidebarWidth - 4
	chatHeight := m.height - 2
	m.sidebar.width = sidebarWidth
	m.sidebar.height = m.height
	m.chat.setSize(chatWidth, chatHeight)
}

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	sbStyle := blurredBorder.Copy().Padding(1, 1)
	chStyle := blurredBorder.Copy().Padding(0, 1)

	if m.focus == focusSidebar {
		sbStyle = focusedBorder.Copy().Padding(1, 1)
	} else {
		chStyle = focusedBorder.Copy().Padding(0, 1)
	}

	sidebarView := sbStyle.
		Width(sidebarWidth).
		Height(m.height - 2).
		Render(m.sidebar.View())

	chatView := chStyle.
		Width(m.width - sidebarWidth - 4).
		Height(m.height - 2).
		Render(m.chat.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, chatView)
}
```

**Step 2: Verify it compiles**

Run: `go build ./tui/`
Expected: Success (no output)

**Step 3: Commit**

```bash
git add tui/
git commit -m "feat: add TUI main model with split layout"
```

---

### Task 9: Main Entry Point

**Files:**
- Create: `main.go`

**Step 1: Write main.go**

```go
// main.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"lazychat/groq"
	"lazychat/store"
	"lazychat/tui"
)

func main() {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: GROQ_API_KEY environment variable is not set")
		fmt.Fprintln(os.Stderr, "Get a free key at https://console.groq.com")
		os.Exit(1)
	}

	dataDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "lazychat")
	s, err := store.New(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating store: %v\n", err)
		os.Exit(1)
	}

	g := groq.NewClient(apiKey)
	m := tui.NewModel(s, g)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 2: Verify it compiles**

Run: `go build -o lazychat .`
Expected: Binary `lazychat` created with no errors.

**Step 3: Run all tests**

Run: `go test ./... -v`
Expected: All tests PASS.

**Step 4: Commit**

```bash
git add main.go
git commit -m "feat: add main entry point"
```

---

### Task 10: Integration Test

**Step 1: Run the app manually**

Run: `GROQ_API_KEY=your-key-here ./lazychat`

Expected behavior:
1. App launches in alt-screen with sidebar (focused) and empty chat panel
2. Press `n` — creates "New Chat", focus switches to chat input
3. Type a message, press Enter — message appears, AI streams a response
4. Press Tab — focus back to sidebar
5. Press `q` — app quits
6. Re-launch — conversation persists in sidebar

**Step 2: Verify JSON file created**

Run: `ls ~/.local/share/lazychat/`
Expected: One `.json` file per conversation created.

**Step 3: Final commit**

```bash
git add -A
git commit -m "feat: lazychat v0.1 — TUI chat client for Groq"
```
