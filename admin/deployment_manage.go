package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/samber/lo"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/suitestore"
	"gitlab.com/navyx/ai/maos/maos-core/k8s"
	"gitlab.com/navyx/ai/maos/maos-core/util"
)

func ListDeployments(ctx context.Context, logger *slog.Logger, ds dbaccess.DataSource, request api.AdminListDeploymentsRequestObject) (api.AdminListDeploymentsResponseObject, error) {
	logger.Info("AdminListDeployments",
		"status", lo.FromPtrOr(request.Params.Status, "<nil>"),
		"reviewer", lo.FromPtrOr(request.Params.Reviewer, "<nil>"),
		"page", lo.FromPtrOr(request.Params.Page, -999),
		"page_size", lo.FromPtrOr(request.Params.PageSize, -999),
	)

	page, _ := lo.Coalesce[*int](request.Params.Page, &defaultPage)
	pageSizePtr, _ := lo.Coalesce[*int](request.Params.PageSize, &defaultPageSize)
	pageSize := lo.Clamp(*pageSizePtr, 1, 100)
	status := dbsqlc.NullDeploymentStatus{}
	if request.Params.Status != nil {
		status.Scan(string(*request.Params.Status))
	}

	res, err := querier.DeploymentListPaginated(ctx, ds, &dbsqlc.DeploymentListPaginatedParams{
		Page:     int64(*page),
		PageSize: int64(pageSize),
		Status:   status,
		Reviewer: request.Params.Reviewer,
		Name:     request.Params.Name,
		ID:       lo.FromPtrOr(request.Params.Id, nil),
	})
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
				Notes:      deserializeNotes(row.Notes),
				Reviewers:  row.Reviewers,
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

func GetDeployment(ctx context.Context, logger *slog.Logger, ds dbaccess.DataSource, request api.AdminGetDeploymentRequestObject) (api.AdminGetDeploymentResponseObject, error) {
	logger.Info("AdminGetDeployment", "id", request.Id)

	deployment, err := querier.DeploymentGetById(ctx, ds, int64(request.Id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminGetDeployment404Response{}, nil
		}

		logger.Error("Cannot get deployment", "error", err)
		return api.AdminGetDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot get deployment: %v", err)},
		}, nil
	}

	if deployment.ConfigSuiteID == nil {
		return api.AdminGetDeployment200JSONResponse{
			Id:         deployment.ID,
			Name:       deployment.Name,
			Status:     api.DeploymentDetailStatus(deployment.Status),
			Reviewers:  deployment.Reviewers,
			Notes:      deserializeNotes(deployment.Notes),
			CreatedBy:  deployment.CreatedBy,
			CreatedAt:  deployment.CreatedAt,
			ApprovedBy: deployment.ApprovedBy,
			ApprovedAt: deployment.ApprovedAt,
			FinishedBy: deployment.FinishedBy,
			FinishedAt: deployment.FinishedAt,
		}, nil
	}

	configs, err := querier.ConfigListBySuiteIdGroupByActor(ctx, ds, *deployment.ConfigSuiteID)
	if err != nil {
		logger.Error("Cannot get configs", "error", err)
		return api.AdminGetDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot get configs: %v", err)},
		}, nil
	}

	filteredConfigs := lo.Filter(configs, func(row *dbsqlc.ConfigListBySuiteIdGroupByActorRow, _ int) bool {
		return row.ActorConfigurable
	})

	resultConfigs := lo.Map(filteredConfigs, func(row *dbsqlc.ConfigListBySuiteIdGroupByActorRow, _ int) api.Config {
		var content map[string]string
		err := json.Unmarshal(row.Content, &content)
		if err != nil {
			logger.Error("Cannot unmarshal content", "error", err)
		}

		// insert kubernetes config
		if row.ActorDeployable {
			InsertMissingKubeConfigsWithDefault(content, string(row.ActorRole), row.ActorMigratable)
		}

		return api.Config{
			Id:              row.ID,
			ActorId:         row.ActorId,
			ActorName:       row.ActorName,
			MinActorVersion: util.SerializeActorVersion(row.MinActorVersion),
			CreatedAt:       row.CreatedAt,
			CreatedBy:       row.CreatedBy,
			Content:         content,
		}
	})

	return api.AdminGetDeployment200JSONResponse{
		Id:         deployment.ID,
		Name:       deployment.Name,
		Status:     api.DeploymentDetailStatus(deployment.Status),
		Notes:      deserializeNotes(deployment.Notes),
		Reviewers:  deployment.Reviewers,
		CreatedBy:  deployment.CreatedBy,
		CreatedAt:  deployment.CreatedAt,
		ApprovedBy: deployment.ApprovedBy,
		ApprovedAt: deployment.ApprovedAt,
		FinishedBy: deployment.FinishedBy,
		FinishedAt: deployment.FinishedAt,
		Configs:    &resultConfigs,
	}, nil
}

