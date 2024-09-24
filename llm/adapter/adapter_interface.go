package adapter

import (
	"context"

	"gitlab.com/navyx/ai/maos/maos-core/llm"
)

type LLMAdapter interface {
	GetCompletion(ctx context.Context, request llm.CompletionRequest) (llm.CompletionResult, error)
}
