package handler_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/handler"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
	"gitlab.com/navyx/ai/maos/maos-core/middleware"
)

func TestGetAgentConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := testhelper.Logger(t)

	setupGetAgentConfigTest := func(t *testing.T) (*pgxpool.Pool, dbaccess.Accessor, *dbsqlc.Agent, *dbsqlc.Config) {
		t.Helper()
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)

		// Create an agent
		agent := fixture.InsertAgent(t, ctx, dbPool, "test-agent")

		// Create a config for the agent
		configContent := map[string]string{"key1": "value1", "key2": "value2"}
		fixture.InsertConfig2(t, ctx, dbPool, agent.ID, nil, "test-user", configContent)

		// Create additional config suites
		inactiveConfigSuite := fixture.InsertConfigSuite(t, ctx, dbPool)
		activeConfigSuite := fixture.InsertConfigSuite(t, ctx, dbPool)
		inactiveConfigSuite2 := fixture.InsertConfigSuite(t, ctx, dbPool)

		// Set the active config suite
		_, err := dbPool.Exec(ctx, "UPDATE config_suites SET deployed_at = EXTRACT(EPOCH FROM NOW()), active = TRUE WHERE id = $1", activeConfigSuite.ID)
		require.NoError(t, err)

		// Set the inactive config suite
		_, err = dbPool.Exec(ctx, "UPDATE config_suites SET deployed_at = EXTRACT(EPOCH FROM NOW()), active = FALSE WHERE id <> $1", activeConfigSuite.ID)
		require.NoError(t, err)

		// Create additional configs for the agent
		fixture.InsertConfig2(t, ctx, dbPool, agent.ID, &inactiveConfigSuite.ID, "test-user", map[string]string{"inactive_key": "inactive_value"})
		fixture.InsertConfig2(t, ctx, dbPool, agent.ID, &inactiveConfigSuite2.ID, "test-user", map[string]string{"inactive_key_2": "inactive_value_2"})
		activeConfig := fixture.InsertConfig2(t, ctx, dbPool, agent.ID, &activeConfigSuite.ID, "test-user", map[string]string{"active_key": "active_value"})

		return dbPool, accessor, agent, activeConfig
	}

	t.Run("Successfully get agent config", func(t *testing.T) {
		t.Parallel()
		_, accessor, agent, _ := setupGetAgentConfigTest(t)

		// Create a mock context with a token
		token := &middleware.Token{AgentId: agent.ID}
		ctx := context.WithValue(context.Background(), middleware.TokenContextKey, token)

		request := api.GetCallerConfigRequestObject{}
		response, err := handler.GetAgentConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.GetCallerConfig200JSONResponse{}, response)

		jsonResponse := response.(api.GetCallerConfig200JSONResponse)
		require.NotNil(t, jsonResponse.Data)
		require.Equal(t, 1, len(jsonResponse.Data))
		require.Equal(t, "active_value", jsonResponse.Data["active_key"])
	})

	t.Run("Active config not version compatible", func(t *testing.T) {
		t.Parallel()
		dbPool, accessor, agent, config := setupGetAgentConfigTest(t)

		// Update the active config with a higher minimum agent version
		_, err := dbPool.Exec(ctx, "UPDATE configs SET min_agent_version = '{2,0,0}' WHERE id = $1", config.ID)
		require.NoError(t, err)

		// Create a mock context with a token and lower agent version
		token := &middleware.Token{AgentId: agent.ID}
		ctx := context.WithValue(context.Background(), middleware.TokenContextKey, token)

		request := api.GetCallerConfigRequestObject{
			Params: api.GetCallerConfigParams{
				XAgentVersion: lo.ToPtr("1.0.0"),
			},
		}
		response, err := handler.GetAgentConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.GetCallerConfig200JSONResponse{}, response)

		jsonResponse := response.(api.GetCallerConfig200JSONResponse)
		require.NotNil(t, jsonResponse.Data)
		require.Equal(t, 1, len(jsonResponse.Data))
		require.Equal(t, "inactive_value_2", jsonResponse.Data["inactive_key_2"])
	})

	t.Run("No compatible config found", func(t *testing.T) {
		t.Parallel()
		dbPool, accessor, agent, _ := setupGetAgentConfigTest(t)

		// Update all configs with a higher minimum agent version
		_, err := dbPool.Exec(ctx, "UPDATE configs SET min_agent_version = '{3,0,0}'")
		require.NoError(t, err)

		// Create a mock context with a token and lower agent version
		token := &middleware.Token{AgentId: agent.ID}
		ctx := context.WithValue(context.Background(), middleware.TokenContextKey, token)

		request := api.GetCallerConfigRequestObject{
			Params: api.GetCallerConfigParams{
				XAgentVersion: lo.ToPtr("1.0.0"),
			},
		}
		response, err := handler.GetAgentConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.GetCallerConfig404Response{}, response)
	})

	t.Run("No token in context", func(t *testing.T) {
		t.Parallel()
		_, accessor, _, _ := setupGetAgentConfigTest(t)

		request := api.GetCallerConfigRequestObject{}
		response, err := handler.GetAgentConfig(context.Background(), logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.GetCallerConfig401Response{}, response)
	})

	t.Run("Agent not found", func(t *testing.T) {
		t.Parallel()
		_, accessor, _, _ := setupGetAgentConfigTest(t)

		// Create a mock context with a non-existent agent ID
		token := &middleware.Token{AgentId: 999999}
		ctx := context.WithValue(context.Background(), middleware.TokenContextKey, token)

		request := api.GetCallerConfigRequestObject{}
		response, err := handler.GetAgentConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.GetCallerConfig404Response{}, response)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool, accessor, agent, _ := setupGetAgentConfigTest(t)

		// Create a mock context with a token
		token := &middleware.Token{AgentId: agent.ID}
		ctx := context.WithValue(context.Background(), middleware.TokenContextKey, token)

		// Close the database pool to simulate a database error
		dbPool.Close()

		request := api.GetCallerConfigRequestObject{}
		response, err := handler.GetAgentConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.GetCallerConfig500JSONResponse{}, response)

		errorResponse := response.(api.GetCallerConfig500JSONResponse)
		require.Contains(t, errorResponse.Error, "Cannot get agent config")
	})

	t.Run("Invalid config content", func(t *testing.T) {
		t.Parallel()
		dbPool, accessor, agent, config := setupGetAgentConfigTest(t)

		// Update the config with invalid JSON content
		_, err := dbPool.Exec(ctx, "UPDATE configs SET content = $1 WHERE id = $2", `"invalid_json"`, config.ID)
		require.NoError(t, err)

		// Create a mock context with a token
		token := &middleware.Token{AgentId: agent.ID}
		ctx := context.WithValue(context.Background(), middleware.TokenContextKey, token)

		request := api.GetCallerConfigRequestObject{}
		response, err := handler.GetAgentConfig(ctx, logger, accessor, request)

		require.NoError(t, err)
		require.IsType(t, api.GetCallerConfig500JSONResponse{}, response)

		errorResponse := response.(api.GetCallerConfig500JSONResponse)
		require.Contains(t, errorResponse.Error, "Cannot unmarshal agent config")
	})
}
