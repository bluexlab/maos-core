package invocation

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func TestInsertInvocation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Test case 1: Successful invocation insertion
	t.Run("Successful invocation insertion", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		agent := fixture.InsertAgent(t, ctx, dbPool, "agent1")

		metadata := map[string]interface{}{"kind": "test"}
		payload := map[string]interface{}{"key1": 16888, "key2": map[string]interface{}{"key3": "value3"}}

		request := api.CreateInvocationAsyncRequestObject{
			Body: &api.CreateInvocationAsyncJSONRequestBody{
				Agent:   agent.Name,
				Meta:    metadata,
				Payload: payload,
			},
		}
		response, err := InsertInvocation(ctx, accessor, agent.ID, request)

		assert.NoError(t, err)
		assert.IsType(t, api.CreateInvocationAsync201JSONResponse{}, response)
		jsonResponse := response.(api.CreateInvocationAsync201JSONResponse)

		// Check if the ID is a valid integer
		id, err := strconv.ParseInt(jsonResponse.Id, 10, 64)
		assert.NoError(t, err)

		// Verify the invocation was created in the database
		invocation, err := accessor.Querier().InvocationFindById(ctx, accessor.Source(), id)
		assert.NoError(t, err)
		assert.NotNil(t, invocation)
		assert.Equal(t, dbsqlc.InvocationState("available"), invocation.State)
		assert.EqualValues(t, 1, invocation.Priority)

		// Verify the metadata
		var storedMeta map[string]interface{}
		err = json.Unmarshal(invocation.Metadata, &storedMeta)
		assert.NoError(t, err)
		require.Equal(t, metadata, storedMeta)

		// Verify the payload
		var storedPayload map[string]interface{}
		err = json.Unmarshal(invocation.Payload, &storedPayload)
		assert.NoError(t, err)
		require.EqualValues(t,
			testhelper.SerializeToJson(t, payload),
			testhelper.SerializeToJson(t, storedPayload),
		)
	})

	// Test case 2: Invalid payload (for JSON marshalling error)
	t.Run("Invalid payload", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		callerAgentID := int64(1)

		invalidPayload := map[string]interface{}{
			"key": make(chan int), // channels are not JSON-serializable
		}

		request := api.CreateInvocationAsyncRequestObject{
			Body: &api.CreateInvocationAsyncJSONRequestBody{
				Meta:    map[string]interface{}{"kind": "test"},
				Payload: invalidPayload,
			},
		}

		response, err := InsertInvocation(ctx, accessor, callerAgentID, request)

		assert.NoError(t, err)
		assert.IsType(t, api.CreateInvocationAsync500JSONResponse{}, response)
		assert.Contains(t, response.(api.CreateInvocationAsync500JSONResponse).Error, "Failed to marshal payload")
	})

	// Test case 3: Invalid agent name
	t.Run("Invalid agent name", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		agent := fixture.InsertAgent(t, ctx, dbPool, "agent1")

		request := api.CreateInvocationAsyncRequestObject{
			Body: &api.CreateInvocationAsyncJSONRequestBody{
				Agent:   "invalid-agent",
				Meta:    map[string]interface{}{"kind": "test"},
				Payload: map[string]interface{}{},
			},
		}
		response, err := InsertInvocation(ctx, accessor, agent.ID, request)

		assert.NoError(t, err)
		assert.IsType(t, api.CreateInvocationAsync400JSONResponse{}, response)
	})

	// Test case 4: Database error
	t.Run("Database error", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		callerAgentID := int64(1)

		request := api.CreateInvocationAsyncRequestObject{
			Body: &api.CreateInvocationAsyncJSONRequestBody{
				Meta:    map[string]interface{}{"kind": "test"},
				Payload: map[string]interface{}{"key": "value"},
			},
		}

		dbPool.Close() // Simulate a database error by closing the connection
		_, err := InsertInvocation(ctx, accessor, callerAgentID, request)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "closed pool")
	})
}
