// tui/chat.go
package tui

import (
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"lazychat/conversation"
	"lazychat/provider"
)

type clipboardMsg struct{ err error }
type clearStatusMsg struct{}
type openPagerMsg struct{ content string }

type escEmptyChatMsg struct{}
type deleteMsgMsg int
type tokenMsg string
type reasoningTokenMsg string
type streamDoneMsg struct{}
type streamErrMsg struct{ err error }
type sendMsg string

const maxInputHeight = 10

type chat struct {
	viewport     viewport.Model
	input        textarea.Model
	messages     []conversation.Message
	streaming    bool
	streamCh     <-chan provider.StreamEvent
	focused      bool
	inputFocused bool
	selectedMsg    int
	confirmDelete  bool
	spinnerFrame   int
	activeModel    string
	mdRenderer     *glamour.TermRenderer
	width          int
	height         int
	err            string

	// Input history
	inputHistory []string // previously sent messages
	historyIdx   int      // current position in history (-1 = not navigating)
	savedInput   string   // text typed before navigating history
}

func newChat() chat {
	ti := textarea.New()
	ti.Placeholder = "Type a message..."
	ti.CharLimit = 0
	ti.MaxHeight = maxInputHeight
	ti.ShowLineNumbers = false
	ti.Prompt = ""
	ti.SetHeight(1)
	ti.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("alt+enter"))
	// Disable textarea's built-in up/down so we handle history navigation
	ti.KeyMap.LineNext = key.NewBinding(key.WithKeys(""))
	ti.KeyMap.LinePrevious = key.NewBinding(key.WithKeys(""))
	ti.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ti.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ti.FocusedStyle.EndOfBuffer = lipgloss.NewStyle()
	ti.BlurredStyle.EndOfBuffer = lipgloss.NewStyle()

	vp := viewport.New(0, 0)

	return chat{
		viewport:    vp,
		input:       ti,
		selectedMsg: -1,
		historyIdx:  -1,
	}
}

func waitForStream(ch <-chan provider.StreamEvent) tea.Cmd {
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
		if event.Reasoning {
			return reasoningTokenMsg(event.Token)
		}
		return tokenMsg(event.Token)
	}
}

// visualLineCount returns the number of visual lines the textarea content
// occupies, accounting for word wrapping.
func (c *chat) visualLineCount() int {
	val := c.input.Value()
	if val == "" {
		return 1
	}
	w := c.input.Width()
	if w <= 0 {
		w = 1
	}
	total := 0
	for _, logical := range strings.Split(val, "\n") {
		lineLen := lipgloss.Width(logical)
		if lineLen == 0 {
			total++
		} else {
			wrapped := (lineLen + w - 1) / w
			// The textarea's wrap function adds a trailing space to each
			// wrapped segment. When content fills exactly to the width,
			// this causes an extra line for the cursor.
			if lineLen%w == 0 {
				wrapped++
			}
			total += wrapped
		}
	}
	if total < 1 {
		total = 1
	}
	return total
}

// inputFrameHeight returns the current height of the input frame (border + content).
func (c *chat) inputFrameHeight() int {
	lines := c.visualLineCount()
	if lines > maxInputHeight {
		lines = maxInputHeight
	}
	return lines + 2 // top border + bottom border
}

