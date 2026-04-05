// tui/skillpicker.go
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"lazychat/store"
)

type skillSelectedMsg struct {
	skill store.Skill
}

type skillPicker struct {
	skills   []store.Skill
	selected int
	viewport viewport.Model
	width    int
	height   int
}

func newSkillPicker(skills []store.Skill) skillPicker {
	return skillPicker{
		skills:   skills,
		viewport: viewport.New(0, 0),
	}
}

func (p skillPicker) Update(msg tea.Msg) (skillPicker, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if p.selected < len(p.skills)-1 {
				p.selected++
				p.refreshViewport()
			}
		case "k", "up", "ctrl+p":
			if p.selected > 0 {
				p.selected--
				p.refreshViewport()
			}
		case "enter", "l":
			if len(p.skills) > 0 {
				s := p.skills[p.selected]
				return p, func() tea.Msg {
					return skillSelectedMsg{skill: s}
				}
			}
		}
	}
	return p, nil
}

func (p *skillPicker) setSize(w, h int) {
	p.width = w
	p.height = h
	p.viewport.Width = w
	p.viewport.Height = h - 6
	p.refreshViewport()
}

func (p *skillPicker) refreshViewport() {
	var b strings.Builder

	for i, s := range p.skills {
		cursor := "  "
		style := normalItemStyle
		if i == p.selected {
			cursor = "> "
			style = selectedItemStyle
		}

		b.WriteString(style.Render(cursor + s.Title))
		if s.Mode != "" {
			b.WriteString(dimStyle.Render(" (" + s.Mode + ")"))
		}
		b.WriteString("\n")
	}

	p.viewport.SetContent(b.String())
}

func (p skillPicker) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Select Skill"))
	b.WriteString("\n\n")

	b.WriteString(p.viewport.View())

	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("[enter]select [esc]back"))

	return b.String()
}
