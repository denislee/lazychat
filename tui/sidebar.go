// tui/sidebar.go
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazychat/conversation"
	"lazychat/store"
)

type selectConvMsg int
type selectConvInputMsg int
type previewConvMsg int
type previewUsageMsg struct{}
type newConvMsg struct{}
type deleteConvMsg int
type selectUsageMsg struct{}

type sidebar struct {
	conversations []conversation.Conversation
	selected      int
	focused       bool
	confirmDelete bool
	width         int
	height        int
	scroll        int // first visible conversation index
	skills        []store.Skill
}

func newSidebar() sidebar {
	return sidebar{focused: true}
}

func (s sidebar) isFixedMode(mode string) bool {
	for _, skill := range s.skills {
		if skill.Mode == mode {
			return true
		}
	}
	return false
}

// maxVisible returns the number of conversation items that fit in the sidebar.
// Overhead: header(2) + footer(4: blank + Usage + 2 help lines) + padding(2) = 8.
func (s sidebar) maxVisible() int {
	n := s.height - 8
	if s.confirmDelete {
		n-- // delete confirmation takes an extra line
	}
	if n < 1 {
		n = 1
	}
	return n
}

func (s *sidebar) ensureVisible() {
	maxVis := s.maxVisible()
	convSelected := s.selected
	if convSelected >= len(s.conversations) {
		// Usage item — show last conversations
		convSelected = len(s.conversations) - 1
	}
	if convSelected < 0 {
		return
	}
	if convSelected < s.scroll {
		s.scroll = convSelected
	}
	if convSelected >= s.scroll+maxVis {
		s.scroll = convSelected - maxVis + 1
	}
	if s.scroll < 0 {
		s.scroll = 0
	}
}

func (s sidebar) Update(msg tea.Msg) (sidebar, tea.Cmd) {
	if !s.focused {
		return s, nil
	}
	maxIdx := len(s.conversations) // last index = Usage item
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if s.confirmDelete {
			switch msg.String() {
			case "y":
				s.confirmDelete = false
				idx := s.selected
				return s, func() tea.Msg { return deleteConvMsg(idx) }
			case "n", "esc", "d":
				s.confirmDelete = false
			}
			return s, nil
		}
		switch msg.String() {
		case "up", "k", "ctrl+p":
			if s.selected > 0 {
				s.selected--
				s.ensureVisible()
				if s.selected < len(s.conversations) {
					idx := s.selected
					return s, func() tea.Msg { return previewConvMsg(idx) }
				} else if s.selected == maxIdx {
					return s, func() tea.Msg { return previewUsageMsg{} }
				}
			}
		case "down", "j", "ctrl+n":
			if s.selected < maxIdx {
				s.selected++
				s.ensureVisible()
				if s.selected < len(s.conversations) {
					idx := s.selected
					return s, func() tea.Msg { return previewConvMsg(idx) }
				} else if s.selected == maxIdx {
					return s, func() tea.Msg { return previewUsageMsg{} }
				}
			}
		case "enter", "l":
			if s.selected == maxIdx {
				return s, func() tea.Msg { return selectUsageMsg{} }
			}
			if s.selected < len(s.conversations) {
				idx := s.selected
				return s, func() tea.Msg { return selectConvMsg(idx) }
			}
		case "i":
			if s.selected < len(s.conversations) {
				idx := s.selected
				return s, func() tea.Msg { return selectConvInputMsg(idx) }
			}
		case "d":
			if s.selected < len(s.conversations) && !s.isFixedMode(s.conversations[s.selected].Mode) {
				s.confirmDelete = true
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
		b.WriteString(dimStyle.Render("Press '^k' to start"))
	} else {
		maxVis := s.maxVisible()
		end := s.scroll + maxVis
		if end > len(s.conversations) {
			end = len(s.conversations)
		}

		for i := s.scroll; i < end; i++ {
			conv := s.conversations[i]
			cursor := "  "
			style := normalItemStyle
			if i == s.selected {
				cursor = "> "
				if s.confirmDelete {
					style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
				} else {
					style = selectedItemStyle
				}
			}
			maxWidth := s.width - 2 - lipgloss.Width(cursor)
			title := lipgloss.NewStyle().MaxWidth(maxWidth).Render(conv.Title)
			b.WriteString(style.Render(cursor + title))
			b.WriteString("\n")
			if i == s.selected && s.confirmDelete {
				confirm := lipgloss.NewStyle().
					Foreground(lipgloss.Color("196")).
					Bold(true).
					Render("  Delete? [y/n]")
				b.WriteString(confirm)
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")
	usageIdx := len(s.conversations)
	cursor := "  "
	style := normalItemStyle
	if s.selected == usageIdx {
		cursor = "> "
		style = selectedItemStyle
	}
	b.WriteString(style.Render(cursor + "Usage"))

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("[d]el [m]odel [^k]mode"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("[q]uit"))

	return b.String()
}
