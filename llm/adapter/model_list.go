package adapter

import "gitlab.com/navyx/ai/maos/maos-core/llm"

const (
	PROVIDER_AZURE = "Azure"
)

var modelList = []llm.Model{
	{
		ID:       "5a265146-4e05-4cd7-a0a9-9adda7bf7a38-azure-gpt4o",
		Provider: PROVIDER_AZURE,
		Name:     "Azure gpt-4o",
	},
	{
		ID:       "bdf5c21b-ad28-4096-9bca-667927b5c742-azure-gpt4",
		Provider: PROVIDER_AZURE,
		Name:     "Azure gpt-4",
	},
}

var modelMap = map[string]llm.Model{}

func init() {
	modelMap = make(map[string]llm.Model)
	for _, model := range modelList {
		modelMap[model.ID] = model
	}
}

func GetModelByID(id string) (llm.Model, bool) {
	model, ok := modelMap[id]
	return model, ok
}

func GetModelList() []llm.Model {
	return modelList
}
