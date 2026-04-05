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
	isNew bool
}

type skillEntry struct {
	title string
	mode  string
	isNew bool
}

type skillPicker struct {
	items    []skillEntry
	selected int
	viewport viewport.Model
	width    int
	height   int
}

func newSkillPicker(skills []store.Skill) skillPicker {
	items := []skillEntry{
		{title: "[new chat]", isNew: true},
	}
	for _, s := range skills {
		items = append(items, skillEntry{
			title: s.Title,
			mode:  s.Mode,
		})
	}

	return skillPicker{
		items:    items,
		viewport: viewport.New(0, 0),
	}
}

func (p skillPicker) Update(msg tea.Msg) (skillPicker, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down", "ctrl+n":
			if p.selected < len(p.items)-1 {
				p.selected++
				p.refreshViewport()
			}
		case "k", "up", "ctrl+p":
			if p.selected > 0 {
				p.selected--
				p.refreshViewport()
			}
		case "enter", "l":
			if len(p.items) > 0 {
				item := p.items[p.selected]
				if item.isNew {
					return p, func() tea.Msg {
						return skillSelectedMsg{isNew: true}
					}
				}
				// Find the actual skill
				var selectedSkill store.Skill
				selectedSkill.Title = item.title
				selectedSkill.Mode = item.mode
				return p, func() tea.Msg {
					return skillSelectedMsg{skill: selectedSkill}
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

	for i, item := range p.items {
		cursor := "  "
		style := normalItemStyle
		if i == p.selected {
			cursor = "> "
			style = selectedItemStyle
		}

		b.WriteString(style.Render(cursor + item.title))
		if item.mode != "" {
			b.WriteString(dimStyle.Render(" (" + item.mode + ")"))
		}
		b.WriteString("\n")
	}

	p.viewport.SetContent(b.String())
}

func (p skillPicker) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Select Chat Mode"))
	b.WriteString("\n\n")

	b.WriteString(p.viewport.View())

	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("[enter]select [esc/h]back"))

	return b.String()
}
