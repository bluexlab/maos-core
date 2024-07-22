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

var (
	defaultPage     = 1
	defaultPageSize = 10
	tokenLength     = 32
)

func ListApiTokens(ctx context.Context, accessor dbaccess.Accessor, request api.AdminListApiTokensRequestObject) (api.AdminListApiTokensResponseObject, error) {
	page, _ := lo.Coalesce[*int](request.Params.Page, &defaultPage)
	pageSize, _ := lo.Coalesce[*int](request.Params.PageSize, &defaultPageSize)
	res, err := accessor.Querier().ApiTokenListByPage(ctx, accessor.Source(), &dbsqlc.ApiTokenListByPageParams{
		Page:     int64(*page),
		PageSize: int64(*pageSize),
	})
	if err != nil {
		return api.AdminListApiTokens500Response{}, err
	}

	data := util.MapSlice(
		res,
		func(row *dbsqlc.ApiTokenListByPageRow) api.ApiToken {
			return api.ApiToken{
				Id:          row.ID,
				AgentId:     row.AgentID,
				CreatedAt:   row.ExpireAt,
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
	params := dbsqlc.ApiTokenInsertParams{
		ID:          GenerateAPIToken(),
		AgentID:     request.Body.AgentId,
		CreatedBy:   request.Body.CreatedBy,
		Permissions: request.Body.Permissions,
		ExpireAt:    request.Body.ExpireAt,
	}

	apiToken, err := accessor.Querier().ApiTokenInsert(ctx, accessor.Source(), &params)
	if err != nil {
		return api.AdminCreateApiToken500Response{}, err
	}

	return api.AdminCreateApiToken201JSONResponse{
		Id:          apiToken.ID,
		AgentId:     apiToken.AgentID,
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
