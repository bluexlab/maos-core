package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/samber/lo"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

func AdminGetAgentConfig(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminGetAgentConfigRequestObject) (api.AdminGetAgentConfigResponseObject, error) {
	logger.Info("AdminGetAgentConfig", "agentId", request.Id)

	config, err := accessor.Querier().ConfigFindByAgentId(ctx, accessor.Pool(), int64(request.Id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminGetAgentConfig404Response{}, nil
		}

		logger.Error("Cannot get agent config", "error", err)
		return api.AdminGetAgentConfig500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot get agent config: %v", err)},
		}, nil
	}

	var content map[string]interface{}
	err = json.Unmarshal(config.Content, &content)
	if err != nil {
		logger.Error("Cannot unmarshal agent config", "error", err)
		return api.AdminGetAgentConfig500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot unmarshal agent config: %v", err)},
		}, nil
	}

	return api.AdminGetAgentConfig200JSONResponse{
		Data: api.Config{
			Id:              config.ID,
			AgentId:         config.AgentId,
			AgentName:       config.AgentName,
			Content:         content,
			MinAgentVersion: config.MinAgentVersion,
			CreatedAt:       config.CreatedAt,
			CreatedBy:       config.CreatedBy,
		},
	}, nil
}

func AdminUpdateAgentConfig(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminUpdateAgentConfigRequestObject) (api.AdminUpdateAgentConfigResponseObject, error) {
	logger.Info("AdminUpdateAgentConfig",
		"agentId", request.Id,
		"content", request.Body.Content,
		"minAgentVersion", lo.FromPtrOr(request.Body.MinAgentVersion, "nil"),
		"user", request.Body.User)

	// Marshal the content to JSON
	contentJSON, err := json.Marshal(request.Body.Content)
	if err != nil {
		logger.Error("Cannot marshal agent config content", "error", err)
		return api.AdminUpdateAgentConfig500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot marshal agent config content: %v", err)},
		}, nil
	}

	// Update the config in the database
	_, err = accessor.Querier().ConfigInsert(ctx, accessor.Pool(), &dbsqlc.ConfigInsertParams{
		AgentId:         int64(request.Id),
		MinAgentVersion: request.Body.MinAgentVersion,
		Content:         contentJSON,
		CreatedBy:       request.Body.User,
	})
	if err != nil {
		logger.Error("Cannot update agent config", "error", err)
		return api.AdminUpdateAgentConfig500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot update agent config: %v", err)},
		}, nil
	}

	// If the update was successful, return a 201 response with no content
	return api.AdminUpdateAgentConfig201Response{}, nil
}
