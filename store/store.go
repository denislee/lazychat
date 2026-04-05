// store/store.go
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"lazychat/conversation"
)

type Store struct {
	dir string
}

func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Save(conv conversation.Conversation) error {
	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, conv.ID+".json"), data, 0644)
}

func (s *Store) Load(id string) (conversation.Conversation, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, id+".json"))
	if err != nil {
		return conversation.Conversation{}, err
	}
	var conv conversation.Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return conversation.Conversation{}, err
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
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var m meta
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
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
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, "config.json"), data, 0644)
}

func (s *Store) LoadConfig() (Config, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, "config.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

const maxHistoryEntries = 500

func (s *Store) SaveHistory(entries []string) error {
	if len(entries) > maxHistoryEntries {
		entries = entries[len(entries)-maxHistoryEntries:]
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, "history.json"), data, 0644)
}

func (s *Store) LoadHistory() []string {
	data, err := os.ReadFile(filepath.Join(s.dir, "history.json"))
	if err != nil {
		return nil
	}
	var entries []string
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil
	}
	return entries
}
