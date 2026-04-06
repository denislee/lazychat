// store/store.go
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"lazychat/internal/conversation"
)

type Store struct {
	dir string
}

func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Save(conv conversation.Conversation) error {
	file, err := os.OpenFile(filepath.Join(s.dir, conv.ID+".json"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for saving: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(conv); err != nil {
		return fmt.Errorf("failed to encode conversation: %w", err)
	}
	return nil
}

func (s *Store) Load(id string) (conversation.Conversation, error) {
	file, err := os.Open(filepath.Join(s.dir, id+".json"))
	if err != nil {
		return conversation.Conversation{}, fmt.Errorf("failed to open conversation file: %w", err)
	}
	defer file.Close()

	var conv conversation.Conversation
	if err := json.NewDecoder(file).Decode(&conv); err != nil {
		return conversation.Conversation{}, fmt.Errorf("failed to decode conversation: %w", err)
	}
	return conv, nil
}

func (s *Store) List() ([]conversation.Conversation, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var convs []conversation.Conversation
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" || e.Name() == "config.json" || e.Name() == "history.json" {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		conv, err := s.Load(id)
		if err != nil {
			continue
		}
		convs = append(convs, conv)
	}
	sort.Slice(convs, func(i, j int) bool {
		return convs[i].CreatedAt.After(convs[j].CreatedAt)
	})
	return convs, nil
}

// ListMeta returns conversations with only metadata (no messages) for fast startup.
func (s *Store) ListMeta() ([]conversation.Conversation, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	type meta struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		CreatedAt time.Time `json:"created_at"`
		Mode      string    `json:"mode,omitempty"`
	}
	var convs []conversation.Conversation
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" || e.Name() == "config.json" || e.Name() == "history.json" {
			continue
		}
		file, err := os.Open(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var m meta
		if err := json.NewDecoder(file).Decode(&m); err != nil {
			file.Close()
			continue
		}
		file.Close()
		convs = append(convs, conversation.Conversation{
			ID:        m.ID,
			Title:     m.Title,
			CreatedAt: m.CreatedAt,
			Mode:      m.Mode,
		})
	}
	sort.Slice(convs, func(i, j int) bool {
		return convs[i].CreatedAt.After(convs[j].CreatedAt)
	})
	return convs, nil
}

func (s *Store) Delete(id string) error {
	return os.Remove(filepath.Join(s.dir, id+".json"))
}

// Config holds user preferences persisted across sessions.
type Skill struct {
	Mode   string `json:"mode"`
	Title  string `json:"title"`
	Prompt string `json:"prompt"`
}

type Config struct {
	Provider string  `json:"provider"`
	Model    string  `json:"model"`
	Skills   []Skill `json:"skills"`
}

func (s *Store) SaveConfig(cfg Config) error {
	file, err := os.OpenFile(filepath.Join(s.dir, "config.json"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config file for saving: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}
	return nil
}

func (s *Store) LoadConfig() (Config, error) {
	file, err := os.Open(filepath.Join(s.dir, "config.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to decode config: %w", err)
	}
	return cfg, nil
}

const maxHistoryEntries = 500

func (s *Store) SaveHistory(entries []string) error {
	if len(entries) > maxHistoryEntries {
		entries = entries[len(entries)-maxHistoryEntries:]
	}
	file, err := os.OpenFile(filepath.Join(s.dir, "history.json"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open history file for saving: %w", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(entries); err != nil {
		return fmt.Errorf("failed to encode history: %w", err)
	}
	return nil
}

func (s *Store) LoadHistory() []string {
	file, err := os.Open(filepath.Join(s.dir, "history.json"))
	if err != nil {
		return nil
	}
	defer file.Close()

	var entries []string
	if err := json.NewDecoder(file).Decode(&entries); err != nil {
		return nil
	}
	return entries
}
