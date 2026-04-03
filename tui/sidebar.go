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

	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("[n]ew [d]el [q]uit"))

	return b.String()
}
