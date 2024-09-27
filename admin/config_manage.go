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
	"gitlab.com/navyx/ai/maos/maos-core/util"
)

func UpdateConfig(
	ctx context.Context,
	logger *slog.Logger,
	accessor dbaccess.Accessor,
	request api.AdminUpdateConfigRequestObject,
) (api.AdminUpdateConfigResponseObject, error) {
	logger.Info("AdminUpdateConfig",
		"id", request.Id,
		"user", request.Body.User,
		"content", request.Body.Content,
	)

	contentJSON, err := json.Marshal(request.Body.Content)
	if err != nil {
		logger.Error("Cannot marshal actor config content", "error", err)
		return api.AdminUpdateConfig500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot marshal actor config content: %v", err)},
		}, nil
	}

	actor, err := accessor.Querier().GetActorByConfigId(ctx, accessor.Source(), request.Id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminUpdateConfig404Response{}, nil
		}
		logger.Error("Cannot get actor by config id", "error", err)
		return api.AdminUpdateConfig500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot get actor by config id: %v", err)},
		}, nil
	}

	err = ValidateKubeConfig(*request.Body.Content, string(actor.Role))
	if err != nil {
		logger.Error("Invalid kube config", "error", err)
		return api.AdminUpdateConfig400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: fmt.Sprintf("Invalid kube config: %v", err)},
		}, nil
	}

	minActorVersion := util.DeserializeActorVersion(request.Body.MinActorVersion)
	if minActorVersion == nil && request.Body.MinActorVersion != nil {
		logger.Error("Invalid actor version", "version", request.Body.MinActorVersion)
		return api.AdminUpdateConfig400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Invalid actor version"},
		}, nil
	}

	updatedConfig, err := accessor.Querier().ConfigUpdateInactiveContentByCreator(
		ctx,
		accessor.Source(),
		&dbsqlc.ConfigUpdateInactiveContentByCreatorParams{
			ID:              request.Id,
			Updater:         request.Body.User,
			Content:         contentJSON,
			MinActorVersion: minActorVersion,
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
			ActorId:         updatedConfig.ActorId,
			ActorName:       updatedConfig.ActorName,
			Content:         content,
			MinActorVersion: util.SerializeActorVersion(updatedConfig.MinActorVersion),
			CreatedAt:       updatedConfig.CreatedAt,
			CreatedBy:       updatedConfig.CreatedBy,
			UpdatedAt:       updatedConfig.UpdatedAt,
			UpdatedBy:       updatedConfig.UpdatedBy,
		},
	}, nil
}
