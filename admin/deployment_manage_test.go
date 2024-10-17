package admin_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/admin"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
	"gitlab.com/navyx/ai/maos/maos-core/k8s"
)

type mockK8sController struct {
	k8s.Controller
	updatedDeploymentSets [][]k8s.DeploymentParams
	migrationParams       [][]k8s.MigrationParams
	migrationResults      map[string]interface{}
}

func (m *mockK8sController) UpdateDeploymentSet(ctx context.Context, deploymentSet []k8s.DeploymentParams) error {
	m.updatedDeploymentSets = append(m.updatedDeploymentSets, deploymentSet)
	return nil
}

func (m *mockK8sController) TriggerRollingRestart(ctx context.Context, deploymentName string) error {
	return nil
}

func (m *mockK8sController) RunMigrations(ctx context.Context, migrations []k8s.MigrationParams) (map[string]interface{}, error) {
	m.migrationParams = append(m.migrationParams, migrations)
	return m.migrationResults, nil
}

func TestListDeploymentsWithDB(t *testing.T) {
	t.Parallel()
	logger := testhelper.Logger(t)

	t.Run("Successful listing", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		fixture.InsertDeployment(t, ctx, dbPool, "deployment1", []string{"user1", "user2"})
		fixture.InsertDeployment(t, ctx, dbPool, "deployment2", []string{"user3", "user4"})

		request := api.AdminListDeploymentsRequestObject{}
		response, err := admin.ListDeployments(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminListDeployments200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListDeployments200JSONResponse)
		require.NotNil(t, jsonResponse.Data)
		require.Len(t, jsonResponse.Data, 2)
		require.Equal(t, int64(2), jsonResponse.Meta.Total)

		actualNames := lo.Map(jsonResponse.Data, func(d api.Deployment, _ int) string { return d.Name })
		require.Equal(t, []string{"deployment2", "deployment1"}, actualNames)

		for _, deployment := range jsonResponse.Data {
			require.NotZero(t, deployment.CreatedAt)
			require.Equal(t, "tester", deployment.CreatedBy)
			require.Equal(t, api.DeploymentStatusDraft, deployment.Status)

			// Check reviewers
			if deployment.Name == "deployment1" {
				require.ElementsMatch(t, []string{"user1", "user2"}, deployment.Reviewers)
			} else if deployment.Name == "deployment2" {
				require.ElementsMatch(t, []string{"user3", "user4"}, deployment.Reviewers)
			}
		}
	})

	t.Run("Custom page and page size", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		lo.RepeatBy(25, func(i int) *dbsqlc.Deployment {
			return fixture.InsertDeployment(t, ctx, dbPool, fmt.Sprintf("deployment-%03d", i), nil)
		})

		request := api.AdminListDeploymentsRequestObject{
			Params: api.AdminListDeploymentsParams{
				Page:     lo.ToPtr(2),
				PageSize: lo.ToPtr(10),
			},
		}
		response, err := admin.ListDeployments(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminListDeployments200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListDeployments200JSONResponse)
		require.NotNil(t, jsonResponse.Data)
		require.Len(t, jsonResponse.Data, 10)
		require.Equal(t, int64(25), jsonResponse.Meta.Total)

		expectedNames := lo.Map(lo.Range(10), func(i int, _ int) string { return fmt.Sprintf("deployment-%03d", 24-i-10) })
		actualNames := lo.Map(jsonResponse.Data, func(d api.Deployment, _ int) string { return d.Name })
		require.Equal(t, expectedNames, actualNames)

		for i := 0; i < len(jsonResponse.Data)-1; i++ {
			require.True(t, jsonResponse.Data[i].Id > jsonResponse.Data[i+1].Id)
			require.True(t, jsonResponse.Data[i].CreatedAt >= jsonResponse.Data[i+1].CreatedAt)
		}

		for i, deployment := range jsonResponse.Data {
			require.NotEmpty(t, deployment.Id)
			require.Equal(t, fmt.Sprintf("deployment-%03d", 24-i-10), deployment.Name)
			require.NotZero(t, deployment.CreatedAt)
			require.NotEmpty(t, deployment.CreatedBy)
			require.NotEmpty(t, deployment.Status)
		}
	})

	t.Run("Filter by status", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		// Insert deployments with different statuses
		fixture.InsertDeployment(t, ctx, dbPool, "deployment-draft", nil)
		fixture.InsertDeployment(t, ctx, dbPool, "deployment-reviewing", nil)
		fixture.InsertDeployment(t, ctx, dbPool, "deployment-approved", nil)

		// Update statuses
		_, err := dbPool.Exec(ctx, "UPDATE deployments SET status = 'reviewing' WHERE name = 'deployment-reviewing'")
		require.NoError(t, err)
		_, err = dbPool.Exec(ctx, "UPDATE deployments SET status = 'approved' WHERE name = 'deployment-approved'")
		require.NoError(t, err)

		// Filter by 'reviewing' status
		reviewingStatus := api.AdminListDeploymentsParamsStatus("reviewing")
		request := api.AdminListDeploymentsRequestObject{
			Params: api.AdminListDeploymentsParams{
				Status: &reviewingStatus,
			},
		}
		response, err := admin.ListDeployments(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminListDeployments200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListDeployments200JSONResponse)
		require.NotNil(t, jsonResponse.Data)
		require.Len(t, jsonResponse.Data, 1)
		require.Equal(t, "deployment-reviewing", jsonResponse.Data[0].Name)
		require.Equal(t, api.DeploymentStatusReviewing, jsonResponse.Data[0].Status)
	})

	t.Run("Filter by reviewer", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		// Insert deployments with different reviewers
		fixture.InsertDeployment(t, ctx, dbPool, "deployment-reviewer1", []string{"reviewer1", "reviewer2"})
		fixture.InsertDeployment(t, ctx, dbPool, "deployment-reviewer2", []string{"reviewer2", "reviewer3"})
		fixture.InsertDeployment(t, ctx, dbPool, "deployment-reviewer3", []string{"reviewer3", "reviewer4"})

		// Filter by 'reviewer2'
		reviewer := "reviewer2"
		request := api.AdminListDeploymentsRequestObject{
			Params: api.AdminListDeploymentsParams{
				Reviewer: &reviewer,
			},
		}
		response, err := admin.ListDeployments(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminListDeployments200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListDeployments200JSONResponse)
		require.NotNil(t, jsonResponse.Data)
		require.Len(t, jsonResponse.Data, 2)

		deploymentNames := []string{jsonResponse.Data[0].Name, jsonResponse.Data[1].Name}
		require.Contains(t, deploymentNames, "deployment-reviewer1")
		require.Contains(t, deploymentNames, "deployment-reviewer2")
		require.NotContains(t, deploymentNames, "deployment-reviewer3")

		for _, deployment := range jsonResponse.Data {
			require.Contains(t, deployment.Reviewers, "reviewer2")
		}
	})

	t.Run("Filter by name", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		// Insert deployments with different names
		fixture.InsertDeployment(t, ctx, dbPool, "deployment-alpha", nil)
		fixture.InsertDeployment(t, ctx, dbPool, "deployment-beta", nil)
		fixture.InsertDeployment(t, ctx, dbPool, "other-gamma", nil)

		// Filter by 'deployment'
		name := "deployment"
		request := api.AdminListDeploymentsRequestObject{
			Params: api.AdminListDeploymentsParams{
				Name: &name,
			},
		}
		response, err := admin.ListDeployments(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminListDeployments200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListDeployments200JSONResponse)
		require.NotNil(t, jsonResponse.Data)
		require.Len(t, jsonResponse.Data, 2)

		deploymentNames := []string{jsonResponse.Data[0].Name, jsonResponse.Data[1].Name}
		require.Contains(t, deploymentNames, "deployment-alpha")
		require.Contains(t, deploymentNames, "deployment-beta")
		require.NotContains(t, deploymentNames, "other-gamma")

		for _, deployment := range jsonResponse.Data {
			require.Contains(t, deployment.Name, "deployment")
		}
	})

	t.Run("Filter by partial name", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		// Insert deployments with different names
		fixture.InsertDeployment(t, ctx, dbPool, "partial-match-1", nil)
		fixture.InsertDeployment(t, ctx, dbPool, "partial-match-2", nil)
		fixture.InsertDeployment(t, ctx, dbPool, "no-m-atch", nil)

		// Filter by partial name 'match'
		name := "match"
		request := api.AdminListDeploymentsRequestObject{
			Params: api.AdminListDeploymentsParams{
				Name: &name,
			},
		}
		response, err := admin.ListDeployments(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminListDeployments200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListDeployments200JSONResponse)
		require.NotNil(t, jsonResponse.Data)
		require.Len(t, jsonResponse.Data, 2)

		deploymentNames := []string{jsonResponse.Data[0].Name, jsonResponse.Data[1].Name}
		require.Contains(t, deploymentNames, "partial-match-1")
		require.Contains(t, deploymentNames, "partial-match-2")
		require.NotContains(t, deploymentNames, "no-match")

		for _, deployment := range jsonResponse.Data {
			require.Contains(t, deployment.Name, "match")
		}
	})

	t.Run("Filter by id list", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		defer dbPool.Close()

		// Insert deployments with different IDs
		deployment1 := fixture.InsertDeployment(t, ctx, dbPool, "deployment-1", nil)
		deployment2 := fixture.InsertDeployment(t, ctx, dbPool, "deployment-2", nil)
		deployment3 := fixture.InsertDeployment(t, ctx, dbPool, "deployment-3", nil)

		// Filter by IDs of deployment1 and deployment2
		idList := []int64{deployment1.ID, deployment2.ID}
		request := api.AdminListDeploymentsRequestObject{
			Params: api.AdminListDeploymentsParams{
				Id: &idList,
			},
		}
		response, err := admin.ListDeployments(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminListDeployments200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListDeployments200JSONResponse)
		require.NotNil(t, jsonResponse.Data)
		require.Len(t, jsonResponse.Data, 2)

		deploymentIDs := []int64{jsonResponse.Data[0].Id, jsonResponse.Data[1].Id}
		require.Contains(t, deploymentIDs, deployment1.ID)
		require.Contains(t, deploymentIDs, deployment2.ID)
		require.NotContains(t, deploymentIDs, deployment3.ID)

		for _, deployment := range jsonResponse.Data {
			require.Contains(t, idList, deployment.Id)
		}
	})

	t.Run("Database pool closed", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)

		fixture.InsertDeployment(t, ctx, dbPool, "deployment1", nil)
		dbPool.Close()

		request := api.AdminListDeploymentsRequestObject{}
		response, err := admin.ListDeployments(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminListDeployments500JSONResponse{}, response)
		errorResponse := response.(api.AdminListDeployments500JSONResponse)
		require.Contains(t, errorResponse.Error, "Cannot list deployments: closed pool")
	})
}

