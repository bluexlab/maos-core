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

// ModelDeploymentMap is a map of model ID to Azure deployment name.
// The predefined deployment name should be set in the environment variable.
// After package initialization, the deployment name will be replaced by the real value.
var ModelDeploymentMap = map[string]string{
	"5a265146-4e05-4cd7-a0a9-9adda7bf7a38-azure-gpt4o": "AOAI_GPT4O_DEPLOYMENT",
	"bdf5c21b-ad28-4096-9bca-667927b5c742-azure-gpt4":  "AOAI_GPT4_DEPLOYMENT",
}

func init() {
	newMap := make(map[string]string)
	for k, v := range ModelDeploymentMap {
		deployment := os.Getenv(v)
		if deployment == "" {
			slog.Error("deployment not found for model", "name", k)
		}
		newMap[k] = deployment
	}

	ModelDeploymentMap = newMap
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
	deploymentName, err := GetAzureDeploymentByModelID(request.ModelID)
	if err != nil {
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

	resp, err := a.client.GetChatCompletions(ctx, body, nil)
	if err != nil {
		return llm.CompletionResult{}, err
	}
	return FromGetChatCompletionsResponse(resp), nil
}

func GetAzureDeploymentByModelID(modelID string) (string, error) {
	deploymentName, ok := ModelDeploymentMap[modelID]
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
				return &azopenai.ChatRequestAssistantMessage{
					Content: to.Ptr(content.Text),
				}
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
	getRoleAndContent := func(choice azopenai.ChatChoice) (string, string) {
		role, content := "", ""
		if choice.Message != nil && choice.Message.Role != nil {
			role = string(*choice.Message.Role)
		}
		if choice.Message != nil && choice.Message.Content != nil {
			content = string(*choice.Message.Content)
		}
		return role, content
	}

	return llm.CompletionResult{
		Messages: lo.Map(
			resp.Choices,
			func(msg azopenai.ChatChoice, _ int) llm.Message {
				role, content := getRoleAndContent(msg)
				return llm.Message{
					Role: role,
					Content: []llm.Content{
						{
							Text: content,
						},
					},
				}
			},
		),
	}
}
