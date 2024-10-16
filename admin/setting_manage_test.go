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

		// Setup setting
		_, err := dbPool.Exec(ctx, "INSERT INTO settings (key, value) VALUES ($1, $2)", "system", []byte(`{
			"display_name": "test-maos",
			"deployment_approve_required": true,
			"enable_secrets_backup": true,
			"secrets_backup_public_key": "test-key",
			"secrets_backup_bucket": "test-bucket",
			"secrets_backup_prefix": "test-prefix"
		}`))
		require.NoError(t, err)

		response, err := admin.GetSetting(ctx, logger, dbPool, api.AdminGetSettingRequestObject{})
		assert.NoError(t, err)
		require.IsType(t, api.AdminGetSetting200JSONResponse{}, response)

		jsonResponse := response.(api.AdminGetSetting200JSONResponse)
		assert.Equal(t, "test-maos", jsonResponse.DisplayName)
		assert.True(t, jsonResponse.DeploymentApproveRequired)
		assert.True(t, jsonResponse.EnableSecretsBackup)
		assert.Equal(t, "test-key", *jsonResponse.SecretsBackupPublicKey)
		assert.Equal(t, "test-bucket", *jsonResponse.SecretsBackupBucket)
		assert.Equal(t, "test-prefix", *jsonResponse.SecretsBackupPrefix)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		request := api.AdminGetSettingRequestObject{}

		dbPool.Close()
		response, err := admin.GetSetting(ctx, logger, dbPool, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminGetSetting500JSONResponse{}, response)
	})

	t.Run("Missing setting", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		response, err := admin.GetSetting(ctx, logger, dbPool, api.AdminGetSettingRequestObject{})

		assert.NoError(t, err)
		require.IsType(t, api.AdminGetSetting200JSONResponse{}, response)

		jsonResponse := response.(api.AdminGetSetting200JSONResponse)
		assert.Empty(t, jsonResponse.DisplayName)
		assert.False(t, jsonResponse.DeploymentApproveRequired)
		assert.False(t, jsonResponse.EnableSecretsBackup)
		assert.Nil(t, jsonResponse.SecretsBackupPublicKey)
		assert.Nil(t, jsonResponse.SecretsBackupBucket)
		assert.Nil(t, jsonResponse.SecretsBackupPrefix)
	})
}

