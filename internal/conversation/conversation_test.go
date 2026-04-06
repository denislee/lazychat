// conversation/conversation_test.go
package conversation

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	before := time.Now()
	conv := New("My Chat")
	after := time.Now()

	if conv.Title != "My Chat" {
		t.Errorf("expected title 'My Chat', got '%s'", conv.Title)
	}
	if conv.ID == "" {
		t.Error("expected non-empty ID")
	}
	if len(conv.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(conv.Messages))
	}
	if conv.CreatedAt.Before(before) || conv.CreatedAt.After(after) {
		t.Error("created_at not within expected time range")
	}
}

func TestNewSanitizesTitle(t *testing.T) {
	conv := New("Hello World!")
	if conv.ID == "" {
		t.Error("expected non-empty ID")
	}
	for _, c := range conv.ID {
		if c == ' ' {
			t.Error("ID should not contain spaces")
		}
	}
}
