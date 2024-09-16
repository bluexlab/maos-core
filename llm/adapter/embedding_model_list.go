package adapter

import "gitlab.com/navyx/ai/maos/maos-core/llm"

var embeddingModelList = []llm.EmbeddingModel{
	{
		ID:        "6baf223e-d321-41d2-bb33-c8328320e1e3-azure-text-embedding-ada-002",
		Provider:  PROVIDER_AZURE,
		Name:      "Azure text-embedding-ada-002",
		Dimension: 1536,
	},
	{
		ID:        "d68a09df-3589-4273-b032-04488d9b230d-azure-text-embedding-3-small",
		Provider:  PROVIDER_AZURE,
		Name:      "Azure text-embedding-3-small",
		Dimension: 1536,
	},
	{
		ID:        "38f8d506-b9ab-4c5b-b7c7-051fd4849bbf-azure-text-embedding-3-large",
		Provider:  PROVIDER_AZURE,
		Name:      "Azure text-embedding-3-large",
		Dimension: 3072,
	},
	{
		ID:        "c6bbe66a-ac3e-4687-99b6-3a7d64bdf97a-voyage-large-2-instruct",
		Provider:  PROVIDER_VOYAGE,
		Name:      "voyage-large-2-instruct",
		Dimension: 1024,
	},
	{
		ID:        "84586745-b0ce-4d54-860f-bdbf579224d4-voyage-finance-2",
		Provider:  PROVIDER_VOYAGE,
		Name:      "voyage-finance-2",
		Dimension: 1024,
	},
	{
		ID:        "369e2def-108c-4355-aea6-25cd3dc90b6f-voyage-multilingual-2",
		Provider:  PROVIDER_VOYAGE,
		Name:      "voyage-multilingual-2",
		Dimension: 1024,
	},
	{
		ID:        "e9b1d228-ec9d-4972-bfca-ce9593e80866-voyage-law-2",
		Provider:  PROVIDER_VOYAGE,
		Name:      "voyage-law-2",
		Dimension: 1024,
	},
	{
		ID:        "fb6aa59b-6791-41c5-9262-0ebab269a196-voyage-code-2",
		Provider:  PROVIDER_VOYAGE,
		Name:      "voyage-code-2",
		Dimension: 1536,
	},
	{
		ID:        "7a571008-601a-463f-a3b9-1ba48de38984-voyage-large-2",
		Provider:  PROVIDER_VOYAGE,
		Name:      "voyage-large-2",
		Dimension: 1536,
	},
	{
		ID:        "285d60de-ac64-4461-a59d-be86d82fff75-voyage-2",
		Provider:  PROVIDER_VOYAGE,
		Name:      "voyage-2",
		Dimension: 1024,
	},
}

var embeddingModelMap = map[string]llm.EmbeddingModel{}

func init() {
	embeddingModelMap = make(map[string]llm.EmbeddingModel)
	for _, model := range embeddingModelList {
		embeddingModelMap[model.ID] = model
	}
}

func GetEmbeddingModelList() []llm.EmbeddingModel {
	return embeddingModelList
}
func GetEmbeddingModelByID(id string) (llm.EmbeddingModel, bool) {
	model, ok := embeddingModelMap[id]
	return model, ok
}
