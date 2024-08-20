package admin_test

import (
	"context"
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
		content := map[string]string{
			"key1": "value1",
			"key2": "42",
		}
		content0 := map[string]string{"key1": "value2"}
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
		assert.Equal(t, agent.Name, jsonResponse.Data.AgentName)
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