func TestGetDeployment(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := testhelper.Logger(t)

	t.Run("Successful retrieval with default Kube configs", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// Create two actors
		actor1 := fixture.InsertActor2(t, ctx, dbPool, "actor1", "agent", true, true, true, false)
		actor2 := fixture.InsertActor2(t, ctx, dbPool, "actor2", "agent", true, false, true, false)

		// Create a config suite
		createResponse, err := admin.CreateDeployment(ctx, logger, dbPool, api.AdminCreateDeploymentRequestObject{
			Body: &api.AdminCreateDeploymentJSONRequestBody{
				Name:      "test-deployment",
				User:      "test-user",
				Reviewers: &[]string{"reviewer1", "reviewer2"},
			},
		})
		require.NoError(t, err)
		createdDeployment := createResponse.(api.AdminCreateDeployment201JSONResponse).Data

		// Create configs for each actor
		config1Content := map[string]string{"key1": "value1", "key2": "value2"}
		config2Content := map[string]string{"key3": "value3", "key4": "value4"}
		fixture.InsertConfig2(t, ctx, dbPool, actor1.ID, createdDeployment.ConfigSuiteId, "testuser", config1Content)
		fixture.InsertConfig2(t, ctx, dbPool, actor2.ID, createdDeployment.ConfigSuiteId, "testuser", config2Content)

		request := api.AdminGetDeploymentRequestObject{Id: createdDeployment.Id}
		response, err := admin.GetDeployment(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminGetDeployment200JSONResponse{}, response)

		retrievedDeployment := response.(api.AdminGetDeployment200JSONResponse)
		require.Equal(t, createdDeployment.Id, retrievedDeployment.Id)
		require.Equal(t, createdDeployment.Name, retrievedDeployment.Name)
		require.Equal(t, createdDeployment.CreatedBy, retrievedDeployment.CreatedBy)
		require.Equal(t, createdDeployment.CreatedAt, retrievedDeployment.CreatedAt)
		require.Equal(t, string(createdDeployment.Status), string(retrievedDeployment.Status))
		require.Equal(t, createdDeployment.Reviewers, retrievedDeployment.Reviewers)
		require.Equal(t, createdDeployment.ApprovedBy, retrievedDeployment.ApprovedBy)
		require.Equal(t, createdDeployment.ApprovedAt, retrievedDeployment.ApprovedAt)
		require.Equal(t, createdDeployment.FinishedBy, retrievedDeployment.FinishedBy)
		require.Equal(t, createdDeployment.FinishedAt, retrievedDeployment.FinishedAt)

		// Check configs
		require.NotNil(t, retrievedDeployment.Configs)
		require.Len(t, *retrievedDeployment.Configs, 2)

		configMap := make(map[int64]api.Config)
		for _, config := range *retrievedDeployment.Configs {
			configMap[config.ActorId] = config
		}

		// Check config for actor1
		config1 := configMap[actor1.ID]
		require.NotZero(t, config1.Id)
		require.Equal(t, actor1.ID, config1.ActorId)
		require.Equal(t, actor1.Name, config1.ActorName)
		for key, value := range config1Content {
			require.Equal(t, value, config1.Content[key])
		}
		// Check for default Kube configs
		for key, defaultValue := range admin.KubeConfigsWithDefault {
			require.Equal(t, defaultValue, config1.Content[key])
		}
		require.NotZero(t, config1.CreatedAt)
		require.NotEmpty(t, config1.CreatedBy)

		// Check config for actor2
		config2 := configMap[actor2.ID]
		require.NotZero(t, config2.Id)
		require.Equal(t, actor2.ID, config2.ActorId)
		require.Equal(t, actor2.Name, config2.ActorName)
		for key, value := range config2Content {
			require.Equal(t, value, config2.Content[key])
		}
		// config should not have default kube configs
		for key := range admin.KubeConfigsWithDefault {
			require.Equal(t, "", config2.Content[key])
		}
		require.NotZero(t, config2.CreatedAt)
		require.NotEmpty(t, config2.CreatedBy)
	})

	t.Run("Retrieval with Kube configs", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		actor := fixture.InsertActor2(t, ctx, dbPool, "actor-kube", "agent", true, true, true, false)

		// Create a config suite
		createResponse, err := admin.CreateDeployment(ctx, logger, dbPool, api.AdminCreateDeploymentRequestObject{
			Body: &api.AdminCreateDeploymentJSONRequestBody{
				Name: "test-deployment-kube",
				User: "test-user",
			},
		})
		require.NoError(t, err)
		createdDeployment := createResponse.(api.AdminCreateDeployment201JSONResponse).Data

		// Create config with custom Kube settings
		customKubeConfig := map[string]string{
			"KUBE_REPLICAS":          "3",
			"KUBE_DOCKER_IMAGE":      "custom-image:latest",
			"KUBE_IMAGE_PULL_SECRET": "custom-image-pull-secret",
			"KUBE_CPU_REQUEST":       "250m",
			"KUBE_MEMORY_REQUEST":    "512Mi",
			"KUBE_CPU_LIMIT":         "500m",
			"KUBE_MEMORY_LIMIT":      "1Gi",
		}
		fixture.InsertConfig2(t, ctx, dbPool, actor.ID, createdDeployment.ConfigSuiteId, "testuser", customKubeConfig)

		request := api.AdminGetDeploymentRequestObject{Id: createdDeployment.Id}
		response, err := admin.GetDeployment(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminGetDeployment200JSONResponse{}, response)

		retrievedDeployment := response.(api.AdminGetDeployment200JSONResponse)
		require.NotNil(t, retrievedDeployment.Configs)
		require.Len(t, *retrievedDeployment.Configs, 1)

		config := (*retrievedDeployment.Configs)[0]
		for key, value := range customKubeConfig {
			require.Equal(t, value, config.Content[key])
		}
	})

	t.Run("Deployment not found", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		request := api.AdminGetDeploymentRequestObject{
			Id: 999999, // Non-existent ID
		}

		response, err := admin.GetDeployment(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminGetDeployment404Response{}, response)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// Close the database pool to simulate a database error
		dbPool.Close()

		request := api.AdminGetDeploymentRequestObject{
			Id: 1,
		}

		response, err := admin.GetDeployment(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminGetDeployment500JSONResponse{}, response)

		errorResponse := response.(api.AdminGetDeployment500JSONResponse)
		require.Contains(t, errorResponse.Error, "Cannot get deployment")
	})
}

