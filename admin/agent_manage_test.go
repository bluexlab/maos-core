package admin_test

import (
	"context"
	"fmt"
	"log/slog"
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
			// Add checks for new fields
			assert.NotNil(t, agent.Enabled)
			assert.NotNil(t, agent.Deployable)
			assert.NotNil(t, agent.Configurable)
			assert.NotNil(t, agent.Renameable) // Add check for renameable
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
			// Add checks for new fields
			assert.NotNil(t, agent.Enabled)
			assert.NotNil(t, agent.Deployable)
			assert.NotNil(t, agent.Configurable)
			assert.NotNil(t, agent.Renameable) // Add check for renameable
		}
	})

	t.Run("Agent with API token", func(t *testing.T) {
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
		assert.True(t, agentResponse.Enabled)
		assert.False(t, agentResponse.Deployable)
		assert.False(t, agentResponse.Configurable)
		assert.False(t, agentResponse.Renameable)
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
	t.Run("Successful agent creation with all fields", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		request := api.AdminCreateAgentRequestObject{
			Body: &api.AdminCreateAgentJSONRequestBody{
				Name:         "TestAgent",
				Enabled:      lo.ToPtr(true),
				Deployable:   lo.ToPtr(true),
				Configurable: lo.ToPtr(true),
			},
		}

		response, err := admin.CreateAgent(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminCreateAgent201JSONResponse{}, response)
		jsonResponse := response.(api.AdminCreateAgent201JSONResponse)
		assert.NotEmpty(t, jsonResponse.Id)
		assert.Equal(t, request.Body.Name, jsonResponse.Name)
		assert.NotZero(t, jsonResponse.CreatedAt)

		// Check new fields
		assert.True(t, jsonResponse.Enabled)
		assert.True(t, jsonResponse.Deployable)
		assert.True(t, jsonResponse.Configurable)
		assert.True(t, jsonResponse.Renameable)

		// Verify the agent was created in the database
		agent, err := accessor.Querier().AgentFindById(ctx, accessor.Source(), jsonResponse.Id)
		assert.NoError(t, err)
		assert.Equal(t, jsonResponse.Id, agent.ID)
		assert.Equal(t, jsonResponse.Name, agent.Name)
		assert.Equal(t, jsonResponse.CreatedAt, agent.CreatedAt)
		assert.Equal(t, jsonResponse.Enabled, agent.Enabled)
		assert.Equal(t, jsonResponse.Deployable, agent.Deployable)
		assert.Equal(t, jsonResponse.Configurable, agent.Configurable)

		// Verify the queue was created in the database
		queue, err := accessor.Querier().QueueFindById(ctx, accessor.Source(), agent.QueueID)
		assert.NoError(t, err)
		assert.Equal(t, agent.Name, queue.Name)
		assert.Equal(t, []byte(`{"type": "agent"}`), queue.Metadata)
	})

	t.Run("Successful agent creation with partial fields", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		request := api.AdminCreateAgentRequestObject{
			Body: &api.AdminCreateAgentJSONRequestBody{
				Name:       "TestAgent",
				Deployable: lo.ToPtr(true),
			},
		}

		response, err := admin.CreateAgent(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminCreateAgent201JSONResponse{}, response)
		jsonResponse := response.(api.AdminCreateAgent201JSONResponse)
		assert.NotEmpty(t, jsonResponse.Id)
		assert.Equal(t, request.Body.Name, jsonResponse.Name)
		assert.NotZero(t, jsonResponse.CreatedAt)

		// Check new fields
		assert.True(t, jsonResponse.Enabled)
		assert.True(t, jsonResponse.Deployable)
		assert.False(t, jsonResponse.Configurable)
		assert.True(t, jsonResponse.Renameable)

		// Verify the agent was created in the database
		agent, err := accessor.Querier().AgentFindById(ctx, accessor.Source(), jsonResponse.Id)
		assert.NoError(t, err)
		assert.Equal(t, jsonResponse.Id, agent.ID)
		assert.Equal(t, jsonResponse.Name, agent.Name)
		assert.Equal(t, jsonResponse.CreatedAt, agent.CreatedAt)
		assert.Equal(t, jsonResponse.Enabled, agent.Enabled)
		assert.Equal(t, jsonResponse.Deployable, agent.Deployable)
		assert.False(t, jsonResponse.Configurable)

		// Verify the queue was created in the database
		queue, err := accessor.Querier().QueueFindById(ctx, accessor.Source(), agent.QueueID)
		assert.NoError(t, err)
		assert.Equal(t, agent.Name, queue.Name)
		assert.Equal(t, []byte(`{"type": "agent"}`), queue.Metadata)
	})

	// Test case: Database error
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

