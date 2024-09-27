package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/samber/lo"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
	"gitlab.com/navyx/ai/maos/maos-core/util"
)

func TestListApiTokensWithDB(t *testing.T) {
	t.Parallel()

	expireAt := time.Now().Add(24 * time.Hour).Unix()

	t.Run("Successful listing", func(t *testing.T) {
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)
		actor1 := fixture.InsertActor(t, ctx, dbPool, "actor1")
		actor2 := fixture.InsertActor(t, ctx, dbPool, "actor2")
		fixture.InsertToken(t, ctx, dbPool, "token001", actor1.ID, expireAt, []string{"invocation:read", "invocation:create"})
		fixture.InsertToken(t, ctx, dbPool, "token002", actor2.ID, expireAt, []string{"admin"})

		request := api.AdminListApiTokensRequestObject{
			Params: api.AdminListApiTokensParams{
				Page:     nil,
				PageSize: nil,
			},
		}

		response, err := ListApiTokens(ctx, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminListApiTokens200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListApiTokens200JSONResponse)
		assert.NotNil(t, jsonResponse.Data)
		assert.Len(t, jsonResponse.Data, 2)
		require.Equal(t, 1, jsonResponse.Meta.TotalPages)

		expectedResponse := []api.ApiToken{
			{
				Id:          "token001",
				ActorId:     actor1.ID,
				ExpireAt:    expireAt,
				CreatedBy:   "test",
				Permissions: []api.Permission{api.InvocationRead, api.InvocationCreate},
			},
			{
				Id:          "token002",
				ActorId:     actor2.ID,
				ExpireAt:    expireAt,
				CreatedBy:   "test",
				Permissions: []api.Permission{api.Admin},
			},
		}

		for i := 0; i < len(expectedResponse); i++ {
			testhelper.AssertEqualIgnoringFields(t, expectedResponse[i], jsonResponse.Data[i], "CreatedAt")
		}
	})

	t.Run("Custom page and page size", func(t *testing.T) {
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		page := 2
		pageSize := 10
		request := api.AdminListApiTokensRequestObject{
			Params: api.AdminListApiTokensParams{
				Page:     &page,
				PageSize: &pageSize,
			},
		}

		createdAt := time.Now().Unix() - 100000

		lo.RepeatBy(21, func(i int) *dbsqlc.ApiToken {
			_, token := fixture.InsertActorToken(t, ctx, dbPool, fmt.Sprintf("token-%03d", i), expireAt, []string{"read"}, createdAt+int64(i))
			return token
		})

		accessor := dbaccess.New(dbPool)
		response, err := ListApiTokens(ctx, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminListApiTokens200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListApiTokens200JSONResponse)
		assert.NotNil(t, jsonResponse.Data)
		assert.Len(t, jsonResponse.Data, 10)
		assert.Equal(t, 3, jsonResponse.Meta.TotalPages)

		expectedTokenIds := lo.RepeatBy(10, func(i int) string { return fmt.Sprintf("token-%03d", 10-i) })
		assert.Equal(t,
			expectedTokenIds,
			util.MapSlice(jsonResponse.Data, func(t api.ApiToken) string {
				return t.Id
			}),
		)
	})

	t.Run("Database error", func(t *testing.T) {
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		request := api.AdminListApiTokensRequestObject{}

		accessor := dbaccess.New(dbPool)
		dbPool.Close()

		response, err := ListApiTokens(ctx, accessor, request)

		assert.NoError(t, err)
		assert.EqualValues(t,
			api.AdminListApiTokens500JSONResponse{
				N500JSONResponse: api.N500JSONResponse{Error: "Cannot list API tokens: closed pool"},
			},
			response)
	})
}

