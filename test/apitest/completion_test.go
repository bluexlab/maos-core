package apitest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
	"gitlab.com/navyx/ai/maos/maos-core/llm"
	"gitlab.com/navyx/ai/maos/maos-core/llm/adapter"
)

// MockAdapter is a mock implementation of the LLMAdapter interface
type MockAdapter struct {
	mock.Mock
}

func (m *MockAdapter) GetCompletion(ctx context.Context, request llm.CompletionRequest) (llm.CompletionResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(llm.CompletionResult), args.Error(1)
}

func TestCreateCompletion(t *testing.T) {
	ctx := context.Background()

	// Setup
	server, accessor, _ := SetupHttpTestWithDb(t, ctx)
	actor := fixture.InsertActor(t, ctx, accessor.Source(), "test-actor")
	fixture.InsertToken(t, ctx, accessor.Source(), "test-token", actor.ID, 0, []string{"create:completion"})

	// Create a mock adapter
	mockAdapter := new(MockAdapter)

	// Override the CreateAdapter function to return our mock
	originalCreateAdapter := adapter.CreateAdapter
	adapter.CreateAdapter = func(modelId string) (adapter.LLMAdapter, error) {
		return mockAdapter, nil
	}
	defer func() { adapter.CreateAdapter = originalCreateAdapter }()

	t.Run("Successful completion", func(t *testing.T) {
		// Set up mock expectations
		expectedRequest := llm.CompletionRequest{
			ModelID: "test-model",
			Messages: []llm.Message{
				{
					Role: "user",
					Content: []llm.Content{
						{Text: "Hello, AI!"},
						{Image: []byte("Hello, AI!")},
						{ImageURL: "https://example.com/image.png"},
					},
				},
			},
			Temperature: nil,
			MaxTokens:   lo.ToPtr(int32(8000)),
		}

		mockResponse := llm.CompletionResult{
			Messages: []llm.Message{
				{
					Role: "assistant",
					Content: []llm.Content{
						{Text: "Hello, human! How can I assist you today?"},
					},
				},
			},
		}

		mockAdapter.On("GetCompletion", mock.Anything, expectedRequest).Return(mockResponse, nil)

		// Prepare the request body
		requestBody := api.CreateCompletionJSONRequestBody{
			ModelId: "test-model",
			Messages: []api.Message{
				{
					Role: api.MessageRole("user"),
					Content: []api.MessageContent{
						{}, {}, {},
					},
				},
			},
		}
		requestBody.Messages[0].Content[0].MergeMessageContent0(api.MessageContent0{Text: "Hello, AI!"})
		requestBody.Messages[0].Content[1].MergeMessageContent1(api.MessageContent1{Image: "SGVsbG8sIEFJIQ=="})
		requestBody.Messages[0].Content[2].MergeMessageContent2(api.MessageContent2{ImageUrl: "https://example.com/image.png"})

		// Send the request
		resp, resBody := PostHttp(t, server.URL+"/v1/completion", testhelper.SerializeToJson(t, requestBody), "test-token")

		// Assert the response
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.CreateCompletion200JSONResponse
		err := json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)

		assert.Len(t, response.Messages, 1)
		assert.Equal(t, "assistant", string(response.Messages[0].Role))
		assert.Len(t, response.Messages[0].Content, 1)

		content, err := response.Messages[0].Content[0].AsMessageContent0()
		require.NoError(t, err)
		assert.Equal(t, "Hello, human! How can I assist you today?", content.Text)

		// Verify that the mock expectations were met
		mockAdapter.AssertExpectations(t)
	})

	t.Run("Unauthorized access", func(t *testing.T) {
		requestBody := api.CreateCompletionJSONRequestBody{
			ModelId: "test-model",
			Messages: []api.Message{{
				Role:    api.MessageRole("user"),
				Content: []api.MessageContent{{}},
			}},
		}

		// Send the request with an invalid token
		resp, _ := PostHttp(t, server.URL+"/v1/completion", testhelper.SerializeToJson(t, requestBody), "invalid-token")

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		// No need to verify mock expectations as the request should not reach the adapter
	})

	t.Run("Bad request", func(t *testing.T) {
		originalCreateAdapter := adapter.CreateAdapter
		adapter.CreateAdapter = func(modelId string) (adapter.LLMAdapter, error) {
			require.Equal(t, "invalid-model", modelId)
			return nil, fmt.Errorf("invalid model")
		}
		defer func() { adapter.CreateAdapter = originalCreateAdapter }()

		// Prepare an invalid request body (missing required field)
		invalidRequestBody := api.CreateCompletionJSONRequestBody{
			ModelId: "invalid-model",
			Messages: []api.Message{{
				Role:    api.MessageRole("user"),
				Content: []api.MessageContent{{}},
			}},
		}
		invalidRequestBody.Messages[0].Content[0].FromMessageContent0(api.MessageContent0{Text: "Hello, AI!"})

		// Send the request with the invalid body
		resp, resBody := PostHttp(t, server.URL+"/v1/completion", testhelper.SerializeToJson(t, invalidRequestBody), "test-token")

		// Assert the response
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Contains(t, resBody, "Model invalid-model not found")
	})

	t.Run("Bad request - Invalid message content", func(t *testing.T) {
		// Prepare a request body with invalid message content
		invalidRequestBody := api.CreateCompletionJSONRequestBody{
			TraceId: "123",
			ModelId: "test-model",
			Messages: []api.Message{{
				Role:    api.MessageRole("user"),
				Content: []api.MessageContent{{}},
			}},
		}
		// Set an empty content, which should be invalid
		invalidRequestBody.Messages[0].Content[0].FromMessageContent0(api.MessageContent0{})

		// Send the request with the invalid body
		resp, resBody := PostHttp(t, server.URL+"/v1/completion", testhelper.SerializeToJson(t, invalidRequestBody), "test-token")

		// Assert the response
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Contains(t, resBody, "Invalid message content")
	})

	t.Run("Bad request - Invalid base64 image encoding", func(t *testing.T) {
		// Prepare a request body with invalid base64 image encoding
		invalidRequestBody := api.CreateCompletionJSONRequestBody{
			TraceId: "123",
			ModelId: "test-model",
			Messages: []api.Message{{
				Role:    api.MessageRole("user"),
				Content: []api.MessageContent{{}},
			}},
		}
		// Set an invalid base64 encoded image
		invalidRequestBody.Messages[0].Content[0].FromMessageContent1(api.MessageContent1{Image: "invalid-base64"})

		// Send the request with the invalid body
		resp, resBody := PostHttp(t, server.URL+"/v1/completion", testhelper.SerializeToJson(t, invalidRequestBody), "test-token")

		// Assert the response
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Contains(t, resBody, "Invalid base64 image encoding")
	})
}

