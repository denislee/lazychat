// provider/provider.go
package provider

import "lazychat/conversation"

// StreamEvent represents a single event from a streaming response.
type StreamEvent struct {
	Token     string
	Err       error
	Done      bool
	Reasoning bool // true when the token is part of the model's thinking/reasoning
}

// RateLimitInfo holds rate limit data from API response headers.
type RateLimitInfo struct {
	LimitRequests     string
	LimitTokens       string
	RemainingRequests string
	RemainingTokens   string
	ResetRequests     string
	ResetTokens       string
}

// Provider is the interface that all LLM backends must implement.
type Provider interface {
	Name() string
	AvailableModels() []string
	GetModel() string
	SetModel(model string)
	GetRateLimit() RateLimitInfo
	FetchUsage() (RateLimitInfo, error)
	StreamChat(messages []conversation.Message) <-chan StreamEvent
}
