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
		actor := fixture.InsertActor(t, ctx, dbPool, "TestActor")
		configSuite := fixture.InsertConfigSuite(t, ctx, dbPool)
		initialContent := map[string]string{
			"key1": "value1",
			"key2": "42",
		}
		config := fixture.InsertConfig2(t, ctx, dbPool, actor.ID, &configSuite.ID, user, initialContent)

		updatedContent := map[string]string{
			"key1": "newValue1",
			"key3": "newValue3",
		}
		minActorVersion := "2.0.0"

		request := api.AdminUpdateConfigRequestObject{
			Id: config.ID,
			Body: &api.AdminUpdateConfigJSONRequestBody{
				Content:         &updatedContent,
				MinActorVersion: &minActorVersion,
				User:            user,
			},
		}

		response, err := admin.UpdateConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminUpdateConfig200JSONResponse{}, response)

		jsonResponse := response.(api.AdminUpdateConfig200JSONResponse)
		require.Equal(t, config.ID, jsonResponse.Data.Id)
		require.Equal(t, config.ActorId, jsonResponse.Data.ActorId)
		require.Equal(t, updatedContent, jsonResponse.Data.Content)
		require.Equal(t, user, jsonResponse.Data.CreatedBy)
		require.Equal(t, actor.Name, jsonResponse.Data.ActorName)
		require.NotZero(t, jsonResponse.Data.CreatedAt)
	})

	t.Run("Config suite is deployed", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)
		defer dbPool.Close()

		user := "testuser"
		actor := fixture.InsertActor(t, ctx, dbPool, "TestActor3")
		configSuite := fixture.InsertConfigSuite(t, ctx, dbPool)

		// Set the config suite as deployed
		_, err := dbPool.Exec(ctx, "UPDATE config_suites SET deployed_at = 16888 WHERE id = $1", configSuite.ID)
		require.NoError(t, err)

		initialContent := map[string]string{
			"key1": "value1",
		}
		config := fixture.InsertConfig2(t, ctx, dbPool, actor.ID, &configSuite.ID, user, initialContent)

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

		actor := fixture.InsertActor(t, ctx, dbPool, "TestActor2")
		configSuite := fixture.InsertConfigSuite(t, ctx, dbPool)
		config := fixture.InsertConfig2(t, ctx, dbPool, actor.ID, &configSuite.ID, "testuser", map[string]string{"key": "value"})

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

		actor := fixture.InsertActor(t, ctx, dbPool, "TestActor3")

		// Insert a deployment with reviewers
		deployment, err := accessor.Querier().DeploymentInsertWithConfigSuite(ctx, accessor.Source(), &dbsqlc.DeploymentInsertWithConfigSuiteParams{
			CreatedBy: "creator",
			Name:      "TestDeployment",
			Reviewers: []string{"reviewer1", "reviewer2"},
		})
		require.NoError(t, err)

		config := fixture.InsertConfig2(t, ctx, dbPool, actor.ID, deployment.ConfigSuiteID, "creator", map[string]string{"key": "value"})

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
		updatedConfig, err := accessor.Querier().ConfigFindByActorId(ctx, accessor.Source(), actor.ID)
		require.NoError(t, err)
		require.Equal(t, `{"key": "value"}`, string(updatedConfig.Content))
	})

	t.Run("Update config if user is reviewer", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)
		defer dbPool.Close()

		actor := fixture.InsertActor(t, ctx, dbPool, "TestActor3")

		// Insert a deployment with reviewers
		deployment, err := accessor.Querier().DeploymentInsertWithConfigSuite(ctx, accessor.Source(), &dbsqlc.DeploymentInsertWithConfigSuiteParams{
			CreatedBy: "creator",
			Name:      "TestDeployment",
			Reviewers: []string{"reviewer1", "reviewer2"},
		})
		require.NoError(t, err)

		config := fixture.InsertConfig2(t, ctx, dbPool, actor.ID, deployment.ConfigSuiteID, "creator", map[string]string{"key": "value"})

		updatedContent := map[string]string{"key": "newValue"}
		user := "reviewer1"

		request := api.AdminUpdateConfigRequestObject{
			Id: config.ID,
			Body: &api.AdminUpdateConfigJSONRequestBody{
				Content: &updatedContent,
				User:    user,
			},
		}

		response, err := admin.UpdateConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminUpdateConfig200JSONResponse{}, response)

		jsonResponse := response.(api.AdminUpdateConfig200JSONResponse)
		require.Equal(t, config.ID, jsonResponse.Data.Id)
		require.Equal(t, config.ActorId, jsonResponse.Data.ActorId)
		require.Equal(t, updatedContent, jsonResponse.Data.Content)
		require.NotNil(t, jsonResponse.Data.UpdatedBy)
		require.Equal(t, user, *jsonResponse.Data.UpdatedBy)
		require.NotZero(t, jsonResponse.Data.UpdatedAt)
	})

	t.Run("Invalid Kubernetes config", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)
		defer dbPool.Close()

		user := "testuser"
		actor := fixture.InsertActor2(t, ctx, dbPool, "TestActor", "agent", true, true, true, false)
		configSuite := fixture.InsertConfigSuite(t, ctx, dbPool)
		initialContent := map[string]string{
			"KUBE_REPLICAS": "1",
		}
		config := fixture.InsertConfig2(t, ctx, dbPool, actor.ID, &configSuite.ID, user, initialContent)

		invalidKubeConfig := map[string]string{
			"KUBE_REPLICAS": "invalid",
		}

		request := api.AdminUpdateConfigRequestObject{
			Id: config.ID,
			Body: &api.AdminUpdateConfigJSONRequestBody{
				Content: &invalidKubeConfig,
				User:    user,
			},
		}

		response, err := admin.UpdateConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminUpdateConfig400JSONResponse{}, response)
		jsonResponse := response.(api.AdminUpdateConfig400JSONResponse)
		require.Contains(t, jsonResponse.Error, "invalid replicas")
	})

	t.Run("Valid Kubernetes config", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)
		defer dbPool.Close()

		user := "testuser"
		actor := fixture.InsertActor(t, ctx, dbPool, "TestActor")
		configSuite := fixture.InsertConfigSuite(t, ctx, dbPool)
		initialContent := map[string]string{
			"KUBE_REPLICAS": "1",
		}
		config := fixture.InsertConfig2(t, ctx, dbPool, actor.ID, &configSuite.ID, user, initialContent)

		validKubeConfig := map[string]string{
			"KUBE_REPLICAS":       "2",
			"KUBE_DOCKER_IMAGE":   "myregistry.com/myimage:latest",
			"KUBE_CPU_REQUEST":    "200m",
			"KUBE_MEMORY_REQUEST": "256Mi",
		}

		request := api.AdminUpdateConfigRequestObject{
			Id: config.ID,
			Body: &api.AdminUpdateConfigJSONRequestBody{
				Content: &validKubeConfig,
				User:    user,
			},
		}

		response, err := admin.UpdateConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminUpdateConfig200JSONResponse{}, response)
		jsonResponse := response.(api.AdminUpdateConfig200JSONResponse)
		require.Equal(t, validKubeConfig["KUBE_REPLICAS"], jsonResponse.Data.Content["KUBE_REPLICAS"])
		require.Equal(t, validKubeConfig["KUBE_DOCKER_IMAGE"], jsonResponse.Data.Content["KUBE_DOCKER_IMAGE"])
		require.Equal(t, validKubeConfig["KUBE_CPU_REQUEST"], jsonResponse.Data.Content["KUBE_CPU_REQUEST"])
		require.Equal(t, validKubeConfig["KUBE_MEMORY_REQUEST"], jsonResponse.Data.Content["KUBE_MEMORY_REQUEST"])
	})
}
