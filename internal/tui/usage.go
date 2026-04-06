// tui/usage.go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"lazychat/internal/provider"
)

type usageView struct {
	info         provider.RateLimitInfo
	providerName string
	model        string
	loading      bool
	err          string
	width        int
	height       int
}

func newUsageView() usageView {
	return usageView{}
}

func (u *usageView) setSize(w, h int) {
	u.width = w
	u.height = h
}

func (u usageView) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("API Usage"))
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Provider: " + u.providerName))
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Model: " + u.model))
	b.WriteString("\n\n")

	if u.loading {
		b.WriteString(dimStyle.Render("Fetching usage data..."))
		return b.String()
	}

	if u.err != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Error: " + u.err))
		b.WriteString("\n\n")
	}

	if u.info.LimitRequests == "" && u.info.LimitTokens == "" {
		b.WriteString(dimStyle.Render("No usage data yet. Send a message first or press 'r' to refresh."))
		return b.String()
	}

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).Width(22)
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	dimValStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	row := func(label, value string) string {
		return labelStyle.Render(label) + valStyle.Render(value) + "\n"
	}
	dimRow := func(label, value string) string {
		return labelStyle.Render(label) + dimValStyle.Render(value) + "\n"
	}

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).Render("Requests"))
	b.WriteString("\n")
	b.WriteString(row("  Remaining:", fmt.Sprintf("%s / %s", u.info.RemainingRequests, u.info.LimitRequests)))
	b.WriteString(dimRow("  Resets in:", u.info.ResetRequests))
	b.WriteString("\n")

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).Render("Tokens"))
	b.WriteString("\n")
	b.WriteString(row("  Remaining:", fmt.Sprintf("%s / %s", u.info.RemainingTokens, u.info.LimitTokens)))
	b.WriteString(dimRow("  Resets in:", u.info.ResetTokens))

	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("[r]efresh [h]back"))

	return b.String()
}