func GetDeploymentResult(ctx context.Context, logger *slog.Logger, ds dbaccess.DataSource, request api.AdminGetDeploymentResultRequestObject) (api.AdminGetDeploymentResultResponseObject, error) {
	logger.Info("AdminGetDeploymentResult", "id", request.Id)

	deployment, err := querier.DeploymentGetById(ctx, ds, int64(request.Id))
	if err != nil {
		return api.AdminGetDeploymentResult404Response{}, nil
	}

	var logs map[string]map[string]interface{}
	if deployment.MigrationLogs != nil {
		err = json.Unmarshal(deployment.MigrationLogs, &logs)
		if err != nil {
			logger.Error("Cannot unmarshal logs", "error", err)
		}
	}

	return api.AdminGetDeploymentResult200JSONResponse{
		Status: string(deployment.Status),
		Error:  deployment.LastError,
		Logs:   &logs,
	}, nil
}

func CreateDeployment(ctx context.Context, logger *slog.Logger, ds dbaccess.DataSource, request api.AdminCreateDeploymentRequestObject) (api.AdminCreateDeploymentResponseObject, error) {
	logger.Info("AdminCreateDeployment", "request", request.Body)

	if request.Body.Name == "" || request.Body.User == "" {
		return api.AdminCreateDeployment400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Missing required field"},
		}, nil
	}

	deployment, err := querier.DeploymentInsertWithConfigSuite(ctx, ds, &dbsqlc.DeploymentInsertWithConfigSuiteParams{
		Name:      request.Body.Name,
		Reviewers: lo.FromPtrOr(request.Body.Reviewers, nil),
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
			Id:            deployment.ID,
			Name:          deployment.Name,
			Status:        api.DeploymentStatus(deployment.Status),
			Reviewers:     deployment.Reviewers,
			ConfigSuiteId: deployment.ConfigSuiteID,
			CreatedBy:     deployment.CreatedBy,
			CreatedAt:     deployment.CreatedAt,
			ApprovedBy:    deployment.ApprovedBy,
			ApprovedAt:    deployment.ApprovedAt,
			FinishedBy:    deployment.FinishedBy,
			FinishedAt:    deployment.FinishedAt,
		},
	}, nil
}

func UpdateDeployment(ctx context.Context, logger *slog.Logger, ds dbaccess.DataSource, request api.AdminUpdateDeploymentRequestObject) (api.AdminUpdateDeploymentResponseObject, error) {
	logger.Info("AdminUpdateDeployment", "id", request.Id, "name", lo.FromPtrOr(request.Body.Name, "<nil>"), "reviewers", lo.FromPtrOr(request.Body.Reviewers, nil))

	deployment, err := querier.DeploymentUpdate(ctx, ds, &dbsqlc.DeploymentUpdateParams{
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
			Id:            deployment.ID,
			Name:          deployment.Name,
			Status:        api.DeploymentStatus(deployment.Status),
			Reviewers:     deployment.Reviewers,
			Notes:         deserializeNotes(deployment.Notes),
			ConfigSuiteId: deployment.ConfigSuiteID,
			CreatedBy:     deployment.CreatedBy,
			CreatedAt:     deployment.CreatedAt,
			ApprovedBy:    deployment.ApprovedBy,
			ApprovedAt:    deployment.ApprovedAt,
			FinishedBy:    deployment.FinishedBy,
			FinishedAt:    deployment.FinishedAt,
		},
	}, nil
}