func TestListCompletionModels(t *testing.T) {
	ctx := context.Background()
	server, accessor, _ := SetupHttpTestWithDb(t, ctx)
	actor := fixture.InsertActor(t, ctx, accessor.Source(), "test-actor")
	fixture.InsertToken(t, ctx, accessor.Source(), "test-token", actor.ID, 0, []string{"read:completion"})

	t.Run("Successful listing of completion models", func(t *testing.T) {
		// Send the request
		resp, resBody := GetHttp(t, server.URL+"/v1/completion/models?trace_id=123", "test-token")

		// Assert the response
		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse the response body
		var response api.ListCompletionModels200JSONResponse
		err := json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)

		// Assert the response content
		require.NotEmpty(t, response.Data)

		// Expected model list based on adapter.GetModelList()
		expectedModels := adapter.GetModelList()

		require.Equal(t, len(expectedModels), len(response.Data))

		for i, model := range response.Data {
			require.Equal(t, expectedModels[i].ID, model.Id)
			require.Equal(t, expectedModels[i].Name, model.Name)
			require.Equal(t, expectedModels[i].Provider, model.Provider)
		}
	})

	t.Run("Unauthorized request", func(t *testing.T) {
		// Send the request without a token
		resp, _ := GetHttp(t, server.URL+"/v1/completion/models?trace_id=123", "")

		// Assert the response
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
