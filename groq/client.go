// groq/client.go
package groq

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"lazychat/conversation"
)

const (
	apiURL = "https://api.groq.com/openai/v1/chat/completions"
	Model  = "llama-3.3-70b-versatile"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

type chatRequest struct {
	Model    string                 `json:"model"`
	Messages []conversation.Message `json:"messages"`
	Stream   bool                   `json:"stream"`
}

type streamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// StreamEvent represents a single event from the SSE stream.
type StreamEvent struct {
	Token string
	Err   error
	Done  bool
}

// parseSSELine parses a single SSE line and returns (token, done, skip).
func parseSSELine(line string) (string, bool, bool) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false, true
	}
	data := strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return "", true, false
	}
	var sr streamResponse
	if err := json.Unmarshal([]byte(data), &sr); err != nil {
		return "", false, true
	}
	if len(sr.Choices) == 0 || sr.Choices[0].Delta.Content == "" {
		return "", false, true
	}
	return sr.Choices[0].Delta.Content, false, false
}

// StreamChat sends messages to Groq and returns a channel of streaming events.
func (c *Client) StreamChat(messages []conversation.Message) <-chan StreamEvent {
	ch := make(chan StreamEvent)
	go func() {
		defer close(ch)

		reqBody := chatRequest{
			Model:    Model,
			Messages: messages,
			Stream:   true,
		}
		body, err := json.Marshal(reqBody)
		if err != nil {
			ch <- StreamEvent{Err: err}
			return
		}

		req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
		if err != nil {
			ch <- StreamEvent{Err: err}
			return
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			ch <- StreamEvent{Err: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			ch <- StreamEvent{Err: fmt.Errorf("groq API error: %s", resp.Status)}
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			token, done, skip := parseSSELine(scanner.Text())
			if done {
				ch <- StreamEvent{Done: true}
				return
			}
			if skip {
				continue
			}
			ch <- StreamEvent{Token: token}
		}
		if err := scanner.Err(); err != nil {
			ch <- StreamEvent{Err: err}
		}
	}()
	return ch
}
