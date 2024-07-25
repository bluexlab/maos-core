package handler

import (
	"context"
	"encoding/json"
	"log/slog"

	"gitlab.com/navyx/ai/maos/maos-core/admin"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/invocation"
)

func NewAPIHandler(logger *slog.Logger, accessor dbaccess.Accessor) *APIHandler {
	return &APIHandler{
		logger:            logger,
		accessor:          accessor,
		invocationManager: invocation.NewManager(logger, accessor),
	}
}

type APIHandler struct {
	logger            *slog.Logger
	accessor          dbaccess.Accessor
	invocationManager *invocation.Manager
}

func (s *APIHandler) Start(ctx context.Context) error {
	return s.invocationManager.Start(ctx)
}

func (s *APIHandler) Close(ctx context.Context) error {
	return s.invocationManager.Close(ctx)
}

// GetCallerConfig implements the GET /v1/config endpoint
func (s *APIHandler) GetCallerConfig(ctx context.Context, request api.GetCallerConfigRequestObject) (api.GetCallerConfigResponseObject, error) {
	config := api.GetCallerConfig200JSONResponse{
		"key1": "value1",
		"key2": "value2",
	}
	return config, nil
}

// CreateInvocation implements POST /v1/invocations endpoint
func (s *APIHandler) CreateInvocationAsync(ctx context.Context, request api.CreateInvocationAsyncRequestObject) (api.CreateInvocationAsyncResponseObject, error) {
	token := ValidatePermissions(ctx, "CreateInvocationAsync")
	if token == nil {
		return api.CreateInvocationAsync401Response{}, nil
	}
	return s.invocationManager.InsertInvocation(ctx, token.AgentId, request)
}

func (s *APIHandler) CreateInvocationSync(ctx context.Context, request api.CreateInvocationSyncRequestObject) (api.CreateInvocationSyncResponseObject, error) {
	panic("not implemented")
}

func (s *APIHandler) GetInvocationById(ctx context.Context, request api.GetInvocationByIdRequestObject) (api.GetInvocationByIdResponseObject, error) {
	panic("not implemented")
}

// GetNextInvocation implements the GET /v1/invocation/next endpoint
func (s *APIHandler) GetNextInvocation(ctx context.Context, request api.GetNextInvocationRequestObject) (api.GetNextInvocationResponseObject, error) {
	token := ValidatePermissions(ctx, "GetNextInvocation")
	if token == nil {
		return api.GetNextInvocation401Response{}, nil
	}
	return s.invocationManager.GetNextInvocation(ctx, token.AgentId, token.QueueId, request)
}

// ReturnInvocationResponse implements the POST /v1/invocation/{invoke_id}/response endpoint
func (s *APIHandler) ReturnInvocationResponse(ctx context.Context, request api.ReturnInvocationResponseRequestObject) (api.ReturnInvocationResponseResponseObject, error) {
	// Process the invocation response
	// You can access the invoke_id and result from the request object
	invokeID := request.InvokeId
	result := request.Body.Result

	// Here you would typically store or process the result
	// For this example, we'll just log it
	resultJSON, _ := json.Marshal(result)
	s.logger.Info("Received result", "invocation", invokeID, "result", string(resultJSON))

	return api.ReturnInvocationResponse200Response{}, nil
}

// ReturnInvocationError implements the POST /v1/invocation/{invoke_id}/error endpoint
func (s *APIHandler) ReturnInvocationError(ctx context.Context, request api.ReturnInvocationErrorRequestObject) (api.ReturnInvocationErrorResponseObject, error) {
	// Process the invocation error
	// You can access the invoke_id and error from the request object
	invokeID := request.InvokeId
	errorDetails := request.Body.Error

	// Here you would typically store or process the error
	// For this example, we'll just log it
	errorJSON, _ := json.Marshal(errorDetails)
	s.logger.Info("Received error for", "invocation", invokeID, "error", string(errorJSON))

	return api.ReturnInvocationError200Response{}, nil
}

func (s *APIHandler) ListEmbeddingModels(ctx context.Context, request api.ListEmbeddingModelsRequestObject) (api.ListEmbeddingModelsResponseObject, error) {
	panic("not implemented")
}

func (s *APIHandler) CreateEmbedding(ctx context.Context, request api.CreateEmbeddingRequestObject) (api.CreateEmbeddingResponseObject, error) {
	panic("not implemented")
}

func (s *APIHandler) CreateCompletion(ctx context.Context, request api.CreateCompletionRequestObject) (api.CreateCompletionResponseObject, error) {
	panic("not implemented")
}

func (s *APIHandler) ListCompletionModels(ctx context.Context, request api.ListCompletionModelsRequestObject) (api.ListCompletionModelsResponseObject, error) {
	panic("not implemented")
}

func (s *APIHandler) CreateRerank(ctx context.Context, request api.CreateRerankRequestObject) (api.CreateRerankResponseObject, error) {
	panic("not implemented")
}

func (s *APIHandler) ListRerankModels(ctx context.Context, request api.ListRerankModelsRequestObject) (api.ListRerankModelsResponseObject, error) {
	panic("not implemented")
}

func (s *APIHandler) ListCollection(ctx context.Context, request api.ListCollectionRequestObject) (api.ListCollectionResponseObject, error) {
	panic("not implemented")
}

func (s *APIHandler) CreateCollection(ctx context.Context, request api.CreateCollectionRequestObject) (api.CreateCollectionResponseObject, error) {
	panic("not implemented")
}

func (s *APIHandler) QueryCollection(ctx context.Context, request api.QueryCollectionRequestObject) (api.QueryCollectionResponseObject, error) {
	panic("not implemented")
}

func (s *APIHandler) UpsertCollection(ctx context.Context, request api.UpsertCollectionRequestObject) (api.UpsertCollectionResponseObject, error) {
	panic("not implemented")
}

func (s *APIHandler) ListVectoreStores(ctx context.Context, request api.ListVectoreStoresRequestObject) (api.ListVectoreStoresResponseObject, error) {
	panic("not implemented")
}

func (s *APIHandler) AdminListAgents(ctx context.Context, request api.AdminListAgentsRequestObject) (api.AdminListAgentsResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminListAgents")
	if token == nil {
		return api.AdminListAgents401Response{}, nil
	}
	return admin.ListAgents(ctx, s.accessor, request)
}

func (s *APIHandler) AdminCreateAgent(ctx context.Context, request api.AdminCreateAgentRequestObject) (api.AdminCreateAgentResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminCreateAgent")
	if token == nil {
		return api.AdminCreateAgent401Response{}, nil
	}
	panic("not implemented")
}

func (s *APIHandler) AdminListApiTokens(ctx context.Context, request api.AdminListApiTokensRequestObject) (api.AdminListApiTokensResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminListApiTokens")
	if token == nil {
		return api.AdminListApiTokens401Response{}, nil
	}
	return admin.ListApiTokens(ctx, s.accessor, request)
}

func (s *APIHandler) AdminCreateApiToken(ctx context.Context, request api.AdminCreateApiTokenRequestObject) (api.AdminCreateApiTokenResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminCreateApiToken")
	if token == nil {
		return api.AdminCreateApiToken401Response{}, nil
	}
	return admin.CreateApiToken(ctx, s.accessor, request)
}