func TestCreateDeployment(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := testhelper.Logger(t)

	t.Run("Successful creation", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		request := api.AdminCreateDeploymentRequestObject{
			Body: &api.AdminCreateDeploymentJSONRequestBody{
				Name: "test-deployment",
				User: "test-user",
			},
		}

		response, err := admin.CreateDeployment(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminCreateDeployment201JSONResponse{}, response)

		createdDeployment := response.(api.AdminCreateDeployment201JSONResponse)
		require.NotEmpty(t, createdDeployment.Data.Id)
		require.Equal(t, "test-deployment", createdDeployment.Data.Name)
		require.Equal(t, "test-user", createdDeployment.Data.CreatedBy)
		require.NotZero(t, createdDeployment.Data.CreatedAt)
		require.Equal(t, api.DeploymentStatus("draft"), createdDeployment.Data.Status)
		require.Nil(t, createdDeployment.Data.ApprovedBy)
		require.Zero(t, createdDeployment.Data.ApprovedAt)
		require.Nil(t, createdDeployment.Data.FinishedBy)
		require.Zero(t, createdDeployment.Data.FinishedAt)

		// Verify the deployment was actually inserted in the database using DeploymentList
		listRequest := api.AdminListDeploymentsRequestObject{
			Params: api.AdminListDeploymentsParams{
				Page:     lo.ToPtr(1),
				PageSize: lo.ToPtr(10),
			},
		}
		listResponse, err := admin.ListDeployments(ctx, logger, dbPool, listRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminListDeployments200JSONResponse{}, listResponse)

		listJsonResponse := listResponse.(api.AdminListDeployments200JSONResponse)
		require.NotEmpty(t, listJsonResponse.Data)

		foundDeployment, found := lo.Find(listJsonResponse.Data, func(d api.Deployment) bool {
			return d.Id == createdDeployment.Data.Id
		})

		require.True(t, found)
		require.NotNil(t, foundDeployment)
		require.Equal(t, createdDeployment.Data.Id, foundDeployment.Id)
		require.Equal(t, createdDeployment.Data.Name, foundDeployment.Name)
		require.Equal(t, createdDeployment.Data.CreatedBy, foundDeployment.CreatedBy)
		require.Equal(t, createdDeployment.Data.CreatedAt, foundDeployment.CreatedAt)
		require.Equal(t, createdDeployment.Data.Status, foundDeployment.Status)

		// Check if config suite was created
		configSuite, err := querier.ConfigSuiteGetById(ctx, dbPool, *createdDeployment.Data.ConfigSuiteId)
		require.NoError(t, err)
		require.NotNil(t, configSuite)
		require.Equal(t, createdDeployment.Data.CreatedBy, configSuite.CreatedBy)

		// Get all actors
		actors, err := querier.ActorListPagenated(ctx, dbPool, &dbsqlc.ActorListPagenatedParams{Page: 1, PageSize: 1000})
		require.NoError(t, err)

		// Check if configs were created for all actors
		for _, actor := range actors {
			config, err := querier.ConfigFindByActorIdAndSuiteId(ctx, dbPool, &dbsqlc.ConfigFindByActorIdAndSuiteIdParams{
				ActorId:       actor.ID,
				ConfigSuiteID: *createdDeployment.Data.ConfigSuiteId,
			})
			require.NoError(t, err)
			require.NotNil(t, config)
			require.Equal(t, createdDeployment.Data.CreatedBy, config.CreatedBy)
		}
	})

	t.Run("Create deployment with existing configs", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		logger := testhelper.Logger(t)

		// Create 3 actors
		actor1 := fixture.InsertActor(t, ctx, dbPool, "actor1")
		actor2 := fixture.InsertActor(t, ctx, dbPool, "actor2")
		actor3 := fixture.InsertActor(t, ctx, dbPool, "actor3")

		// Create existing configs for actor1 and actor2
		existingContent1 := []byte(`{"key": "value1"}`)
		existingContent2 := []byte(`{"key": "value2"}`)
		_, err := querier.ConfigInsert(ctx, dbPool, &dbsqlc.ConfigInsertParams{
			ActorId:         actor1.ID,
			Content:         existingContent1,
			MinActorVersion: []int32{1, 0, 0},
			CreatedBy:       "test-user",
		})
		require.NoError(t, err)
		_, err = querier.ConfigInsert(ctx, dbPool, &dbsqlc.ConfigInsertParams{
			ActorId:   actor2.ID,
			Content:   existingContent2,
			CreatedBy: "test-user",
		})
		require.NoError(t, err)

		// Create a new deployment
		request := api.AdminCreateDeploymentRequestObject{
			Body: &api.AdminCreateDeploymentJSONRequestBody{
				Name: "test-deployment",
				User: "test-user",
			},
		}

		response, err := admin.CreateDeployment(ctx, logger, dbPool, request)
		require.NoError(t, err)
		require.IsType(t, api.AdminCreateDeployment201JSONResponse{}, response)

		createdDeployment := response.(api.AdminCreateDeployment201JSONResponse)
		require.NotEmpty(t, createdDeployment.Data.Id)
		require.NotEmpty(t, createdDeployment.Data.ConfigSuiteId)

		// Check if configs were created for all actors
		config1, err := querier.ConfigFindByActorIdAndSuiteId(ctx, dbPool, &dbsqlc.ConfigFindByActorIdAndSuiteIdParams{
			ActorId:       actor1.ID,
			ConfigSuiteID: *createdDeployment.Data.ConfigSuiteId,
		})
		require.NoError(t, err)
		require.Equal(t, existingContent1, config1.Content)
		require.Equal(t, []int32{1, 0, 0}, config1.MinActorVersion)

		config2, err := querier.ConfigFindByActorIdAndSuiteId(ctx, dbPool, &dbsqlc.ConfigFindByActorIdAndSuiteIdParams{
			ActorId:       actor2.ID,
			ConfigSuiteID: *createdDeployment.Data.ConfigSuiteId,
		})
		require.NoError(t, err)
		require.Equal(t, existingContent2, config2.Content)
		require.Nil(t, config2.MinActorVersion)

		config3, err := querier.ConfigFindByActorIdAndSuiteId(ctx, dbPool, &dbsqlc.ConfigFindByActorIdAndSuiteIdParams{
			ActorId:       actor3.ID,
			ConfigSuiteID: *createdDeployment.Data.ConfigSuiteId,
		})
		require.NoError(t, err)
		require.NotNil(t, config3)
		require.Equal(t, []byte("{}"), config3.Content) // New config should have empty JSON object
		require.Nil(t, config3.MinActorVersion)
	})

	t.Run("Create deployment with existing configs and active config suite", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		logger := testhelper.Logger(t)

		// Create 3 actors
		actor1 := fixture.InsertActor(t, ctx, dbPool, "actor1")
		actor2 := fixture.InsertActor(t, ctx, dbPool, "actor2")
		actor3 := fixture.InsertActor(t, ctx, dbPool, "actor3")

		// Create active config suite
		suite := fixture.InsertConfigSuite(t, ctx, dbPool)
		_, err := querier.ConfigSuiteActivate(ctx, dbPool, &dbsqlc.ConfigSuiteActivateParams{
			UpdatedBy: "test-user",
			ID:        suite.ID,
		})
		require.NoError(t, err)

		// Create existing active configs for actor1 and actor2
		_, err = querier.ConfigInsert(ctx, dbPool, &dbsqlc.ConfigInsertParams{
			ActorId:       actor1.ID,
			ConfigSuiteID: &suite.ID,
			Content:       []byte(`{"key": "active-value1"}`),
			CreatedBy:     "test-user",
		})
		require.NoError(t, err)
		_, err = querier.ConfigInsert(ctx, dbPool, &dbsqlc.ConfigInsertParams{
			ActorId:       actor2.ID,
			ConfigSuiteID: &suite.ID,
			Content:       []byte(`{"key": "active-value2"}`),
			CreatedBy:     "test-user",
		})
		require.NoError(t, err)

		// Create non-active configs for actor1 and actor2
		existingContent1 := []byte(`{"key": "value1"}`)
		existingContent2 := []byte(`{"key": "value2"}`)
		_, err = querier.ConfigInsert(ctx, dbPool, &dbsqlc.ConfigInsertParams{
			ActorId:         actor1.ID,
			Content:         existingContent1,
			MinActorVersion: []int32{1, 0, 0},
			CreatedBy:       "test-user",
		})
		require.NoError(t, err)
		_, err = querier.ConfigInsert(ctx, dbPool, &dbsqlc.ConfigInsertParams{
			ActorId:   actor2.ID,
			Content:   existingContent2,
			CreatedBy: "test-user",
		})
		require.NoError(t, err)

		// Create a new deployment
		request := api.AdminCreateDeploymentRequestObject{
			Body: &api.AdminCreateDeploymentJSONRequestBody{
				Name: "test-deployment",
				User: "test-user",
			},
		}

		response, err := admin.CreateDeployment(ctx, logger, dbPool, request)
		require.NoError(t, err)
		require.IsType(t, api.AdminCreateDeployment201JSONResponse{}, response)

		createdDeployment := response.(api.AdminCreateDeployment201JSONResponse)
		require.NotEmpty(t, createdDeployment.Data.Id)
		require.NotEmpty(t, createdDeployment.Data.ConfigSuiteId)

		// Check if configs were created for all actors
		config1, err := querier.ConfigFindByActorIdAndSuiteId(ctx, dbPool, &dbsqlc.ConfigFindByActorIdAndSuiteIdParams{
			ActorId:       actor1.ID,
			ConfigSuiteID: *createdDeployment.Data.ConfigSuiteId,
		})
		require.NoError(t, err)
		require.Equal(t, []byte(`{"key": "active-value1"}`), config1.Content)

		config2, err := querier.ConfigFindByActorIdAndSuiteId(ctx, dbPool, &dbsqlc.ConfigFindByActorIdAndSuiteIdParams{
			ActorId:       actor2.ID,
			ConfigSuiteID: *createdDeployment.Data.ConfigSuiteId,
		})
		require.NoError(t, err)
		require.Equal(t, []byte(`{"key": "active-value2"}`), config2.Content)

		config3, err := querier.ConfigFindByActorIdAndSuiteId(ctx, dbPool, &dbsqlc.ConfigFindByActorIdAndSuiteIdParams{
			ActorId:       actor3.ID,
			ConfigSuiteID: *createdDeployment.Data.ConfigSuiteId,
		})
		require.NoError(t, err)
		require.NotNil(t, config3)
		require.Equal(t, []byte("{}"), config3.Content) // New config should have empty JSON object
	})

	t.Run("Create deployment by cloning from existing and not from active config suite", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		logger := testhelper.Logger(t)

		// Create 3 actors
		actor1 := fixture.InsertActor(t, ctx, dbPool, "actor1")
		actor2 := fixture.InsertActor(t, ctx, dbPool, "actor2")
		actor3 := fixture.InsertActor(t, ctx, dbPool, "actor3")

		// Create active config suite
		suite := fixture.InsertConfigSuite(t, ctx, dbPool)
		_, err := querier.ConfigSuiteActivate(ctx, dbPool, &dbsqlc.ConfigSuiteActivateParams{
			UpdatedBy: "test-user",
			ID:        suite.ID,
		})
		require.NoError(t, err)

		// Create existing active configs for actor1 and actor2
		_, err = querier.ConfigInsert(ctx, dbPool, &dbsqlc.ConfigInsertParams{
			ActorId:       actor1.ID,
			ConfigSuiteID: &suite.ID,
			Content:       []byte(`{"key": "active-value1"}`),
			CreatedBy:     "test-user",
		})
		require.NoError(t, err)
		_, err = querier.ConfigInsert(ctx, dbPool, &dbsqlc.ConfigInsertParams{
			ActorId:       actor2.ID,
			ConfigSuiteID: &suite.ID,
			Content:       []byte(`{"key": "active-value2"}`),
			CreatedBy:     "test-user",
		})
		require.NoError(t, err)

		// Create non-active configs for actor1 and actor2
		existingContent1 := []byte(`{"key": "value1"}`)
		existingContent2 := []byte(`{"key": "value2"}`)
		_, err = querier.ConfigInsert(ctx, dbPool, &dbsqlc.ConfigInsertParams{
			ActorId:         actor1.ID,
			Content:         existingContent1,
			MinActorVersion: []int32{1, 0, 0},
			CreatedBy:       "test-user",
		})
		require.NoError(t, err)
		_, err = querier.ConfigInsert(ctx, dbPool, &dbsqlc.ConfigInsertParams{
			ActorId:   actor2.ID,
			Content:   existingContent2,
			CreatedBy: "test-user",
		})
		require.NoError(t, err)

		// Create a new deployment
		deployment, err := querier.DeploymentInsertWithConfigSuite(ctx, dbPool, &dbsqlc.DeploymentInsertWithConfigSuiteParams{
			Name:      "deployment16888",
			CreatedBy: "test-user",
		})
		require.NoError(t, err)
		dbPool.Exec(ctx, "UPDATE configs SET content = $1 WHERE config_suite_id = $2 AND actor_id = $3", `{"key":"org1"}`, deployment.ConfigSuiteID, actor1.ID)
		dbPool.Exec(ctx, "UPDATE configs SET content = $1 WHERE config_suite_id = $2 AND actor_id = $3", `{"key":"org2"}`, deployment.ConfigSuiteID, actor2.ID)

		// Create a new deployment
		request := api.AdminCreateDeploymentRequestObject{
			Body: &api.AdminCreateDeploymentJSONRequestBody{
				Name:      "test-deployment",
				User:      "test-user",
				CloneFrom: &deployment.ID,
			},
		}

		response, err := admin.CreateDeployment(ctx, logger, dbPool, request)
		require.NoError(t, err)
		require.IsType(t, api.AdminCreateDeployment201JSONResponse{}, response)

		createdDeployment := response.(api.AdminCreateDeployment201JSONResponse)
		require.NotEmpty(t, createdDeployment.Data.Id)
		require.NotEmpty(t, createdDeployment.Data.ConfigSuiteId)

		// Check if configs were created for all actors
		config1, err := querier.ConfigFindByActorIdAndSuiteId(ctx, dbPool, &dbsqlc.ConfigFindByActorIdAndSuiteIdParams{
			ActorId:       actor1.ID,
			ConfigSuiteID: *createdDeployment.Data.ConfigSuiteId,
		})
		require.NoError(t, err)
		require.Equal(t, []byte(`{"key": "org1"}`), config1.Content)

		config2, err := querier.ConfigFindByActorIdAndSuiteId(ctx, dbPool, &dbsqlc.ConfigFindByActorIdAndSuiteIdParams{
			ActorId:       actor2.ID,
			ConfigSuiteID: *createdDeployment.Data.ConfigSuiteId,
		})
		require.NoError(t, err)
		require.Equal(t, []byte(`{"key": "org2"}`), config2.Content)

		config3, err := querier.ConfigFindByActorIdAndSuiteId(ctx, dbPool, &dbsqlc.ConfigFindByActorIdAndSuiteIdParams{
			ActorId:       actor3.ID,
			ConfigSuiteID: *createdDeployment.Data.ConfigSuiteId,
		})
		require.NoError(t, err)
		require.NotNil(t, config3)
		require.Equal(t, []byte("{}"), config3.Content) // New config should have empty JSON object
	})

	t.Run("Create deployment with invalid clone from", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		dbPool := testhelper.TestDB(ctx, t)
		logger := testhelper.Logger(t)

		// Create 3 actors
		fixture.InsertActor(t, ctx, dbPool, "actor1")
		fixture.InsertActor(t, ctx, dbPool, "actor2")

		request := api.AdminCreateDeploymentRequestObject{
			Body: &api.AdminCreateDeploymentJSONRequestBody{
				Name:      "test-deployment",
				User:      "test-user",
				CloneFrom: lo.ToPtr(int64(9999999)),
			},
		}

		response, err := admin.CreateDeployment(ctx, logger, dbPool, request)
		require.NoError(t, err)
		require.IsType(t, api.AdminCreateDeployment400JSONResponse{}, response)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// Close the database pool to simulate a database error
		dbPool.Close()

		request := api.AdminCreateDeploymentRequestObject{
			Body: &api.AdminCreateDeploymentJSONRequestBody{
				Name: "test-deployment",
				User: "test-user",
			},
		}

		response, err := admin.CreateDeployment(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminCreateDeployment500JSONResponse{}, response)

		errorResponse := response.(api.AdminCreateDeployment500JSONResponse)
		require.Contains(t, errorResponse.Error, "Cannot create deployment")
	})
}

