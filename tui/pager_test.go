package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"lazychat/conversation"
)

func TestPagerOpenAndView(t *testing.T) {
	p := newPager()
	p.open("Hello world\nLine two\nLine three", 80, 20)

	if !p.active {
		t.Fatal("pager should be active after open")
	}
	if len(p.lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(p.lines))
	}

	view := p.View()
	if view == "" {
		t.Fatal("pager View() returned empty string")
	}
	if !strings.Contains(view, "Hello world") {
		t.Error("pager View() missing content 'Hello world'")
	}
	if !strings.Contains(view, "NORMAL") {
		t.Error("pager View() missing NORMAL mode indicator")
	}
	t.Logf("View output:\n%s", view)
}

func TestPagerKeyNavigation(t *testing.T) {
	p := newPager()
	p.open("line1\nline2\nline3\nline4\nline5", 80, 20)

	if p.cursor != 0 {
		t.Fatalf("expected cursor=0, got %d", p.cursor)
	}

	// Press j
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.cursor != 1 {
		t.Fatalf("after j: expected cursor=1, got %d", p.cursor)
	}

	// Press k
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.cursor != 0 {
		t.Fatalf("after k: expected cursor=0, got %d", p.cursor)
	}

	// Press G (go to end)
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if p.cursor != 4 {
		t.Fatalf("after G: expected cursor=4, got %d", p.cursor)
	}

	// Press g (go to top)
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if p.cursor != 0 {
		t.Fatalf("after g: expected cursor=0, got %d", p.cursor)
	}
}

func TestPagerVisualSelect(t *testing.T) {
	p := newPager()
	p.open("line1\nline2\nline3", 80, 20)

	// Enter visual mode
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if p.selectFrom != 0 {
		t.Fatalf("expected selectFrom=0, got %d", p.selectFrom)
	}

	// Move down
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	text := p.selectedText()
	if text != "line1\nline2" {
		t.Fatalf("expected selected text 'line1\\nline2', got %q", text)
	}

	view := p.View()
	if !strings.Contains(view, "VISUAL") {
		t.Error("pager View() missing VISUAL mode indicator")
	}
}

func TestPagerClose(t *testing.T) {
	p := newPager()
	p.open("hello", 80, 20)

	p, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if p.active {
		t.Error("pager should be inactive after esc")
	}
	if cmd == nil {
		t.Error("expected pagerCloseMsg command")
	}
}

func TestOpenPagerMsgFromChat(t *testing.T) {
	// Verify the openPagerMsg type exists and can be created
	msg := openPagerMsg{content: "test content"}
	if msg.content != "test content" {
		t.Error("openPagerMsg content mismatch")
	}
}

func TestChatLKeyOpensP(t *testing.T) {
	c := newChat()
	c.focused = true
	c.inputFocused = false
	c.messages = []conversation.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "world response here"},
	}
	c.selectedMsg = 1
	c.setSize(80, 30)

	updatedChat, cmd := c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	_ = updatedChat

	if cmd == nil {
		t.Fatal("pressing 'l' with selected message should return a command")
	}

	// Execute the command to get the message
	msg := cmd()
	switch m := msg.(type) {
	case openPagerMsg:
		if m.content != "world response here" {
			t.Fatalf("expected content 'world response here', got %q", m.content)
		}
		t.Log("openPagerMsg received with correct content")
	default:
		t.Fatalf("expected openPagerMsg, got %T", msg)
	}
}

func TestWrapLine(t *testing.T) {
	// Short line stays as-is
	lines := wrapLine("hello world", 80)
	if len(lines) != 1 || lines[0] != "hello world" {
		t.Fatalf("short line: expected 1 line, got %d: %q", len(lines), lines)
	}

	// Long line wraps at word boundary
	long := "the quick brown fox jumps over the lazy dog and keeps on running"
	lines = wrapLine(long, 30)
	if len(lines) < 2 {
		t.Fatalf("expected >1 lines for width 30, got %d: %q", len(lines), lines)
	}
	for _, l := range lines {
		if len(l) > 30 {
			t.Errorf("wrapped line exceeds width: %q (%d chars)", l, len(l))
		}
	}
	joined := strings.Join(lines, " ")
	if joined != long {
		t.Errorf("wrapped content differs:\n  got:  %q\n  want: %q", joined, long)
	}

	// Empty line
	lines = wrapLine("", 40)
	if len(lines) != 1 || lines[0] != "" {
		t.Fatalf("empty line: expected [\"\"], got %q", lines)
	}
}

func TestPagerWordWrap(t *testing.T) {
	long := "this is a very long line that should definitely be wrapped when displayed in a narrow pager view"
	p := newPager()
	p.open(long, 30, 20)

	if len(p.lines) < 2 {
		t.Fatalf("expected long line to wrap into multiple lines, got %d", len(p.lines))
	}

	view := p.View()
	if !strings.Contains(view, "this") {
		t.Error("wrapped view missing start of content")
	}
	if !strings.Contains(view, "view") {
		t.Error("wrapped view missing end of content")
	}
	t.Logf("Wrapped pager (%d visual lines):\n%s", len(p.lines), view)
}

func TestChatLKeyNoSelection(t *testing.T) {
	c := newChat()
	c.focused = true
	c.inputFocused = false
	c.messages = []conversation.Message{
		{Role: "user", Content: "hello"},
	}
	c.selectedMsg = -1 // no selection
	c.setSize(80, 30)

	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if cmd != nil {
		t.Fatal("pressing 'l' without selection should NOT return a command")
	}
}
