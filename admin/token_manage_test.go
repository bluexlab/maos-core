package admin

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/samber/lo"

	"github.com/stretchr/testify/assert"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
	"gitlab.com/navyx/ai/maos/maos-core/util"
)

func TestListApiTokensWithDB(t *testing.T) {
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

		lo.RepeatBy(21, func(i int) *dbsqlc.ApiTokens {
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

		assert.Error(t, err)
		assert.Equal(t, "closed pool", err.Error())
		assert.IsType(t, api.AdminListApiTokens500Response{}, response)
	})
}
