package adapter_test

import (
	"context"
	"os"
	"testing"

	"gitlab.com/navyx/ai/maos/maos-core/llm"
	"gitlab.com/navyx/ai/maos/maos-core/llm/adapter"
)

func TestAzureOpenAIWithText(t *testing.T) {
	t.Skip()
	client, err := adapter.NewAzureAdapter(
		os.Getenv("AOAI_ENDPOINT"),
		os.Getenv("AOAI_API_KEY"),
	)
	if err != nil {
		t.Fatal(err)
	}

	req := llm.CompletionRequest{
		ModelID: "5a265146-4e05-4cd7-a0a9-9adda7bf7a38-azure-gpt4o",
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

func TestAzureOpenAIWithImage(t *testing.T) {
	t.Skip()
	img, err := os.ReadFile("test_data/phoenix.jpg")
	if err != nil {
		t.Fatal(err)
	}

	client, err := adapter.NewAzureAdapter(
		os.Getenv("AOAI_ENDPOINT"),
		os.Getenv("AOAI_API_KEY"),
	)
	if err != nil {
		t.Fatal(err)
	}

	req := llm.CompletionRequest{
		ModelID: "5a265146-4e05-4cd7-a0a9-9adda7bf7a38-azure-gpt4o",
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
