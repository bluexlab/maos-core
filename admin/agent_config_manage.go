package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
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

	var content map[string]string
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
