package adapter

type MessageRequest struct {
	Model         string    `json:"model"`
	System        *string   `json:"system,omitempty"` // System prompt
	Messages      []Message `json:"messages"`
	MaxTokens     int32     `json:"max_tokens"`
	StopSequences []string  `json:"stop_sequences,omitempty"`
	Temperature   *float32  `json:"temperature,omitempty"`
}

type Message struct {
	Role    string    `json:"role"` // "user", "assistant"
	Content []Content `json:"content"`
}

type Content struct {
	Type   string  `json:"type"` // "text", "image"
	Text   *string `json:"text,omitempty"`
	Source *Source `json:"source,omitempty"`
}

type Source struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/jpeg", "image/png", "image/gif", "image/webp"
	Data      []byte `json:"data"`       // image data
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
