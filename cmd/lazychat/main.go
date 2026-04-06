package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"lazychat/internal/gemini"
	"lazychat/internal/geminicli"
	"lazychat/internal/groq"
	"lazychat/internal/provider"
	"lazychat/internal/store"
	"lazychat/internal/tui"
)

func main() {
	var providers []provider.Provider

	if key := os.Getenv("GROQ_API_KEY"); key != "" {
		providers = append(providers, groq.NewClient(key))
	}
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		providers = append(providers, gemini.NewClient(key))
	}
	if geminicli.Available() {
		providers = append(providers, geminicli.NewClient())
	}

	if len(providers) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no providers available")
		fmt.Fprintln(os.Stderr, "Set GROQ_API_KEY or GEMINI_API_KEY, or install the gemini CLI")
		os.Exit(1)
	}

	dataDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "lazychat")
	s, err := store.New(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating store: %v\n", err)
		os.Exit(1)
	}

	m := tui.NewModel(s, providers...)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