func (c chat) Update(msg tea.Msg) (chat, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.focused {
			if c.inputFocused {
				switch msg.String() {
				case "enter":
					val := strings.TrimSpace(c.input.Value())
					if val != "" {
						c.inputHistory = append(c.inputHistory, val)
						c.historyIdx = -1
						c.savedInput = ""
						c.input.SetValue("")
						c.resizeInput()
						return c, func() tea.Msg { return sendMsg(val) }
					}
					return c, nil
				case "up", "ctrl+p":
					if len(c.inputHistory) > 0 {
						if c.historyIdx == -1 {
							c.savedInput = c.input.Value()
							c.historyIdx = len(c.inputHistory) - 1
						} else if c.historyIdx > 0 {
							c.historyIdx--
						}
						c.input.SetValue(c.inputHistory[c.historyIdx])
						c.input.CursorEnd()
						c.resizeInput()
					}
					return c, nil
				case "down":
					if c.historyIdx != -1 {
						if c.historyIdx < len(c.inputHistory)-1 {
							c.historyIdx++
							c.input.SetValue(c.inputHistory[c.historyIdx])
						} else {
							c.historyIdx = -1
							c.input.SetValue(c.savedInput)
						}
						c.input.CursorEnd()
						c.resizeInput()
					}
					return c, nil
				case "esc", "ctrl+]":
					if len(c.messages) == 0 {
						return c, func() tea.Msg { return escEmptyChatMsg{} }
					}
					c.historyIdx = -1
					c.savedInput = ""
					c.inputFocused = false
					c.input.Blur()
					c.resetInputHeight()
					return c, nil
				}

				// Expand height before update so the textarea's internal viewport
				// doesn't scroll when text wraps at a small height.
				c.input.SetHeight(maxInputHeight)
				var cmd tea.Cmd
				c.input, cmd = c.input.Update(msg)
				cmds = append(cmds, cmd)
				c.resizeInput()
				return c, tea.Batch(cmds...)
			} else if c.confirmDelete {
				switch msg.String() {
				case "y":
					c.confirmDelete = false
					if c.selectedMsg >= 0 && c.selectedMsg < len(c.messages) {
						idx := c.selectedMsg
						return c, func() tea.Msg { return deleteMsgMsg(idx) }
					}
					return c, nil
				case "n", "esc", "d":
					c.confirmDelete = false
					c.refreshViewport()
					return c, nil
				}
				return c, nil
			} else {
				switch msg.String() {
				case "i":
					c.inputFocused = true
					c.input.Focus()
					c.resizeInput()
					return c, nil
				case "n":
					return c, func() tea.Msg { return newConvMsg{} }
				case "d":
					if c.selectedMsg >= 0 && c.selectedMsg < len(c.messages) {
						c.confirmDelete = true
						c.refreshViewport()
					}
					return c, nil
				case "j", "down":
					if len(c.messages) > 0 {
						if c.selectedMsg < len(c.messages)-1 {
							c.selectedMsg++
						}
						c.refreshViewport()
						c.ensureVisible()
					}
					return c, nil
				case "k", "up":
					if len(c.messages) > 0 {
						if c.selectedMsg == -1 {
							c.selectedMsg = len(c.messages) - 1
						} else if c.selectedMsg > 0 {
							c.selectedMsg--
						}
						c.refreshViewport()
						c.ensureVisible()
					}
					return c, nil
				case "g":
					if len(c.messages) > 0 {
						c.selectedMsg = 0
						c.refreshViewport()
						c.ensureVisible()
					}
					return c, nil
				case "G":
					if len(c.messages) > 0 {
						c.selectedMsg = len(c.messages) - 1
						c.refreshViewport()
						c.ensureVisible()
					}
					return c, nil
				case "y":
					if c.selectedMsg >= 0 && c.selectedMsg < len(c.messages) {
						content := c.messages[c.selectedMsg].Content
						return c, func() tea.Msg {
							cmd := exec.Command("wl-copy")
							cmd.Stdin = strings.NewReader(content)
							err := cmd.Run()
							return clipboardMsg{err: err}
						}
					}
					return c, nil
				case "l":
					if c.selectedMsg >= 0 && c.selectedMsg < len(c.messages) {
						content := c.messages[c.selectedMsg].Content
						return c, func() tea.Msg { return openPagerMsg{content: content} }
					}
					return c, nil
				}
			}
		}

	case spinnerTickMsg:
		if c.streaming {
			c.spinnerFrame++
			c.refreshViewport()
		}

	case reasoningTokenMsg:
		if len(c.messages) > 0 {
			last := &c.messages[len(c.messages)-1]
			if last.Role == "assistant" && last.Content == "" && !last.Reasoning {
				// First reasoning token — mark the empty assistant msg as reasoning
				last.Reasoning = true
			}
			last.Content += string(msg)
			c.refreshViewport()
			c.viewport.GotoBottom()
		}
		return c, waitForStream(c.streamCh)

	case tokenMsg:
		if len(c.messages) > 0 {
			last := &c.messages[len(c.messages)-1]
			if last.Role == "assistant" && last.Reasoning {
				// Reasoning is done, create a new message for the answer
				c.messages = append(c.messages, conversation.Message{
					Role:  "assistant",
					Model: last.Model,
				})
				last = &c.messages[len(c.messages)-1]
			}
			last.Content += string(msg)
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
		// Put the error into the assistant message so it's visible inline
		if len(c.messages) > 0 && c.messages[len(c.messages)-1].Role == "assistant" {
			c.messages[len(c.messages)-1].Content = "Error: " + msg.err.Error()
		}
		c.refreshViewport()
		c.viewport.GotoBottom()
	}

	if c.focused && c.inputFocused {
		var cmd tea.Cmd
		c.input, cmd = c.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Don't pass key messages to viewport — we handle scrolling via ensureVisible
	if c.focused && !c.inputFocused {
		if _, isKey := msg.(tea.KeyMsg); !isKey {
			var cmd tea.Cmd
			c.viewport, cmd = c.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return c, tea.Batch(cmds...)
}

func (c *chat) formatMsgLine(msg conversation.Message) string {
	switch msg.Role {
	case "user":
		return userStyle.Render("You: ") + msg.Content
	case "assistant":
		label := msg.Model
		if label == "" {
			label = c.activeModel
		}
		if msg.Reasoning {
			return reasoningLabelStyle.Render(label+" thinking:") + "\n" + reasoningStyle.Render(msg.Content)
		}
		content := msg.Content
		if c.mdRenderer != nil && content != "" {
			rendered, err := c.mdRenderer.Render(content)
			if err == nil {
				content = strings.TrimSpace(rendered)
			}
		}
		return modelNameStyle.Render(label+":") + "\n" + content
	}
	return msg.Content
}

func (c *chat) refreshViewport() {
	var b strings.Builder
	w := c.viewport.Width
	for i, msg := range c.messages {
		isSelected := c.selectedMsg != -1 && i == c.selectedMsg

		// Show spinner on empty assistant message while streaming
		isThinking := c.streaming && i == len(c.messages)-1 && msg.Role == "assistant" && msg.Content == ""
		var line string
		if isThinking {
			frame := spinnerFrames[c.spinnerFrame%len(spinnerFrames)]
			label := msg.Model
			if label == "" {
				label = c.activeModel
			}
			line = modelNameStyle.Render(label+": ") +
				lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(frame+" thinking...")
		} else {
			line = c.formatMsgLine(msg)
		}

		if isSelected {
			borderColor := lipgloss.Color("170")
			if c.confirmDelete {
				borderColor = lipgloss.Color("196")
			}
			styled := lipgloss.NewStyle().
				Width(w - 3).
				PaddingLeft(1).
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(borderColor).
				Render(line)
			b.WriteString(styled)
			if c.confirmDelete {
				b.WriteString("\n")
				confirm := lipgloss.NewStyle().
					Foreground(lipgloss.Color("196")).
					Bold(true).
					PaddingLeft(3).
					Render("Delete this message? [y]es [n]o")
				b.WriteString(confirm)
			}
		} else {
			styled := lipgloss.NewStyle().
				Width(w).
				PaddingLeft(3).
				Render(line)
			b.WriteString(styled)
		}
		b.WriteString("\n\n")
	}
	if c.err != "" {
		b.WriteString(dimStyle.Render("Error: " + c.err))
		b.WriteString("\n")
	}
	c.viewport.SetContent(b.String())
}

func (c *chat) ensureVisible() {
	if c.selectedMsg < 0 || len(c.messages) == 0 {
		return
	}

	// Count lines before the selected message to find its position
	w := c.viewport.Width
	linesBefore := 0
	for i, msg := range c.messages {
		if i == c.selectedMsg {
			break
		}
		line := c.formatMsgLine(msg)
		rendered := lipgloss.NewStyle().Width(w).PaddingLeft(3).Render(line)
		linesBefore += strings.Count(rendered, "\n") + 1
		linesBefore += 2 // blank line separator
	}

	// Height of the selected message
	line := c.formatMsgLine(c.messages[c.selectedMsg])
	rendered := lipgloss.NewStyle().Width(w-3).PaddingLeft(1).
		BorderLeft(true).BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("170")).Render(line)
	msgHeight := strings.Count(rendered, "\n") + 1

	yOffset := c.viewport.YOffset
	vpHeight := c.viewport.Height

	if linesBefore < yOffset {
		c.viewport.SetYOffset(linesBefore)
	} else if linesBefore+msgHeight > yOffset+vpHeight {
		c.viewport.SetYOffset(linesBefore + msgHeight - vpHeight)
	}
}

// resetInputHeight shrinks the input back to 1 line (used when blurring).
func (c *chat) resetInputHeight() {
	c.input.SetHeight(1)
	c.viewport.Height = c.height - 3 // 1 line + 2 borders
}

// resizeInput adjusts the textarea height and viewport to fit the content.
func (c *chat) resizeInput() {
	lines := c.visualLineCount()
	if lines > maxInputHeight {
		lines = maxInputHeight
	}
	c.input.SetHeight(lines)
	frameH := lines + 2 // borders
	c.viewport.Height = c.height - frameH
}

func (c *chat) setSize(w, h int) {
	oldWidth := c.width
	c.width = w
	c.height = h
	inputInner := w - 4 // 2 for border sides + 2 for padding
	c.viewport.Width = w - 2
	c.input.SetWidth(inputInner)
	c.resizeInput()
	// Only recreate renderer when width actually changes
	if w != oldWidth {
		r, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(w-8),
		)
		if err == nil {
			c.mdRenderer = r
		}
	}
}

func (c chat) View() string {
	inputInner := c.width - 4
	frameFocused := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("170")).
		Width(inputInner)
	frameBlurred := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(inputInner)

	var bottom string
	if c.inputFocused {
		bottom = frameFocused.Render(c.input.View())
	} else {
		content := dimStyle.Render("[i]nput [h]back [j/k]scroll [y]ank [l]view [d]el")
		bottom = frameBlurred.Render(content)
	}

	bottomHeight := lipgloss.Height(bottom)
	vpHeight := c.height - bottomHeight
	if vpHeight < 0 {
		vpHeight = 0
	}
	c.viewport.Height = vpHeight
	vpView := c.viewport.View()

	return lipgloss.JoinVertical(lipgloss.Left, vpView, bottom)
}
