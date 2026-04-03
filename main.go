package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"lazychat/groq"
	"lazychat/store"
	"lazychat/tui"
)

func main() {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: GROQ_API_KEY environment variable is not set")
		fmt.Fprintln(os.Stderr, "Get a free key at https://console.groq.com")
		os.Exit(1)
	}

	dataDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "lazychat")
	s, err := store.New(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating store: %v\n", err)
		os.Exit(1)
	}

	g := groq.NewClient(apiKey)
	m := tui.NewModel(s, g)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
