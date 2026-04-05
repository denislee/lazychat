// groq/client_test.go
package groq

import (
	"strings"
	"testing"

	"lazychat/provider"
)

func TestParseSSELine(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		wantToken     string
		wantDone      bool
		wantSkip      bool
		wantReasoning bool
	}{
		{"token", `data: {"choices":[{"delta":{"content":"hello"}}]}`, "hello", false, false, false},
		{"reasoning", `data: {"choices":[{"delta":{"reasoning":"thinking..."}}]}`, "thinking...", false, false, true},
		{"done", "data: [DONE]", "", true, false, false},
		{"empty line", "", "", false, true, false},
		{"comment", ": comment", "", false, true, false},
		{"no prefix", "something", "", false, true, false},
		{"empty content", `data: {"choices":[{"delta":{"content":""}}]}`, "", false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, skip := parseSSELine(tt.line)
			if skip != tt.wantSkip {
				t.Errorf("skip = %v, want %v", skip, tt.wantSkip)
			}
			if skip {
				return
			}
			if event.Token != tt.wantToken {
				t.Errorf("token = %q, want %q", event.Token, tt.wantToken)
			}
			if event.Done != tt.wantDone {
				t.Errorf("done = %v, want %v", event.Done, tt.wantDone)
			}
			if event.Reasoning != tt.wantReasoning {
				t.Errorf("reasoning = %v, want %v", event.Reasoning, tt.wantReasoning)
			}
		})
	}
}

func TestCompoundParser(t *testing.T) {
	tests := []struct {
		name   string
		tokens []string
		want   []provider.StreamEvent
	}{
		{
			name:   "think block becomes reasoning",
			tokens: []string{"<think>", "hello world", "</think>"},
			want: []provider.StreamEvent{
				{Token: "hello world", Reasoning: true},
			},
		},
		{
			name:   "content after think is not reasoning",
			tokens: []string{"<think>", "thinking", "</think>", "answer"},
			want: []provider.StreamEvent{
				{Token: "thinking", Reasoning: true},
				{Token: "answer"},
			},
		},
		{
			name:   "tags split across tokens",
			tokens: []string{"<thi", "nk>", "reason", "</thi", "nk>", "done"},
			want: []provider.StreamEvent{
				{Token: "reason", Reasoning: true},
				{Token: "done"},
			},
		},
		{
			name:   "tool and output stripped",
			tokens: []string{"<tool>", "search(q)", "</tool>", "<output>", "results", "</output>"},
			want: []provider.StreamEvent{
				{Token: "search(q)"},
				{Token: "results"},
			},
		},
		{
			name:   "full compound response",
			tokens: []string{"<think>", "let me think", "</think>", "\n", "<tool>", "search(arm linux)", "</tool>", "\n", "<output>", "ARM info...", "</output>"},
			want: []provider.StreamEvent{
				{Token: "let me think", Reasoning: true},
				{Token: "\n"},
				{Token: "search(arm linux)"},
				{Token: "\n"},
				{Token: "ARM info..."},
			},
		},
		{
			name:   "no tags passes through",
			tokens: []string{"just plain text"},
			want: []provider.StreamEvent{
				{Token: "just plain text"},
			},
		},
		{
			name:   "angle bracket not a tag",
			tokens: []string{"x < y and a > b"},
			want: []provider.StreamEvent{
				{Token: "x "},
				{Token: "<"},
				{Token: " y and a > b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p compoundParser
			var got []provider.StreamEvent
			for _, tok := range tt.tokens {
				got = append(got, p.feed(tok)...)
			}
			got = append(got, p.flush()...)

			// Filter out empty-token events
			var filtered []provider.StreamEvent
			for _, ev := range got {
				if ev.Token != "" {
					filtered = append(filtered, ev)
				}
			}

			if len(filtered) != len(tt.want) {
				var gotStrs, wantStrs []string
				for _, e := range filtered {
					r := ""
					if e.Reasoning {
						r = " (reasoning)"
					}
					gotStrs = append(gotStrs, e.Token+r)
				}
				for _, e := range tt.want {
					r := ""
					if e.Reasoning {
						r = " (reasoning)"
					}
					wantStrs = append(wantStrs, e.Token+r)
				}
				t.Fatalf("event count = %d, want %d\ngot:  [%s]\nwant: [%s]",
					len(filtered), len(tt.want),
					strings.Join(gotStrs, ", "), strings.Join(wantStrs, ", "))
			}
			for i, ev := range filtered {
				if ev.Token != tt.want[i].Token {
					t.Errorf("event[%d].Token = %q, want %q", i, ev.Token, tt.want[i].Token)
				}
				if ev.Reasoning != tt.want[i].Reasoning {
					t.Errorf("event[%d].Reasoning = %v, want %v", i, ev.Reasoning, tt.want[i].Reasoning)
				}
			}
		})
	}
}

func TestWrapCompound(t *testing.T) {
	in := make(chan provider.StreamEvent)
	out := wrapCompound(in)

	go func() {
		in <- provider.StreamEvent{Token: "<think>"}
		in <- provider.StreamEvent{Token: "reasoning"}
		in <- provider.StreamEvent{Token: "</think>"}
		in <- provider.StreamEvent{Token: "answer"}
		in <- provider.StreamEvent{Done: true}
		close(in)
	}()

	var events []provider.StreamEvent
	for ev := range out {
		events = append(events, ev)
	}

	// Should get: reasoning(true), answer(false), done
	var content []provider.StreamEvent
	for _, ev := range events {
		if ev.Token != "" || ev.Done {
			content = append(content, ev)
		}
	}

	if len(content) != 3 {
		t.Fatalf("got %d events, want 3", len(content))
	}
	if content[0].Token != "reasoning" || !content[0].Reasoning {
		t.Errorf("event[0] = %+v, want reasoning token", content[0])
	}
	if content[1].Token != "answer" || content[1].Reasoning {
		t.Errorf("event[1] = %+v, want content token", content[1])
	}
	if !content[2].Done {
		t.Errorf("event[2] = %+v, want done", content[2])
	}
}
