package adapter_test

import (
	"context"
	"os"
	"testing"

	"gitlab.com/navyx/ai/maos/maos-core/llm"
	"gitlab.com/navyx/ai/maos/maos-core/llm/adapter"
)

func TestAnthropicClaudeWithText(t *testing.T) {
	t.Skip()
	client := adapter.NewAnthropicAdapter()

	maxTokens := int32(1000)
	req := llm.CompletionRequest{
		ModelID:   "93d07ee3-c9fb-4f0e-9fc1-df1a7af10b6c-anthropic-claude-3.5-sonnet-20240620",
		MaxTokens: &maxTokens,
		Messages: []llm.Message{
			{
				Role: "system",
				Content: []llm.Content{
					{Text: "A chatbot that helps you with your daily tasks."},
				},
			},
			{
				Role: "user",
				Content: []llm.Content{
					{Text: "What time is it?"},
				},
			},
		},
	}
	result, err := client.GetCompletion(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	_ = result
}

func TestAnthropicClaudeWithImage(t *testing.T) {
	t.Skip()
	img, err := os.ReadFile("test_data/phoenix.jpg")
	if err != nil {
		t.Fatal(err)
	}

	client := adapter.NewAnthropicAdapter()

	maxTokens := int32(1000)
	req := llm.CompletionRequest{
		ModelID:   "93d07ee3-c9fb-4f0e-9fc1-df1a7af10b6c-anthropic-claude-3.5-sonnet-20240620",
		MaxTokens: &maxTokens,
		Messages: []llm.Message{
			{
				Role: "system",
				Content: []llm.Content{
					{Text: "A chatbot that helps you with your daily tasks."},
				},
			},
			{
				Role: "user",
				Content: []llm.Content{
					{Text: "Describe what you see in this image."},
					{Image: img},
				},
			},
		},
	}
	result, err := client.GetCompletion(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	_ = result
}
