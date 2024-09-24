package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/samber/lo"
	"gitlab.com/navyx/ai/maos/maos-core/llm"
	"gitlab.com/navyx/ai/maos/maos-core/util"
)

var VoyageEmbeddingModelMap = map[string]string{
	"c6bbe66a-ac3e-4687-99b6-3a7d64bdf97a-voyage-large-2-instruct": "voyage-large-2-instruct",
	"84586745-b0ce-4d54-860f-bdbf579224d4-voyage-finance-2":        "voyage-finance-2",
	"369e2def-108c-4355-aea6-25cd3dc90b6f-voyage-multilingual-2":   "voyage-multilingual-2",
	"e9b1d228-ec9d-4972-bfca-ce9593e80866-voyage-law-2":            "voyage-law-2",
	"fb6aa59b-6791-41c5-9262-0ebab269a196-voyage-code-2":           "voyage-code-2",
	"7a571008-601a-463f-a3b9-1ba48de38984-voyage-large-2":          "voyage-large-2",
	"285d60de-ac64-4461-a59d-be86d82fff75-voyage-2":                "voyage-2",
}

type VoyageEmbeddingAdapter struct {
	httpClient *http.Client
	apiKey     string
}

func NewVoyageEmbeddingAdapter() *VoyageEmbeddingAdapter {
	return &VoyageEmbeddingAdapter{
		httpClient: &http.Client{},
		apiKey:     os.Getenv("VOYAGE_API_KEY"),
	}
}

func (a *VoyageEmbeddingAdapter) GetEmbedding(ctx context.Context, request llm.EmbeddingRequest) (llm.EmbeddingResult, error) {
	voyageRequest, err := ToVoyageEmbeddingRequest(request)
	if err != nil {
		return llm.EmbeddingResult{}, err
	}

	body, err := util.NewObjectJsonReader(voyageRequest)
	if err != nil {
		return llm.EmbeddingResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", os.Getenv("VOYAGE_ENDPOINT")+"/embeddings", body)
	if err != nil {
		return llm.EmbeddingResult{}, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.apiKey))
	req.Header.Add("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return llm.EmbeddingResult{}, err
	}
	defer resp.Body.Close()

	status := resp.StatusCode
	if status != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return llm.EmbeddingResult{}, fmt.Errorf("status code %d", status)
		}
		return llm.EmbeddingResult{}, fmt.Errorf("status code %d, body %s", status, string(body))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return llm.EmbeddingResult{}, err
	}

	var response VoyageEmbeddingResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return llm.EmbeddingResult{}, err
	}

	return llm.EmbeddingResult{
		Data: lo.Map(
			response.Data,
			func(item VoyageEmbeddingData, _ int) llm.Embedding {
				return llm.Embedding{
					Index:     item.Index,
					Embedding: item.Embedding,
				}
			},
		),
	}, nil
}

func GetVoyageEmbeddingModelByModelID(modelID string) (string, error) {
	model, ok := VoyageEmbeddingModelMap[modelID]
	if !ok {
		return "", fmt.Errorf("model not found for model ID %s", modelID)
	}
	return model, nil
}

func ToVoyageEmbeddingRequest(request llm.EmbeddingRequest) (VoyageEmbeddingRequest, error) {
	model, err := GetVoyageEmbeddingModelByModelID(request.ModelID)
	if err != nil {
		return VoyageEmbeddingRequest{}, err
	}

	return VoyageEmbeddingRequest{
		Model:     model,
		Input:     request.Input,
		InputType: request.InputType,
	}, nil
}
