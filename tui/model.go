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

	sbStyle := blurredBorder.Padding(1, 1)
	chStyle := blurredBorder.Padding(0, 1)

	if m.focus == focusSidebar {
		sbStyle = focusedBorder.Padding(1, 1)
	} else {
		chStyle = focusedBorder.Padding(0, 1)
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
