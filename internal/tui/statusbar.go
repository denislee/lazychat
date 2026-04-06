// tui/statusbar.go
package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type netActivity int

const (
	netIdle netActivity = iota
	netSending
	netStreaming
	netFetchingUsage
	netError
)

type statusBar struct {
	width        int
	provider     string
	model        string
	activity     netActivity
	tokenCount   int
	spinnerFrame int
	lastError    string
	flashMsg     string
}

type spinnerTickMsg struct{}

func spinnerTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

func (s *statusBar) View() string {
	barStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252")).
		Width(s.width)

	provStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("170")).
		Bold(true)

	actStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("42"))

	dimBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("245"))

	left := provStyle.Render(" " + s.provider + "/" + s.model)

	errStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("196")).
		Bold(true)

	var right string

	if s.flashMsg != "" {
		right = actStyle.Render(s.flashMsg + " ")
		gap := s.width - lipgloss.Width(left) - lipgloss.Width(right)
		if gap < 0 {
			gap = 0
		}
		pad := lipgloss.NewStyle().Background(lipgloss.Color("236")).Width(gap).Render("")
		return barStyle.Render(left + pad + right)
	}

	switch s.activity {
	case netIdle:
		right = dimBarStyle.Render("idle ")
	case netError:
		errText := s.lastError
		maxLen := s.width - lipgloss.Width(left) - 2
		if len(errText) > maxLen && maxLen > 3 {
			errText = errText[:maxLen-3] + "..."
		}
		right = errStyle.Render(errText + " ")
	case netSending:
		frame := spinnerFrames[s.spinnerFrame%len(spinnerFrames)]
		right = actStyle.Render(fmt.Sprintf("%s connecting... ", frame))
	case netStreaming:
		frame := spinnerFrames[s.spinnerFrame%len(spinnerFrames)]
		right = actStyle.Render(fmt.Sprintf("%s streaming (%d tokens) ", frame, s.tokenCount))
	case netFetchingUsage:
		frame := spinnerFrames[s.spinnerFrame%len(spinnerFrames)]
		right = actStyle.Render(fmt.Sprintf("%s fetching usage... ", frame))
	}

	gap := s.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	pad := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Width(gap).
		Render("")

	return barStyle.Render(left + pad + right)
}
