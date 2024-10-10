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

func TestListActorsWithDB(t *testing.T) {
	t.Parallel()
	logger := testhelper.Logger(t)
	ctx := context.Background()

	t.Run("Successful listing", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		// Setup actors
		fixture.InsertActor(t, ctx, dbPool, "actor1")
		fixture.InsertActor(t, ctx, dbPool, "actor2")

		request := api.AdminListActorsRequestObject{}

		response, err := admin.ListActors(ctx, logger, accessor, request)

		assert.NoError(t, err)
		require.IsType(t, api.AdminListActors200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListActors200JSONResponse)
		assert.NotNil(t, jsonResponse.Data)
		assert.Len(t, jsonResponse.Data, 2)
		assert.Equal(t, 1, jsonResponse.Meta.TotalPages)

		actualNames := lo.Map(jsonResponse.Data, func(a api.Actor, _ int) string { return a.Name })
		assert.Equal(t, []string{"actor1", "actor2"}, actualNames)

		for _, actor := range jsonResponse.Data {
			assert.NotEmpty(t, actor.Id)
			assert.NotEmpty(t, actor.Name)
			assert.NotZero(t, actor.CreatedAt)
			assert.NotEmpty(t, actor.Role)
			// Add checks for new fields
			assert.NotNil(t, actor.Enabled)
			assert.NotNil(t, actor.Deployable)
			assert.NotNil(t, actor.Configurable)
			assert.NotNil(t, actor.Renameable) // Add check for renameable
		}
	})

	t.Run("Custom page and page size", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		// Setup actors
		lo.RepeatBy(21, func(i int) *dbsqlc.Actor {
			return fixture.InsertActor(t, ctx, dbPool, fmt.Sprintf("actor-%03d", i))
		})

		request := api.AdminListActorsRequestObject{
			Params: api.AdminListActorsParams{
				Page:     lo.ToPtr(2),
				PageSize: lo.ToPtr(10),
			},
		}

		response, err := admin.ListActors(ctx, logger, accessor, request)

		assert.NoError(t, err)
		require.IsType(t, api.AdminListActors200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListActors200JSONResponse)
		assert.NotNil(t, jsonResponse.Data)
		assert.Len(t, jsonResponse.Data, 10)
		assert.Equal(t, 3, jsonResponse.Meta.TotalPages)

		expectedNames := lo.Map(lo.Range(10), func(i int, _ int) string { return fmt.Sprintf("actor-%03d", i+10) })
		actualNames := lo.Map(jsonResponse.Data, func(a api.Actor, _ int) string { return a.Name })
		assert.Equal(t, expectedNames, actualNames)

		for _, actor := range jsonResponse.Data {
			assert.NotEmpty(t, actor.Id)
			assert.NotEmpty(t, actor.Name)
			assert.NotZero(t, actor.CreatedAt)
			assert.NotEmpty(t, actor.Role)
			assert.NotNil(t, actor.Enabled)
			assert.NotNil(t, actor.Deployable)
			assert.NotNil(t, actor.Configurable)
			assert.NotNil(t, actor.Renameable) // Add check for renameable
		}
	})

	t.Run("Actor with API token", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		accessor := dbaccess.New(dbPool)

		// Setup actor
		actor := fixture.InsertActor(t, ctx, dbPool, "actor-with-token")

		// Add API token to the actor
		_, err := accessor.Querier().ApiTokenInsert(ctx, accessor.Source(), &dbsqlc.ApiTokenInsertParams{
			ID:          "test-token",
			ActorId:     actor.ID,
			Permissions: []string{"read"},
		})
		assert.NoError(t, err)

		request := api.AdminListActorsRequestObject{}
		response, err := admin.ListActors(ctx, logger, accessor, request)

		assert.NoError(t, err)
		require.IsType(t, api.AdminListActors200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListActors200JSONResponse)
		assert.NotNil(t, jsonResponse.Data)
		assert.Len(t, jsonResponse.Data, 1)

		actorResponse := jsonResponse.Data[0]
		assert.Equal(t, actor.ID, actorResponse.Id)
		assert.Equal(t, actor.Name, actorResponse.Name)
		assert.EqualValues(t, actor.Role, actorResponse.Role)
		assert.True(t, actorResponse.Enabled)
		assert.False(t, actorResponse.Deployable)
		assert.False(t, actorResponse.Configurable)
		assert.False(t, actorResponse.Renameable)
	})

	t.Run("Database pool closed", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		accessor := dbaccess.New(dbPool)

		fixture.InsertActor(t, ctx, dbPool, "actor1")
		dbPool.Close()

		request := api.AdminListActorsRequestObject{}

		response, err := admin.ListActors(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminListActors500JSONResponse{}, response)
		errorResponse := response.(api.AdminListActors500JSONResponse)
		assert.Contains(t, errorResponse.Error, "closed pool")
	})
}

