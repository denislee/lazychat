// groq/client_test.go
package groq

import (
	"testing"
)

func TestParseSSELine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		want     string
		wantDone bool
		wantSkip bool
	}{
		{"token", `data: {"choices":[{"delta":{"content":"hello"}}]}`, "hello", false, false},
		{"done", "data: [DONE]", "", true, false},
		{"empty line", "", "", false, true},
		{"comment", ": comment", "", false, true},
		{"no prefix", "something", "", false, true},
		{"empty content", `data: {"choices":[{"delta":{"content":""}}]}`, "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, done, skip := parseSSELine(tt.line)
			if token != tt.want {
				t.Errorf("token = %q, want %q", token, tt.want)
			}
			if done != tt.wantDone {
				t.Errorf("done = %v, want %v", done, tt.wantDone)
			}
			if skip != tt.wantSkip {
				t.Errorf("skip = %v, want %v", skip, tt.wantSkip)
			}
		})
	}
}