func SubmitDeployment(ctx context.Context, logger *slog.Logger, ds dbaccess.DataSource, request api.AdminSubmitDeploymentRequestObject) (api.AdminSubmitDeploymentResponseObject, error) {
	logger.Info("AdminSubmitDeployment", "id", request.Id)

	tx, err := ds.Begin(ctx)
	if err != nil {
		logger.Error("Cannot start transaction", "error", err)
		return api.AdminSubmitDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot start transaction: %v", err)},
		}, nil
	}
	defer tx.Rollback(ctx)

	deployment, err := querier.DeploymentGetById(ctx, tx, int64(request.Id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminSubmitDeployment404Response{}, nil
		}

		logger.Error("Cannot get deployment", "error", err)
		return api.AdminSubmitDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot get deployment: %v", err)},
		}, nil
	}

	if deployment.Status != "draft" {
		logger.Info("Cannot submit deployment", "error", "Deployment is not in draft status", "id", deployment.ID)
		return api.AdminSubmitDeployment400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Deployment must be in draft status to be submitted"},
		}, nil
	}
	if deployment.ConfigSuiteID == nil {
		logger.Info("Cannot submit deployment", "error", "Deployment has no config suite", "id", deployment.ID)
		return api.AdminSubmitDeployment400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Deployment must have a config suite to be submitted"},
		}, nil
	}
	if len(deployment.Reviewers) == 0 {
		logger.Info("Cannot submit deployment", "error", "Deployment has no reviewers", "id", deployment.ID)
		return api.AdminSubmitDeployment400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Deployment must have at least one reviewer"},
		}, nil
	}

	_, err = querier.DeploymentSubmitForReview(ctx, tx, int64(request.Id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminSubmitDeployment404Response{}, nil
		}

		logger.Error("Cannot submit deployment", "error", err)
		return api.AdminSubmitDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot submit deployment: %v", err)},
		}, nil
	}

	tx.Commit(ctx)

	// TODO: Notify reviewers (implement this functionality)

	logger.Info("Deployment submitted successfully", "id", deployment.ID, "status", deployment.Status)
	return api.AdminSubmitDeployment200Response{}, nil
}

func RejectDeployment(ctx context.Context, logger *slog.Logger, ds dbaccess.DataSource, request api.AdminRejectDeploymentRequestObject) (api.AdminRejectDeploymentResponseObject, error) {
	logger.Info("AdminRejectDeployment", "id", request.Id, "user", request.Body.User)

	if request.Body == nil || request.Body.User == "" {
		return api.AdminRejectDeployment400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Missing required field"},
		}, nil
	}

	_, err := querier.DeploymentReject(ctx, ds, &dbsqlc.DeploymentRejectParams{
		ID:         int64(request.Id),
		RejectedBy: request.Body.User,
		Notes:      json.RawMessage(fmt.Sprintf(`{"reason": "%s"}`, request.Body.Reason)),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminRejectDeployment404Response{}, nil
		}

		logger.Error("Cannot reject deployment", "error", err)
		return api.AdminRejectDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot reject deployment: %v", err)},
		}, nil
	}

	return api.AdminRejectDeployment201Response{}, nil
}

