package adapter

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/samber/lo"
	"gitlab.com/navyx/ai/maos/maos-core/llm"
)

type AzureAdapter struct {
	client *azopenai.Client
}

// AzureModelDeploymentMap is a map of model ID to Azure deployment name.
// The predefined deployment name should be set in the environment variable.
// After package initialization, the deployment name will be replaced by the real value.
var AzureModelDeploymentMap = map[string]string{
	"5a265146-4e05-4cd7-a0a9-9adda7bf7a38-azure-gpt4o": "AOAI_GPT4O_DEPLOYMENT",
	"bdf5c21b-ad28-4096-9bca-667927b5c742-azure-gpt4":  "AOAI_GPT4_DEPLOYMENT",
}

func init() {
	newMap := make(map[string]string)
	for k, v := range AzureModelDeploymentMap {
		deployment := os.Getenv(v)
		if deployment == "" {
			slog.Error("deployment not found for model", "name", k)
		}
		newMap[k] = deployment
	}

	AzureModelDeploymentMap = newMap
}

func NewAzureAdapter(endpoint, credential string) (*AzureAdapter, error) {
	client, err := azopenai.NewClientWithKeyCredential(
		endpoint,
		azcore.NewKeyCredential(credential),
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &AzureAdapter{client: client}, nil
}

func (a *AzureAdapter) GetCompletion(ctx context.Context, request llm.CompletionRequest) (llm.CompletionResult, error) {
	slog.Info("AzureAdapter Getting completion", "modelID", request.ModelID)

	deploymentName, err := GetAzureDeploymentByModelID(request.ModelID)
	if err != nil {
		slog.Error("AzureAdapter GetAzureDeploymentByModelID failed", "error", err)
		return llm.CompletionResult{}, err
	}

	body := azopenai.ChatCompletionsOptions{
		DeploymentName: to.Ptr(deploymentName),
		MaxTokens:      request.MaxTokens,
		Temperature:    request.Temperature,
	}
	if len(request.StopSequences) != 0 {
		body.Stop = request.StopSequences
	}
	for _, msg := range request.Messages {
		classifications, err := ToChatRequestMessageClassification(msg)
		if err != nil {
			return llm.CompletionResult{}, err
		}
		body.Messages = append(body.Messages, classifications...)
	}
	for _, tool := range request.Tools {
		body.Tools = append(body.Tools, &azopenai.ChatCompletionsFunctionToolDefinition{
			Type: to.Ptr("function"),
			Function: &azopenai.FunctionDefinition{
				Name:        to.Ptr(tool.Name),
				Description: to.Ptr(tool.Description),
				Parameters:  tool.Parameters,
			},
		})
	}

	resp, err := a.client.GetChatCompletions(ctx, body, nil)
	if err != nil {
		slog.Error("AzureAdapter Getting completion", "error", err)
		return llm.CompletionResult{}, err
	}
	return FromGetChatCompletionsResponse(resp), nil
}

func GetAzureDeploymentByModelID(modelID string) (string, error) {
	deploymentName, ok := AzureModelDeploymentMap[modelID]
	if !ok {
		return "", fmt.Errorf("model not found")
	}

	return deploymentName, nil
}

func ToChatRequestMessageClassification(msg llm.Message) ([]azopenai.ChatRequestMessageClassification, error) {
	chatRole := azopenai.ChatRole(msg.Role)
	if chatRole == azopenai.ChatRoleUser {
		contents := lo.Map(
			msg.Content,
			func(content llm.Content, _ int) azopenai.ChatCompletionRequestMessageContentPartClassification {
				return ToChatRequestMessageContent(content)
			},
		)
		return []azopenai.ChatRequestMessageClassification{
			&azopenai.ChatRequestUserMessage{
				Content: azopenai.NewChatRequestUserMessageContent(contents),
			},
		}, nil
	} else if chatRole == azopenai.ChatRoleAssistant {
		results := lo.Map(
			msg.Content,
			func(content llm.Content, _ int) azopenai.ChatRequestMessageClassification {
				assistantMsg := &azopenai.ChatRequestAssistantMessage{}
				if content.ToolCall != nil {
					assistantMsg.ToolCalls = []azopenai.ChatCompletionsToolCallClassification{
						&azopenai.ChatCompletionsFunctionToolCall{
							ID:   to.Ptr(content.ToolCall.ID),
							Type: to.Ptr("function"),
							Function: &azopenai.FunctionCall{
								Name:      to.Ptr(content.ToolCall.FunctionName),
								Arguments: to.Ptr(content.ToolCall.Arguments),
							},
						},
					}
				} else {
					assistantMsg.Content = to.Ptr(content.Text)
				}

				return assistantMsg
			},
		)
		return results, nil
	} else if chatRole == azopenai.ChatRoleSystem {
		results := lo.Map(
			msg.Content,
			func(content llm.Content, _ int) azopenai.ChatRequestMessageClassification {
				return &azopenai.ChatRequestSystemMessage{
					Content: to.Ptr(content.Text),
				}
			},
		)
		return results, nil
	} else if chatRole == azopenai.ChatRoleTool {
		results := lo.Map(
			msg.Content,
			func(content llm.Content, _ int) azopenai.ChatRequestMessageClassification {
				return &azopenai.ChatRequestToolMessage{
					Content:    to.Ptr(content.ToolResult.Result),
					ToolCallID: to.Ptr(content.ToolResult.ID),
				}
			},
		)
		return results, nil
	}
	return nil, fmt.Errorf("invalid role")
}

func ToChatRequestMessageContent(content llm.Content) azopenai.ChatCompletionRequestMessageContentPartClassification {
	// Image Content
	if len(content.Image) != 0 {
		enocdeLen := base64.StdEncoding.EncodedLen(len(content.Image))
		encoded := make([]byte, enocdeLen)
		base64.StdEncoding.Encode(encoded, content.Image)
		mimeType := http.DetectContentType(content.Image)
		dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
		return &azopenai.ChatCompletionRequestMessageContentPartImage{
			ImageURL: &azopenai.ChatCompletionRequestMessageContentPartImageURL{
				URL: to.Ptr(dataURL),
			},
		}
	}

	if content.ImageURL != "" {
		return &azopenai.ChatCompletionRequestMessageContentPartImage{
			ImageURL: &azopenai.ChatCompletionRequestMessageContentPartImageURL{
				URL: to.Ptr(content.ImageURL),
			},
		}
	}

	// Text Context
	return &azopenai.ChatCompletionRequestMessageContentPartText{
		Text: to.Ptr(content.Text),
	}
}

func FromGetChatCompletionsResponse(resp azopenai.GetChatCompletionsResponse) llm.CompletionResult {
	// Convert azopenai.ChatChoice into llm.Message.
	getRolesAndContents := func(choice azopenai.ChatChoice) (string, []llm.Content) {
		role := ""
		if choice.Message != nil && choice.Message.Role != nil {
			role = string(*choice.Message.Role)
		}
		if role == "" {
			return "", nil
		}

		contents := make([]llm.Content, 0)
		if choice.Message.Content != nil {
			contents = append(contents, llm.Content{
				Text: string(*choice.Message.Content),
			})
		}

		for _, callInterface := range choice.Message.ToolCalls {
			call, _ := callInterface.(*azopenai.ChatCompletionsFunctionToolCall)
			if call != nil {
				contents = append(contents, llm.Content{
					ToolCall: &llm.ToolCall{
						ID:           *call.ID,
						FunctionName: *call.Function.Name,
						Arguments:    *call.Function.Arguments,
					},
				})
			}
		}

		return role, contents
	}

	return llm.CompletionResult{
		Messages: lo.Map(
			resp.Choices,
			func(msg azopenai.ChatChoice, _ int) llm.Message {
				role, contents := getRolesAndContents(msg)
				return llm.Message{
					Role:    role,
					Content: contents,
				}
			},
		),
	}
}
