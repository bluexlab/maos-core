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

func GetActorConfig(
	ctx context.Context,
	logger *slog.Logger,
	accessor dbaccess.Accessor,
	request api.GetCallerConfigRequestObject,
) (api.GetCallerConfigResponseObject, error) {
	logger.Info("GetActorConfig", "ActorVersion", request.Params.XActorVersion)

	token := GetContextToken(ctx)
	if token == nil {
		return api.GetCallerConfig401Response{}, nil
	}

	// parse actor version to []int32
	actorVersion := util.DeserializeActorVersion(request.Params.XActorVersion)
	if actorVersion == nil && request.Params.XActorVersion != nil {
		return api.GetCallerConfig400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{
				Error: fmt.Sprintf("Cannot parse actor version: %s", *request.Params.XActorVersion),
			},
		}, nil
	}

	// get active version compatible config
	config, err := accessor.Querier().ConfigActorActiveConfig(
		ctx,
		accessor.Source(),
		&dbsqlc.ConfigActorActiveConfigParams{
			ActorId:      token.ActorId,
			ActorVersion: actorVersion,
		},
	)

	if err == nil {
		return parseConfigsAndReturn(ctx, logger, config.Content)
	}

	if err != nil {
		if err != pgx.ErrNoRows {
			return api.GetCallerConfig500JSONResponse{
				N500JSONResponse: api.N500JSONResponse{
					Error: fmt.Sprintf("Cannot get actor config: %v", err),
				},
			}, nil
		}
	}

	// active config not found, get latest version compatible retired config
	config, err = accessor.Querier().ConfigActorRetiredAndVersionCompatibleConfig(
		ctx,
		accessor.Source(),
		&dbsqlc.ConfigActorRetiredAndVersionCompatibleConfigParams{
			ActorId:      token.ActorId,
			ActorVersion: actorVersion,
		})

	if err != nil {
		if err == pgx.ErrNoRows {
			return api.GetCallerConfig404Response{}, nil
		}
		return api.GetCallerConfig500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{
				Error: fmt.Sprintf("Cannot get actor config: %v", err),
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
				Error: fmt.Sprintf("Cannot unmarshal actor config: %v", err),
			},
		}, nil
	}

	return api.GetCallerConfig200JSONResponse{
		Data: api.Configuration(content),
	}, nil
}
