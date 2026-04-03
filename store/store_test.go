// store/store_test.go
package store

import (
	"os"
	"testing"

	"lazychat/conversation"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return s
}

func TestSaveAndLoad(t *testing.T) {
	s := testStore(t)
	conv := conversation.New("Test Chat")
	conv.Messages = append(conv.Messages, conversation.Message{
		Role: "user", Content: "hello",
	})

	if err := s.Save(conv); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := s.Load(conv.ID)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.Title != "Test Chat" {
		t.Errorf("expected title 'Test Chat', got '%s'", loaded.Title)
	}
	if len(loaded.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(loaded.Messages))
	}
}

func TestList(t *testing.T) {
	s := testStore(t)

	convs, err := s.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(convs) != 0 {
		t.Errorf("expected 0 conversations, got %d", len(convs))
	}

	s.Save(conversation.New("Chat 1"))
	s.Save(conversation.New("Chat 2"))

	convs, err = s.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(convs) != 2 {
		t.Errorf("expected 2 conversations, got %d", len(convs))
	}
}

func TestDelete(t *testing.T) {
	s := testStore(t)
	conv := conversation.New("To Delete")
	s.Save(conv)

	if err := s.Delete(conv.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err := s.Load(conv.ID)
	if !os.IsNotExist(err) {
		t.Errorf("expected file not found error, got: %v", err)
	}
}
