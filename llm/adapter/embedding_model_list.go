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
