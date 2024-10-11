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
	DisplayName               *string `json:"display_name,omitempty"`
	DeploymentApproveRequired *bool   `json:"deployment_approve_required,omitempty"`
}

func GetSetting(ctx context.Context, logger *slog.Logger, ds dbaccess.DataSource, request api.AdminGetSettingRequestObject) (api.AdminGetSettingResponseObject, error) {
	logger.Info("GetSetting")

	setting, err := querier.SettingGetSystem(ctx, ds)
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminGetSetting200JSONResponse{
				DisplayName:               "",
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
		DisplayName:               lo.FromPtrOr(settingContent.DisplayName, ""),
		DeploymentApproveRequired: lo.FromPtrOr(settingContent.DeploymentApproveRequired, false),
	}, nil
}

func UpdateSetting(ctx context.Context, logger *slog.Logger, ds dbaccess.DataSource, request api.AdminUpdateSettingRequestObject) (api.AdminUpdateSettingResponseObject, error) {
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
		DisplayName:               request.Body.DisplayName,
		DeploymentApproveRequired: request.Body.DeploymentApproveRequired,
	}

	// Marshal updated setting
	updatedSettingBytes, err := json.Marshal(settingContent)
	if err != nil {
		logger.Error("Failed to marshal updated setting", "error", err)
		return return500Error()
	}

	// Save update to database
	setting, err := querier.SettingUpdateSystem(ctx, ds, updatedSettingBytes)
	if err != nil {
		logger.Error("Failed to update setting", "error", err)
		return return500Error()
	}

	updatedSetting, err := deserializeSetting(setting.Value, logger)
	if err != nil {
		return return500Error()
	}

	return api.AdminUpdateSetting200JSONResponse{
		DisplayName:               lo.FromPtrOr(updatedSetting.DisplayName, ""),
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
