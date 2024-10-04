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

func TestAnthropicClaudeWithText(t *testing.T) {
	t.Skip()
	client := adapter.NewAnthropicAdapter(os.Getenv("ANTHROPIC_API_KEY"))

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

func TestAnthropicClaudeWitTool(t *testing.T) {
	t.Skip()

	client := adapter.NewAnthropicAdapter(os.Getenv("ANTHROPIC_API_KEY"))

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

	maxTokens := int32(1000)
	req := llm.CompletionRequest{
		ModelID:   "93d07ee3-c9fb-4f0e-9fc1-df1a7af10b6c-anthropic-claude-3.5-sonnet-20240620",
		MaxTokens: &maxTokens,
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
	var err error
	var keepRunning = true
	for keepRunning {
		result, err = client.GetCompletion(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}

		for _, m := range result.Messages {
			var newMessage *llm.Message

			for _, c := range m.Content {
				if newMessage == nil {
					newMessage = &llm.Message{
						Role: m.Role,
					}
				}
				keepRunning = c.ToolCall != nil
				newMessage.Content = append(newMessage.Content, c)
				if c.ToolCall == nil {
					continue
				}

				sum, err := add(json.RawMessage(c.ToolCall.Arguments))
				if err != nil {
					t.Fatal(err)
				}
				req.Messages = append(req.Messages, *newMessage)
				newMessage = nil

				toolResultMessage := llm.Message{
					Role: "tool",
					Content: []llm.Content{
						{
							ToolResult: &llm.ToolResult{
								ID:     c.ToolCall.ID,
								Result: sum,
							},
						},
					},
				}
				req.Messages = append(req.Messages, toolResultMessage)
			}

			if newMessage != nil {
				req.Messages = append(req.Messages, *newMessage)
			}
		}
	}
	fmt.Println(result.Messages[0].Content[0].Text)
}

func TestAnthropicClaudeWithImage(t *testing.T) {
	t.Skip()
	img, err := os.ReadFile("test_data/phoenix.jpg")
	if err != nil {
		t.Fatal(err)
	}

	client := adapter.NewAnthropicAdapter(os.Getenv("ANTHROPIC_API_KEY"))

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
