// tui/styles.go
package tui

import "github.com/charmbracelet/lipgloss"

var (
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

	modelNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99"))

	reasoningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)

	reasoningLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("243")).
				Italic(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	focusedBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("170"))

	blurredBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
)
