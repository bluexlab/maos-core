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

func ListDeployments(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminListDeploymentsRequestObject) (api.AdminListDeploymentsResponseObject, error) {
	logger.Info("AdminListDeployments",
		"name", request.Params.Name,
		"page", lo.FromPtrOr(request.Params.Page, -999),
		"page_size", lo.FromPtrOr(request.Params.PageSize, -999),
	)

	page, _ := lo.Coalesce[*int](request.Params.Page, &defaultPage)
	pageSizePtr, _ := lo.Coalesce[*int](request.Params.PageSize, &defaultPageSize)
	pageSize := lo.Clamp(*pageSizePtr, 1, 100)

	res, err := accessor.Querier().DeploymentListPaginated(ctx, accessor.Source(), &dbsqlc.DeploymentListPaginatedParams{Page: int64(*page), PageSize: int64(pageSize)})
	if err != nil {
		logger.Error("Cannot list deployments", "error", err)
		return api.AdminListDeployments500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot list deployments: %v", err)},
		}, nil
	}

	data := util.MapSlice(
		res,
		func(row *dbsqlc.DeploymentListPaginatedRow) api.Deployment {
			return api.Deployment{
				Id:         row.ID,
				Name:       row.Name,
				Status:     api.DeploymentStatus(row.Status),
				CreatedBy:  row.CreatedBy,
				CreatedAt:  row.CreatedAt,
				ApprovedBy: row.ApprovedBy,
				ApprovedAt: row.ApprovedAt,
				FinishedBy: row.FinishedBy,
				FinishedAt: row.FinishedAt,
			}
		},
	)
	response := api.AdminListDeployments200JSONResponse{Data: data}
	if len(res) > 0 {
		response.Meta.Total = res[0].TotalCount
		response.Meta.Page = *page
		response.Meta.PageSize = pageSize
	}
	return response, nil
}

func CreateDeployment(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminCreateDeploymentRequestObject) (api.AdminCreateDeploymentResponseObject, error) {
	logger.Info("AdminCreateDeployment", "request", request.Body)

	if request.Body.Name == "" || request.Body.User == "" {
		return api.AdminCreateDeployment400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Missing required field"},
		}, nil
	}

	deployment, err := accessor.Querier().DeploymentInsert(ctx, accessor.Source(), &dbsqlc.DeploymentInsertParams{
		Name:      request.Body.Name,
		CreatedBy: request.Body.User,
	})
	if err != nil {
		logger.Error("Cannot create deployment", "error", err)
		return api.AdminCreateDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot create deployment: %v", err)},
		}, nil
	}

	return api.AdminCreateDeployment201JSONResponse{
		Data: api.Deployment{
			Id:         deployment.ID,
			Name:       deployment.Name,
			Status:     api.DeploymentStatus(deployment.Status),
			CreatedBy:  deployment.CreatedBy,
			CreatedAt:  deployment.CreatedAt,
			ApprovedBy: deployment.ApprovedBy,
			ApprovedAt: deployment.ApprovedAt,
			FinishedBy: deployment.FinishedBy,
			FinishedAt: deployment.FinishedAt,
		},
	}, nil
}

func UpdateDeployment(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminUpdateDeploymentRequestObject) (api.AdminUpdateDeploymentResponseObject, error) {
	logger.Info("AdminUpdateDeployment", "id", request.Id, "name", lo.FromPtrOr(request.Body.Name, "<nil>"), "reviewers", lo.FromPtrOr(request.Body.Reviewers, nil))

	deployment, err := accessor.Querier().DeploymentUpdate(ctx, accessor.Source(), &dbsqlc.DeploymentUpdateParams{
		ID:        int64(request.Id),
		Name:      request.Body.Name,
		Reviewers: lo.FromPtrOr(request.Body.Reviewers, nil),
	})

	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminUpdateDeployment404Response{}, nil
		}

		logger.Error("Cannot update deployment", "error", err)
		return api.AdminUpdateDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot update deployment: %v", err)},
		}, nil
	}

	return api.AdminUpdateDeployment200JSONResponse{
		Data: api.Deployment{
			Id:         deployment.ID,
			Name:       deployment.Name,
			Status:     api.DeploymentStatus(deployment.Status),
			CreatedBy:  deployment.CreatedBy,
			CreatedAt:  deployment.CreatedAt,
			ApprovedBy: deployment.ApprovedBy,
			ApprovedAt: deployment.ApprovedAt,
			FinishedBy: deployment.FinishedBy,
			FinishedAt: deployment.FinishedAt,
		},
	}, nil
}

func SubmitDeployment(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminSubmitDeploymentRequestObject) (api.AdminSubmitDeploymentResponseObject, error) {
	logger.Info("AdminSubmitDeployment", "id", request.Id)

	deployment, err := accessor.Querier().DeploymentSubmitForReview(ctx, accessor.Source(), int64(request.Id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminSubmitDeployment404Response{}, nil
		}

		logger.Error("Cannot submit deployment", "error", err)
		return api.AdminSubmitDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot submit deployment: %v", err)},
		}, nil
	}

	// TODO: Notify reviewers (implement this functionality)

	logger.Info("Deployment submitted successfully", "id", deployment.ID, "status", deployment.Status)
	return api.AdminSubmitDeployment200Response{}, nil
}

func DeleteDeployment(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminDeleteDeploymentRequestObject) (api.AdminDeleteDeploymentResponseObject, error) {
	logger.Info("AdminDeleteDeployment", "id", request.Id)

	_, err := accessor.Querier().DeploymentDelete(ctx, accessor.Source(), int64(request.Id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminDeleteDeployment404Response{}, nil
		}

		logger.Error("Cannot delete deployment", "error", err)
		return api.AdminDeleteDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot delete deployment: %v", err)},
		}, nil
	}

	return api.AdminDeleteDeployment200Response{}, nil
}
