// store/store.go
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	return conv, json.Unmarshal(data, &conv)
}

func (s *Store) List() ([]conversation.Conversation, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var convs []conversation.Conversation
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
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

func (s *Store) Delete(id string) error {
	return os.Remove(filepath.Join(s.dir, id+".json"))
}
