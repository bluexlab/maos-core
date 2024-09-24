package adapter

type VoyageEmbeddingRequest struct {
	Input     []string `json:"input"`
	Model     string   `json:"model"`
	InputType *string  `json:"input_type,omitempty"`
}

type VoyageEmbeddingResponse struct {
	Object string                `json:"object"`
	Data   []VoyageEmbeddingData `json:"data"`
	Model  string                `json:"model"`
	Usage  struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

type VoyageEmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}
