package admin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/samber/lo"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/util"
)

func ListApiTokens(ctx context.Context, accessor dbaccess.Accessor, request api.AdminListApiTokensRequestObject) (api.AdminListApiTokensResponseObject, error) {
	page, _ := lo.Coalesce[*int](request.Params.Page, &defaultPage)
	pageSize, _ := lo.Coalesce[*int](request.Params.PageSize, &defaultPageSize)
	res, err := accessor.Querier().ApiTokenListByPage(ctx, accessor.Source(), &dbsqlc.ApiTokenListByPageParams{
		Page:     max(int64(*page), 1),
		PageSize: min(max(int64(*pageSize), 1), 100),
	})
	if err != nil {
		return api.AdminListApiTokens500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot list API tokens: %s", err.Error())},
		}, nil
	}

	data := util.MapSlice(
		res,
		func(row *dbsqlc.ApiTokenListByPageRow) api.ApiToken {
			return api.ApiToken{
				Id:          row.ID,
				AgentId:     row.AgentId,
				CreatedAt:   row.CreatedAt,
				CreatedBy:   row.CreatedBy,
				ExpireAt:    row.ExpireAt,
				Permissions: util.MapSlice(row.Permissions, func(p string) api.Permission { return api.Permission(p) }),
			}
		},
	)
	response := api.AdminListApiTokens200JSONResponse{Data: data}
	response.Meta.TotalPages = int((res[0].TotalCount + int64(*pageSize) - 1) / int64(*pageSize))
	return response, nil
}

func CreateApiToken(ctx context.Context, accessor dbaccess.Accessor, request api.AdminCreateApiTokenRequestObject) (api.AdminCreateApiTokenResponseObject, error) {
	if request.Body.AgentId == 0 ||
		request.Body.CreatedBy == "" ||
		request.Body.ExpireAt == 0 {
		return api.AdminCreateApiToken400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Missing required fields"},
		}, nil
	}

	params := dbsqlc.ApiTokenInsertParams{
		ID:          GenerateAPIToken(),
		AgentId:     request.Body.AgentId,
		CreatedBy:   request.Body.CreatedBy,
		Permissions: request.Body.Permissions,
		ExpireAt:    request.Body.ExpireAt,
	}

	apiToken, err := accessor.Querier().ApiTokenInsert(ctx, accessor.Source(), &params)
	if err != nil {
		return api.AdminCreateApiToken500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot insert API tokens: %s", err.Error())},
		}, nil
	}

	return api.AdminCreateApiToken201JSONResponse{
		Id:          apiToken.ID,
		AgentId:     apiToken.AgentId,
		CreatedAt:   apiToken.ExpireAt,
		CreatedBy:   apiToken.CreatedBy,
		ExpireAt:    apiToken.ExpireAt,
		Permissions: util.MapSlice(apiToken.Permissions, func(p string) api.Permission { return api.Permission(p) }),
	}, nil
}

func GenerateAPIToken() string {
	// Calculate the number of random bytes needed
	// We'll generate slightly more than needed to account for base64 encoding
	numBytes := (tokenLength*3)/4 + 1

	// Generate random bytes
	randomBytes := make([]byte, numBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(fmt.Errorf("failed to generate random bytes: %v", err))
	}

	// Encode to base64
	encoded := base64.RawURLEncoding.EncodeToString(randomBytes)

	// Trim to desired length and add prefix
	token := "ma-" + encoded[:tokenLength]

	return token
}