func PublishDeployment(
	ctx context.Context,
	logger *slog.Logger,
	ds dbaccess.DataSource,
	suiteStore suitestore.SuiteStore,
	controller k8s.Controller,
	request api.AdminPublishDeploymentRequestObject) (api.AdminPublishDeploymentResponseObject, error) {
	logger.Info("AdminPublishDeployment", "id", request.Id, "user", request.Body.User)

	if request.Body == nil || request.Body.User == "" {
		return api.AdminPublishDeployment400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Missing required field"},
		}, nil
	}

	errMessage, err := querier.SetDeploymentDeploying(ctx, ds, &dbsqlc.SetDeploymentDeployingParams{
		ID:         int64(request.Id),
		ApprovedBy: request.Body.User,
	})
	if err != nil {
		logger.Error("Cannot set deployment to deploying", "error", err)
		return api.AdminPublishDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot publish deployment. Cannot set deployment to deploying: %v", err)},
		}, nil
	}
	if errMessage != "" {
		logger.Error("Cannot set deployment to deploying", "error", errMessage)
		return api.AdminPublishDeployment400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: fmt.Sprintf("Cannot publish deployment. Cannot set deployment to deploying: %s", errMessage)},
		}, nil
	}

	// run deployment migrations and update deployment in background
	go runDeploymentMigrationsAndUpdateDeployment(logger, controller, suiteStore, ds, int64(request.Id), request.Body.User)

	return api.AdminPublishDeployment201Response{}, nil
}

func DeleteDeployment(ctx context.Context, logger *slog.Logger, ds dbaccess.DataSource, request api.AdminDeleteDeploymentRequestObject) (api.AdminDeleteDeploymentResponseObject, error) {
	logger.Info("AdminDeleteDeployment", "id", request.Id)

	_, err := querier.DeploymentDelete(ctx, ds, int64(request.Id))
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

func RestartDeployment(
	ctx context.Context,
	logger *slog.Logger,
	ds dbaccess.DataSource,
	controller k8s.Controller,
	request api.AdminRestartDeploymentRequestObject) (api.AdminRestartDeploymentResponseObject, error) {
	logger.Info("AdminRestartDeployment", "id", request.Id, "user", request.Body.User)

	if request.Body == nil || request.Body.User == "" {
		return api.AdminRestartDeployment401Response{}, nil
	}

	tx, err := ds.Begin(ctx)
	if err != nil {
		logger.Error("Cannot start transaction", "error", err)
		return api.AdminRestartDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot restart deployment. Cannot start transaction: %v", err)},
		}, nil
	}
	defer tx.Rollback(ctx)

	// Query deployment and check status
	deployment, err := querier.DeploymentGetById(ctx, tx, int64(request.Id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminRestartDeployment404Response{}, nil
		}
		logger.Error("Cannot get deployment", "error", err)
		return api.AdminRestartDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot restart deployment. Cannot get deployment: %v", err)},
		}, nil
	}

	if deployment.Status != "deployed" {
		return api.AdminRestartDeployment404Response{}, nil
	}

	if deployment.ConfigSuiteID == nil {
		logger.Error("Cannot restart deployment", "error", "Deployment has no config suite", "id", deployment.ID)
		return api.AdminRestartDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: "Cannot restart deployment. Deployment has no config suite"},
		}, nil
	}

	// update kubernetes actor deployments
	configs, err := querier.ConfigListBySuiteIdGroupByActor(ctx, tx, *deployment.ConfigSuiteID)
	if err != nil {
		logger.Error("Cannot get configs", "error", err)
		return api.AdminRestartDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot get configs: %v", err)},
		}, nil
	}

	// rotate actor api keys
	// it generates new api keys for each actor
	// and set the old ones to expire after 15 minutes
	apiTokens, err := rotateActorApiKeys(ctx, tx, configs, request.Body.User)
	if err != nil {
		logger.Error("Cannot rotate actor api keys", "error", err)
		return api.AdminRestartDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot rotate actor api keys: %v", err)},
		}, nil
	}

	err = updateKubernetesDeployments(ctx, controller, deployment, configs, apiTokens)
	if err != nil {
		logger.Error("Cannot update kubernetes deployments", "error", err)
		return api.AdminRestartDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot update kubernetes deployments: %v", err)},
		}, nil
	}

	tx.Commit(ctx)

	return api.AdminRestartDeployment201Response{}, nil
}

