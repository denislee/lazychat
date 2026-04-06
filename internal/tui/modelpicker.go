// tui/modelpicker.go
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazychat/internal/provider"
)

type modelChangedMsg struct {
	provider string
	model    string
}

type modelEntry struct {
	provider string
	model    string
}

type modelPicker struct {
	entries  []modelEntry
	selected int
	current  modelEntry
	viewport viewport.Model
	width    int
	height   int
}

func newModelPicker(providers []provider.Provider) modelPicker {
	var entries []modelEntry
	for _, p := range providers {
		for _, m := range p.AvailableModels() {
			entries = append(entries, modelEntry{provider: p.Name(), model: m})
		}
	}

	var current modelEntry
	if len(providers) > 0 {
		current = modelEntry{provider: providers[0].Name(), model: providers[0].GetModel()}
	}

	return modelPicker{
		entries:  entries,
		current:  current,
		viewport: viewport.New(0, 0),
	}
}

func (p modelPicker) Update(msg tea.Msg) (modelPicker, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if p.selected < len(p.entries)-1 {
				p.selected++
				p.refreshViewport()
				p.ensureVisible()
			}
		case "k", "up", "ctrl+p":
			if p.selected > 0 {
				p.selected--
				p.refreshViewport()
				p.ensureVisible()
			}
		case "enter", "l":
			e := p.entries[p.selected]
			p.current = e
			return p, func() tea.Msg {
				return modelChangedMsg{provider: e.provider, model: e.model}
			}
		}
	}
	return p, nil
}

func (p *modelPicker) setSize(w, h int) {
	p.width = w
	p.height = h
	// Reserve lines for header ("Select Model\n\n") and footer
	p.viewport.Width = w
	p.viewport.Height = h - 6
	p.refreshViewport()
}

func (p *modelPicker) refreshViewport() {
	var b strings.Builder

	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	prevProvider := ""

	for i, e := range p.entries {
		if e.provider != prevProvider {
			if prevProvider != "" {
				b.WriteString("\n")
			}
			b.WriteString(sectionStyle.Render(strings.ToUpper(e.provider)))
			b.WriteString("\n")
			prevProvider = e.provider
		}

		cursor := "  "
		style := normalItemStyle
		if i == p.selected {
			cursor = "> "
			style = selectedItemStyle
		}

		label := e.model
		if e == p.current {
			label += " (active)"
		}

		b.WriteString(style.Render(cursor + label))
		b.WriteString("\n")
	}

	p.viewport.SetContent(b.String())
}

func (p *modelPicker) ensureVisible() {
	// Count lines before the selected entry
	linesBefore := 0
	prevProvider := ""
	for i, e := range p.entries {
		if e.provider != prevProvider {
			if prevProvider != "" {
				linesBefore++ // blank line between sections
			}
			linesBefore++ // section header
			prevProvider = e.provider
		}
		if i == p.selected {
			break
		}
		linesBefore++ // entry line
	}

	yOffset := p.viewport.YOffset
	vpHeight := p.viewport.Height

	if linesBefore < yOffset {
		p.viewport.SetYOffset(linesBefore)
	} else if linesBefore+1 > yOffset+vpHeight {
		p.viewport.SetYOffset(linesBefore + 1 - vpHeight)
	}
}

func (p modelPicker) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Select Model"))
	b.WriteString("\n\n")

	b.WriteString(p.viewport.View())

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Current: "))
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")).
		Render(p.current.provider + "/" + p.current.model))

	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("[enter]select [m]back"))

	return b.String()
}
