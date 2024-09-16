package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/samber/lo"
	"gitlab.com/navyx/ai/maos/maos-core/llm"
)

// AzureEmbeddingModelDeploymentMap is a map of model ID to Azure deployment name.
// The predefined deployment name should be set in the environment variable.
// After package initialization, the deployment name will be replaced by the real value.
var AzureEmbeddingModelDeploymentMap = map[string]string{
	"6baf223e-d321-41d2-bb33-c8328320e1e3-azure-text-embedding-ada-002": "AOAI_TEXT_EMBEDDING_ADA_002_DEPLOYMENT",
	"d68a09df-3589-4273-b032-04488d9b230d-azure-text-embedding-3-small": "AOAI_TEXT_EMBEDDING_3_SMALL_DEPLOYMENT",
	"38f8d506-b9ab-4c5b-b7c7-051fd4849bbf-azure-text-embedding-3-large": "AOAI_TEXT_EMBEDDING_3_LARGE_DEPLOYMENT",
}

func init() {
	newMap := make(map[string]string)
	for k, v := range AzureEmbeddingModelDeploymentMap {
		deployment := os.Getenv(v)
		if deployment == "" {
			slog.Error("deployment not found for model", "name", k)
		}
		newMap[k] = deployment
	}
	AzureEmbeddingModelDeploymentMap = newMap
}

type AzureEmbeddingAdapter struct {
	client *azopenai.Client
}

func NewAzureEmbeddingAdapter(endpoint, credential string) (*AzureEmbeddingAdapter, error) {
	client, err := azopenai.NewClientWithKeyCredential(
		endpoint,
		azcore.NewKeyCredential(credential),
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &AzureEmbeddingAdapter{client: client}, nil
}

func GetAzureEmbeddingDeploymentByModelID(modelID string) (string, error) {
	deploymentName, ok := AzureEmbeddingModelDeploymentMap[modelID]
	if !ok {
		return "", fmt.Errorf("deployment not found for model %s", modelID)
	}
	return deploymentName, nil
}

func (a *AzureEmbeddingAdapter) GetEmbedding(ctx context.Context, request llm.EmbeddingRequest) (llm.EmbeddingResult, error) {
	deploymentName, err := GetAzureEmbeddingDeploymentByModelID(request.ModelID)
	if err != nil {
		return llm.EmbeddingResult{}, err
	}

	body := azopenai.EmbeddingsOptions{
		DeploymentName: to.Ptr(deploymentName),
		Input:          request.Input,
	}

	resp, err := a.client.GetEmbeddings(ctx, body, nil)
	if err != nil {
		return llm.EmbeddingResult{}, err
	}

	return llm.EmbeddingResult{
		Data: lo.Map(resp.Embeddings.Data, func(item azopenai.EmbeddingItem, _ int) llm.Embedding {
			return llm.Embedding{
				Embedding: lo.Map(item.Embedding, func(item float32, _ int) float64 {
					return float64(item)
				}),
				Index: int(*item.Index),
			}
		}),
	}, nil
}