func TestUpdateDeployment(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := testhelper.Logger(t)

	t.Run("Successful update", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// First, create a deployment to update
		createdDeployment, err := querier.DeploymentInsert(ctx, dbPool, &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
			Reviewers: []string{"initial-reviewer1", "initial-reviewer2"},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Update the deployment
		updateRequest := api.AdminUpdateDeploymentRequestObject{
			Id: createdDeployment.ID,
			Body: &api.AdminUpdateDeploymentJSONRequestBody{
				Name:      lo.ToPtr("updated-deployment"),
				Reviewers: &[]string{"reviewer1", "reviewer2", "reviewer3"},
			},
		}
		updateResponse, err := admin.UpdateDeployment(ctx, logger, dbPool, updateRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminUpdateDeployment200JSONResponse{}, updateResponse)

		updatedDeployment := updateResponse.(api.AdminUpdateDeployment200JSONResponse)
		require.Equal(t, "updated-deployment", updatedDeployment.Data.Name)
		require.EqualValues(t, createdDeployment.Status, updatedDeployment.Data.Status)
		require.Equal(t, createdDeployment.CreatedBy, updatedDeployment.Data.CreatedBy)
		require.Equal(t, createdDeployment.CreatedAt, updatedDeployment.Data.CreatedAt)
		require.Equal(t, createdDeployment.ApprovedBy, updatedDeployment.Data.ApprovedBy)
		require.Equal(t, createdDeployment.ApprovedAt, updatedDeployment.Data.ApprovedAt)
		require.Equal(t, createdDeployment.FinishedBy, updatedDeployment.Data.FinishedBy)
		require.Equal(t, createdDeployment.FinishedAt, updatedDeployment.Data.FinishedAt)
		require.ElementsMatch(t, []string{"reviewer1", "reviewer2", "reviewer3"}, updatedDeployment.Data.Reviewers)

		// Get deployment from DB and compare
		dbDeployment, err := querier.DeploymentGetById(ctx, dbPool, createdDeployment.ID)
		require.NoError(t, err)
		require.Equal(t, "updated-deployment", dbDeployment.Name)
		require.ElementsMatch(t, []string{"reviewer1", "reviewer2", "reviewer3"}, dbDeployment.Reviewers)
	})

	t.Run("Update without name", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// First, create a deployment to update
		createdDeployment, err := querier.DeploymentInsert(ctx, dbPool, &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Update the deployment without name
		updateRequest := api.AdminUpdateDeploymentRequestObject{
			Id: createdDeployment.ID,
			Body: &api.AdminUpdateDeploymentJSONRequestBody{
				Reviewers: &[]string{"reviewer1", "reviewer2"},
			},
		}
		updateResponse, err := admin.UpdateDeployment(ctx, logger, dbPool, updateRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminUpdateDeployment200JSONResponse{}, updateResponse)

		updatedDeployment := updateResponse.(api.AdminUpdateDeployment200JSONResponse)
		require.Equal(t, "test-deployment", updatedDeployment.Data.Name)
		require.EqualValues(t, createdDeployment.Status, updatedDeployment.Data.Status)
		require.Equal(t, createdDeployment.CreatedBy, updatedDeployment.Data.CreatedBy)
		require.Equal(t, createdDeployment.CreatedAt, updatedDeployment.Data.CreatedAt)
		require.Equal(t, createdDeployment.ApprovedBy, updatedDeployment.Data.ApprovedBy)
		require.Equal(t, createdDeployment.ApprovedAt, updatedDeployment.Data.ApprovedAt)
		require.Equal(t, createdDeployment.FinishedBy, updatedDeployment.Data.FinishedBy)
		require.Equal(t, createdDeployment.FinishedAt, updatedDeployment.Data.FinishedAt)

		// Get deployment from DB and compare
		dbDeployment, err := querier.DeploymentGetById(ctx, dbPool, createdDeployment.ID)
		require.NoError(t, err)
		require.Equal(t, "test-deployment", dbDeployment.Name)
		require.Equal(t, []string{"reviewer1", "reviewer2"}, dbDeployment.Reviewers)
	})

	t.Run("Update without reviewers", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// First, create a deployment to update
		createdDeployment, err := querier.DeploymentInsert(ctx, dbPool, &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
			Reviewers: []string{"initial-reviewer1", "initial-reviewer2"},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Update the deployment without reviewers
		updateRequest := api.AdminUpdateDeploymentRequestObject{
			Id: createdDeployment.ID,
			Body: &api.AdminUpdateDeploymentJSONRequestBody{
				Name: lo.ToPtr("updated-deployment"),
			},
		}
		updateResponse, err := admin.UpdateDeployment(ctx, logger, dbPool, updateRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminUpdateDeployment200JSONResponse{}, updateResponse)

		updatedDeployment := updateResponse.(api.AdminUpdateDeployment200JSONResponse)
		require.Equal(t, "updated-deployment", updatedDeployment.Data.Name)
		require.EqualValues(t, createdDeployment.Status, updatedDeployment.Data.Status)
		require.Equal(t, createdDeployment.CreatedBy, updatedDeployment.Data.CreatedBy)
		require.Equal(t, createdDeployment.CreatedAt, updatedDeployment.Data.CreatedAt)
		require.Equal(t, createdDeployment.ApprovedBy, updatedDeployment.Data.ApprovedBy)
		require.Equal(t, createdDeployment.ApprovedAt, updatedDeployment.Data.ApprovedAt)
		require.Equal(t, createdDeployment.FinishedBy, updatedDeployment.Data.FinishedBy)
		require.Equal(t, createdDeployment.FinishedAt, updatedDeployment.Data.FinishedAt)

		// Get deployment from DB and compare
		dbDeployment, err := querier.DeploymentGetById(ctx, dbPool, createdDeployment.ID)
		require.NoError(t, err)
		require.Equal(t, "updated-deployment", dbDeployment.Name)
		require.Equal(t, []string{"initial-reviewer1", "initial-reviewer2"}, dbDeployment.Reviewers)
	})

	t.Run("Only draft deployment can be updated", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// First, create a deployment to update
		createRequest := api.AdminCreateDeploymentRequestObject{
			Body: &api.AdminCreateDeploymentJSONRequestBody{
				Name: "test-deployment",
				User: "test-user",
			},
		}
		createResponse, err := admin.CreateDeployment(ctx, logger, dbPool, createRequest)
		require.NoError(t, err)
		createdDeployment := createResponse.(api.AdminCreateDeployment201JSONResponse)

		// Submit the deployment for review to change its status
		_, err = querier.DeploymentSubmitForReview(ctx, dbPool, int64(createdDeployment.Data.Id))
		require.NoError(t, err)

		// Attempt to update the deployment
		updateRequest := api.AdminUpdateDeploymentRequestObject{
			Id: createdDeployment.Data.Id,
			Body: &api.AdminUpdateDeploymentJSONRequestBody{
				Name: lo.ToPtr("updated-deployment"),
			},
		}
		updateResponse, err := admin.UpdateDeployment(ctx, logger, dbPool, updateRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminUpdateDeployment404Response{}, updateResponse)

		// Get deployment from DB and compare
		dbDeployment, err := querier.DeploymentGetById(ctx, dbPool, int64(createdDeployment.Data.Id))
		require.NoError(t, err)
		require.Equal(t, "test-deployment", dbDeployment.Name)
		require.EqualValues(t, "reviewing", dbDeployment.Status)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// First, create a deployment to update
		createdDeployment, err := querier.DeploymentInsert(ctx, dbPool, &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Close the database pool to simulate a database error
		dbPool.Close()

		// Attempt to update the deployment
		updateRequest := api.AdminUpdateDeploymentRequestObject{
			Id: createdDeployment.ID,
			Body: &api.AdminUpdateDeploymentJSONRequestBody{
				Name: lo.ToPtr("updated-deployment"),
			},
		}
		updateResponse, err := admin.UpdateDeployment(ctx, logger, dbPool, updateRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminUpdateDeployment500JSONResponse{}, updateResponse)

		errorResponse := updateResponse.(api.AdminUpdateDeployment500JSONResponse)
		require.Contains(t, errorResponse.Error, "Cannot update deployment")
	})
}

