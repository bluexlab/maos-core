package admin

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/samber/lo"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
)

type SettingType struct {
	ClusterName               *string `json:"cluster_name,omitempty"`
	DeploymentApproveRequired *bool   `json:"deployment_approve_required,omitempty"`
}

func GetSetting(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminGetSettingRequestObject) (api.AdminGetSettingResponseObject, error) {
	logger.Info("GetSetting")

	setting, err := accessor.Querier().SettingGetSystem(ctx, accessor.Source())
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminGetSetting200JSONResponse{
				ClusterName:               "",
				DeploymentApproveRequired: false,
			}, nil
		}

		logger.Error("Failed to retrieve setting", "error", err)
		return api.AdminGetSetting500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{
				Error: "Internal server error",
			},
		}, nil
	}

	settingContent, err := deserializeSetting(setting.Value, logger)
	if err != nil {
		return api.AdminGetSetting500JSONResponse{N500JSONResponse: api.N500JSONResponse{Error: "Internal server error"}}, nil
	}

	return api.AdminGetSetting200JSONResponse{
		ClusterName:               lo.FromPtrOr(settingContent.ClusterName, ""),
		DeploymentApproveRequired: lo.FromPtrOr(settingContent.DeploymentApproveRequired, false),
	}, nil
}

func UpdateSetting(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminUpdateSettingRequestObject) (api.AdminUpdateSettingResponseObject, error) {
	logger.Info("UpdateSetting", "request", request.Body)

	return500Error := func() (api.AdminUpdateSettingResponseObject, error) {
		return api.AdminUpdateSetting500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{
				Error: "Internal server error",
			},
		}, nil
	}

	// Validate the request
	if request.Body == nil {
		return api.AdminUpdateSetting400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{
				Error: "Invalid request body",
			},
		}, nil
	}

	settingContent := SettingType{
		ClusterName:               request.Body.ClusterName,
		DeploymentApproveRequired: request.Body.DeploymentApproveRequired,
	}

	// Marshal updated setting
	updatedSettingBytes, err := json.Marshal(settingContent)
	if err != nil {
		logger.Error("Failed to marshal updated setting", "error", err)
		return return500Error()
	}

	// Save update to database
	setting, err := accessor.Querier().SettingUpdateSystem(ctx, accessor.Source(), updatedSettingBytes)
	if err != nil {
		logger.Error("Failed to update setting", "error", err)
		return return500Error()
	}

	updatedSetting, err := deserializeSetting(setting.Value, logger)
	if err != nil {
		return return500Error()
	}

	return api.AdminUpdateSetting200JSONResponse{
		ClusterName:               lo.FromPtrOr(updatedSetting.ClusterName, ""),
		DeploymentApproveRequired: lo.FromPtrOr(updatedSetting.DeploymentApproveRequired, false),
	}, nil
}

func deserializeSetting(content []byte, logger *slog.Logger) (SettingType, error) {
	var settingContent SettingType
	err := json.Unmarshal(content, &settingContent)
	if err != nil {
		logger.Error("Failed to unmarshal setting", "error", err)
		return SettingType{}, err
	}
	return settingContent, nil
}