func TestUpdateSetting(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	t.Run("Successful update", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		// Setup initial setting
		_, err := dbPool.Exec(ctx, "INSERT INTO settings (key, value) VALUES ($1, $2)", "system", []byte(`{
			"display_name": "initial-maos",
			"deployment_approve_required": false,
			"enable_secrets_backup": false,
			"secrets_backup_public_key": "initial-key",
			"secrets_backup_bucket": "initial-bucket",
			"secrets_backup_prefix": "initial-prefix"
		}`))
		require.NoError(t, err)

		// Prepare update request
		updateRequest := api.AdminUpdateSettingRequestObject{
			Body: &api.AdminUpdateSettingJSONRequestBody{
				DisplayName:               lo.ToPtr("updated-maos"),
				DeploymentApproveRequired: lo.ToPtr(true),
				EnableSecretsBackup:       lo.ToPtr(true),
				SecretsBackupPublicKey:    lo.ToPtr("updated-key"),
				SecretsBackupBucket:       lo.ToPtr("updated-bucket"),
				SecretsBackupPrefix:       lo.ToPtr("updated-prefix"),
			},
		}

		response, err := admin.UpdateSetting(ctx, logger, dbPool, updateRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminUpdateSetting200Response{}, response)

		updatedResponse, err := admin.GetSetting(ctx, logger, dbPool, api.AdminGetSettingRequestObject{})
		require.IsType(t, api.AdminGetSetting200JSONResponse{}, updatedResponse)

		updatedJsonResponse := updatedResponse.(api.AdminGetSetting200JSONResponse)
		assert.Equal(t, "updated-maos", updatedJsonResponse.DisplayName)
		assert.True(t, updatedJsonResponse.DeploymentApproveRequired)
		assert.True(t, updatedJsonResponse.EnableSecretsBackup)
		assert.Equal(t, "updated-key", *updatedJsonResponse.SecretsBackupPublicKey)
		assert.Equal(t, "updated-bucket", *updatedJsonResponse.SecretsBackupBucket)
		assert.Equal(t, "updated-prefix", *updatedJsonResponse.SecretsBackupPrefix)

		// Verify the update in the database
		var settingValue []byte
		err = dbPool.QueryRow(ctx, "SELECT value FROM settings WHERE key = 'system'").Scan(&settingValue)
		require.NoError(t, err)

		var settingContent admin.SettingType
		err = json.Unmarshal(settingValue, &settingContent)
		require.NoError(t, err)
		assert.Equal(t, "updated-maos", *settingContent.DisplayName)
		assert.True(t, *settingContent.DeploymentApproveRequired)
		assert.True(t, *settingContent.EnableSecretsBackup)
		assert.Equal(t, "updated-key", *settingContent.SecretsBackupPublicKey)
		assert.Equal(t, "updated-bucket", *settingContent.SecretsBackupBucket)
		assert.Equal(t, "updated-prefix", *settingContent.SecretsBackupPrefix)
	})

	t.Run("Update with only display name", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		// Setup initial setting
		_, err := dbPool.Exec(ctx, "INSERT INTO settings (key, value) VALUES ($1, $2)", "system", []byte(`{
			"display_name": "initial-maos",
			"deployment_approve_required": true,
			"enable_secrets_backup": true,
			"secrets_backup_public_key": "initial-key",
			"secrets_backup_bucket": "initial-bucket",
			"secrets_backup_prefix": "initial-prefix"
		}`))
		require.NoError(t, err)

		// Prepare update request with only display name
		updateRequest := api.AdminUpdateSettingRequestObject{
			Body: &api.AdminUpdateSettingJSONRequestBody{
				DisplayName: lo.ToPtr("updated-maos-name"),
			},
		}

		response, err := admin.UpdateSetting(ctx, logger, dbPool, updateRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminUpdateSetting200Response{}, response)

		updatedResponse, err := admin.GetSetting(ctx, logger, dbPool, api.AdminGetSettingRequestObject{})
		require.IsType(t, api.AdminGetSetting200JSONResponse{}, updatedResponse)

		updatedJsonResponse := updatedResponse.(api.AdminGetSetting200JSONResponse)
		assert.Equal(t, "updated-maos-name", updatedJsonResponse.DisplayName)
		assert.True(t, updatedJsonResponse.DeploymentApproveRequired)               // Should remain unchanged
		assert.True(t, updatedJsonResponse.EnableSecretsBackup)                     // Should remain unchanged
		assert.Equal(t, "initial-key", *updatedJsonResponse.SecretsBackupPublicKey) // Should remain unchanged
		assert.Equal(t, "initial-bucket", *updatedJsonResponse.SecretsBackupBucket) // Should remain unchanged
		assert.Equal(t, "initial-prefix", *updatedJsonResponse.SecretsBackupPrefix) // Should remain unchanged

		// Verify the update in the database
		var settingValue []byte
		err = dbPool.QueryRow(ctx, "SELECT value FROM settings WHERE key = 'system'").Scan(&settingValue)
		require.NoError(t, err)

		var settingContent admin.SettingType
		err = json.Unmarshal(settingValue, &settingContent)
		require.NoError(t, err)
		assert.Equal(t, "updated-maos-name", *settingContent.DisplayName)
		assert.True(t, *settingContent.DeploymentApproveRequired)              // Should remain unchanged
		assert.True(t, *settingContent.EnableSecretsBackup)                    // Should remain unchanged
		assert.Equal(t, "initial-key", *settingContent.SecretsBackupPublicKey) // Should remain unchanged
		assert.Equal(t, "initial-bucket", *settingContent.SecretsBackupBucket) // Should remain unchanged
		assert.Equal(t, "initial-prefix", *settingContent.SecretsBackupPrefix) // Should remain unchanged
	})

	t.Run("Update with partial request", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		// Setup initial setting
		initialSetting := admin.SettingType{
			DisplayName:               lo.ToPtr("initial-maos"),
			DeploymentApproveRequired: lo.ToPtr(true),
			EnableSecretsBackup:       lo.ToPtr(false),
			SecretsBackupPublicKey:    lo.ToPtr("initial-key"),
			SecretsBackupBucket:       lo.ToPtr("initial-bucket"),
			SecretsBackupPrefix:       lo.ToPtr("initial-prefix"),
		}
		initialSettingBytes, err := json.Marshal(initialSetting)
		require.NoError(t, err)
		_, err = dbPool.Exec(ctx, "INSERT INTO settings (key, value) VALUES ($1, $2)", "system", initialSettingBytes)
		require.NoError(t, err)

		// Prepare partial update request
		updateRequest := api.AdminUpdateSettingRequestObject{
			Body: &api.AdminUpdateSettingJSONRequestBody{
				DisplayName:         lo.ToPtr("updated-maos"),
				EnableSecretsBackup: lo.ToPtr(true),
				SecretsBackupBucket: lo.ToPtr("updated-bucket"),
			},
		}

		response, err := admin.UpdateSetting(ctx, logger, dbPool, updateRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminUpdateSetting200Response{}, response)

		updatedResponse, err := admin.GetSetting(ctx, logger, dbPool, api.AdminGetSettingRequestObject{})
		require.IsType(t, api.AdminGetSetting200JSONResponse{}, updatedResponse)

		updatedJsonResponse := updatedResponse.(api.AdminGetSetting200JSONResponse)
		assert.Equal(t, "updated-maos", updatedJsonResponse.DisplayName)
		assert.True(t, updatedJsonResponse.DeploymentApproveRequired) // Should remain unchanged
		assert.True(t, updatedJsonResponse.EnableSecretsBackup)
		assert.Equal(t, "initial-key", *updatedJsonResponse.SecretsBackupPublicKey) // Should remain unchanged
		assert.Equal(t, "updated-bucket", *updatedJsonResponse.SecretsBackupBucket)
		assert.Equal(t, "initial-prefix", *updatedJsonResponse.SecretsBackupPrefix) // Should remain unchanged

		// Verify the update in the database
		var settingValue []byte
		err = dbPool.QueryRow(ctx, "SELECT value FROM settings WHERE key = 'system'").Scan(&settingValue)
		require.NoError(t, err)

		var settingContent admin.SettingType
		err = json.Unmarshal(settingValue, &settingContent)
		require.NoError(t, err)

		// Check updated fields
		assert.Equal(t, "updated-maos", *settingContent.DisplayName)
		assert.True(t, *settingContent.EnableSecretsBackup)
		assert.Equal(t, "updated-bucket", *settingContent.SecretsBackupBucket)

		// Check unchanged fields
		assert.True(t, *settingContent.DeploymentApproveRequired)
		assert.Equal(t, "initial-key", *settingContent.SecretsBackupPublicKey)
		assert.Equal(t, "initial-prefix", *settingContent.SecretsBackupPrefix)
	})

	t.Run("Update without initial setting", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		// Prepare update request
		updateRequest := api.AdminUpdateSettingRequestObject{
			Body: &api.AdminUpdateSettingJSONRequestBody{
				DisplayName:               lo.ToPtr("new-maos"),
				DeploymentApproveRequired: lo.ToPtr(true),
				EnableSecretsBackup:       lo.ToPtr(true),
				SecretsBackupPublicKey:    lo.ToPtr("new-key"),
				SecretsBackupBucket:       lo.ToPtr("new-bucket"),
				SecretsBackupPrefix:       lo.ToPtr("new-prefix"),
			},
		}

		response, err := admin.UpdateSetting(ctx, logger, dbPool, updateRequest)
		assert.NoError(t, err)
		require.IsType(t, api.AdminUpdateSetting200Response{}, response)

		updatedResponse, err := admin.GetSetting(ctx, logger, dbPool, api.AdminGetSettingRequestObject{})
		require.IsType(t, api.AdminGetSetting200JSONResponse{}, updatedResponse)

		updatedJsonResponse := updatedResponse.(api.AdminGetSetting200JSONResponse)
		assert.Equal(t, "new-maos", updatedJsonResponse.DisplayName)
		assert.True(t, updatedJsonResponse.DeploymentApproveRequired)
		assert.True(t, updatedJsonResponse.EnableSecretsBackup)
		assert.Equal(t, "new-key", *updatedJsonResponse.SecretsBackupPublicKey)
		assert.Equal(t, "new-bucket", *updatedJsonResponse.SecretsBackupBucket)
		assert.Equal(t, "new-prefix", *updatedJsonResponse.SecretsBackupPrefix)

		// Verify the update in the database
		var settingValue []byte
		err = dbPool.QueryRow(ctx, "SELECT value FROM settings WHERE key = 'system'").Scan(&settingValue)
		require.NoError(t, err)

		var settingContent admin.SettingType
		err = json.Unmarshal(settingValue, &settingContent)
		require.NoError(t, err)
		assert.Equal(t, "new-maos", *settingContent.DisplayName)
		assert.True(t, *settingContent.DeploymentApproveRequired)
		assert.True(t, *settingContent.EnableSecretsBackup)
		assert.Equal(t, "new-key", *settingContent.SecretsBackupPublicKey)
		assert.Equal(t, "new-bucket", *settingContent.SecretsBackupBucket)
		assert.Equal(t, "new-prefix", *settingContent.SecretsBackupPrefix)
	})

	t.Run("Invalid request body", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		// Prepare invalid update request (nil body)
		updateRequest := api.AdminUpdateSettingRequestObject{}

		response, err := admin.UpdateSetting(ctx, logger, dbPool, updateRequest)
		assert.NoError(t, err)
		assert.IsType(t, api.AdminUpdateSetting400JSONResponse{}, response)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		updateRequest := api.AdminUpdateSettingRequestObject{
			Body: &api.AdminUpdateSettingJSONRequestBody{
				DisplayName: lo.ToPtr("test-maos"),
			},
		}

		dbPool.Close()
		response, err := admin.UpdateSetting(ctx, logger, dbPool, updateRequest)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminUpdateSetting500JSONResponse{}, response)
	})
}
