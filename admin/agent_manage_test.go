package admin_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
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

	type testCase struct {
		name          string
		setupAgents   func(context.Context, *pgxpool.Pool) []*dbsqlc.Agent
		request       api.AdminListAgentsRequestObject
		expectedLen   int
		expectedNames []string
		expectedPages int
		expectedError string
	}

	tests := []testCase{
		{
			name: "Successful listing",
			setupAgents: func(ctx context.Context, dbPool *pgxpool.Pool) []*dbsqlc.Agent {
				return []*dbsqlc.Agent{
					fixture.InsertAgent(t, ctx, dbPool, "agent1"),
					fixture.InsertAgent(t, ctx, dbPool, "agent2"),
				}
			},
			request:       api.AdminListAgentsRequestObject{},
			expectedLen:   2,
			expectedNames: []string{"agent1", "agent2"},
			expectedPages: 1,
		},
		{
			name: "Custom page and page size",
			setupAgents: func(ctx context.Context, dbPool *pgxpool.Pool) []*dbsqlc.Agent {
				return lo.RepeatBy(21, func(i int) *dbsqlc.Agent {
					return fixture.InsertAgent(t, ctx, dbPool, fmt.Sprintf("agent-%03d", i))
				})
			},
			request: api.AdminListAgentsRequestObject{
				Params: api.AdminListAgentsParams{
					Page:     lo.ToPtr(2),
					PageSize: lo.ToPtr(10),
				},
			},
			expectedLen:   10,
			expectedNames: lo.Map(lo.Range(10), func(i int, _ int) string { return fmt.Sprintf("agent-%03d", i+10) }),
			expectedPages: 3,
		},
		{
			name: "Database pool closed",
			setupAgents: func(ctx context.Context, dbPool *pgxpool.Pool) []*dbsqlc.Agent {
				agents := []*dbsqlc.Agent{
					fixture.InsertAgent(t, ctx, dbPool, "agent1"),
				}
				dbPool.Close()
				return agents
			},
			request:       api.AdminListAgentsRequestObject{},
			expectedError: "closed pool",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			dbPool := testhelper.TestDB(ctx, t)
			defer dbPool.Close()

			accessor := dbaccess.New(dbPool)

			_ = tt.setupAgents(ctx, dbPool)

			response, err := admin.ListAgents(ctx, logger, accessor, tt.request)

			if tt.expectedError != "" {
				assert.NoError(t, err)
				assert.IsType(t, api.AdminListAgents500JSONResponse{}, response)
			} else {
				assert.NoError(t, err)
				require.IsType(t, api.AdminListAgents200JSONResponse{}, response)

				jsonResponse := response.(api.AdminListAgents200JSONResponse)
				assert.NotNil(t, jsonResponse.Data)
				assert.Len(t, jsonResponse.Data, tt.expectedLen)
				assert.Equal(t, tt.expectedPages, jsonResponse.Meta.TotalPages)

				actualNames := lo.Map(jsonResponse.Data, func(a api.Agent, _ int) string { return a.Name })
				assert.Equal(t, tt.expectedNames, actualNames)

				for _, agent := range jsonResponse.Data {
					assert.NotEmpty(t, agent.Id)
					assert.NotEmpty(t, agent.Name)
					assert.NotZero(t, agent.CreatedAt)
				}
			}
		})
	}
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
