package admin_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/admin"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func TestUpdateConfig(t *testing.T) {
	t.Parallel()
	logger := testhelper.Logger(t)

	ctx := context.Background()

	t.Run("Successful update config", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)
		defer dbPool.Close()

		user := "testuser"
		agent := fixture.InsertAgent(t, ctx, dbPool, "TestAgent")
		configSuite := fixture.InsertConfigSuite(t, ctx, dbPool)
		initialContent := map[string]string{
			"key1": "value1",
			"key2": "42",
		}
		config := fixture.InsertConfig2(t, ctx, dbPool, agent.ID, &configSuite.ID, user, initialContent)

		updatedContent := map[string]string{
			"key1": "newValue1",
			"key3": "newValue3",
		}
		minAgentVersion := "2.0.0"

		request := api.AdminUpdateConfigRequestObject{
			Id: config.ID,
			Body: &api.AdminUpdateConfigJSONRequestBody{
				Content:         &updatedContent,
				MinAgentVersion: &minAgentVersion,
				User:            user,
			},
		}

		response, err := admin.UpdateConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminUpdateConfig200JSONResponse{}, response)

		jsonResponse := response.(api.AdminUpdateConfig200JSONResponse)
		require.Equal(t, config.ID, jsonResponse.Data.Id)
		require.Equal(t, config.AgentId, jsonResponse.Data.AgentId)
		require.Equal(t, updatedContent, jsonResponse.Data.Content)
		require.Equal(t, minAgentVersion, *jsonResponse.Data.MinAgentVersion)
		require.Equal(t, user, jsonResponse.Data.CreatedBy)
		require.Equal(t, agent.Name, jsonResponse.Data.AgentName)
		require.NotZero(t, jsonResponse.Data.CreatedAt)
	})

	t.Run("Config suite is deployed", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)
		defer dbPool.Close()

		user := "testuser"
		agent := fixture.InsertAgent(t, ctx, dbPool, "TestAgent3")
		configSuite := fixture.InsertConfigSuite(t, ctx, dbPool)

		// Set the config suite as deployed
		_, err := dbPool.Exec(ctx, "UPDATE config_suites SET deployed_at = 16888 WHERE id = $1", configSuite.ID)
		require.NoError(t, err)

		initialContent := map[string]string{
			"key1": "value1",
		}
		config := fixture.InsertConfig2(t, ctx, dbPool, agent.ID, &configSuite.ID, user, initialContent)

		updatedContent := map[string]string{
			"key1": "newValue1",
		}

		request := api.AdminUpdateConfigRequestObject{
			Id: config.ID,
			Body: &api.AdminUpdateConfigJSONRequestBody{
				Content: &updatedContent,
				User:    user,
			},
		}

		response, err := admin.UpdateConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminUpdateConfig404Response{}, response)
	})

	t.Run("Config not found", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)
		defer dbPool.Close()

		request := api.AdminUpdateConfigRequestObject{
			Id: 999999, // Non-existent config ID
			Body: &api.AdminUpdateConfigJSONRequestBody{
				Content: &map[string]string{"key": "value"},
				User:    "testuser",
			},
		}

		response, err := admin.UpdateConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminUpdateConfig404Response{}, response)
	})

	t.Run("Database error", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)
		defer dbPool.Close()

		agent := fixture.InsertAgent(t, ctx, dbPool, "TestAgent2")
		configSuite := fixture.InsertConfigSuite(t, ctx, dbPool)
		config := fixture.InsertConfig2(t, ctx, dbPool, agent.ID, &configSuite.ID, "testuser", map[string]string{"key": "value"})

		request := api.AdminUpdateConfigRequestObject{
			Id: config.ID,
			Body: &api.AdminUpdateConfigJSONRequestBody{
				Content: &map[string]string{"key": "newValue"},
				User:    "testuser",
			},
		}

		dbPool.Close() // Simulate database error by closing the connection

		response, err := admin.UpdateConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminUpdateConfig500JSONResponse{}, response)
	})

	t.Run("Update config failed if user is not the creator or reviewer", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)
		defer dbPool.Close()

		agent := fixture.InsertAgent(t, ctx, dbPool, "TestAgent3")

		// Insert a deployment with reviewers
		deployment, err := accessor.Querier().DeploymentInsertWithConfigSuite(ctx, accessor.Source(), &dbsqlc.DeploymentInsertWithConfigSuiteParams{
			CreatedBy: "creator",
			Name:      "TestDeployment",
			Reviewers: []string{"reviewer1", "reviewer2"},
		})
		require.NoError(t, err)

		config := fixture.InsertConfig2(t, ctx, dbPool, agent.ID, deployment.ConfigSuiteID, "creator", map[string]string{"key": "value"})

		updatedContent := map[string]string{"key": "newValue"}
		unauthorizedUser := "unauthorized_user"

		request := api.AdminUpdateConfigRequestObject{
			Id: config.ID,
			Body: &api.AdminUpdateConfigJSONRequestBody{
				Content: &updatedContent,
				User:    unauthorizedUser,
			},
		}

		response, err := admin.UpdateConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminUpdateConfig404Response{}, response)

		// Verify that the config was not updated
		updatedConfig, err := accessor.Querier().ConfigFindByAgentId(ctx, accessor.Source(), agent.ID)
		require.NoError(t, err)
		require.Equal(t, `{"key": "value"}`, string(updatedConfig.Content))
	})

	t.Run("Update config if user is reviewer", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)
		defer dbPool.Close()

		agent := fixture.InsertAgent(t, ctx, dbPool, "TestAgent3")

		// Insert a deployment with reviewers
		deployment, err := accessor.Querier().DeploymentInsertWithConfigSuite(ctx, accessor.Source(), &dbsqlc.DeploymentInsertWithConfigSuiteParams{
			CreatedBy: "creator",
			Name:      "TestDeployment",
			Reviewers: []string{"reviewer1", "reviewer2"},
		})
		require.NoError(t, err)

		config := fixture.InsertConfig2(t, ctx, dbPool, agent.ID, deployment.ConfigSuiteID, "creator", map[string]string{"key": "value"})

		updatedContent := map[string]string{"key": "newValue"}
		user := "reviewer1"
		minAgentVersion := "2.0.0"

		request := api.AdminUpdateConfigRequestObject{
			Id: config.ID,
			Body: &api.AdminUpdateConfigJSONRequestBody{
				Content:         &updatedContent,
				MinAgentVersion: &minAgentVersion,
				User:            user,
			},
		}

		response, err := admin.UpdateConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminUpdateConfig200JSONResponse{}, response)

		jsonResponse := response.(api.AdminUpdateConfig200JSONResponse)
		require.Equal(t, config.ID, jsonResponse.Data.Id)
		require.Equal(t, config.AgentId, jsonResponse.Data.AgentId)
		require.Equal(t, updatedContent, jsonResponse.Data.Content)
		require.Equal(t, minAgentVersion, *jsonResponse.Data.MinAgentVersion)
		require.NotNil(t, jsonResponse.Data.UpdatedBy)
		require.Equal(t, user, *jsonResponse.Data.UpdatedBy)
		require.NotZero(t, jsonResponse.Data.UpdatedAt)
	})
}
