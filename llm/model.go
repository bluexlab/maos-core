package llm

type Model struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Name     string `json:"name"`
}

// ModelListResponse represents the response for the model list endpoint
type ModelListResponse struct {
	Data []Model `json:"data"`
}

// Message represents a message used for generating completions
type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

// Content represents the content within a message
type Content struct {
	Text  string `json:"text,omitempty"`
	Image []byte `json:"image,omitempty"`
}

// CompletionRequest represents the request body for the completion endpoint
type CompletionRequest struct {
	ModelID       string    `json:"model_id"`
	Messages      []Message `json:"messages"`
	StopSequences []string  `json:"stop_sequences,omitempty"`
	Temperature   *float32  `json:"temperature,omitempty"`
	MaxTokens     *int32    `json:"max_tokens,omitempty"`
}

type CompletionResult struct {
	Messages []Message `json:"messages"`
}
