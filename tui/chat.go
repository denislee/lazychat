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