func TestSubmitDeployment(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := testhelper.Logger(t)

	t.Run("Submit draft deployment successfully", func(t *testing.T) {
		dbPool := testhelper.TestDB(ctx, t)

		// Create a draft deployment
		createdDeployment, err := querier.DeploymentInsertWithConfigSuite(ctx, dbPool, &dbsqlc.DeploymentInsertWithConfigSuiteParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
			Reviewers: []string{"reviewer1", "reviewer2"},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Submit the deployment for review
		submitRequest := api.AdminSubmitDeploymentRequestObject{Id: createdDeployment.ID}
		submitResponse, err := admin.SubmitDeployment(ctx, logger, dbPool, submitRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminSubmitDeployment200Response{}, submitResponse)

		// Get deployment from DB and verify status
		dbDeployment, err := querier.DeploymentGetById(ctx, dbPool, int64(createdDeployment.ID))
		require.NoError(t, err)
		require.EqualValues(t, "reviewing", dbDeployment.Status)
	})

	t.Run("Submit non-existent deployment", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// Attempt to submit a non-existent deployment
		submitRequest := api.AdminSubmitDeploymentRequestObject{
			Id: 999999, // Non-existent ID
		}
		submitResponse, err := admin.SubmitDeployment(ctx, logger, dbPool, submitRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminSubmitDeployment404Response{}, submitResponse)
	})

	t.Run("Submit non-draft deployment", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// Create a deployment with 'reviewing' status
		createdDeployment, err := querier.DeploymentInsert(ctx, dbPool, &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
			Status:    dbsqlc.NullDeploymentStatus{DeploymentStatus: "reviewing", Valid: true},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Attempt to submit the non-draft deployment
		submitRequest := api.AdminSubmitDeploymentRequestObject{Id: createdDeployment.ID}
		submitResponse, err := admin.SubmitDeployment(ctx, logger, dbPool, submitRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminSubmitDeployment400JSONResponse{}, submitResponse)

		// Verify that the status hasn't changed
		dbDeployment, err := querier.DeploymentGetById(ctx, dbPool, createdDeployment.ID)
		require.NoError(t, err)
		require.EqualValues(t, "reviewing", dbDeployment.Status)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// Create a draft deployment
		createdDeployment, err := querier.DeploymentInsert(ctx, dbPool, &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
			Status:    dbsqlc.NullDeploymentStatus{DeploymentStatus: "draft", Valid: true},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Close the database pool to simulate a database error
		dbPool.Close()

		// Attempt to submit the deployment
		submitRequest := api.AdminSubmitDeploymentRequestObject{Id: createdDeployment.ID}
		submitResponse, err := admin.SubmitDeployment(ctx, logger, dbPool, submitRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminSubmitDeployment500JSONResponse{}, submitResponse)
	})
}

func TestRejectDeployment(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := testhelper.Logger(t)

	setupRejectDeploymentTest := func(t *testing.T, statusValue string) (*pgxpool.Pool, *dbsqlc.Deployment) {
		t.Helper()
		dbPool := testhelper.TestDB(ctx, t)

		var status dbsqlc.NullDeploymentStatus
		status.Scan(statusValue)

		createdDeployment, err := querier.DeploymentInsert(ctx, dbPool, &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
			Status:    status,
			Reviewers: []string{"reviewer1", "reviewer2"},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		return dbPool, createdDeployment
	}

	t.Run("Successfully reject draft deployment", func(t *testing.T) {
		t.Parallel()
		dbPool, createdDeployment := setupRejectDeploymentTest(t, "reviewing")
		rejectRequest := api.AdminRejectDeploymentRequestObject{
			Id: createdDeployment.ID,
			Body: &api.AdminRejectDeploymentJSONRequestBody{
				User:   "reviewer1",
				Reason: "Test rejection reason",
			},
		}
		rejectResponse, err := admin.RejectDeployment(ctx, logger, dbPool, rejectRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminRejectDeployment201Response{}, rejectResponse)

		// Verify that the status has changed to rejected
		dbDeployment, err := querier.DeploymentGetById(ctx, dbPool, int64(createdDeployment.ID))
		require.NoError(t, err)
		require.EqualValues(t, "rejected", dbDeployment.Status)
		require.NotNil(t, dbDeployment.FinishedAt)
		require.Equal(t, "reviewer1", *dbDeployment.FinishedBy)
		require.Contains(t, string(dbDeployment.Notes), "Test rejection reason")
	})

	t.Run("Fail to reject non-draft deployment", func(t *testing.T) {
		t.Parallel()
		dbPool, createdDeployment := setupRejectDeploymentTest(t, "draft")

		rejectRequest := api.AdminRejectDeploymentRequestObject{
			Id: createdDeployment.ID,
			Body: &api.AdminRejectDeploymentJSONRequestBody{
				User:   "reviewer1",
				Reason: "Test rejection reason",
			},
		}
		rejectResponse, err := admin.RejectDeployment(ctx, logger, dbPool, rejectRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminRejectDeployment404Response{}, rejectResponse)

		// Verify that the status hasn't changed
		dbDeployment, err := querier.DeploymentGetById(ctx, dbPool, createdDeployment.ID)
		require.NoError(t, err)
		require.EqualValues(t, "draft", dbDeployment.Status)
	})

	t.Run("Fail to reject with non-reviewer user", func(t *testing.T) {
		t.Parallel()
		dbPool, createdDeployment := setupRejectDeploymentTest(t, "reviewing")

		rejectRequest := api.AdminRejectDeploymentRequestObject{
			Id: createdDeployment.ID,
			Body: &api.AdminRejectDeploymentJSONRequestBody{
				User:   "non-reviewer",
				Reason: "Test rejection reason",
			},
		}
		rejectResponse, err := admin.RejectDeployment(ctx, logger, dbPool, rejectRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminRejectDeployment404Response{}, rejectResponse)

		// Verify that the status hasn't changed
		dbDeployment, err := querier.DeploymentGetById(ctx, dbPool, createdDeployment.ID)
		require.NoError(t, err)
		require.EqualValues(t, "reviewing", dbDeployment.Status)
	})

	t.Run("Missing user in request body", func(t *testing.T) {
		t.Parallel()
		dbPool, createdDeployment := setupRejectDeploymentTest(t, "reviewing")

		rejectRequest := api.AdminRejectDeploymentRequestObject{
			Id:   createdDeployment.ID,
			Body: &api.AdminRejectDeploymentJSONRequestBody{},
		}
		rejectResponse, err := admin.RejectDeployment(ctx, logger, dbPool, rejectRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminRejectDeployment400JSONResponse{}, rejectResponse)
	})

	t.Run("Missing reason in request body", func(t *testing.T) {
		t.Parallel()
		dbPool, createdDeployment := setupRejectDeploymentTest(t, "reviewing")

		rejectRequest := api.AdminRejectDeploymentRequestObject{
			Id: createdDeployment.ID,
			Body: &api.AdminRejectDeploymentJSONRequestBody{
				User: "reviewer1",
			},
		}
		rejectResponse, err := admin.RejectDeployment(ctx, logger, dbPool, rejectRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminRejectDeployment201Response{}, rejectResponse)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool, createdDeployment := setupRejectDeploymentTest(t, "reviewing")

		// Close the database pool to simulate a database error
		dbPool.Close()

		rejectRequest := api.AdminRejectDeploymentRequestObject{
			Id: createdDeployment.ID,
			Body: &api.AdminRejectDeploymentJSONRequestBody{
				User:   "reviewer1",
				Reason: "Test rejection reason",
			},
		}
		rejectResponse, err := admin.RejectDeployment(ctx, logger, dbPool, rejectRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminRejectDeployment500JSONResponse{}, rejectResponse)

		errorResponse := rejectResponse.(api.AdminRejectDeployment500JSONResponse)
		require.Contains(t, errorResponse.Error, "Cannot reject deployment")
	})
}

func TestPublishDeployment(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := testhelper.Logger(t)

	setupDeploymentTest := func(t *testing.T, status string, withMigrations bool) (*pgxpool.Pool, *dbsqlc.Deployment, *dbsqlc.Actor, *dbsqlc.Actor, *testhelper.MockSuiteStore) {
		t.Helper()
		dbPool := testhelper.TestDB(ctx, t)
		suiteStore := testhelper.NewMockSuiteStore()

		// Create two actors
		actor1 := fixture.InsertActor2(t, ctx, dbPool, "actor1", "agent", true, true, true, withMigrations)
		actor2 := fixture.InsertActor2(t, ctx, dbPool, "actor2", "agent", true, false, true, withMigrations)

		// Create a existing deployed deployment and a config suite
		if status != "deployed" {
			existingDeployment, err := querier.DeploymentInsertWithConfigSuite(ctx, dbPool, &dbsqlc.DeploymentInsertWithConfigSuiteParams{
				CreatedBy: "test-user",
				Name:      "existing-deployed-deployment",
				Reviewers: []string{"reviewer1", "reviewer2"},
			})
			require.NoError(t, err)
			require.NotNil(t, existingDeployment)
			// update deployment status
			_, err = dbPool.Exec(ctx, "UPDATE deployments SET status = $1 WHERE id = $2", "deployed", existingDeployment.ID)
			require.NoError(t, err)
		}

		// Create a deployment and a config suite
		createdDeployment, err := querier.DeploymentInsertWithConfigSuite(ctx, dbPool, &dbsqlc.DeploymentInsertWithConfigSuiteParams{
			CreatedBy: "test-user",
			Name:      "test-deployment",
			Reviewers: []string{"reviewer1", "reviewer2"},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// update deployment status
		_, err = dbPool.Exec(ctx, "UPDATE deployments SET status = $1 WHERE id = $2", status, createdDeployment.ID)
		require.NoError(t, err)

		// Create configs for each actor
		actor1EnvVars := map[string]string{
			"key":                    "value1",
			"KUBE_DOCKER_IMAGE":      "actor1-image:latest",
			"KUBE_IMAGE_PULL_SECRET": "custom-image-pull-secret",
			"KUBE_REPLICAS":          "2",
			"KUBE_MEMORY_REQUEST":    "256Mi",
			"KUBE_MEMORY_LIMIT":      "512Mi",
		}
		actor2EnvVars := map[string]string{
			"key":                 "value2",
			"KUBE_DOCKER_IMAGE":   "actor2-image:latest",
			"KUBE_REPLICAS":       "3",
			"KUBE_MEMORY_REQUEST": "256Mi",
			"KUBE_MEMORY_LIMIT":   "512Mi",
		}

		if withMigrations {
			actor1EnvVars["KUBE_MIGRATE_DOCKER_IMAGE"] = "actor1-image:latest"
			actor1EnvVars["KUBE_MIGRATE_IMAGE_PULL_SECRET"] = "custom-image-pull-secret"
			actor1EnvVars["KUBE_MIGRATE_COMMAND"] = "migrate.sh up"
			actor1EnvVars["KUBE_MIGRATE_MEMORY_REQUEST"] = "266Mi"
			actor1EnvVars["KUBE_MIGRATE_MEMORY_LIMIT"] = "522Mi"
			actor2EnvVars["KUBE_MIGRATE_DOCKER_IMAGE"] = "actor2-image:latest"
			actor2EnvVars["KUBE_MIGRATE_IMAGE_PULL_SECRET"] = "custom-image-pull-secret"
			actor2EnvVars["KUBE_MIGRATE_COMMAND"] = "migrate.sh up"
			actor2EnvVars["KUBE_MIGRATE_MEMORY_REQUEST"] = "268Mi"
			actor2EnvVars["KUBE_MIGRATE_MEMORY_LIMIT"] = "532Mi"
		}

		config1 := fixture.InsertConfig2(t, ctx, dbPool, actor1.ID, createdDeployment.ConfigSuiteID, "test-user", actor1EnvVars)
		config2 := fixture.InsertConfig2(t, ctx, dbPool, actor2.ID, createdDeployment.ConfigSuiteID, "test-user", actor2EnvVars)
		require.NotNil(t, config1)
		require.NotNil(t, config2)

		deployment := &dbsqlc.Deployment{
			ID:            createdDeployment.ID,
			Name:          createdDeployment.Name,
			Status:        createdDeployment.Status,
			Reviewers:     createdDeployment.Reviewers,
			ConfigSuiteID: createdDeployment.ConfigSuiteID,
			CreatedBy:     createdDeployment.CreatedBy,
			CreatedAt:     createdDeployment.CreatedAt,
		}
		return dbPool, deployment, actor1, actor2, suiteStore
	}

	t.Run("Successfully publish reviewing deployment", func(t *testing.T) {
		t.Parallel()
		dbPool, createdDeployment, _, _, suiteStore := setupDeploymentTest(t, "reviewing", false)

		// Publish the deployment
		publishRequest := api.AdminPublishDeploymentRequestObject{
			Id:   createdDeployment.ID,
			Body: &api.AdminPublishDeploymentJSONRequestBody{User: "admin"},
		}
		mockController := &mockK8sController{}
		publishResponse, err := admin.PublishDeployment(ctx, logger, dbPool, suiteStore, mockController, publishRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminPublishDeployment201Response{}, publishResponse)

		// Verify that the status has changed to 'deployed'
		dbDeployment, err := querier.DeploymentGetById(ctx, dbPool, int64(createdDeployment.ID))
		require.NoError(t, err)
		require.EqualValues(t, "deploying", dbDeployment.Status)

		// wait for deploying to finish
		require.Eventually(t, func() bool {
			dbDeployment, err = querier.DeploymentGetById(ctx, dbPool, int64(createdDeployment.ID))
			require.NoError(t, err)
			return dbDeployment.Status == "deployed" || dbDeployment.Status == "failed"
		}, 1*time.Second, 50*time.Millisecond)

		require.EqualValues(t, "deployed", dbDeployment.Status)

		// verify config suites was activated
		dbSuite, err := querier.ConfigSuiteGetById(ctx, dbPool, *dbDeployment.ConfigSuiteID)
		require.NoError(t, err)
		require.EqualValues(t, true, dbSuite.Active)
		require.NotNil(t, dbSuite.DeployedAt)
		require.Equal(t, "admin", *dbSuite.UpdatedBy)

		// Verify that UpdateDeploymentSet was called
		require.Len(t, mockController.updatedDeploymentSets, 1)
		updatedSet := mockController.updatedDeploymentSets[0]
		require.Len(t, updatedSet, 1) // We expect 1 actor config

		// Verify that the suite was published to the suite store
		publishedSuites, err := suiteStore.ReadSuites(ctx)
		require.NoError(t, err)
		require.NotNil(t, publishedSuites)
		require.Len(t, publishedSuites, 1) // We expect 2 actor configs

		// Verify the content of the published suite
		publishedSuite := publishedSuites[0]
		for _, actorConfig := range publishedSuite.ConfigSuites {
			switch actorConfig.ActorName {
			case "actor1":
				require.Equal(t, map[string]string{
					"key":                    "value1",
					"KUBE_DOCKER_IMAGE":      "actor1-image:latest",
					"KUBE_IMAGE_PULL_SECRET": "custom-image-pull-secret",
					"KUBE_REPLICAS":          "2",
					"KUBE_MEMORY_REQUEST":    "256Mi",
					"KUBE_MEMORY_LIMIT":      "512Mi",
				}, actorConfig.Configs)
			case "actor2":
				require.Equal(t, map[string]string{
					"key":                 "value2",
					"KUBE_DOCKER_IMAGE":   "actor2-image:latest",
					"KUBE_REPLICAS":       "3",
					"KUBE_MEMORY_REQUEST": "256Mi",
					"KUBE_MEMORY_LIMIT":   "512Mi",
				}, actorConfig.Configs)
			default:
				t.Fatalf("Unexpected actor name: %s", actorConfig.ActorName)
			}
		}
	})

	t.Run("Successfully publish draft deployment", func(t *testing.T) {
		t.Parallel()
		dbPool, createdDeployment, _, _, suiteStore := setupDeploymentTest(t, "draft", false)

		// Publish the deployment
		publishRequest := api.AdminPublishDeploymentRequestObject{
			Id:   createdDeployment.ID,
			Body: &api.AdminPublishDeploymentJSONRequestBody{User: "admin"},
		}
		mockController := &mockK8sController{}
		publishResponse, err := admin.PublishDeployment(ctx, logger, dbPool, suiteStore, mockController, publishRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminPublishDeployment201Response{}, publishResponse)

		// Verify that the status has changed to 'deployed'
		dbDeployment, err := querier.DeploymentGetById(ctx, dbPool, int64(createdDeployment.ID))
		require.NoError(t, err)
		require.EqualValues(t, "deploying", dbDeployment.Status)

		// Verify that the deploying_at field is set
		require.NotNil(t, dbDeployment.DeployingAt)

		require.Eventually(t, func() bool {
			dbDeployment, err = querier.DeploymentGetById(ctx, dbPool, int64(createdDeployment.ID))
			require.NoError(t, err)
			return dbDeployment.Status == "deployed" || dbDeployment.Status == "failed"
		}, 1*time.Second, 50*time.Millisecond)

		require.EqualValues(t, "deployed", dbDeployment.Status)

		// verify config suites was activated
		dbSuite, err := querier.ConfigSuiteGetById(ctx, dbPool, *dbDeployment.ConfigSuiteID)
		require.NoError(t, err)
		require.EqualValues(t, true, dbSuite.Active)
		require.NotNil(t, dbSuite.DeployedAt)
		require.Equal(t, "admin", *dbSuite.UpdatedBy)

		// Verify that UpdateDeploymentSet was called
		require.Len(t, mockController.updatedDeploymentSets, 1)
		updatedSet := mockController.updatedDeploymentSets[0]
		require.Len(t, updatedSet, 1) // We expect 1 actor config

		// Verify that the suite was published to the suite store
		publishedSuites, err := suiteStore.ReadSuites(ctx)
		require.NoError(t, err)
		require.NotNil(t, publishedSuites)
		require.Len(t, publishedSuites, 1) // We expect 2 actor configs
	})

	t.Run("Successfully publish and update k8s deployment", func(t *testing.T) {
		t.Parallel()
		dbPool, createdDeployment, _, _, suiteStore := setupDeploymentTest(t, "reviewing", true)
		mockController := &mockK8sController{}

		// Publish the deployment
		publishRequest := api.AdminPublishDeploymentRequestObject{
			Id:   createdDeployment.ID,
			Body: &api.AdminPublishDeploymentJSONRequestBody{User: "admin"},
		}
		publishResponse, err := admin.PublishDeployment(ctx, logger, dbPool, suiteStore, mockController, publishRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminPublishDeployment201Response{}, publishResponse)

		// Verify that the status has changed to 'deployed'
		dbDeployment, err := querier.DeploymentGetById(ctx, dbPool, int64(createdDeployment.ID))
		require.NoError(t, err)
		require.EqualValues(t, "deploying", dbDeployment.Status)

		require.Eventually(t, func() bool {
			dbDeployment, err = querier.DeploymentGetById(ctx, dbPool, int64(createdDeployment.ID))
			require.NoError(t, err)
			return dbDeployment.Status == "deployed" || dbDeployment.Status == "failed"
		}, 1*time.Second, 50*time.Millisecond)

		require.EqualValues(t, "deployed", dbDeployment.Status)

		// verify migrations
		require.EqualValues(t, 1, len(mockController.migrationParams))
		require.EqualValues(t, k8s.MigrationParams{
			Serial:           createdDeployment.ID,
			Name:             "maos-actor1",
			Image:            "actor1-image:latest",
			Command:          []string{"migrate.sh", "up"},
			ImagePullSecrets: "custom-image-pull-secret",
			EnvVars:          map[string]string{"key": "value1"},
			MemoryRequest:    "266Mi",
			MemoryLimit:      "522Mi",
		}, mockController.migrationParams[0][0])
		require.EqualValues(t, k8s.MigrationParams{
			Serial:           createdDeployment.ID,
			Name:             "maos-actor2",
			Image:            "actor2-image:latest",
			Command:          []string{"migrate.sh", "up"},
			ImagePullSecrets: "custom-image-pull-secret",
			EnvVars:          map[string]string{"key": "value2"},
			MemoryRequest:    "268Mi",
			MemoryLimit:      "532Mi",
		}, mockController.migrationParams[0][1])

		// Verify that UpdateDeploymentSet was called
		require.Len(t, mockController.updatedDeploymentSets, 1)
		updatedSet := mockController.updatedDeploymentSets[0]
		require.Len(t, updatedSet, 1) // We expect 1 actor config

		// Verify the content of the updated deployment set
		deployment := updatedSet[0]
		require.Equal(t, "maos-actor1", deployment.Name)
		require.Equal(t, map[string]string{"key": "value1"}, deployment.EnvVars)
		require.NotEmpty(t, deployment.APIKey)
		require.Equal(t, "actor1-image:latest", deployment.Image)
		require.Equal(t, int32(2), deployment.Replicas)
		require.Equal(t, "256Mi", deployment.MemoryRequest)
		require.Equal(t, "512Mi", deployment.MemoryLimit)
	})

	t.Run("Attempt to publish already deployed deployment", func(t *testing.T) {
		t.Parallel()
		dbPool, createdDeployment, _, _, suiteStore := setupDeploymentTest(t, "deployed", false)

		// Attempt to publish the already deployed deployment
		publishRequest := api.AdminPublishDeploymentRequestObject{
			Id:   createdDeployment.ID,
			Body: &api.AdminPublishDeploymentJSONRequestBody{User: "admin"},
		}
		mockController := &mockK8sController{}
		publishResponse, err := admin.PublishDeployment(ctx, logger, dbPool, suiteStore, mockController, publishRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminPublishDeployment400JSONResponse{}, publishResponse)

		// Verify that the status hasn't changed
		dbDeployment, err := querier.DeploymentGetById(ctx, dbPool, int64(createdDeployment.ID))
		require.NoError(t, err)
		require.EqualValues(t, "deployed", dbDeployment.Status)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool, createdDeployment, _, _, suiteStore := setupDeploymentTest(t, "reviewing", false)

		// Close the database pool to simulate a database error
		dbPool.Close()

		// Attempt to publish the deployment
		publishRequest := api.AdminPublishDeploymentRequestObject{
			Id:   createdDeployment.ID,
			Body: &api.AdminPublishDeploymentJSONRequestBody{User: "admin"},
		}
		mockController := &mockK8sController{}
		publishResponse, err := admin.PublishDeployment(ctx, logger, dbPool, suiteStore, mockController, publishRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminPublishDeployment500JSONResponse{}, publishResponse)

		errorResponse := publishResponse.(api.AdminPublishDeployment500JSONResponse)
		require.Contains(t, errorResponse.Error, "Cannot publish deployment")
	})
}

func TestDeleteDeployment(t *testing.T) {
	ctx := context.Background()
	logger := testhelper.Logger(t)

	t.Run("Successfully delete draft deployment", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// Create a draft deployment
		createdDeployment, err := querier.DeploymentInsert(ctx, dbPool, &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
			Status:    dbsqlc.NullDeploymentStatus{DeploymentStatus: "draft", Valid: true},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Delete the deployment
		deleteRequest := api.AdminDeleteDeploymentRequestObject{Id: createdDeployment.ID}
		deleteResponse, err := admin.DeleteDeployment(ctx, logger, dbPool, deleteRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminDeleteDeployment200Response{}, deleteResponse)

		// Verify that the deployment no longer exists
		_, err = querier.DeploymentGetById(ctx, dbPool, createdDeployment.ID)
		require.Error(t, err)
		require.Equal(t, pgx.ErrNoRows, err)
	})

	t.Run("Attempt to delete non-draft deployment", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// Create a deployment with 'reviewing' status
		createdDeployment, err := querier.DeploymentInsert(ctx, dbPool, &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
			Status:    dbsqlc.NullDeploymentStatus{DeploymentStatus: "reviewing", Valid: true},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		rows, err := dbPool.Query(ctx, "SELECT id,name,status FROM deployments")
		require.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var id int64
			var name, status string
			err := rows.Scan(&id, &name, &status)
			require.NoError(t, err)
			t.Logf("Deployment: ID=%d, Name=%s, Status=%s", id, name, status)
		}
		require.NoError(t, rows.Err())

		// Attempt to delete the non-draft deployment
		deleteRequest := api.AdminDeleteDeploymentRequestObject{Id: createdDeployment.ID}
		deleteResponse, err := admin.DeleteDeployment(ctx, logger, dbPool, deleteRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminDeleteDeployment404Response{}, deleteResponse)

		// Verify that the deployment still exists
		dbDeployment, err := querier.DeploymentGetById(ctx, dbPool, createdDeployment.ID)
		require.NoError(t, err)
		require.EqualValues(t, "reviewing", dbDeployment.Status)
	})

	t.Run("Attempt to delete non-existent deployment", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// Attempt to delete a non-existent deployment
		deleteRequest := api.AdminDeleteDeploymentRequestObject{Id: 9999}
		deleteResponse, err := admin.DeleteDeployment(ctx, logger, dbPool, deleteRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminDeleteDeployment404Response{}, deleteResponse)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// Create a draft deployment
		createdDeployment, err := querier.DeploymentInsert(ctx, dbPool, &dbsqlc.DeploymentInsertParams{
			Name:      "test-deployment",
			CreatedBy: "test-user",
			Status:    dbsqlc.NullDeploymentStatus{DeploymentStatus: "draft", Valid: true},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Close the database pool to simulate a database error
		dbPool.Close()

		// Attempt to delete the deployment
		deleteRequest := api.AdminDeleteDeploymentRequestObject{Id: createdDeployment.ID}
		deleteResponse, err := admin.DeleteDeployment(ctx, logger, dbPool, deleteRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminDeleteDeployment500JSONResponse{}, deleteResponse)

		errorResponse := deleteResponse.(api.AdminDeleteDeployment500JSONResponse)
		require.Contains(t, errorResponse.Error, "Cannot delete deployment")
	})
}

func TestRestartDeployment(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := testhelper.Logger(t)

	setupRestartDeploymentTest := func(t *testing.T, status string) (*pgxpool.Pool, int64) {
		t.Helper()
		dbPool := testhelper.TestDB(ctx, t)

		// Create two actors
		actor1 := fixture.InsertActor2(t, ctx, dbPool, "actor1", "agent", true, true, true, false)
		actor2 := fixture.InsertActor2(t, ctx, dbPool, "actor2", "agent", true, false, true, false)

		// Create a deployment
		createdDeployment, err := querier.DeploymentInsertWithConfigSuite(ctx, dbPool, &dbsqlc.DeploymentInsertWithConfigSuiteParams{
			CreatedBy: "test-user",
			Name:      "test-deployment",
			Reviewers: []string{"reviewer1", "reviewer2"},
		})
		require.NoError(t, err)
		require.NotNil(t, createdDeployment)

		// Update deployment status
		_, err = dbPool.Exec(ctx, "UPDATE deployments SET status = $1 WHERE id = $2", status, createdDeployment.ID)
		require.NoError(t, err)

		// Create configs for each actor
		config1 := fixture.InsertConfig2(t, ctx, dbPool, actor1.ID, createdDeployment.ConfigSuiteID, "test-user", map[string]string{
			"key":                 "value1",
			"KUBE_DOCKER_IMAGE":   "actor1-image:latest",
			"KUBE_REPLICAS":       "2",
			"KUBE_MEMORY_REQUEST": "256Mi",
			"KUBE_MEMORY_LIMIT":   "512Mi",
		})
		config2 := fixture.InsertConfig2(t, ctx, dbPool, actor2.ID, createdDeployment.ConfigSuiteID, "test-user", map[string]string{
			"key":                 "value2",
			"KUBE_DOCKER_IMAGE":   "actor2-image:latest",
			"KUBE_REPLICAS":       "3",
			"KUBE_MEMORY_REQUEST": "256Mi",
			"KUBE_MEMORY_LIMIT":   "512Mi",
		})
		require.NotNil(t, config1)
		require.NotNil(t, config2)

		return dbPool, createdDeployment.ID
	}

	t.Run("Successfully restart deployed deployment", func(t *testing.T) {
		t.Parallel()
		dbPool, createdDeploymentId := setupRestartDeploymentTest(t, "deployed")

		mockController := &mockK8sController{}
		restartRequest := api.AdminRestartDeploymentRequestObject{
			Id:   createdDeploymentId,
			Body: &api.AdminRestartDeploymentJSONRequestBody{User: "admin"},
		}
		restartResponse, err := admin.RestartDeployment(ctx, logger, dbPool, mockController, restartRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminRestartDeployment201Response{}, restartResponse)

		// Verify that the status remains "deployed"
		dbDeployment, err := querier.DeploymentGetById(ctx, dbPool, createdDeploymentId)
		require.NoError(t, err)
		require.EqualValues(t, "deployed", dbDeployment.Status)

		// Verify that UpdateDeploymentSet was called
		require.Len(t, mockController.updatedDeploymentSets, 1)
		updatedSet := mockController.updatedDeploymentSets[0]
		require.Len(t, updatedSet, 1) // We expect 1 actor config
	})

	t.Run("Attempt to restart non-deployed deployment", func(t *testing.T) {
		t.Parallel()
		dbPool, createdDeploymentId := setupRestartDeploymentTest(t, "reviewing")

		mockController := &mockK8sController{}
		restartRequest := api.AdminRestartDeploymentRequestObject{
			Id:   createdDeploymentId,
			Body: &api.AdminRestartDeploymentJSONRequestBody{User: "admin"},
		}
		restartResponse, err := admin.RestartDeployment(ctx, logger, dbPool, mockController, restartRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminRestartDeployment404Response{}, restartResponse)
	})

	t.Run("Attempt to restart non-existent deployment", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		mockController := &mockK8sController{}
		restartRequest := api.AdminRestartDeploymentRequestObject{
			Id:   9999,
			Body: &api.AdminRestartDeploymentJSONRequestBody{User: "admin"},
		}
		restartResponse, err := admin.RestartDeployment(ctx, logger, dbPool, mockController, restartRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminRestartDeployment404Response{}, restartResponse)
	})

	t.Run("Missing user in request body", func(t *testing.T) {
		t.Parallel()
		dbPool, createdDeploymentId := setupRestartDeploymentTest(t, "deployed")

		mockController := &mockK8sController{}
		restartRequest := api.AdminRestartDeploymentRequestObject{
			Id:   createdDeploymentId,
			Body: &api.AdminRestartDeploymentJSONRequestBody{},
		}
		restartResponse, err := admin.RestartDeployment(ctx, logger, dbPool, mockController, restartRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminRestartDeployment401Response{}, restartResponse)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool, createdDeploymentId := setupRestartDeploymentTest(t, "deployed")

		// Close the database pool to simulate a database error
		dbPool.Close()

		mockController := &mockK8sController{}
		restartRequest := api.AdminRestartDeploymentRequestObject{
			Id:   createdDeploymentId,
			Body: &api.AdminRestartDeploymentJSONRequestBody{User: "admin"},
		}
		restartResponse, err := admin.RestartDeployment(ctx, logger, dbPool, mockController, restartRequest)
		require.NoError(t, err)
		require.IsType(t, api.AdminRestartDeployment500JSONResponse{}, restartResponse)

		errorResponse := restartResponse.(api.AdminRestartDeployment500JSONResponse)
		require.Contains(t, errorResponse.Error, "Cannot restart deployment")
	})
}
