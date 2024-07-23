package admin

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

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
		accessor := dbaccess.New(dbPool)
		agent1 := fixture.InsertAgent(t, ctx, dbPool, "agent1")
		agent2 := fixture.InsertAgent(t, ctx, dbPool, "agent2")
		fixture.InsertToken(t, ctx, dbPool, "token001", agent1.ID, expireAt, []string{"invocation:read", "invocation:create"})
		fixture.InsertToken(t, ctx, dbPool, "token002", agent2.ID, expireAt, []string{"admin"})

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
				AgentId:     agent1.ID,
				ExpireAt:    expireAt,
				CreatedBy:   "test",
				Permissions: []api.Permission{api.InvocationRead, api.InvocationCreate},
			},
			{
				Id:          "token002",
				AgentId:     agent2.ID,
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
			_, token := fixture.InsertAgentToken(t, ctx, dbPool, fmt.Sprintf("token-%03d", i), expireAt, []string{"read"}, createdAt+int64(i))
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
		accessor := dbaccess.New(dbPool)
		agent1 := fixture.InsertAgent(t, ctx, dbPool, "agent1")

		request := api.AdminCreateApiTokenRequestObject{
			Body: &api.AdminCreateApiTokenJSONRequestBody{
				AgentId:     agent1.ID,
				CreatedBy:   "admin",
				Permissions: []string{"read", "write"},
				ExpireAt:    expireAt,
			},
		}

		expectedApiToken := dbsqlc.ApiToken{
			AgentID:     request.Body.AgentId,
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
		assert.Equal(t, expectedApiToken.AgentID, jsonResponse.AgentId)
		assert.Equal(t, expectedApiToken.CreatedBy, jsonResponse.CreatedBy)
		assert.Equal(t,
			util.MapSlice(expectedApiToken.Permissions, func(p string) api.Permission { return api.Permission(p) }),
			jsonResponse.Permissions,
		)
		assert.Equal(t, expectedApiToken.ExpireAt, jsonResponse.ExpireAt)

		apiToken, err := accessor.Querier().ApiTokenFindByID(ctx, accessor.Source(), jsonResponse.Id)
		assert.NoError(t, err)
		assert.Equal(t, expectedApiToken.AgentID, apiToken.AgentID)
		assert.Equal(t, expectedApiToken.CreatedBy, apiToken.CreatedBy)
		assert.Equal(t, expectedApiToken.Permissions, apiToken.Permissions)
		assert.Equal(t, expectedApiToken.ExpireAt, apiToken.ExpireAt)
	})

	// Test case 2: Database error
	t.Run("Database error", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)
		accessor := dbaccess.New(dbPool)
		agent1 := fixture.InsertAgent(t, ctx, dbPool, "agent1")
		request := api.AdminCreateApiTokenRequestObject{
			Body: &api.AdminCreateApiTokenJSONRequestBody{
				AgentId:     agent1.ID,
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
