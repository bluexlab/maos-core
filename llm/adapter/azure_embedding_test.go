package adapter_test

import (
	"context"
	"os"
	"testing"

	"gitlab.com/navyx/ai/maos/maos-core/llm"
	"gitlab.com/navyx/ai/maos/maos-core/llm/adapter"
)

func TestAzureEmbeddingAdapter_GetEmbedding(t *testing.T) {
	t.Skip("skipping test")

	adapter, err := adapter.NewAzureEmbeddingAdapter(
		os.Getenv("AOAI_ENDPOINT"),
		os.Getenv("AOAI_API_KEY"),
	)
	if err != nil {
		t.Fatalf("Failed to create AzureEmbeddingAdapter: %v", err)
	}

	modelList := []string{
		"6baf223e-d321-41d2-bb33-c8328320e1e3-azure-text-embedding-ada-002",
		"d68a09df-3589-4273-b032-04488d9b230d-azure-text-embedding-3-small",
		"38f8d506-b9ab-4c5b-b7c7-051fd4849bbf-azure-text-embedding-3-large",
	}

	for _, model := range modelList {
		embeddingRequest := llm.EmbeddingRequest{
			ModelID: model,
			Input: []string{
				"Hello, world!",
				"This is a test",
			},
		}

		embeddingResult, err := adapter.GetEmbedding(context.Background(), embeddingRequest)
		if err != nil {
			t.Fatalf("Failed to get embedding: %v", err)
		}

		if len(embeddingResult.Data) != 2 {
			t.Fatalf("Expected 1 embedding, got %d", len(embeddingResult.Data))
		}

		for _, embedding := range embeddingResult.Data {
			if embedding.Embedding == nil {
				t.Fatalf("Expected embedding, got nil")
			}
		}
	}
}
