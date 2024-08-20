package handler

import (
	"context"
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
	token := ValidatePermissions(ctx, "CreateInvocationSync")
	if token == nil {
		return api.CreateInvocationSync401Response{}, nil
	}
	return s.invocationManager.ExecuteInvocationSync(ctx, token.AgentId, request)
}

func (s *APIHandler) GetInvocationById(ctx context.Context, request api.GetInvocationByIdRequestObject) (api.GetInvocationByIdResponseObject, error) {
	token := ValidatePermissions(ctx, "CreateInvocationSync")
	if token == nil {
		return api.GetInvocationById401Response{}, nil
	}
	return s.invocationManager.GetInvocationById(ctx, token.AgentId, request)
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
	token := ValidatePermissions(ctx, "ReturnInvocationResponse")
	if token == nil {
		return api.ReturnInvocationResponse401Response{}, nil
	}

	return s.invocationManager.ReturnInvocationResponse(ctx, token.AgentId, request)
}

// ReturnInvocationError implements the POST /v1/invocation/{invoke_id}/error endpoint
func (s *APIHandler) ReturnInvocationError(ctx context.Context, request api.ReturnInvocationErrorRequestObject) (api.ReturnInvocationErrorResponseObject, error) {
	token := ValidatePermissions(ctx, "ReturnInvocationResponse")
	if token == nil {
		return api.ReturnInvocationError401Response{}, nil
	}

	return s.invocationManager.ReturnInvocationError(ctx, token.AgentId, request)
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
	return admin.ListAgents(ctx, s.logger, s.accessor, request)
}

func (s *APIHandler) AdminCreateAgent(ctx context.Context, request api.AdminCreateAgentRequestObject) (api.AdminCreateAgentResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminCreateAgent")
	if token == nil {
		return api.AdminCreateAgent401Response{}, nil
	}
	return admin.CreateAgent(ctx, s.logger, s.accessor, request)
}

func (s *APIHandler) AdminGetAgent(ctx context.Context, request api.AdminGetAgentRequestObject) (api.AdminGetAgentResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminGetAgents")
	if token == nil {
		return api.AdminGetAgent401Response{}, nil
	}
	return admin.GetAgent(ctx, s.logger, s.accessor, request)
}

func (s *APIHandler) AdminUpdateAgent(ctx context.Context, request api.AdminUpdateAgentRequestObject) (api.AdminUpdateAgentResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminUpdateAgent")
	if token == nil {
		return api.AdminUpdateAgent401Response{}, nil
	}
	return admin.UpdateAgent(ctx, s.logger, s.accessor, request)
}

func (s *APIHandler) AdminDeleteAgent(ctx context.Context, request api.AdminDeleteAgentRequestObject) (api.AdminDeleteAgentResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminDeleteAgent")
	if token == nil {
		return api.AdminDeleteAgent401Response{}, nil
	}
	return admin.DeleteAgent(ctx, s.logger, s.accessor, request)
}

func (s *APIHandler) AdminGetAgentConfig(ctx context.Context, request api.AdminGetAgentConfigRequestObject) (api.AdminGetAgentConfigResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminGetAgentConfig")
	if token == nil {
		return api.AdminGetAgentConfig401Response{}, nil
	}
	return admin.AdminGetAgentConfig(ctx, s.logger, s.accessor, request)
}

// func (s *APIHandler) AdminUpdateAgentConfig(ctx context.Context, request api.AdminUpdateAgentConfigRequestObject) (api.AdminUpdateAgentConfigResponseObject, error) {
// 	token := ValidatePermissions(ctx, "AdminUpdateAgentConfig")
// 	if token == nil {
// 		return api.AdminUpdateAgentConfig401Response{}, nil
// 	}
// 	return admin.AdminUpdateAgentConfig(ctx, s.logger, s.accessor, request)
// }

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

func (s *APIHandler) AdminListDeployments(ctx context.Context, request api.AdminListDeploymentsRequestObject) (api.AdminListDeploymentsResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminListDeployments")
	if token == nil {
		return api.AdminListDeployments401Response{}, nil
	}

	return admin.ListDeployments(ctx, s.logger, s.accessor, request)
}

func (s *APIHandler) AdminGetDeployment(ctx context.Context, request api.AdminGetDeploymentRequestObject) (api.AdminGetDeploymentResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminGetDeployment")
	if token == nil {
		return api.AdminGetDeployment401Response{}, nil
	}

	return admin.GetDeployment(ctx, s.logger, s.accessor, request)
}

func (s *APIHandler) AdminCreateDeployment(ctx context.Context, request api.AdminCreateDeploymentRequestObject) (api.AdminCreateDeploymentResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminCreateDeployment")
	if token == nil {
		return api.AdminCreateDeployment401Response{}, nil
	}

	return admin.CreateDeployment(ctx, s.logger, s.accessor, request)
}

func (s *APIHandler) AdminUpdateDeployment(ctx context.Context, request api.AdminUpdateDeploymentRequestObject) (api.AdminUpdateDeploymentResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminUpdateDeployment")
	if token == nil {
		return api.AdminUpdateDeployment401Response{}, nil
	}
	return admin.UpdateDeployment(ctx, s.logger, s.accessor, request)
}

func (s *APIHandler) AdminSubmitDeployment(ctx context.Context, request api.AdminSubmitDeploymentRequestObject) (api.AdminSubmitDeploymentResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminSubmitDeployment")
	if token == nil {
		return api.AdminSubmitDeployment401Response{}, nil
	}
	return admin.SubmitDeployment(ctx, s.logger, s.accessor, request)
}

func (s *APIHandler) AdminPublishDeployment(ctx context.Context, request api.AdminPublishDeploymentRequestObject) (api.AdminPublishDeploymentResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminPublishDeployment")
	if token == nil {
		return api.AdminPublishDeployment401Response{}, nil
	}
	return admin.PublishDeployment(ctx, s.logger, s.accessor, request)
}

func (s *APIHandler) AdminDeleteDeployment(ctx context.Context, request api.AdminDeleteDeploymentRequestObject) (api.AdminDeleteDeploymentResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminDeleteDeployment")
	if token == nil {
		return api.AdminDeleteDeployment401Response{}, nil
	}
	return admin.DeleteDeployment(ctx, s.logger, s.accessor, request)
}

func (s *APIHandler) AdminUpdateConfig(ctx context.Context, request api.AdminUpdateConfigRequestObject) (api.AdminUpdateConfigResponseObject, error) {
	token := ValidatePermissions(ctx, "AdminUpdateConfig")
	if token == nil {
		return api.AdminUpdateConfig401Response{}, nil
	}
	return admin.UpdateConfig(ctx, s.logger, s.accessor, request)
}

func (s *APIHandler) GetHealth(ctx context.Context, request api.GetHealthRequestObject) (api.GetHealthResponseObject, error) {
	return api.GetHealth200JSONResponse{Status: "healthy"}, nil
}
