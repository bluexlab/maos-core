package admin_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/admin"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func TestListAgentsWithDB(t *testing.T) {
	t.Parallel()
	logger := testhelper.Logger(t)
	ctx := context.Background()

	t.Run("Successful listing", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		// Setup agents
		fixture.InsertAgent(t, ctx, dbPool, "agent1")
		fixture.InsertAgent(t, ctx, dbPool, "agent2")

		request := api.AdminListAgentsRequestObject{}

		response, err := admin.ListAgents(ctx, logger, accessor, request)

		assert.NoError(t, err)
		require.IsType(t, api.AdminListAgents200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListAgents200JSONResponse)
		assert.NotNil(t, jsonResponse.Data)
		assert.Len(t, jsonResponse.Data, 2)
		assert.Equal(t, 1, jsonResponse.Meta.TotalPages)

		actualNames := lo.Map(jsonResponse.Data, func(a api.Agent, _ int) string { return a.Name })
		assert.Equal(t, []string{"agent1", "agent2"}, actualNames)

		for _, agent := range jsonResponse.Data {
			assert.NotEmpty(t, agent.Id)
			assert.NotEmpty(t, agent.Name)
			assert.NotZero(t, agent.CreatedAt)
			assert.True(t, agent.Updatable)
		}
	})

	t.Run("Custom page and page size", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		// Setup agents
		lo.RepeatBy(21, func(i int) *dbsqlc.Agent {
			return fixture.InsertAgent(t, ctx, dbPool, fmt.Sprintf("agent-%03d", i))
		})

		request := api.AdminListAgentsRequestObject{
			Params: api.AdminListAgentsParams{
				Page:     lo.ToPtr(2),
				PageSize: lo.ToPtr(10),
			},
		}

		response, err := admin.ListAgents(ctx, logger, accessor, request)

		assert.NoError(t, err)
		require.IsType(t, api.AdminListAgents200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListAgents200JSONResponse)
		assert.NotNil(t, jsonResponse.Data)
		assert.Len(t, jsonResponse.Data, 10)
		assert.Equal(t, 3, jsonResponse.Meta.TotalPages)

		expectedNames := lo.Map(lo.Range(10), func(i int, _ int) string { return fmt.Sprintf("agent-%03d", i+10) })
		actualNames := lo.Map(jsonResponse.Data, func(a api.Agent, _ int) string { return a.Name })
		assert.Equal(t, expectedNames, actualNames)

		for _, agent := range jsonResponse.Data {
			assert.NotEmpty(t, agent.Id)
			assert.NotEmpty(t, agent.Name)
			assert.NotZero(t, agent.CreatedAt)
			assert.True(t, agent.Updatable)
		}
	})

	t.Run("Agent with API token is not updatable", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		// Setup agent
		agent := fixture.InsertAgent(t, ctx, dbPool, "agent-with-token")

		// Add API token to the agent
		_, err := accessor.Querier().ApiTokenInsert(ctx, accessor.Source(), &dbsqlc.ApiTokenInsertParams{
			ID:          "test-token",
			AgentId:     agent.ID,
			Permissions: []string{"read"},
		})
		assert.NoError(t, err)

		request := api.AdminListAgentsRequestObject{}
		response, err := admin.ListAgents(ctx, logger, accessor, request)

		assert.NoError(t, err)
		require.IsType(t, api.AdminListAgents200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListAgents200JSONResponse)
		assert.NotNil(t, jsonResponse.Data)
		assert.Len(t, jsonResponse.Data, 1)

		agentResponse := jsonResponse.Data[0]
		assert.Equal(t, agent.ID, agentResponse.Id)
		assert.Equal(t, agent.Name, agentResponse.Name)
		assert.False(t, agentResponse.Updatable)
	})

	t.Run("Database pool closed", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		accessor := dbaccess.New(dbPool)

		fixture.InsertAgent(t, ctx, dbPool, "agent1")
		dbPool.Close()

		request := api.AdminListAgentsRequestObject{}

		response, err := admin.ListAgents(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminListAgents500JSONResponse{}, response)
		errorResponse := response.(api.AdminListAgents500JSONResponse)
		assert.Contains(t, errorResponse.Error, "closed pool")
	})
}

func TestCreateAgentWithDB(t *testing.T) {
	t.Parallel()
	logger := testhelper.Logger(t)

	ctx := context.Background()

	// Test case 1: Successful agent creation
	t.Run("Successful agent creation", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		request := api.AdminCreateAgentRequestObject{
			Body: &api.AdminCreateAgentJSONRequestBody{
				Name: "TestAgent",
			},
		}

		response, err := admin.CreateAgent(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminCreateAgent201JSONResponse{}, response)
		jsonResponse := response.(api.AdminCreateAgent201JSONResponse)
		assert.NotEmpty(t, jsonResponse.Id)
		assert.Equal(t, request.Body.Name, jsonResponse.Name)
		assert.NotZero(t, jsonResponse.CreatedAt)

		// Verify the agent was created in the database
		agent, err := accessor.Querier().AgentFindById(ctx, accessor.Source(), jsonResponse.Id)
		assert.NoError(t, err)
		assert.Equal(t, jsonResponse.Id, agent.ID)
		assert.Equal(t, jsonResponse.Name, agent.Name)
		assert.Equal(t, jsonResponse.CreatedAt, agent.CreatedAt)

		// Verify the queue was created in the database
		queue, err := accessor.Querier().QueueFindById(ctx, accessor.Source(), agent.QueueID)
		assert.NoError(t, err)
		assert.Equal(t, agent.Name, queue.Name)
		assert.Equal(t, []byte(`{"type": "agent"}`), queue.Metadata)
	})

	// Test case 2: Database error
	t.Run("Database error", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		request := api.AdminCreateAgentRequestObject{
			Body: &api.AdminCreateAgentJSONRequestBody{
				Name: "TestAgent",
			},
		}

		dbPool.Close()
		response, err := admin.CreateAgent(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminCreateAgent500JSONResponse{}, response)
	})

	// Test case 3: Duplicate agent name
	t.Run("Duplicate agent name", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		// Insert an agent first
		existingAgent := fixture.InsertAgent(t, ctx, dbPool, "ExistingAgent")

		request := api.AdminCreateAgentRequestObject{
			Body: &api.AdminCreateAgentJSONRequestBody{
				Name: existingAgent.Name,
			},
		}

		response, err := admin.CreateAgent(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminCreateAgent500JSONResponse{}, response)
	})
}
