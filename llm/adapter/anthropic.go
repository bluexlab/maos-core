package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"gitlab.com/navyx/ai/maos/maos-core/llm"
	"gitlab.com/navyx/ai/maos/maos-core/util"
)

var AnthropicModelMap = map[string]string{
	"3db6db92-a091-4944-9f7e-9d43e70218d3-anthropic-claude-3-opus-20240229":     "claude-3-opus-20240229",
	"93d07ee3-c9fb-4f0e-9fc1-df1a7af10b6c-anthropic-claude-3.5-sonnet-20240620": "claude-3-5-sonnet-20240620",
}

type _AnthropicAdapter struct {
	httpClient *http.Client
	apiKey     string
}

func NewAnthropicAdapter(apiKey string) *_AnthropicAdapter {
	return &_AnthropicAdapter{
		httpClient: &http.Client{},
		apiKey:     apiKey,
	}
}

func (a *_AnthropicAdapter) GetCompletion(ctx context.Context, request llm.CompletionRequest) (llm.CompletionResult, error) {
	msgRequest, err := ToAnthropicMessageRequest(request)
	if err != nil {
		return llm.CompletionResult{}, err
	}
	requestBody, err := util.NewObjectJsonReader(&msgRequest)
	if err != nil {
		return llm.CompletionResult{}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", requestBody)
	if err != nil {
		return llm.CompletionResult{}, err
	}
	httpRequest.Header.Add("x-api-key", a.apiKey)
	httpRequest.Header.Add("anthropic-version", "2023-06-01")
	httpRequest.Header.Add("Content-Type", "application/json")

	httpResponse, err := a.httpClient.Do(httpRequest)
	if err != nil {
		return llm.CompletionResult{}, err
	}
	defer httpResponse.Body.Close()

	if statusCodeCategory := httpResponse.StatusCode / 100; statusCodeCategory != 2 && statusCodeCategory != 4 {
		return llm.CompletionResult{}, fmt.Errorf("unexpected status code %d", httpResponse.StatusCode)
	}

	responseBody := &MessageResponse{}
	if err := json.NewDecoder(httpResponse.Body).Decode(responseBody); err != nil {
		return llm.CompletionResult{}, err
	}
	if responseBody.Error != nil {
		return llm.CompletionResult{}, fmt.Errorf("Anthropic API error: %s", *responseBody.Error)
	}
	if len(responseBody.Content) == 0 {
		return llm.CompletionResult{}, fmt.Errorf("no content in response")
	}

	responseText := &strings.Builder{}
	for _, c := range responseBody.Content {
		if c.Text != nil {
			responseText.WriteString(*c.Text)
		}
	}
	result := llm.CompletionResult{
		Messages: []llm.Message{
			{
				Role: responseBody.Role,
			},
		},
	}
	for _, c := range responseBody.Content {
		content, err := FromAnthropicContentMessage(c)
		if err != nil {
			return llm.CompletionResult{}, err
		}
		result.Messages[0].Content = append(result.Messages[0].Content, content)
	}

	return result, nil
}

func GetAnthropicLLMModelByModelID(modelID string) (string, error) {
	model, ok := AnthropicModelMap[modelID]
	if !ok {
		return "", fmt.Errorf("model not found for model ID %s", modelID)
	}
	return model, nil
}

func ToAnthropicMessageRequest(req llm.CompletionRequest) (MessageRequest, error) {
	model, err := GetAnthropicLLMModelByModelID(req.ModelID)
	if err != nil {
		return MessageRequest{}, err
	}

	getContentsText := func(contents []llm.Content) *string {
		if len(contents) == 0 {
			return nil
		}

		strBuilder := strings.Builder{}
		for _, c := range contents {
			if c.Text == "" {
				continue
			}
			strBuilder.WriteString(c.Text)
		}
		str := strBuilder.String()
		return &str
	}
	request := MessageRequest{
		Model:         model,
		StopSequences: req.StopSequences,
	}
	if req.MaxTokens != nil {
		request.MaxTokens = *req.MaxTokens
	}
	if req.Temperature != nil {
		request.Temperature = req.Temperature
	}

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			request.System = getContentsText(msg.Content)
			continue
		}

		reqMsg, err := ToAnthropicMessage(msg)
		if err != nil {
			return MessageRequest{}, err
		}
		request.Messages = append(request.Messages, reqMsg)
	}

	for _, tool := range req.Tools {
		request.Tools = append(request.Tools, Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.Parameters,
		})
	}

	return request, nil
}

func ToAnthropicMessage(msg llm.Message) (Message, error) {
	result := Message{
		Role: msg.Role,
	}

	for _, c := range msg.Content {
		content := Content{}
		if c.Text != "" {
			content.Text = &(c.Text)
			content.Type = "text"
		} else if c.ImageURL != "" {
			return Message{}, fmt.Errorf("image URL is not supported")
		} else if len(c.Image) > 0 {
			content.Type = "image"
			content.Source = &Source{
				Type:      "base64",
				MediaType: http.DetectContentType(c.Image),
				Data:      c.Image,
			}
		} else if c.ToolCall != nil {
			content.Type = "tool_use"
			content.Id = to.Ptr(c.ToolCall.ID)
			content.Name = to.Ptr(c.ToolCall.FunctionName)
			input := json.RawMessage(c.ToolCall.Arguments)
			content.Input = &input
		} else if c.ToolResult != nil {
			result.Role = "user"
			content.Type = "tool_result"
			content.ToolUseId = to.Ptr(c.ToolResult.ID)
			content.IsError = to.Ptr(c.ToolResult.IsError)
			content.Content = []Content{
				{
					Type: "text",
					Text: to.Ptr(c.ToolResult.Result),
				},
			}
		} else {
			return Message{}, fmt.Errorf("content must have text or image")
		}

		result.Content = append(result.Content, content)
	}

	return result, nil
}

func FromAnthropicContentMessage(content Content) (llm.Content, error) {
	result := llm.Content{}

	switch content.Type {
	case "text":
		if content.Text == nil {
			return llm.Content{}, fmt.Errorf("text content is nil")
		}
		result.Text = *content.Text
	case "image":
		if content.Source == nil {
			return llm.Content{}, fmt.Errorf("image source is nil")
		}
		if content.Source.Type != "base64" {
			return llm.Content{}, fmt.Errorf("invalid image source type %s", content.Source.Type)
		}
		result.Image = content.Source.Data
	case "tool_use":
		if content.Id == nil {
			return llm.Content{}, fmt.Errorf("tool use ID is nil")
		}
		if content.Name == nil {
			return llm.Content{}, fmt.Errorf("tool use name is nil")
		}
		if content.Input == nil {
			return llm.Content{}, fmt.Errorf("tool use input is nil")
		}
		result.ToolCall = &llm.ToolCall{
			ID:           *content.Id,
			FunctionName: *content.Name,
			Arguments:    string(*content.Input),
		}
	default:
		return llm.Content{}, fmt.Errorf("invalid content type %s", content.Type)
	}

	return result, nil
}