func TestCreateActorWithDB(t *testing.T) {
	t.Parallel()
	logger := testhelper.Logger(t)

	ctx := context.Background()

	// Test case 1: Successful actor creation
	t.Run("Successful actor creation with all fields", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		request := api.AdminCreateActorRequestObject{
			Body: &api.AdminCreateActorJSONRequestBody{
				Name:         "TestActor",
				Role:         api.ActorCreateRole("agent"),
				Enabled:      lo.ToPtr(true),
				Deployable:   lo.ToPtr(true),
				Configurable: lo.ToPtr(true),
			},
		}

		response, err := admin.CreateActor(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminCreateActor201JSONResponse{}, response)
		jsonResponse := response.(api.AdminCreateActor201JSONResponse)
		assert.NotEmpty(t, jsonResponse.Id)
		assert.Equal(t, request.Body.Name, jsonResponse.Name)
		assert.Equal(t, api.ActorRole("agent"), jsonResponse.Role)
		assert.NotZero(t, jsonResponse.CreatedAt)

		// Check new fields
		assert.True(t, jsonResponse.Enabled)
		assert.True(t, jsonResponse.Deployable)
		assert.True(t, jsonResponse.Configurable)
		assert.True(t, jsonResponse.Renameable)

		// Verify the actor was created in the database
		actor, err := accessor.Querier().ActorFindById(ctx, accessor.Source(), jsonResponse.Id)
		assert.NoError(t, err)
		assert.Equal(t, jsonResponse.Id, actor.ID)
		assert.Equal(t, jsonResponse.Name, actor.Name)
		assert.EqualValues(t, "agent", actor.Role)
		assert.Equal(t, jsonResponse.CreatedAt, actor.CreatedAt)
		assert.Equal(t, jsonResponse.Enabled, actor.Enabled)
		assert.Equal(t, jsonResponse.Deployable, actor.Deployable)
		assert.Equal(t, jsonResponse.Configurable, actor.Configurable)

		// Verify the queue was created in the database
		queue, err := accessor.Querier().QueueFindById(ctx, accessor.Source(), actor.QueueID)
		assert.NoError(t, err)
		assert.Equal(t, actor.Name, queue.Name)
		assert.Equal(t, []byte(`{"type": "actor"}`), queue.Metadata)
	})

	t.Run("Successful actor creation with partial fields", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		request := api.AdminCreateActorRequestObject{
			Body: &api.AdminCreateActorJSONRequestBody{
				Name:         "TestActor",
				Role:         api.ActorCreateRole("agent"),
				Configurable: lo.ToPtr(true),
				Deployable:   lo.ToPtr(true),
			},
		}

		response, err := admin.CreateActor(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminCreateActor201JSONResponse{}, response)
		jsonResponse := response.(api.AdminCreateActor201JSONResponse)
		assert.NotEmpty(t, jsonResponse.Id)
		assert.Equal(t, request.Body.Name, jsonResponse.Name)
		assert.Equal(t, api.ActorRole("agent"), jsonResponse.Role)
		assert.NotZero(t, jsonResponse.CreatedAt)

		// Check new fields
		assert.True(t, jsonResponse.Enabled)
		assert.True(t, jsonResponse.Configurable)
		assert.True(t, jsonResponse.Deployable)
		assert.False(t, jsonResponse.Migratable)
		assert.True(t, jsonResponse.Renameable)

		// Verify the actor was created in the database
		actor, err := accessor.Querier().ActorFindById(ctx, accessor.Source(), jsonResponse.Id)
		assert.NoError(t, err)
		assert.Equal(t, jsonResponse.Id, actor.ID)
		assert.Equal(t, jsonResponse.Name, actor.Name)
		assert.EqualValues(t, "agent", actor.Role)
		assert.Equal(t, jsonResponse.CreatedAt, actor.CreatedAt)
		assert.Equal(t, jsonResponse.Enabled, actor.Enabled)
		assert.Equal(t, jsonResponse.Deployable, actor.Deployable)
		assert.Equal(t, jsonResponse.Configurable, actor.Configurable)
		assert.False(t, jsonResponse.Migratable)

		// Verify the queue was created in the database
		queue, err := accessor.Querier().QueueFindById(ctx, accessor.Source(), actor.QueueID)
		assert.NoError(t, err)
		assert.Equal(t, actor.Name, queue.Name)
		assert.Equal(t, []byte(`{"type": "actor"}`), queue.Metadata)
	})

	// Test case: Database error
	t.Run("Database error", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		request := api.AdminCreateActorRequestObject{
			Body: &api.AdminCreateActorJSONRequestBody{
				Name: "TestActor",
			},
		}

		dbPool.Close()
		response, err := admin.CreateActor(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminCreateActor500JSONResponse{}, response)
	})

	// Test case 3: Duplicate actor name
	t.Run("Duplicate actor name", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		// Insert an actor first
		existingActor := fixture.InsertActor(t, ctx, dbPool, "ExistingActor")

		request := api.AdminCreateActorRequestObject{
			Body: &api.AdminCreateActorJSONRequestBody{
				Name: existingActor.Name,
			},
		}

		response, err := admin.CreateActor(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminCreateActor500JSONResponse{}, response)
	})

	// Add new test cases after the existing ones
	t.Run("Invalid deployable value", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		request := api.AdminCreateActorRequestObject{
			Body: &api.AdminCreateActorJSONRequestBody{
				Name:       "TestActor",
				Role:       api.ActorCreateRole("agent"),
				Deployable: lo.ToPtr(false),
				Migratable: lo.ToPtr(true),
			},
		}

		response, err := admin.CreateActor(ctx, logger, accessor, request)
		assert.NoError(t, err)
		assert.IsType(t, api.AdminCreateActor400JSONResponse{}, response)
	})

	t.Run("Invalid configurable value", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		request := api.AdminCreateActorRequestObject{
			Body: &api.AdminCreateActorJSONRequestBody{
				Name:         "TestActor",
				Role:         api.ActorCreateRole("agent"),
				Deployable:   lo.ToPtr(true),
				Configurable: lo.ToPtr(false),
			},
		}

		response, err := admin.CreateActor(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminCreateActor400JSONResponse{}, response)
	})
}

