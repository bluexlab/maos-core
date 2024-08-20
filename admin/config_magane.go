package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

func UpdateConfig(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminUpdateConfigRequestObject) (api.AdminUpdateConfigResponseObject, error) {
	logger.Info("AdminUpdateConfig", "id", request.Id, "user", request.Body.User, "min_agent_version", request.Body.MinAgentVersion, "content", request.Body.Content)

	contentJSON, err := json.Marshal(request.Body.Content)
	if err != nil {
		logger.Error("Cannot marshal agent config content", "error", err)
		return api.AdminUpdateConfig500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot marshal agent config content: %v", err)},
		}, nil
	}

	updatedConfig, err := accessor.Querier().ConfigUpdateInactiveContentByCreator(ctx, accessor.Source(), &dbsqlc.ConfigUpdateInactiveContentByCreatorParams{
		ID:              request.Id,
		Updater:         request.Body.User,
		Content:         contentJSON,
		MinAgentVersion: request.Body.MinAgentVersion,
	})

	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminUpdateConfig404Response{}, nil
		}
		logger.Error("Cannot update config", "error", err)
		return api.AdminUpdateConfig500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot update config: %v", err)},
		}, nil
	}

	if updatedConfig.ID == 0 {
		return api.AdminUpdateConfig404Response{}, nil
	}

	var content map[string]string
	err = json.Unmarshal(updatedConfig.Content, &content)
	if err != nil {
		logger.Error("Cannot unmarshal content", "error", err)
		return api.AdminUpdateConfig500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot unmarshal content: %v", err)},
		}, nil
	}

	return api.AdminUpdateConfig200JSONResponse{
		Data: api.Config{
			Id:              updatedConfig.ID,
			AgentId:         updatedConfig.AgentId,
			AgentName:       updatedConfig.AgentName,
			Content:         content,
			MinAgentVersion: updatedConfig.MinAgentVersion,
			CreatedAt:       updatedConfig.CreatedAt,
			CreatedBy:       updatedConfig.CreatedBy,
			UpdatedAt:       updatedConfig.UpdatedAt,
			UpdatedBy:       updatedConfig.UpdatedBy,
		},
	}, nil
}
