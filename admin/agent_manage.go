package admin

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/samber/lo"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/util"
)

func ListAgents(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminListAgentsRequestObject) (api.AdminListAgentsResponseObject, error) {
	logger.Info("ListAgents", "request", request)

	page, _ := lo.Coalesce[*int](request.Params.Page, &defaultPage)
	pageSize, _ := lo.Coalesce[*int](request.Params.PageSize, &defaultPageSize)
	res, err := accessor.Querier().AgentListPagenated(ctx, accessor.Source(), &dbsqlc.AgentListPagenatedParams{
		Page:     int64(*page),
		PageSize: int64(*pageSize),
	})
	if err != nil {
		logger.Error("Cannot list agents", "error", err)
		return api.AdminListAgents500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot list agents: %v", err)},
		}, nil
	}

	data := util.MapSlice(
		res,
		func(row *dbsqlc.AgentListPagenatedRow) api.Agent {
			return api.Agent{
				Id:           row.ID,
				Name:         row.Name,
				CreatedAt:    row.CreatedAt,
				Renameable:   row.Renameable,
				TokenCount:   row.TokenCount,
				Enabled:      row.Enabled,
				Deployable:   row.Deployable,
				Configurable: row.Configurable,
			}
		},
	)
	response := api.AdminListAgents200JSONResponse{Data: data}
	if len(res) > 0 {
		response.Meta.TotalPages = int((res[0].TotalCount + int64(*pageSize) - 1) / int64(*pageSize))
	}
	return response, nil
}

func CreateAgent(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminCreateAgentRequestObject) (api.AdminCreateAgentResponseObject, error) {
	logger.Info("CreateAgent", "request", request.Body)

	if request.Body.Name == "" {
		return api.AdminCreateAgent400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Missing required field: name"},
		}, nil
	}

	queue, err := accessor.Querier().QueueInsert(ctx, accessor.Source(), &dbsqlc.QueueInsertParams{
		Name:     request.Body.Name,
		Metadata: []byte(`{"type":"agent"}`),
	})
	if err != nil {
		logger.Error("Cannot create agents", "error", err)
		return api.AdminCreateAgent500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot create agents: %v", err)},
		}, nil
	}

	agent, err := accessor.Querier().AgentInsert(ctx, accessor.Source(), &dbsqlc.AgentInsertParams{
		Name:         request.Body.Name,
		QueueID:      queue.ID,
		Enabled:      lo.FromPtrOr(request.Body.Enabled, true),
		Deployable:   lo.FromPtrOr(request.Body.Deployable, false),
		Configurable: lo.FromPtrOr(request.Body.Configurable, false),
	})
	if err != nil {

		return api.AdminCreateAgent500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot create agents: %v", err)},
		}, nil
	}

	return api.AdminCreateAgent201JSONResponse{
		Id:           agent.ID,
		Name:         agent.Name,
		Enabled:      agent.Enabled,
		Deployable:   agent.Deployable,
		Configurable: agent.Configurable,
		TokenCount:   0,
		CreatedAt:    agent.CreatedAt,
		Renameable:   true,
	}, nil
}

func GetAgent(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminGetAgentRequestObject) (api.AdminGetAgentResponseObject, error) {
	logger.Info("GetAgent", "agentId", request.Id)

	agent, err := accessor.Querier().AgentFindById(ctx, accessor.Source(), int64(request.Id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminGetAgent404Response{}, nil
		}

		logger.Error("Cannot get agent", "error", err)
		return api.AdminGetAgent500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot get agent: %v", err)},
		}, nil
	}

	if agent == nil {
		return api.AdminGetAgent404Response{}, nil
	}

	return api.AdminGetAgent200JSONResponse{
		Data: api.Agent{
			Id:           agent.ID,
			Name:         agent.Name,
			TokenCount:   agent.TokenCount,
			CreatedAt:    agent.CreatedAt,
			Renameable:   agent.Renameable,
			Enabled:      agent.Enabled,
			Deployable:   agent.Deployable,
			Configurable: agent.Configurable,
		},
	}, nil
}

func UpdateAgent(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminUpdateAgentRequestObject) (api.AdminUpdateAgentResponseObject, error) {
	logger.Info("UpdateAgent", "agentId", request.Id, "name", lo.FromPtrOr(request.Body.Name, "<nil>"))

	agent, err := accessor.Querier().AgentUpdate(ctx, accessor.Source(), &dbsqlc.AgentUpdateParams{
		ID:           int64(request.Id),
		Name:         request.Body.Name,
		Enabled:      request.Body.Enabled,
		Deployable:   request.Body.Deployable,
		Configurable: request.Body.Configurable,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminUpdateAgent404Response{}, nil
		}

		logger.Error("Cannot update agent", "error", err)
		return api.AdminUpdateAgent500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot update agent: %v", err)},
		}, nil
	}

	return api.AdminUpdateAgent200JSONResponse{
		Data: api.Agent{
			Id:           agent.ID,
			Name:         agent.Name,
			Enabled:      agent.Enabled,
			Deployable:   agent.Deployable,
			Configurable: agent.Configurable,
			CreatedAt:    agent.CreatedAt,
		},
	}, nil
}

func DeleteAgent(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminDeleteAgentRequestObject) (api.AdminDeleteAgentResponseObject, error) {
	logger.Info("DeleteAgent", "agentId", request.Id)

	agent, err := accessor.Querier().AgentDelete(ctx, accessor.Source(), int64(request.Id))
	if err != nil {
		logger.Error("Cannot delete agent", "error", err)
		return api.AdminDeleteAgent500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot delete agent: %v", err)},
		}, nil
	}

	if agent == "NOTFOUND" {
		return api.AdminDeleteAgent404Response{}, nil
	}
	if agent == "REFERENCED" {
		return api.AdminDeleteAgent409Response{}, nil
	}
	if agent == "DONE" {
		return api.AdminDeleteAgent200Response{}, nil
	}
	return api.AdminDeleteAgent500JSONResponse{
		N500JSONResponse: api.N500JSONResponse{Error: "Cannot delete agent"},
	}, nil
}