func TestUpdateActor(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	t.Run("Successful update", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		// Insert an actor first
		existingActor := fixture.InsertActor(t, ctx, dbPool, "ExistingActor")

		request := api.AdminUpdateActorRequestObject{
			Id: existingActor.ID,
			Body: &api.AdminUpdateActorJSONRequestBody{
				Name:         lo.ToPtr("UpdatedActor"),
				Role:         lo.ToPtr(api.AdminUpdateActorJSONBodyRole("portal")),
				Enabled:      lo.ToPtr(false),
				Deployable:   lo.ToPtr(true),
				Configurable: lo.ToPtr(true),
			},
		}

		response, err := admin.UpdateActor(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminUpdateActor200JSONResponse{}, response)

		jsonResponse := response.(api.AdminUpdateActor200JSONResponse)
		assert.Equal(t, existingActor.ID, jsonResponse.Data.Id)
		assert.Equal(t, "UpdatedActor", jsonResponse.Data.Name)
		assert.Equal(t, api.ActorRole("portal"), jsonResponse.Data.Role)
		assert.False(t, jsonResponse.Data.Enabled)
		assert.True(t, jsonResponse.Data.Deployable)
		assert.True(t, jsonResponse.Data.Configurable)
	})

	t.Run("Successful update with partial parameters", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		// Insert an actor first
		existingActor := fixture.InsertActor(t, ctx, dbPool, "ExistingActor")

		// Only update name and enabled fields
		request := api.AdminUpdateActorRequestObject{
			Id: existingActor.ID,
			Body: &api.AdminUpdateActorJSONRequestBody{
				Name:    lo.ToPtr("PartiallyUpdatedActor"),
				Enabled: lo.ToPtr(false),
			},
		}

		response, err := admin.UpdateActor(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminUpdateActor200JSONResponse{}, response)

		jsonResponse := response.(api.AdminUpdateActor200JSONResponse)
		assert.Equal(t, existingActor.ID, jsonResponse.Data.Id)
		assert.Equal(t, "PartiallyUpdatedActor", jsonResponse.Data.Name)
		assert.Equal(t, api.ActorRole("agent"), jsonResponse.Data.Role)
		assert.False(t, jsonResponse.Data.Enabled)

		// Check that other fields remain unchanged
		assert.Equal(t, existingActor.Deployable, jsonResponse.Data.Deployable)
		assert.Equal(t, existingActor.Configurable, jsonResponse.Data.Configurable)

		// Verify the actor was updated in the database
		updatedActor, err := accessor.Querier().ActorFindById(ctx, accessor.Source(), existingActor.ID)
		assert.NoError(t, err)
		assert.Equal(t, "PartiallyUpdatedActor", updatedActor.Name)
		assert.False(t, updatedActor.Enabled)
		assert.Equal(t, existingActor.Deployable, updatedActor.Deployable)
		assert.Equal(t, existingActor.Configurable, updatedActor.Configurable)
	})

	t.Run("Actor not found", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		request := api.AdminUpdateActorRequestObject{
			Id: 999999, // Non-existent ID
			Body: &api.AdminUpdateActorJSONRequestBody{
				Name: lo.ToPtr("UpdatedActor"),
			},
		}

		response, err := admin.UpdateActor(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminUpdateActor404Response{}, response)
	})

	t.Run("Database error", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		// Insert an actor first
		existingActor := fixture.InsertActor(t, ctx, dbPool, "ExistingActor")

		request := api.AdminUpdateActorRequestObject{
			Id: existingActor.ID,
			Body: &api.AdminUpdateActorJSONRequestBody{
				Name: lo.ToPtr("UpdatedActor"),
			},
		}

		dbPool.Close() // Simulate database error
		response, err := admin.UpdateActor(ctx, logger, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminUpdateActor500JSONResponse{}, response)
	})
}
