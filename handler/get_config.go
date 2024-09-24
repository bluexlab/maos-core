package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/util"
)

func GetAgentConfig(
	ctx context.Context,
	logger *slog.Logger,
	accessor dbaccess.Accessor,
	request api.GetCallerConfigRequestObject,
) (api.GetCallerConfigResponseObject, error) {
	logger.Info("GetAgentConfig", "AgentVersion", request.Params.XAgentVersion)

	token := GetContextToken(ctx)
	if token == nil {
		return api.GetCallerConfig401Response{}, nil
	}

	// parse agent version to []int32
	agentVersion := util.DeserializeAgentVersion(request.Params.XAgentVersion)
	if agentVersion == nil && request.Params.XAgentVersion != nil {
		return api.GetCallerConfig400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{
				Error: fmt.Sprintf("Cannot parse agent version: %s", *request.Params.XAgentVersion),
			},
		}, nil
	}

	// get active version compatible config
	config, err := accessor.Querier().ConfigAgentActiveConfig(
		ctx,
		accessor.Source(),
		&dbsqlc.ConfigAgentActiveConfigParams{
			AgentId:      token.AgentId,
			AgentVersion: agentVersion,
		},
	)

	if err == nil {
		return parseConfigsAndReturn(ctx, logger, config.Content)
	}

	if err != nil {
		if err != pgx.ErrNoRows {
			return api.GetCallerConfig500JSONResponse{
				N500JSONResponse: api.N500JSONResponse{
					Error: fmt.Sprintf("Cannot get agent config: %v", err),
				},
			}, nil
		}
	}

	// active config not found, get latest version compatible retired config
	config, err = accessor.Querier().ConfigAgentRetiredAndVersionCompatibleConfig(
		ctx,
		accessor.Source(),
		&dbsqlc.ConfigAgentRetiredAndVersionCompatibleConfigParams{
			AgentId:      token.AgentId,
			AgentVersion: agentVersion,
		})

	if err != nil {
		if err == pgx.ErrNoRows {
			return api.GetCallerConfig404Response{}, nil
		}
		return api.GetCallerConfig500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{
				Error: fmt.Sprintf("Cannot get agent config: %v", err),
			},
		}, nil
	}

	return parseConfigsAndReturn(ctx, logger, config.Content)
}

func parseConfigsAndReturn(
	ctx context.Context,
	logger *slog.Logger,
	bytes []byte,
) (api.GetCallerConfigResponseObject, error) {
	content := make(map[string]string)
	err := json.Unmarshal(bytes, &content)
	if err != nil {
		return api.GetCallerConfig500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{
				Error: fmt.Sprintf("Cannot unmarshal agent config: %v", err),
			},
		}, nil
	}

	return api.GetCallerConfig200JSONResponse{
		Data: api.Configuration(content),
	}, nil
}
