package adapter_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/samber/lo"
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

func TestAzureOpenAIWithTool(t *testing.T) {
	t.Skip()

	client, err := adapter.NewAzureAdapter(
		os.Getenv("AOAI_ENDPOINT"),
		os.Getenv("AOAI_API_KEY"),
	)
	if err != nil {
		t.Fatal(err)
	}

	add := func(argument json.RawMessage) (string, error) {
		var args struct {
			Nums []float64 `json:"nums"`
		}
		if err := json.Unmarshal(argument, &args); err != nil {
			return "", err
		}

		numbers := lo.Map(args.Nums, func(n float64, _ int) int64 { return int64(n) })
		sum := lo.Sum(numbers)

		return fmt.Sprintf("%d", sum), nil
	}

	req := llm.CompletionRequest{
		ModelID: "5a265146-4e05-4cd7-a0a9-9adda7bf7a38-azure-gpt4o",
		Messages: []llm.Message{
			{
				Role: "system",
				Content: []llm.Content{
					{Text: "The assistant will help the user to calculate the total amount of an order."},
				},
			},
			{
				Role: "user",
				Content: []llm.Content{
					{Text: `御品佛跳牆 980
清蒸籠虎斑 880
鰻魚香米糕 880
淮山養生雞 780
富貴雙方   380
豬蹄筍絲  450
貴妃鮑   150
蒜蓉腿   220
叉燒肉 430
御炸蝦球 249
花枝丸 100
螺旋貝 105
元進莊油雞 430
北海道貝柱 880
金園排骨 300元`},
				},
			},
		},
		Tools: []llm.Tool{
			{
				Name:        "add",
				Description: "Add numbers",
				Parameters:  []byte(`{"type":"object","properties":{"nums":{"type":"array", "items": {"type": "number"}}}}`),
			},
		},
	}

	var result llm.CompletionResult
	var keepRunning = true
	for keepRunning {
		result, err = client.GetCompletion(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}

		contents := lo.FlatMap(result.Messages, func(m llm.Message, _ int) []llm.Content {
			return m.Content
		})
		if len(contents) == 0 {
			keepRunning = false
			break
		}

		// toolResults := make([]llm.Content, 0)
		for _, content := range contents {
			if content.ToolCall == nil {
				keepRunning = false
				break
			}

			req.Messages = append(req.Messages, llm.Message{
				Role:    "assistant",
				Content: []llm.Content{content},
			})

			sum, err := add(json.RawMessage(content.ToolCall.Arguments))
			if err != nil {
				t.Fatal(err)
			}

			req.Messages = append(req.Messages, llm.Message{
				Role: "tool",
				Content: []llm.Content{
					{
						ToolResult: &llm.ToolResult{
							ID:     content.ToolCall.ID,
							Result: sum,
						},
					},
				},
			})
		}
	}
	fmt.Println(result.Messages[0].Content[0].Text)
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

func TestAzureOpenAIWithImageURL(t *testing.T) {
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
					{Text: "Describe what you see in this image."},
					{ImageURL: "https://gw.alicdn.com/imgextra/O1CN01AZsz9a1nzKZ6LmwMX_!!6000000005160-2-yinhe.png_468x468Q75.jpg_.webp"},
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