func TestUpdateAgent(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	t.Run("Successful update", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		// Insert an agent first
		existingAgent := fixture.InsertAgent(t, ctx, dbPool, "ExistingAgent")

		request := api.AdminUpdateAgentRequestObject{
			Id: existingAgent.ID,
			Body: &api.AdminUpdateAgentJSONRequestBody{
				Name:         lo.ToPtr("UpdatedAgent"),
				Enabled:      lo.ToPtr(false),
				Deployable:   lo.ToPtr(true),
				Configurable: lo.ToPtr(true),
			},
		}

		response, err := admin.UpdateAgent(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminUpdateAgent200JSONResponse{}, response)

		jsonResponse := response.(api.AdminUpdateAgent200JSONResponse)
		assert.Equal(t, existingAgent.ID, jsonResponse.Data.Id)
		assert.Equal(t, "UpdatedAgent", jsonResponse.Data.Name)
		assert.False(t, jsonResponse.Data.Enabled)
		assert.True(t, jsonResponse.Data.Deployable)
		assert.True(t, jsonResponse.Data.Configurable)
	})

	t.Run("Successful update with partial parameters", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		// Insert an agent first
		existingAgent := fixture.InsertAgent(t, ctx, dbPool, "ExistingAgent")

		// Only update name and enabled fields
		request := api.AdminUpdateAgentRequestObject{
			Id: existingAgent.ID,
			Body: &api.AdminUpdateAgentJSONRequestBody{
				Name:    lo.ToPtr("PartiallyUpdatedAgent"),
				Enabled: lo.ToPtr(false),
			},
		}

		response, err := admin.UpdateAgent(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminUpdateAgent200JSONResponse{}, response)

		jsonResponse := response.(api.AdminUpdateAgent200JSONResponse)
		assert.Equal(t, existingAgent.ID, jsonResponse.Data.Id)
		assert.Equal(t, "PartiallyUpdatedAgent", jsonResponse.Data.Name)
		assert.False(t, jsonResponse.Data.Enabled)

		// Check that other fields remain unchanged
		assert.Equal(t, existingAgent.Deployable, jsonResponse.Data.Deployable)
		assert.Equal(t, existingAgent.Configurable, jsonResponse.Data.Configurable)

		// Verify the agent was updated in the database
		updatedAgent, err := accessor.Querier().AgentFindById(ctx, accessor.Source(), existingAgent.ID)
		assert.NoError(t, err)
		assert.Equal(t, "PartiallyUpdatedAgent", updatedAgent.Name)
		assert.False(t, updatedAgent.Enabled)
		assert.Equal(t, existingAgent.Deployable, updatedAgent.Deployable)
		assert.Equal(t, existingAgent.Configurable, updatedAgent.Configurable)
	})

	t.Run("Agent not found", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		request := api.AdminUpdateAgentRequestObject{
			Id: 999999, // Non-existent ID
			Body: &api.AdminUpdateAgentJSONRequestBody{
				Name: lo.ToPtr("UpdatedAgent"),
			},
		}

		response, err := admin.UpdateAgent(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminUpdateAgent404Response{}, response)
	})

	t.Run("Database error", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		// Insert an agent first
		existingAgent := fixture.InsertAgent(t, ctx, dbPool, "ExistingAgent")

		request := api.AdminUpdateAgentRequestObject{
			Id: existingAgent.ID,
			Body: &api.AdminUpdateAgentJSONRequestBody{
				Name: lo.ToPtr("UpdatedAgent"),
			},
		}

		dbPool.Close() // Simulate database error
		response, err := admin.UpdateAgent(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminUpdateAgent500JSONResponse{}, response)
	})
}
