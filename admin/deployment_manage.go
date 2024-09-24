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

func ListDeployments(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminListDeploymentsRequestObject) (api.AdminListDeploymentsResponseObject, error) {
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

	res, err := accessor.Querier().DeploymentListPaginated(ctx, accessor.Source(), &dbsqlc.DeploymentListPaginatedParams{
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

func GetDeployment(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminGetDeploymentRequestObject) (api.AdminGetDeploymentResponseObject, error) {
	logger.Info("AdminGetDeployment", "id", request.Id)

	deployment, err := accessor.Querier().DeploymentGetById(ctx, accessor.Source(), int64(request.Id))
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

	configs, err := accessor.Querier().ConfigListBySuiteIdGroupByAgent(ctx, accessor.Source(), *deployment.ConfigSuiteID)
	if err != nil {
		logger.Error("Cannot get configs", "error", err)
		return api.AdminGetDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot get configs: %v", err)},
		}, nil
	}

	filteredConfigs := lo.Filter(configs, func(row *dbsqlc.ConfigListBySuiteIdGroupByAgentRow, _ int) bool {
		return row.AgentConfigurable
	})

	resultConfigs := lo.Map(filteredConfigs, func(row *dbsqlc.ConfigListBySuiteIdGroupByAgentRow, _ int) api.Config {
		var content map[string]string
		err := json.Unmarshal(row.Content, &content)
		if err != nil {
			logger.Error("Cannot unmarshal content", "error", err)
		}

		// insert kubernetes config
		if row.AgentDeployable {
			InsertMissingKubeConfigsWithDefault(content)
		}

		return api.Config{
			Id:              row.ID,
			AgentId:         row.AgentId,
			AgentName:       row.AgentName,
			MinAgentVersion: util.SerializeAgentVersion(row.MinAgentVersion),
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

func CreateDeployment(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminCreateDeploymentRequestObject) (api.AdminCreateDeploymentResponseObject, error) {
	logger.Info("AdminCreateDeployment", "request", request.Body)

	if request.Body.Name == "" || request.Body.User == "" {
		return api.AdminCreateDeployment400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Missing required field"},
		}, nil
	}

	deployment, err := accessor.Querier().DeploymentInsertWithConfigSuite(ctx, accessor.Source(), &dbsqlc.DeploymentInsertWithConfigSuiteParams{
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

func SubmitDeployment(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminSubmitDeploymentRequestObject) (api.AdminSubmitDeploymentResponseObject, error) {
	logger.Info("AdminSubmitDeployment", "id", request.Id)

	tx, err := accessor.Source().Begin(ctx)
	if err != nil {
		logger.Error("Cannot start transaction", "error", err)
		return api.AdminSubmitDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot start transaction: %v", err)},
		}, nil
	}
	defer tx.Rollback(ctx)

	deployment, err := accessor.Querier().DeploymentGetById(ctx, tx, int64(request.Id))
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

	_, err = accessor.Querier().DeploymentSubmitForReview(ctx, tx, int64(request.Id))
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

func RejectDeployment(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminRejectDeploymentRequestObject) (api.AdminRejectDeploymentResponseObject, error) {
	logger.Info("AdminRejectDeployment", "id", request.Id, "user", request.Body.User)

	if request.Body == nil || request.Body.User == "" {
		return api.AdminRejectDeployment400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Missing required field"},
		}, nil
	}

	_, err := accessor.Querier().DeploymentReject(ctx, accessor.Source(), &dbsqlc.DeploymentRejectParams{
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
	accessor dbaccess.Accessor,
	suiteStore suitestore.SuiteStore,
	controller k8s.Controller,
	request api.AdminPublishDeploymentRequestObject) (api.AdminPublishDeploymentResponseObject, error) {
	logger.Info("AdminPublishDeployment", "id", request.Id, "user", request.Body.User)

	if request.Body == nil || request.Body.User == "" {
		return api.AdminPublishDeployment400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Missing required field"},
		}, nil
	}

	tx, err := accessor.Source().Begin(ctx)
	if err != nil {
		logger.Error("Cannot start transaction", "error", err)
		return api.AdminPublishDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot publish deployment. Cannot start transaction: %v", err)},
		}, nil
	}
	defer tx.Rollback(ctx)

	// Query deployment and check status
	deployment, err := accessor.Querier().DeploymentGetById(ctx, tx, int64(request.Id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.AdminPublishDeployment404Response{}, nil
		}
		logger.Error("Cannot get deployment", "error", err)
		return api.AdminPublishDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot publish deployment. Cannot get deployment: %v", err)},
		}, nil
	}

	if deployment.Status != "draft" && deployment.Status != "reviewing" {
		return api.AdminPublishDeployment400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Deployment must be in draft or reviewing status to be published"},
		}, nil
	}

	if deployment.ConfigSuiteID == nil {
		logger.Error("Cannot publish deployment", "error", "Deployment has no config suite", "id", deployment.ID)
		return api.AdminPublishDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: "Cannot publish deployment. Deployment has no config suite"},
		}, nil
	}

	// Activate config suite
	suiteId, err := accessor.Querier().ConfigSuiteActivate(ctx, tx, &dbsqlc.ConfigSuiteActivateParams{
		ID:        *deployment.ConfigSuiteID,
		UpdatedBy: request.Body.User,
	})
	if err != nil {
		logger.Error("Cannot activate config suite", "error", err)
		return api.AdminPublishDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot publish deployment. Cannot activate config suite: %v", err)},
		}, nil
	}

	// Update deployment status to deployed
	_, err = accessor.Querier().DeploymentPublish(ctx, tx, &dbsqlc.DeploymentPublishParams{
		ID:         int64(request.Id),
		ApprovedBy: request.Body.User,
	})
	if err != nil {
		logger.Error("Cannot publish deployment", "error", err)
		return api.AdminPublishDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot publish deployment: %v", err)},
		}, nil
	}

	// publish config suite to S3
	err = publishConfigSuiteToS3(ctx, suiteId, tx, accessor, suiteStore)
	if err != nil {
		logger.Error("Cannot serialize config suite", "error", err)
		return api.AdminPublishDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot serialize config suite: %v", err)},
		}, nil
	}

	// update kubernetes agent deployments
	configs, err := accessor.Querier().ConfigListBySuiteIdGroupByAgent(ctx, tx, *deployment.ConfigSuiteID)
	if err != nil {
		logger.Error("Cannot get configs", "error", err)
		return api.AdminPublishDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot get configs: %v", err)},
		}, nil
	}

	// rotate agent api keys
	// it generates new api keys for each agent
	// and set the old ones to expire after 15 minutes
	apiTokens, err := rotateAgentApiKeys(ctx, accessor, tx, configs, request.Body.User)
	if err != nil {
		logger.Error("Cannot rotate agent api keys", "error", err)
		return api.AdminPublishDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot rotate agent api keys: %v", err)},
		}, nil
	}

	err = updateKubernetesDeployments(ctx, controller, deployment, configs, apiTokens)
	if err != nil {
		logger.Error("Cannot update kubernetes deployments", "error", err)
		return api.AdminPublishDeployment500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot update kubernetes deployments: %v", err)},
		}, nil
	}

	tx.Commit(ctx)

	return api.AdminPublishDeployment201Response{}, nil
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