func TestCreateApiTokenWithDB(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	expireAt := time.Now().Add(24 * time.Hour).Unix()

	// Test case 1: Successful API token creation
	t.Run("Successful API token creation", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)
		actor1 := fixture.InsertActor(t, ctx, dbPool, "actor1")

		request := api.AdminCreateApiTokenRequestObject{
			Body: &api.AdminCreateApiTokenJSONRequestBody{
				ActorId:     actor1.ID,
				CreatedBy:   "admin",
				Permissions: []string{"read", "write"},
				ExpireAt:    expireAt,
			},
		}

		expectedApiToken := dbsqlc.ApiToken{
			ActorId:     request.Body.ActorId,
			CreatedBy:   request.Body.CreatedBy,
			Permissions: request.Body.Permissions,
			ExpireAt:    request.Body.ExpireAt,
		}

		response, err := CreateApiToken(ctx, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminCreateApiToken201JSONResponse{}, response)
		jsonResponse := response.(api.AdminCreateApiToken201JSONResponse)
		assert.Equal(t, tokenLength+3, len(jsonResponse.Id))
		assert.True(t, strings.HasPrefix(jsonResponse.Id, "ma-"))
		assert.Equal(t, expectedApiToken.ActorId, jsonResponse.ActorId)
		assert.Equal(t, expectedApiToken.CreatedBy, jsonResponse.CreatedBy)
		assert.Equal(t,
			util.MapSlice(expectedApiToken.Permissions, func(p string) api.Permission { return api.Permission(p) }),
			jsonResponse.Permissions,
		)
		assert.Equal(t, expectedApiToken.ExpireAt, jsonResponse.ExpireAt)

		apiToken, err := accessor.Querier().ApiTokenFindByID(ctx, accessor.Source(), jsonResponse.Id)
		assert.NoError(t, err)
		assert.Equal(t, expectedApiToken.ActorId, apiToken.ActorId)
		assert.Equal(t, expectedApiToken.CreatedBy, apiToken.CreatedBy)
		assert.Equal(t, expectedApiToken.Permissions, apiToken.Permissions)
		assert.Equal(t, expectedApiToken.ExpireAt, apiToken.ExpireAt)
	})

	// Test case 2: Database error
	t.Run("Database error", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)
		actor1 := fixture.InsertActor(t, ctx, dbPool, "actor1")
		request := api.AdminCreateApiTokenRequestObject{
			Body: &api.AdminCreateApiTokenJSONRequestBody{
				ActorId:     actor1.ID,
				CreatedBy:   "admin",
				Permissions: []string{"read"},
				ExpireAt:    expireAt,
			},
		}

		dbPool.Close()
		response, err := CreateApiToken(ctx, accessor, request)

		assert.NoError(t, err)
		assert.EqualValues(t,
			api.AdminCreateApiToken500JSONResponse{
				N500JSONResponse: api.N500JSONResponse{Error: "Cannot insert API tokens: closed pool"},
			},
			response)
	})
}

func TestDeleteApiToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Test case 1: Successful deletion
	t.Run("Successful deletion", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)
		actor := fixture.InsertActor(t, ctx, dbPool, "actor1")
		token := fixture.InsertToken(t, ctx, dbPool, "token001", actor.ID, time.Now().Add(24*time.Hour).Unix(), []string{"read"})

		request := api.AdminDeleteApiTokenRequestObject{
			Id: token.ID,
		}

		response, err := DeleteApiToken(ctx, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminDeleteApiToken204Response{}, response)

		// Verify token is deleted
		_, err = accessor.Querier().ApiTokenFindByID(ctx, accessor.Source(), token.ID)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, pgx.ErrNoRows))
	})

	// Test case 2: Token not found
	t.Run("Token not found", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()
		accessor := dbaccess.New(dbPool)

		request := api.AdminDeleteApiTokenRequestObject{
			Id: "non-existent-token",
		}

		response, err := DeleteApiToken(ctx, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminDeleteApiToken204Response{}, response)
	})

	// Test case 3: Database error
	t.Run("Database error", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)
		actor := fixture.InsertActor(t, ctx, dbPool, "actor1")
		token := fixture.InsertToken(t, ctx, dbPool, "token001", actor.ID, time.Now().Add(24*time.Hour).Unix(), []string{"read"})

		request := api.AdminDeleteApiTokenRequestObject{
			Id: token.ID,
		}

		dbPool.Close()
		response, err := DeleteApiToken(ctx, accessor, request)

		assert.NoError(t, err)
		assert.IsType(t, api.AdminDeleteApiToken500JSONResponse{}, response)
		jsonResponse := response.(api.AdminDeleteApiToken500JSONResponse)
		assert.Contains(t, jsonResponse.Error, "closed pool")
	})
}
