package admin

import (
	"context"

	"github.com/samber/lo"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/util"
)

func ListAgents(ctx context.Context, accessor dbaccess.Accessor, request api.AdminListAgentsRequestObject) (api.AdminListAgentsResponseObject, error) {
	page, _ := lo.Coalesce[*int](request.Params.Page, &defaultPage)
	pageSize, _ := lo.Coalesce[*int](request.Params.PageSize, &defaultPageSize)
	res, err := accessor.Querier().AgentListPagenated(ctx, accessor.Source(), &dbsqlc.AgentListPagenatedParams{
		Page:     int64(*page),
		PageSize: int64(*pageSize),
	})
	if err != nil {
		return api.AdminListAgents500Response{}, err
	}

	data := util.MapSlice(
		res,
		func(row *dbsqlc.AgentListPagenatedRow) api.Agent {
			return api.Agent{
				Id:        row.ID,
				Name:      row.Name,
				CreatedAt: row.CreatedAt,
			}
		},
	)
	response := api.AdminListAgents200JSONResponse{Data: data}
	if len(res) > 0 {
		response.Meta.TotalPages = int((res[0].TotalCount + int64(*pageSize) - 1) / int64(*pageSize))
	}
	return response, nil
}

func CreateaAgent(ctx context.Context, accessor dbaccess.Accessor, request api.AdminCreateAgentRequestObject) (api.AdminCreateAgentResponseObject, error) {
	queue, err := accessor.Querier().QueueInsert(ctx, accessor.Source(), &dbsqlc.QueueInsertParams{
		Name:     request.Body.Name,
		Metadata: []byte(`{"type":"agent"}`),
	})
	if err != nil {
		return api.AdminCreateAgent500Response{}, err
	}

	agent, err := accessor.Querier().AgentInsert(ctx, accessor.Source(), &dbsqlc.AgentInsertParams{
		Name:    request.Body.Name,
		QueueID: queue.ID,
	})
	if err != nil {
		return api.AdminCreateAgent500Response{}, err
	}

	return api.AdminCreateAgent201JSONResponse{
		Id:        agent.ID,
		Name:      agent.Name,
		CreatedAt: agent.CreatedAt,
	}, nil
}
