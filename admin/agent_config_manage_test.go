package admin_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/admin"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func TestAdminGetAgentConfig(t *testing.T) {
	t.Parallel()
	logger := testhelper.Logger(t)

	ctx := context.Background()
	dbPool := testhelper.TestDB(ctx, t)
	defer dbPool.Close()

	accessor := dbaccess.New(dbPool)

	t.Run("Successful get agent config", func(t *testing.T) {
		agent := fixture.InsertAgent(t, ctx, dbPool, "TestAgent")
		agent2 := fixture.InsertAgent(t, ctx, dbPool, "TestAgent2")
		content := map[string]interface{}{
			"key1": "value1",
			"key2": "42",
		}
		content0 := map[string]interface{}{"key1": "value2"}
		fixture.InsertConfig(t, ctx, dbPool, agent.ID, content0)
		fixture.InsertConfig(t, ctx, dbPool, agent2.ID, content0)
		config := fixture.InsertConfig(t, ctx, dbPool, agent.ID, content)

		request := api.AdminGetAgentConfigRequestObject{
			Id: int(agent.ID),
		}

		response, err := admin.AdminGetAgentConfig(ctx, logger, accessor, request)

		assert.NoError(t, err)
		require.IsType(t, api.AdminGetAgentConfig200JSONResponse{}, response)

		jsonResponse := response.(api.AdminGetAgentConfig200JSONResponse)
		assert.Equal(t, config.ID, jsonResponse.Data.Id)
		assert.Equal(t, config.AgentId, jsonResponse.Data.AgentId)
		assert.Equal(t, content, jsonResponse.Data.Content)
		assert.NotZero(t, jsonResponse.Data.CreatedAt)
	})

	t.Run("Agent not found", func(t *testing.T) {
		request := api.AdminGetAgentConfigRequestObject{
			Id: 999999, // Non-existent agent ID
		}

		response, err := admin.AdminGetAgentConfig(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminGetAgentConfig404Response{}, response)
	})

	t.Run("Database error", func(t *testing.T) {
		request := api.AdminGetAgentConfigRequestObject{
			Id: 1,
		}

		dbPool.Close() // Simulate database error by closing the connection

		response, err := admin.AdminGetAgentConfig(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminGetAgentConfig500JSONResponse{}, response)
	})
}

func TestAdminUpdateAgentConfig(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	dbPool := testhelper.TestDB(ctx, t)
	defer dbPool.Close()

	accessor := dbaccess.New(dbPool)

	t.Run("Successful update agent config", func(t *testing.T) {
		agent := fixture.InsertAgent(t, ctx, dbPool, "TestAgent")
		content := map[string]interface{}{
			"key1": "value1",
			"key2": 42.0,
		}
		minAgentVersion := "1.0.0"
		user := "testuser"

		request := api.AdminUpdateAgentConfigRequestObject{
			Id: agent.ID,
			Body: &api.AdminUpdateAgentConfigJSONRequestBody{
				Content:         content,
				MinAgentVersion: &minAgentVersion,
				User:            user,
			},
		}

		response, err := admin.AdminUpdateAgentConfig(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminUpdateAgentConfig201Response{}, response)

		// Verify the config was inserted correctly
		config, err := accessor.Querier().ConfigFindByAgentId(ctx, accessor.Pool(), agent.ID)
		assert.NoError(t, err)
		assert.Equal(t, agent.ID, config.AgentId)
		assert.Equal(t, minAgentVersion, *config.MinAgentVersion)
		assert.Equal(t, user, config.CreatedBy)

		var insertedContent map[string]interface{}
		err = json.Unmarshal(config.Content, &insertedContent)
		assert.NoError(t, err)
		assert.Equal(t, content, insertedContent)
	})

	t.Run("Invalid content", func(t *testing.T) {
		agent := fixture.InsertAgent(t, ctx, dbPool, "TestAgent2")
		invalidContent := map[string]interface{}{
			"key": make(chan int), // channels are not JSON serializable
		}
		user := "testuser"

		request := api.AdminUpdateAgentConfigRequestObject{
			Id: agent.ID,
			Body: &api.AdminUpdateAgentConfigJSONRequestBody{
				Content: invalidContent,
				User:    user,
			},
		}

		response, err := admin.AdminUpdateAgentConfig(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminUpdateAgentConfig500JSONResponse{}, response)
	})

	t.Run("Database error", func(t *testing.T) {
		agent := fixture.InsertAgent(t, ctx, dbPool, "TestAgent3")
		content := map[string]interface{}{
			"key": "value",
		}
		user := "testuser"

		request := api.AdminUpdateAgentConfigRequestObject{
			Id: agent.ID,
			Body: &api.AdminUpdateAgentConfigJSONRequestBody{
				Content: content,
				User:    user,
			},
		}

		dbPool.Close() // Simulate database error by closing the connection

		response, err := admin.AdminUpdateAgentConfig(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminUpdateAgentConfig500JSONResponse{}, response)
	})
}
