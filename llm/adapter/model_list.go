package adapter

import (
	"fmt"

	"gitlab.com/navyx/ai/maos/maos-core/llm"
)

const (
	PROVIDER_AZURE     = "Azure"
	PROVIDER_ANTHROPIC = "Anthropic"
	PROVIDER_VOYAGE    = "VoyageAI"
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
	{
		ID:       "3db6db92-a091-4944-9f7e-9d43e70218d3-anthropic-claude-3-opus-20240229",
		Provider: PROVIDER_ANTHROPIC,
		Name:     "Anthropic Claude 3 Opus 20240229",
	},
	{
		ID:       "93d07ee3-c9fb-4f0e-9fc1-df1a7af10b6c-anthropic-claude-3.5-sonnet-20240620",
		Provider: PROVIDER_ANTHROPIC,
		Name:     "Anthropic Claude 3.5 Sonnet 20240620",
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

// CreateAdapter creates an adapter for the given model ID
// This is a variable so we can inject it for testing
var CreateAdapter = func(modelId string) (LLMAdapter, error) {
	model, ok := GetModelByID(modelId)
	if !ok {
		return nil, fmt.Errorf("model %s not found", modelId)
	}

	switch model.Provider {
	case PROVIDER_AZURE:
		return NewAzureAdapter(model.ID, model.Provider)
	case PROVIDER_ANTHROPIC:
		return NewAnthropicAdapter(), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", model.Provider)
	}
}
