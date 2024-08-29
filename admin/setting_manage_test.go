package admin_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/admin"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func TestGetSettingWithDB(t *testing.T) {
	t.Parallel()
	logger := testhelper.Logger(t)
	ctx := context.Background()

	t.Run("Successful setting retrieval", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		// Setup setting
		_, err := accessor.Source().Exec(ctx, "INSERT INTO settings (key, value) VALUES ($1, $2)", "system", []byte(`{"display_name": "test-maos", "deployment_approve_required": true}`))
		require.NoError(t, err)

		response, err := admin.GetSetting(ctx, logger, accessor, api.AdminGetSettingRequestObject{})
		assert.NoError(t, err)
		require.IsType(t, api.AdminGetSetting200JSONResponse{}, response)

		jsonResponse := response.(api.AdminGetSetting200JSONResponse)
		assert.Equal(t, "test-maos", jsonResponse.DisplayName)
		assert.True(t, jsonResponse.DeploymentApproveRequired)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		request := api.AdminGetSettingRequestObject{}

		dbPool.Close()
		response, err := admin.GetSetting(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminGetSetting500JSONResponse{}, response)
	})

	t.Run("Missing setting", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		response, err := admin.GetSetting(ctx, logger, accessor, api.AdminGetSettingRequestObject{})

		assert.NoError(t, err)
		require.IsType(t, api.AdminGetSetting200JSONResponse{}, response)

		jsonResponse := response.(api.AdminGetSetting200JSONResponse)
		assert.Empty(t, jsonResponse.DisplayName)
		assert.False(t, jsonResponse.DeploymentApproveRequired)
	})
}

func TestUpdateSetting(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	t.Run("Successful update", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		// Setup initial setting
		_, err := accessor.Source().Exec(ctx, "INSERT INTO settings (key, value) VALUES ($1, $2)", "system", []byte(`{"display_name": "initial-maos", "deployment_approve_required": false}`))
		require.NoError(t, err)

		// Prepare update request
		updateRequest := api.AdminUpdateSettingRequestObject{
			Body: &api.AdminUpdateSettingJSONRequestBody{
				DisplayName:               lo.ToPtr("updated-maos"),
				DeploymentApproveRequired: lo.ToPtr(true),
			},
		}

		response, err := admin.UpdateSetting(ctx, logger, accessor, updateRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminUpdateSetting200JSONResponse{}, response)

		jsonResponse := response.(api.AdminUpdateSetting200JSONResponse)
		assert.Equal(t, "updated-maos", jsonResponse.DisplayName)
		assert.True(t, jsonResponse.DeploymentApproveRequired)

		// Verify the update in the database
		var settingValue []byte
		err = accessor.Source().QueryRow(ctx, "SELECT value FROM settings WHERE key = 'system'").Scan(&settingValue)
		require.NoError(t, err)

		var settingContent admin.SettingType
		err = json.Unmarshal(settingValue, &settingContent)
		require.NoError(t, err)
		assert.Equal(t, "updated-maos", *settingContent.DisplayName)
		assert.True(t, *settingContent.DeploymentApproveRequired)
	})

	t.Run("Update with only display name", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		// Setup initial setting
		_, err := accessor.Source().Exec(ctx, "INSERT INTO settings (key, value) VALUES ($1, $2)", "system", []byte(`{"display_name": "initial-maos", "deployment_approve_required": true}`))
		require.NoError(t, err)

		// Prepare update request with only display name
		updateRequest := api.AdminUpdateSettingRequestObject{
			Body: &api.AdminUpdateSettingJSONRequestBody{
				DisplayName: lo.ToPtr("updated-maos-name"),
			},
		}

		response, err := admin.UpdateSetting(ctx, logger, accessor, updateRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminUpdateSetting200JSONResponse{}, response)

		jsonResponse := response.(api.AdminUpdateSetting200JSONResponse)
		assert.Equal(t, "updated-maos-name", jsonResponse.DisplayName)
		assert.True(t, jsonResponse.DeploymentApproveRequired) // Should remain unchanged

		// Verify the update in the database
		var settingValue []byte
		err = accessor.Source().QueryRow(ctx, "SELECT value FROM settings WHERE key = 'system'").Scan(&settingValue)
		require.NoError(t, err)

		var settingContent admin.SettingType
		err = json.Unmarshal(settingValue, &settingContent)
		require.NoError(t, err)
		assert.Equal(t, "updated-maos-name", *settingContent.DisplayName)
		assert.True(t, *settingContent.DeploymentApproveRequired) // Should remain unchanged
	})

	t.Run("Update without initial setting", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		// Prepare update request
		updateRequest := api.AdminUpdateSettingRequestObject{
			Body: &api.AdminUpdateSettingJSONRequestBody{
				DisplayName:               lo.ToPtr("new-maos"),
				DeploymentApproveRequired: lo.ToPtr(true),
			},
		}

		response, err := admin.UpdateSetting(ctx, logger, accessor, updateRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminUpdateSetting200JSONResponse{}, response)

		jsonResponse := response.(api.AdminUpdateSetting200JSONResponse)
		assert.Equal(t, "new-maos", jsonResponse.DisplayName)
		assert.True(t, jsonResponse.DeploymentApproveRequired)

		// Verify the update in the database
		var settingValue []byte
		err = accessor.Source().QueryRow(ctx, "SELECT value FROM settings WHERE key = 'system'").Scan(&settingValue)
		require.NoError(t, err)

		var settingContent admin.SettingType
		err = json.Unmarshal(settingValue, &settingContent)
		require.NoError(t, err)
		assert.Equal(t, "new-maos", *settingContent.DisplayName)
		assert.True(t, *settingContent.DeploymentApproveRequired)
	})

	t.Run("Invalid request body", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		// Prepare invalid update request (nil body)
		updateRequest := api.AdminUpdateSettingRequestObject{}

		response, err := admin.UpdateSetting(ctx, logger, accessor, updateRequest)
		assert.NoError(t, err)
		assert.IsType(t, api.AdminUpdateSetting400JSONResponse{}, response)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		updateRequest := api.AdminUpdateSettingRequestObject{
			Body: &api.AdminUpdateSettingJSONRequestBody{
				DisplayName: lo.ToPtr("test-maos"),
			},
		}

		dbPool.Close()
		response, err := admin.UpdateSetting(ctx, logger, accessor, updateRequest)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminUpdateSetting500JSONResponse{}, response)
	})
}
