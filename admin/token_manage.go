package admin

import (
	"context"

	"github.com/samber/lo"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/util"
)

var (
	defaultPage     = 1
	defaultPageSize = 10
)

func ListApiTokens(ctx context.Context, accessor dbaccess.Accessor, request api.AdminListApiTokensRequestObject) (api.AdminListApiTokensResponseObject, error) {
	page, _ := lo.Coalesce[*int](request.Params.Page, &defaultPage)
	pageSize, _ := lo.Coalesce[*int](request.Params.PageSize, &defaultPageSize)
	res, err := accessor.ApiTokenListByPage(ctx, dbaccess.ApiTokenListByPageParams{
		Page:     int64(*page),
		PageSize: int64(*pageSize),
	})
	if err != nil {
		return api.AdminListApiTokens500Response{}, err
	}

	data := util.MapSlice(
		res,
		func(row *dbaccess.ApiTokenListRow) api.ApiToken {
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
	panic("not implemented")
}
