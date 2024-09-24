package adapter

import "encoding/json"

type MessageRequest struct {
	Model         string    `json:"model"`
	System        *string   `json:"system,omitempty"` // System prompt
	Messages      []Message `json:"messages"`
	Tools         []Tool    `json:"tools,omitempty"`
	MaxTokens     int32     `json:"max_tokens"`
	StopSequences []string  `json:"stop_sequences,omitempty"`
	Temperature   *float32  `json:"temperature,omitempty"`
}

type Message struct {
	Role    string    `json:"role"` // "user", "assistant"
	Content []Content `json:"content"`
}

type Content struct {
	Type string `json:"type"` // "text", "image"，"tool_use", "tool_result"

	Text *string `json:"text,omitempty"` // Text content

	Source *Source `json:"source,omitempty"` // Image content

	Id    *string          `json:"id,omitempty"`    // Tool Use
	Name  *string          `json:"name,omitempty"`  // Tool Use
	Input *json.RawMessage `json:"input,omitempty"` // Tool Use

	ToolUseId *string   `json:"tool_use_id,omitempty"` // Tool Result
	IsError   *bool     `json:"is_error,omitempty"`    // Tool Result
	Content   []Content `json:"content,omitempty"`     // Tool Result
}

type Source struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/jpeg", "image/png", "image/gif", "image/webp"
	Data      []byte `json:"data"`       // image data
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type MessageResponse struct {
	ID           string                `json:"id"`   // ID of the message
	Type         string                `json:"type"` // "message", "error"
	Role         string                `json:"role"` // "assistant"
	Content      []Content             `json:"content"`
	Model        string                `json:"model"`
	StopReason   *string               `json:"stop_reason,omitempty"` // "end_turn", "max_tokens", "stop_sequence", "tool_use"
	StopSequence *string               `json:"stop_sequence,omitempty"`
	Usage        MessageResponseUsage  `json:"usage"`
	Error        *MessageResponseError `json:"error,omitempty"`
}

type MessageResponseUsage struct {
	InputTokens  int32 `json:"input_tokens"`
	OutputTokens int32 `json:"output_tokens"`
}

type MessageResponseError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