func ListPodMetrics(
	ctx context.Context,
	controller k8s.Controller,
	request api.AdminListPodMetricsRequestObject,
) (api.AdminListPodMetricsResponseObject, error) {
	metrics, err := controller.ListRunningPodsWithMetrics(ctx)
	if err != nil {
		return api.AdminListPodMetrics500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot list pod metrics: %v", err)},
		}, nil
	}

	podMetrics := make([]api.PodMetrics, 0, len(metrics))
	for _, metric := range metrics {
		podMetric := api.PodMetrics{
			Name:   metric.Pod.Name,
			Cpu:    metric.Metrics.Containers[0].Usage.Cpu().MilliValue(),
			Memory: metric.Metrics.Containers[0].Usage.Memory().Value(),
		}
		podMetrics = append(podMetrics, podMetric)
	}

	return api.AdminListPodMetrics200JSONResponse{
		Pods: podMetrics,
	}, nil
}

func deserializeNotes(content []byte) *map[string]interface{} {
	var notesMap map[string]interface{}
	err := json.Unmarshal(content, &notesMap)
	if err != nil {
		return nil
	}
	return &notesMap
}

func publishConfigSuiteToS3(ctx context.Context, configSuiteId int64, tx pgx.Tx, ds dbaccess.DataSource, suiteStore suitestore.SuiteStore) error {
	configs, err := querier.ConfigListBySuiteIdGroupByActor(ctx, ds, configSuiteId)
	if err != nil {
		return err
	}

	publishingConfigs := make([]suitestore.ActorConfig, 0, len(configs))
	for _, config := range configs {
		var configContent map[string]string
		err := json.Unmarshal(config.Content, &configContent)
		if err != nil {
			return err
		}

		if config.ActorConfigurable {
			publishingConfigs = append(publishingConfigs, suitestore.ActorConfig{
				ActorName: config.ActorName,
				Configs:   configContent,
			})
		}
	}

	return suiteStore.WriteSuite(ctx, publishingConfigs)
}

func runDeploymentMigrationsAndUpdateDeployment(
	logger *slog.Logger,
	controller k8s.Controller,
	suiteStore suitestore.SuiteStore,
	ds dbaccess.DataSource,
	deploymentId int64,
	user string,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	err := doRunDeploymentMigrationsAndUpdateDeployment(ctx, logger, controller, suiteStore, ds, deploymentId, user)
	if err != nil {
		logger.Error("Cannot run deployment migrations", "error", err)
		// store error to deployment
		err = querier.UpdateDeploymentLastError(ctx, ds, &dbsqlc.UpdateDeploymentLastErrorParams{
			ID:        deploymentId,
			LastError: err.Error(),
		})
		if err != nil {
			logger.Error("Cannot update deployment last error", "error", err)
		}
	}
}