func deserializeNotes(content []byte) *map[string]interface{} {
	var notesMap map[string]interface{}
	err := json.Unmarshal(content, &notesMap)
	if err != nil {
		return nil
	}
	return &notesMap
}

func publishConfigSuiteToS3(ctx context.Context, configSuiteId int64, tx pgx.Tx, accessor dbaccess.Accessor, suiteStore suitestore.SuiteStore) error {
	configs, err := accessor.Querier().ConfigListBySuiteIdGroupByAgent(ctx, tx, configSuiteId)
	if err != nil {
		return err
	}

	publishingConfigs := make([]suitestore.AgentConfig, 0, len(configs))
	for _, config := range configs {
		var configContent map[string]string
		err := json.Unmarshal(config.Content, &configContent)
		if err != nil {
			return err
		}

		publishingConfigs = append(publishingConfigs, suitestore.AgentConfig{
			AgentName: config.AgentName,
			Configs:   configContent,
		})
	}

	return suiteStore.WriteSuite(ctx, publishingConfigs)
}

func updateKubernetesDeployments(
	ctx context.Context,
	controller k8s.Controller,
	deployment *dbsqlc.Deployment,
	configs []*dbsqlc.ConfigListBySuiteIdGroupByAgentRow,
	apiTokens map[int64]string,
) error {
	deploymentSet := make([]k8s.DeploymentParams, 0, len(configs))

	for _, config := range configs {
		var content map[string]string
		err := json.Unmarshal(config.Content, &content)
		if err != nil {
			return fmt.Errorf("failed to unmarshal config content: %v", err)
		}

		if !config.AgentDeployable || content["KUBE_DOCKER_IMAGE"] == "" {
			continue
		}

		// Prepare deployment params
		params := k8s.DeploymentParams{
			Name:          "maos-" + config.AgentName,
			Replicas:      getReplicasFromContent(content),
			Labels:        map[string]string{"app": config.AgentName},
			Image:         content["KUBE_DOCKER_IMAGE"],
			EnvVars:       filterNonKubeConfigs(content),
			APIKey:        apiTokens[config.AgentId],
			MemoryRequest: content["KUBE_MEMORY_REQUEST"],
			MemoryLimit:   content["KUBE_MEMORY_LIMIT"],
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

func rotateAgentApiKeys(
	ctx context.Context,
	accessor dbaccess.Accessor,
	tx pgx.Tx,
	configs []*dbsqlc.ConfigListBySuiteIdGroupByAgentRow,
	createdBy string,
) (map[int64]string, error) {
	apiTokens := make(map[int64]string)

	for _, config := range configs {
		newApiToken := GenerateAPIToken()

		expirationTime := time.Now().Add(60 * 24 * time.Hour)
		_, err := accessor.Querier().ApiTokenRotate(ctx, tx, &dbsqlc.ApiTokenRotateParams{
			ID:          newApiToken,
			AgentId:     config.AgentId,
			NewExpireAt: int64(expirationTime.Unix()),
			CreatedBy:   createdBy,
			Permissions: []string{"read:invocation"}, // TODO: read permissions from agent config
		})
		if err != nil {
			return nil, fmt.Errorf("failed to rorate API key of agent %s: %v", config.AgentName, err)
		}

		apiTokens[config.AgentId] = newApiToken
	}

	return apiTokens, nil
}
