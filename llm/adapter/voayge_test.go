package adapter

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/samber/lo"
	"gitlab.com/navyx/ai/maos/maos-core/llm"
)

func TestVoyageEmbeddingAdapter_GetEmbedding(t *testing.T) {
	t.Skip()

	adapter := NewVoyageEmbeddingAdapter()
	ctx := context.Background()

	docRequest := llm.EmbeddingRequest{
		ModelID:   "7a571008-601a-463f-a3b9-1ba48de38984-voyage-large-2",
		Input:     []string{"The capital of Taiwan is Taipei.", "The capital of Japan is Tokyo."},
		InputType: lo.ToPtr("document"),
	}

	queryResult, err := adapter.GetEmbedding(ctx, docRequest)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(queryResult.Data) != 2 {
		t.Fatalf("expected 2 embeddings, got %d", len(queryResult.Data))
	}

	if queryResult.Data[0].Index != 0 || queryResult.Data[1].Index != 1 {
		t.Fatalf("unexpected indices in result: %v", queryResult.Data)
	}

	if len(queryResult.Data[0].Embedding) == 0 || len(queryResult.Data[1].Embedding) == 0 {
		t.Fatalf("expected non-empty embeddings, got empty embeddings")
	}

	queryRequest := llm.EmbeddingRequest{
		ModelID:   "7a571008-601a-463f-a3b9-1ba48de38984-voyage-large-2",
		Input:     []string{"What is the capital of Taiwan?"},
		InputType: lo.ToPtr("query"),
	}

	result, err := adapter.GetEmbedding(ctx, queryRequest)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Data) != 1 {
		t.Fatalf("expected 1 embedding, got %d", len(result.Data))
	}

	if result.Data[0].Index != 0 {
		t.Fatalf("unexpected indices in result: %v", result.Data)
	}

	if len(result.Data[0].Embedding) == 0 {
		t.Fatalf("expected non-empty embeddings, got empty embeddings")
	}

	// Function to compute cosine similarity
	cosineSimilarity := func(vec1, vec2 []float64) float64 {
		if len(vec1) != len(vec2) {
			t.Fatalf("vectors must be of same length")
		}

		var dotProduct, normA, normB float64
		for i := 0; i < len(vec1); i++ {
			dotProduct += vec1[i] * vec2[i]
			normA += vec1[i] * vec1[i]
			normB += vec2[i] * vec2[i]
		}

		if normA == 0 || normB == 0 {
			t.Fatalf("vectors must not be zero vectors")
		}

		return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
	}

	// Compute cosine similarity between the two document embeddings
	docEmbedding1 := queryResult.Data[0].Embedding
	docEmbedding2 := queryResult.Data[1].Embedding
	queryEmbedding := result.Data[0].Embedding

	querySimilarity1 := cosineSimilarity(docEmbedding1, queryEmbedding)
	querySimilarity2 := cosineSimilarity(docEmbedding2, queryEmbedding)

	fmt.Printf("Cosine similarity between the two document embeddings and the query are: %f, %f", querySimilarity1, querySimilarity2)
}

func TestVoyageEmbeddingAdapter_GetEmbedding_With_Different_Type(t *testing.T) {
	t.Skip()

	adapter := NewVoyageEmbeddingAdapter()
	ctx := context.Background()

	docRequest := llm.EmbeddingRequest{
		ModelID:   "7a571008-601a-463f-a3b9-1ba48de38984-voyage-large-2",
		Input:     []string{"The capital of Taiwan is Taipei."},
		InputType: lo.ToPtr("document"),
	}

	queryRequest := llm.EmbeddingRequest{
		ModelID:   "7a571008-601a-463f-a3b9-1ba48de38984-voyage-large-2",
		Input:     []string{"The capital of Taiwan is Taipei."},
		InputType: lo.ToPtr("query"),
	}

	docResult, err := adapter.GetEmbedding(ctx, docRequest)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	queryResult, err := adapter.GetEmbedding(ctx, queryRequest)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(docResult.Data) != 1 {
		t.Fatalf("expected 1 embedding, got %d", len(docResult.Data))
	}

	if len(queryResult.Data) != 1 {
		t.Fatalf("expected 1 embedding, got %d", len(queryResult.Data))
	}

	fmt.Printf("docResult: %v, queryResult: %v", docResult, queryResult)
}
