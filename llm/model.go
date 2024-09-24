package llm

import "encoding/json"

type Model struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Name     string `json:"name"`
}

type EmbeddingModel struct {
	ID        string `json:"id"`
	Provider  string `json:"provider"`
	Name      string `json:"name"`
	Dimension int    `json:"dimension"`
}

// ModelListResponse represents the response for the model list endpoint
type ModelListResponse struct {
	Data []Model `json:"data"`
}

// EmbeddingListResponse represents the response for the embedding model list endpoint
type EmbeddingListResponse struct {
	Data []EmbeddingModel `json:"data"`
}

// Message represents a message used for generating completions
type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

// Content represents the content within a message
type Content struct {
	Text       string      `json:"text,omitempty"`
	Image      []byte      `json:"image,omitempty"`
	ImageURL   string      `json:"image_url,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
	ToolCall   *ToolCall   `json:"tool_call,omitempty"`
}

type ToolResult struct {
	ID      string `json:"tool_call_id"`
	Result  string `json:"result"`
	IsError bool   `json:"is_error"`
}

type ToolCall struct {
	ID           string `json:"id"`
	FunctionName string `json:"name"`
	Arguments    string `json:"arguments"`
}

// CompletionRequest represents the request body for the completion endpoint
type CompletionRequest struct {
	ModelID       string    `json:"model_id"`
	Messages      []Message `json:"messages"`
	Tools         []Tool    `json:"tools"`
	StopSequences []string  `json:"stop_sequences,omitempty"`
	Temperature   *float32  `json:"temperature,omitempty"`
	MaxTokens     *int32    `json:"max_tokens,omitempty"`
}

type CompletionResult struct {
	Messages []Message `json:"messages"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// EmbeddingRequest represents the request body for the embedding endpoint
type EmbeddingRequest struct {
	ModelID   string   `json:"model_id"`
	Input     []string `json:"input"`
	InputType *string  `json:"input_type"` // default: null. Other options are "query" and "document".
}

// EmbeddingResult represents the response body for the embedding endpoint
type EmbeddingResult struct {
	Data []Embedding `json:"data"`
}

// Embedding represents the embedding of a text
type Embedding struct {
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}
