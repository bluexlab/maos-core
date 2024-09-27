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

func ListActors(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminListActorsRequestObject) (api.AdminListActorsResponseObject, error) {
	logger.Info("ListActors", "request", request)

	page, _ := lo.Coalesce[*int](request.Params.Page, &defaultPage)
	pageSize, _ := lo.Coalesce[*int](request.Params.PageSize, &defaultPageSize)
	res, err := accessor.Querier().ActorListPagenated(ctx, accessor.Source(), &dbsqlc.ActorListPagenatedParams{
		Page:     int64(*page),
		PageSize: int64(*pageSize),
	})
	if err != nil {
		logger.Error("Cannot list actors", "error", err)
		return api.AdminListActors500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot list actors: %v", err)},
		}, nil
	}

	data := util.MapSlice(
		res,
		func(row *dbsqlc.ActorListPagenatedRow) api.Actor {
			return api.Actor{
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
	response := api.AdminListActors200JSONResponse{Data: data}
	if len(res) > 0 {
		response.Meta.TotalPages = int((res[0].TotalCount + int64(*pageSize) - 1) / int64(*pageSize))
	}
	return response, nil
}

func CreateActor(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminCreateActorRequestObject) (api.AdminCreateActorResponseObject, error) {
	logger.Info("CreateActor", "request", request.Body)

	if request.Body.Name == "" {
		return api.AdminCreateActor400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Missing required field: name"},
		}, nil
	}

	queue, err := accessor.Querier().QueueInsert(ctx, accessor.Source(), &dbsqlc.QueueInsertParams{
		Name:     request.Body.Name,
		Metadata: []byte(`{"type":"actor"}`),
	})
	if err != nil {
		logger.Error("Cannot create actors", "error", err)
		return api.AdminCreateActor500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot create actors: %v", err)},
		}, nil
	}

	actor, err := accessor.Querier().ActorInsert(ctx, accessor.Source(), &dbsqlc.ActorInsertParams{
		Name:         request.Body.Name,
		QueueID:      queue.ID,
		Enabled:      lo.FromPtrOr(request.Body.Enabled, true),
		Deployable:   lo.FromPtrOr(request.Body.Deployable, false),
		Configurable: lo.FromPtrOr(request.Body.Configurable, false),
	})
	if err != nil {

		return api.AdminCreateActor500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot create actors: %v", err)},
		}, nil
	}

	return api.AdminCreateActor201JSONResponse{
		Id:           actor.ID,
		Name:         actor.Name,
		Enabled:      actor.Enabled,
		Deployable:   actor.Deployable,
		Configurable: actor.Configurable,
		TokenCount:   0,
		CreatedAt:    actor.CreatedAt,
		Renameable:   true,
	}, nil
}

func GetActor(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminGetActorRequestObject) (api.AdminGetActorResponseObject, error) {
	logger.Info("GetActor", "actorId", request.Id)

	actor, err := accessor.Querier().ActorFindById(ctx, accessor.Source(), int64(request.Id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminGetActor404Response{}, nil
		}

		logger.Error("Cannot get actor", "error", err)
		return api.AdminGetActor500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot get actor: %v", err)},
		}, nil
	}

	if actor == nil {
		return api.AdminGetActor404Response{}, nil
	}

	return api.AdminGetActor200JSONResponse{
		Data: api.Actor{
			Id:           actor.ID,
			Name:         actor.Name,
			TokenCount:   actor.TokenCount,
			CreatedAt:    actor.CreatedAt,
			Renameable:   actor.Renameable,
			Enabled:      actor.Enabled,
			Deployable:   actor.Deployable,
			Configurable: actor.Configurable,
		},
	}, nil
}

func UpdateActor(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminUpdateActorRequestObject) (api.AdminUpdateActorResponseObject, error) {
	logger.Info("UpdateActor", "actorId", request.Id, "name", lo.FromPtrOr(request.Body.Name, "<nil>"))

	actor, err := accessor.Querier().ActorUpdate(ctx, accessor.Source(), &dbsqlc.ActorUpdateParams{
		ID:           int64(request.Id),
		Name:         request.Body.Name,
		Enabled:      request.Body.Enabled,
		Deployable:   request.Body.Deployable,
		Configurable: request.Body.Configurable,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminUpdateActor404Response{}, nil
		}

		logger.Error("Cannot update actor", "error", err)
		return api.AdminUpdateActor500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot update actor: %v", err)},
		}, nil
	}

	return api.AdminUpdateActor200JSONResponse{
		Data: api.Actor{
			Id:           actor.ID,
			Name:         actor.Name,
			Enabled:      actor.Enabled,
			Deployable:   actor.Deployable,
			Configurable: actor.Configurable,
			CreatedAt:    actor.CreatedAt,
		},
	}, nil
}

func DeleteActor(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminDeleteActorRequestObject) (api.AdminDeleteActorResponseObject, error) {
	logger.Info("DeleteActor", "actorId", request.Id)

	actor, err := accessor.Querier().ActorDelete(ctx, accessor.Source(), int64(request.Id))
	if err != nil {
		logger.Error("Cannot delete actor", "error", err)
		return api.AdminDeleteActor500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot delete actor: %v", err)},
		}, nil
	}

	if actor == "NOTFOUND" {
		return api.AdminDeleteActor404Response{}, nil
	}
	if actor == "REFERENCED" {
		return api.AdminDeleteActor409Response{}, nil
	}
	if actor == "DONE" {
		return api.AdminDeleteActor200Response{}, nil
	}
	return api.AdminDeleteActor500JSONResponse{
		N500JSONResponse: api.N500JSONResponse{Error: "Cannot delete actor"},
	}, nil
}