func doRunDeploymentMigrationsAndUpdateDeployment(
	ctx context.Context,
	logger *slog.Logger,
	controller k8s.Controller,
	suiteStore suitestore.SuiteStore,
	ds dbaccess.DataSource,
	deploymentId int64,
	user string,
) error {
	deployment, err := querier.DeploymentGetById(ctx, ds, deploymentId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("deployment not found")
		}
		logger.Error("Cannot get deployment", "error", err)
		return fmt.Errorf("Cannot get deployment: %v", err)
	}

	configs, err := querier.ConfigListBySuiteIdGroupByActor(ctx, ds, *deployment.ConfigSuiteID)
	if err != nil {
		logger.Error("Cannot get configs", "error", err)
		return fmt.Errorf("Cannot get configs: %v", err)
	}

	// run migrations
	err = runDeploymentMigrations(ctx, logger, controller, ds, deploymentId, configs)
	if err != nil {
		logger.Error("Cannot run deployment migrations", "error", err)
		return fmt.Errorf("Cannot run deployment migrations: %v", err)
	}

	// migration finished, update deployment status to deployed
	tx, err := ds.Begin(ctx)
	if err != nil {
		logger.Error("Cannot start transaction", "error", err)
		return fmt.Errorf("Cannot publish deployment. Cannot start transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Query deployment again and check status
	deployment, err = querier.DeploymentGetById(ctx, tx, deploymentId)
	if err != nil {
		logger.Error("Cannot get deployment", "error", err)
		return fmt.Errorf("Cannot get deployment: %v", err)
	}

	if deployment.Status != "deploying" {
		return fmt.Errorf("deployment must be in deploying status to be deployed")
	}

	if deployment.ConfigSuiteID == nil {
		logger.Error("Cannot publish deployment", "error", "Deployment has no config suite", "id", deployment.ID)
		return fmt.Errorf("Cannot publish deployment. Deployment has no config suite")
	}

	// Activate config suite
	suiteId, err := querier.ConfigSuiteActivate(ctx, tx, &dbsqlc.ConfigSuiteActivateParams{
		ID:        *deployment.ConfigSuiteID,
		UpdatedBy: user,
	})
	if err != nil {
		logger.Error("Cannot activate config suite", "error", err)
		return fmt.Errorf("Cannot activate config suite: %v", err)
	}

	// Update deployment status to deployed
	_, err = querier.DeploymentPublish(ctx, tx, &dbsqlc.DeploymentPublishParams{
		ID:         int64(deploymentId),
		ApprovedBy: user,
	})
	if err != nil {
		logger.Error("Cannot publish deployment", "error", err)
		return fmt.Errorf("Cannot publish deployment: %v", err)
	}

	// publish config suite to S3
	err = publishConfigSuiteToS3(ctx, suiteId, tx, tx, suiteStore)
	if err != nil {
		logger.Error("Cannot publish config suite to S3", "error", err)
		return fmt.Errorf("Cannot publish config suite to S3: %v", err)
	}

	// rotate actor api keys
	// it generates new api keys for each actor
	// and set the old ones to expire after 15 minutes
	apiTokens, err := rotateActorApiKeys(ctx, tx, configs, user)
	if err != nil {
		logger.Error("Cannot rotate actor api keys", "error", err)
		return fmt.Errorf("Cannot rotate actor api keys: %v", err)
	}

	err = updateKubernetesDeployments(ctx, controller, deployment, configs, apiTokens)
	if err != nil {
		logger.Error("Cannot update kubernetes deployments", "error", err)
		return fmt.Errorf("Cannot update kubernetes deployments: %v", err)
	}

	return tx.Commit(ctx)
}

func runDeploymentMigrations(
	ctx context.Context,
	logger *slog.Logger,
	controller k8s.Controller,
	ds dbaccess.DataSource,
	deploymentId int64,
	configs []*dbsqlc.ConfigListBySuiteIdGroupByActorRow,
) error {
	migrationParams := make([]k8s.MigrationParams, 0)
	for _, config := range configs {
		if !config.ActorMigratable {
			continue
		}

		var content map[string]string
		err := json.Unmarshal(config.Content, &content)
		if err != nil {
			return fmt.Errorf("failed to unmarshal config content: %v", err)
		}

		if content["KUBE_MIGRATE_DOCKER_IMAGE"] == "" {
			return fmt.Errorf("KUBE_MIGRATE_DOCKER_IMAGE is blank")
		}
		command, err := util.TokenizeCommand(content["KUBE_MIGRATE_COMMAND"])
		if err != nil {
			return fmt.Errorf("failed to tokenize KUBE_MIGRATE_COMMAND: %v", err)
		}

		migrationParams = append(migrationParams, k8s.MigrationParams{
			Serial:           deploymentId,
			Name:             "maos-" + config.ActorName,
			Image:            content["KUBE_MIGRATE_DOCKER_IMAGE"],
			ImagePullSecrets: content["KUBE_MIGRATE_PULL_IMAGE_SECRET"],
			EnvVars:          filterNonKubeConfigs(content),
			Command:          command,
			MemoryRequest:    content["KUBE_MIGRATE_MEMORY_REQUEST"],
			MemoryLimit:      content["KUBE_MIGRATE_MEMORY_LIMIT"],
		})
	}

	migrationLogs, errMigration := controller.RunMigrations(ctx, migrationParams)
	if errMigration != nil {
		logger.Error("failed to run migrations", "error", errMigration)
	}

	logsBytes, err := json.Marshal(migrationLogs)
	if err != nil {
		logger.Error("failed to marshal migration logs", "error", err)
	} else {
		// write migration logs to database
		err = querier.UpdateDeploymentMigrationLogs(ctx, ds, &dbsqlc.UpdateDeploymentMigrationLogsParams{
			ID:            deploymentId,
			MigrationLogs: logsBytes,
		})
		if err != nil {
			logger.Error("failed to write migration logs to database", "error", err)
		}
	}

	return errMigration
}

func updateKubernetesDeployments(
	ctx context.Context,
	controller k8s.Controller,
	deployment *dbsqlc.Deployment,
	configs []*dbsqlc.ConfigListBySuiteIdGroupByActorRow,
	apiTokens map[int64]string,
) error {
	deploymentSet := make([]k8s.DeploymentParams, 0, len(configs))

	for _, config := range configs {
		var content map[string]string
		err := json.Unmarshal(config.Content, &content)
		if err != nil {
			return fmt.Errorf("failed to unmarshal config content: %v", err)
		}

		if !config.ActorDeployable || content["KUBE_DOCKER_IMAGE"] == "" {
			continue
		}

		hasService := config.ActorRole == "portal" || config.ActorRole == "service"
		servicePort := int64(0)
		if hasService {
			servicePort, err = strconv.ParseInt(content["KUBE_SERVICE_PORT"], 10, 32)
			if err != nil {
				return fmt.Errorf("failed to parse KUBE_SERVICE_PORT: %v", err)
			}
		}
		hasIngress := config.ActorRole == "portal"

		// Prepare deployment params
		params := k8s.DeploymentParams{
			Name:          "maos-" + config.ActorName,
			Replicas:      getReplicasFromContent(content),
			Labels:        map[string]string{"app": config.ActorName},
			Image:         content["KUBE_DOCKER_IMAGE"],
			EnvVars:       filterNonKubeConfigs(content),
			APIKey:        apiTokens[config.ActorId],
			MemoryRequest: content["KUBE_MEMORY_REQUEST"],
			MemoryLimit:   content["KUBE_MEMORY_LIMIT"],
			HasService:    hasService,
			ServicePort:   int32(servicePort),
			HasIngress:    hasIngress,
			IngressHost:   content["KUBE_INGRESS_HOST"],
			BodyLimit:     content["KUBE_INGRESS_BODY_LIMIT"],
		}

		deploymentSet = append(deploymentSet, params)
	}

	// Update the deployment set using the Controller interface
	err := controller.UpdateDeploymentSet(ctx, deploymentSet)
	if err != nil {
		return fmt.Errorf("failed to update deployment set: %v", err)
	}

	return nil
}

// New helper function to filter out KUBE configs
func filterNonKubeConfigs(content map[string]string) map[string]string {
	filteredContent := make(map[string]string)
	for key, value := range content {
		if !strings.HasPrefix(key, "KUBE_") {
			filteredContent[key] = value
		}
	}
	return filteredContent
}

func getReplicasFromContent(content map[string]string) int32 {
	replicasStr, exists := content["KUBE_REPLICAS"]
	if !exists {
		return 1 // Default to 1 if not specified
	}

	replicas, err := strconv.Atoi(replicasStr)
	if err != nil || replicas < 1 {
		return 1 // Default to 1 if invalid
	}

	return int32(replicas)
}

func rotateActorApiKeys(
	ctx context.Context,
	tx pgx.Tx,
	configs []*dbsqlc.ConfigListBySuiteIdGroupByActorRow,
	createdBy string,
) (map[int64]string, error) {
	apiTokens := make(map[int64]string)

	for _, config := range configs {
		newApiToken := GenerateAPIToken()

		expirationTime := time.Now().Add(60 * 24 * time.Hour)
		_, err := querier.ApiTokenRotate(ctx, tx, &dbsqlc.ApiTokenRotateParams{
			ID:          newApiToken,
			ActorId:     config.ActorId,
			NewExpireAt: int64(expirationTime.Unix()),
			CreatedBy:   createdBy,
			Permissions: []string{"read:invocation"}, // TODO: read permissions from actor config
		})
		if err != nil {
			return nil, fmt.Errorf("failed to rorate API key of actor %s: %v", config.ActorName, err)
		}

		apiTokens[config.ActorId] = newApiToken
	}

	return apiTokens, nil
}
