package apitest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
	"gitlab.com/navyx/ai/maos/maos-core/k8s"
)

func TestAdminListDeploymentsEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	setupTestData := func(t *testing.T, ctx context.Context, accessor dbaccess.Accessor) {
		fixture.InsertDeployment(t, ctx, accessor.Source(), "deployment1", []string{"user1"})
		fixture.InsertDeployment(t, ctx, accessor.Source(), "deployment2", []string{"user2"})
		actor1 := fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
		actor2 := fixture.InsertActor(t, ctx, accessor.Source(), "actor2")
		fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", actor1.ID, 0, []string{"admin"})
		fixture.InsertToken(t, ctx, accessor.Source(), "user-token", actor2.ID, 0, []string{"user"})
	}

	assertResponseMatches := func(t *testing.T, expected, actual api.AdminListDeployments200JSONResponse) {
		require.Equal(t, len(expected.Data), len(actual.Data))
		require.Equal(t, expected.Meta.Total, actual.Meta.Total)
		require.Equal(t, expected.Meta.Page, actual.Meta.Page)
		require.Equal(t, expected.Meta.PageSize, actual.Meta.PageSize)

		for i, expectedDeployment := range expected.Data {
			require.Equal(t, expectedDeployment.Name, actual.Data[i].Name)
			require.NotZero(t, actual.Data[i].Id)
			require.NotZero(t, actual.Data[i].CreatedAt)
		}
	}

	t.Run("Valid admin token", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		setupTestData(t, ctx, accessor)

		resp, resBody := GetHttp(t, server.URL+"/v1/admin/deployments?page=1&page_size=15", "admin-token")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.AdminListDeployments200JSONResponse
		err := json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)

		expectedResponse := api.AdminListDeployments200JSONResponse{}
		err = json.Unmarshal([]byte(`{"data":[{"id":2,"name":"deployment2"},{"id":1,"name":"deployment1"}],"meta":{"total":2,"page":1,"page_size":15}}`), &expectedResponse)
		require.NoError(t, err)

		assertResponseMatches(t, expectedResponse, response)
	})

	t.Run("Filter by id list", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		// Insert deployments and get their IDs
		deployment1 := fixture.InsertDeployment(t, ctx, accessor.Source(), "deployment1", []string{"user1"})
		deployment2 := fixture.InsertDeployment(t, ctx, accessor.Source(), "deployment2", []string{"user2"})
		fixture.InsertDeployment(t, ctx, accessor.Source(), "deployment3", []string{"user3"})

		actor := fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
		fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", actor.ID, 0, []string{"admin"})

		// Construct URL with id list
		url := fmt.Sprintf("%s/v1/admin/deployments?id=%d&id=%d", server.URL, deployment1.ID, deployment2.ID)

		resp, resBody := GetHttp(t, url, "admin-token")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.AdminListDeployments200JSONResponse
		err := json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)

		require.Len(t, response.Data, 2)
		require.Equal(t, int64(2), response.Meta.Total)

		// Check that only the requested deployments are returned
		deploymentIDs := []int64{response.Data[0].Id, response.Data[1].Id}
		require.Contains(t, deploymentIDs, deployment1.ID)
		require.Contains(t, deploymentIDs, deployment2.ID)

		// Check that deployment names match
		deploymentNames := []string{response.Data[0].Name, response.Data[1].Name}
		require.Contains(t, deploymentNames, "deployment1")
		require.Contains(t, deploymentNames, "deployment2")
	})

	t.Run("Valid admin token with pagination", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		setupTestData(t, ctx, accessor)

		resp, resBody := GetHttp(t, server.URL+"/v1/admin/deployments?page=1&page_size=1", "admin-token")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.AdminListDeployments200JSONResponse
		err := json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)

		expectedResponse := api.AdminListDeployments200JSONResponse{}
		err = json.Unmarshal([]byte(`{"data":[{"id":2,"name":"deployment2"}],"meta":{"total":2,"page":1,"page_size":1}}`), &expectedResponse)
		require.NoError(t, err)

		assertResponseMatches(t, expectedResponse, response)
	})

	t.Run("Non-admin token", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		setupTestData(t, ctx, accessor)

		resp, _ := GetHttp(t, server.URL+"/v1/admin/deployments", "user-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid token", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		setupTestData(t, ctx, accessor)

		resp, _ := GetHttp(t, server.URL+"/v1/admin/deployments", "invalid_token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestAdminGetDeploymentEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name           string
		deploymentID   int64
		token          string
		setupFunc      func(context.Context, *testing.T, dbaccess.Accessor) int64
		expectedStatus int
		expectedBody   string
	}{
		{
			name:         "Valid admin token and existing deployment",
			deploymentID: 1,
			token:        "admin-token",
			setupFunc: func(ctx context.Context, t *testing.T, accessor dbaccess.Accessor) int64 {
				deployment := fixture.InsertDeployment(t, ctx, accessor.Source(), "test_deployment", []string{"admin"})
				return deployment.ID
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":1,"name":"test_deployment","status":"draft","created_by":"tester"}`,
		},
		{
			name:           "Valid admin token but non-existent deployment",
			deploymentID:   999,
			token:          "admin-token",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Non-admin token",
			deploymentID:   1,
			token:          "user-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token",
			deploymentID:   1,
			token:          "invalid_token",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, _ := SetupHttpTestWithDb(t, ctx)

			actor1 := fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
			actor2 := fixture.InsertActor(t, ctx, accessor.Source(), "actor2")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", actor1.ID, 0, []string{"admin"})
			fixture.InsertToken(t, ctx, accessor.Source(), "user-token", actor2.ID, 0, []string{"user"})

			if tt.setupFunc != nil {
				tt.deploymentID = tt.setupFunc(ctx, t, accessor)
			}

			resp, resBody := GetHttp(t, fmt.Sprintf("%s/v1/admin/deployments/%d", server.URL, tt.deploymentID), tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			switch tt.expectedStatus {
			case http.StatusOK:
				var response api.AdminGetDeployment200JSONResponse
				err := json.Unmarshal([]byte(resBody), &response)
				require.NoError(t, err)

				expectedResponse := api.AdminGetDeployment200JSONResponse{}
				err = json.Unmarshal([]byte(tt.expectedBody), &expectedResponse)
				require.NoError(t, err)

				require.Equal(t, expectedResponse.Id, response.Id)
				require.Equal(t, expectedResponse.Name, response.Name)
				require.Equal(t, expectedResponse.Status, response.Status)
				require.Equal(t, expectedResponse.CreatedBy, response.CreatedBy)
				require.NotZero(t, response.CreatedAt)

			case http.StatusNotFound:
				require.Empty(t, resBody)

			case http.StatusUnauthorized:

			default:
				t.Fatalf("Unexpected status code: %d", tt.expectedStatus)
			}
		})
	}
}

func TestAdminCreateDeploymentEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name           string
		body           string
		token          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid admin token",
			body:           `{"name":"new_deployment","user":"admin"}`,
			token:          "admin-token",
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"id":1,"name":"new_deployment","status":"draft","created_by":"admin"}`,
		},
		{
			name:           "Invalid body",
			body:           `{"invalid_json"}`,
			token:          "admin-token",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid character",
		},
		{
			name:           "Missing required fields",
			body:           `{"name":"new_deployment"}`,
			token:          "admin-token",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing required field",
		},
		{
			name:           "Non-admin token",
			body:           `{"name":"new_deployment","user":"user"}`,
			token:          "user-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid token",
			body:           `{"name":"new_deployment","user":"admin"}`,
			token:          "invalid_token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, _ := SetupHttpTestWithDb(t, ctx)

			actor1 := fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
			actor2 := fixture.InsertActor(t, ctx, accessor.Source(), "actor2")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", actor1.ID, 0, []string{"admin"})
			fixture.InsertToken(t, ctx, accessor.Source(), "user-token", actor2.ID, 0, []string{"user"})

			resp, resBody := PostHttp(t, server.URL+"/v1/admin/deployments", tt.body, tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusCreated {
				var response api.AdminCreateDeployment201JSONResponse
				err := json.Unmarshal([]byte(resBody), &response)
				require.NoError(t, err)

				require.NotZero(t, response.Data.Id)
				require.Equal(t, "new_deployment", response.Data.Name)
				require.Equal(t, api.DeploymentStatusDraft, response.Data.Status)
				require.Equal(t, "admin", response.Data.CreatedBy)
				require.NotZero(t, response.Data.CreatedAt)

				// Verify the deployment was actually created in the database
				deployments, err := accessor.Querier().DeploymentListPaginated(ctx, accessor.Source(), &dbsqlc.DeploymentListPaginatedParams{})
				require.NoError(t, err)
				require.Len(t, deployments, 1)
				createdDeployment := deployments[0]
				require.Equal(t, response.Data.Id, createdDeployment.ID)
				require.Equal(t, "new_deployment", createdDeployment.Name)
				require.Equal(t, "admin", createdDeployment.CreatedBy)
			} else {
				if tt.expectedBody != "" {
					resJson := testhelper.JsonToMap(t, resBody)
					require.Contains(t, resJson, "error")
					require.Contains(t, resJson["error"], tt.expectedBody)
				}
			}
		})
	}
}

func TestAdminUpdateDeployment(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		deploymentID   int64
		body           string
		token          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid update",
			deploymentID:   1,
			body:           `{"name":"updated_deployment","status":"in_review"}`,
			token:          "admin-token",
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "Invalid token",
			deploymentID:   1,
			body:           `{"name":"updated_deployment"}`,
			token:          "invalid_token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Non-existent deployment",
			deploymentID:   9999,
			body:           `{"name":"updated_deployment"}`,
			token:          "admin-token",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid reviewers",
			deploymentID:   1,
			body:           `{"reviewers":"reviewer,reviewer2"}`,
			token:          "admin-token",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "cannot unmarshal string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, _ := SetupHttpTestWithDb(t, ctx)

			actor1 := fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", actor1.ID, 0, []string{"admin"})

			// Create a deployment to update
			deployment, err := accessor.Querier().DeploymentInsert(ctx, accessor.Source(), &dbsqlc.DeploymentInsertParams{
				Name:      "original_deployment",
				Status:    dbsqlc.NullDeploymentStatus{DeploymentStatus: "draft", Valid: true},
				CreatedBy: "admin",
			})
			require.NoError(t, err)

			resp, resBody := PatchHttp(t, fmt.Sprintf("%s/v1/admin/deployments/%d", server.URL, tt.deploymentID), tt.body, tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var response api.AdminUpdateDeployment200JSONResponse
				err := json.Unmarshal([]byte(resBody), &response)
				require.NoError(t, err)

				require.Equal(t, deployment.ID, response.Data.Id)
				require.Equal(t, "updated_deployment", response.Data.Name)
				require.Equal(t, api.DeploymentStatusDraft, response.Data.Status)
				require.Equal(t, "admin", response.Data.CreatedBy)
				require.NotZero(t, response.Data.CreatedAt)

				// Verify the deployment was actually updated in the database
				updatedDeployment, err := accessor.Querier().DeploymentGetById(ctx, accessor.Source(), deployment.ID)
				require.NoError(t, err)
				require.Equal(t, "updated_deployment", updatedDeployment.Name)
				require.EqualValues(t, api.DeploymentStatusDraft, updatedDeployment.Status)
			} else {
				if tt.expectedBody != "" {
					resJson := testhelper.JsonToMap(t, resBody)
					require.Contains(t, resJson, "error")
					require.Contains(t, resJson["error"], tt.expectedBody)
				}
			}
		})
	}
}

func TestSubmitDeployment(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		deploymentID   int
		token          string
		expectedStatus int
	}{
		{
			name:           "Submit draft deployment successfully",
			deploymentID:   1,
			token:          "admin-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Submit non-existent deployment",
			deploymentID:   999,
			token:          "admin-token",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Submit deployment without authentication",
			deploymentID:   1,
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, _ := SetupHttpTestWithDb(t, ctx)

			actor1 := fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", actor1.ID, 0, []string{"admin"})

			// Create a draft deployment
			draftDeployment, err := accessor.Querier().DeploymentInsertWithConfigSuite(ctx, accessor.Source(), &dbsqlc.DeploymentInsertWithConfigSuiteParams{
				Name:      "draft_deployment",
				CreatedBy: "admin",
				Reviewers: []string{"reviewer1", "reviewer2"},
			})
			require.NoError(t, err)

			// Create a non-draft deployment
			_, err = accessor.Querier().DeploymentInsert(ctx, accessor.Source(), &dbsqlc.DeploymentInsertParams{
				Name:      "non_draft_deployment",
				Status:    dbsqlc.NullDeploymentStatus{DeploymentStatus: "reviewing", Valid: true},
				CreatedBy: "admin",
				Reviewers: []string{"reviewer1", "reviewer2"},
			})
			require.NoError(t, err)

			var deploymentID int
			if tt.deploymentID == 1 {
				deploymentID = int(draftDeployment.ID)
			} else if tt.deploymentID == 2 {
				deploymentID = 2
			} else {
				deploymentID = tt.deploymentID
			}

			resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/admin/deployments/%d/submit", server.URL, deploymentID), "", tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				// Verify the deployment status was actually updated in the database
				updatedDeployment, err := accessor.Querier().DeploymentGetById(ctx, accessor.Source(), draftDeployment.ID)
				require.NoError(t, err)
				require.EqualValues(t, api.DeploymentStatusReviewing, updatedDeployment.Status)
			}
		})
	}
}

func TestAdminRejectDeploymentEndpoint(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	setupTest := func(t *testing.T) (*httptest.Server, dbaccess.Accessor, int64) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		actor1 := fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
		fixture.InsertToken(t, ctx, accessor.Source(), "reviewer-token", actor1.ID, 0, []string{"admin"})
		fixture.InsertToken(t, ctx, accessor.Source(), "non-reviewer-token", actor1.ID, 0, []string{"admin"})

		deployment, err := accessor.Querier().DeploymentInsertWithConfigSuite(ctx, accessor.Source(), &dbsqlc.DeploymentInsertWithConfigSuiteParams{
			CreatedBy: "test-user",
			Name:      "test-deployment",
			Reviewers: []string{"reviewer1", "reviewer2"},
		})
		require.NoError(t, err)
		require.NotNil(t, deployment)

		_, err = accessor.Source().Exec(ctx, "UPDATE deployments SET status = 'reviewing' WHERE id = $1", deployment.ID)
		require.NoError(t, err)

		return server, accessor, deployment.ID
	}

	t.Run("Valid reviewer token and reviewing deployment", func(t *testing.T) {
		server, accessor, deploymentID := setupTest(t)

		body := api.AdminRejectDeploymentJSONRequestBody{
			User: "reviewer1",
		}
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)

		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/admin/deployments/%d/reject", server.URL, deploymentID), string(jsonBody), "reviewer-token")
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		updatedDeployment, err := accessor.Querier().DeploymentGetById(ctx, accessor.Source(), deploymentID)
		require.NoError(t, err)
		require.EqualValues(t, api.DeploymentStatusRejected, updatedDeployment.Status)
		require.NotNil(t, updatedDeployment.FinishedAt)
		require.Equal(t, "reviewer1", *updatedDeployment.FinishedBy)
	})

	t.Run("Non-reviewer user", func(t *testing.T) {
		server, _, deploymentID := setupTest(t)

		body := api.AdminRejectDeploymentJSONRequestBody{
			User: "non-reviewer",
		}
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)

		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/admin/deployments/%d/reject", server.URL, deploymentID), string(jsonBody), "non-reviewer-token")
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Missing user in request body", func(t *testing.T) {
		server, _, deploymentID := setupTest(t)

		body := api.AdminRejectDeploymentJSONRequestBody{
			User: "",
		}
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)

		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/admin/deployments/%d/reject", server.URL, deploymentID), string(jsonBody), "reviewer-token")
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		server, _, deploymentID := setupTest(t)

		body := api.AdminRejectDeploymentJSONRequestBody{
			User: "reviewer1",
		}
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)

		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/admin/deployments/%d/reject", server.URL, deploymentID), string(jsonBody), "")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Non-existent deployment", func(t *testing.T) {
		server, _, _ := setupTest(t)

		body := api.AdminRejectDeploymentJSONRequestBody{
			User: "reviewer1",
		}
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)

		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/admin/deployments/%d/reject", server.URL, 999), string(jsonBody), "reviewer-token")
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestAdminPublishDeploymentEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("Valid admin token and draft deployment", func(t *testing.T) {
		server, accessor, mockK8sController := SetupHttpTestWithDbAndK8s(t, ctx)

		actor := fixture.InsertActor(t, ctx, accessor.Source(), "admin")
		fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", actor.ID, 0, []string{"admin"})

		mockK8sController.On("RunMigrations", mock.Anything, []k8s.MigrationParams{}).Return(nil, nil)

		mockK8sController.On("UpdateDeploymentSet", mock.Anything, []k8s.DeploymentParams{}).Return(nil)
		deployment, err := accessor.Querier().DeploymentInsertWithConfigSuite(ctx, accessor.Source(), &dbsqlc.DeploymentInsertWithConfigSuiteParams{
			CreatedBy: "test-user",
			Name:      "test-deployment",
			Reviewers: []string{"reviewer1", "reviewer2"},
		})
		require.NoError(t, err)
		require.NotNil(t, deployment)

		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/admin/deployments/%d/publish", server.URL, deployment.ID), `{"user":"admin"}`, "admin-token")
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		// Verify the deployment status was actually updated in the database
		updatedDeployment, err := accessor.Querier().DeploymentGetById(ctx, accessor.Source(), deployment.ID)
		require.NoError(t, err)
		require.EqualValues(t, api.DeploymentStatusDeploying, updatedDeployment.Status)

		require.Eventually(t, func() bool {
			updatedDeployment, err = accessor.Querier().DeploymentGetById(ctx, accessor.Source(), deployment.ID)
			require.NoError(t, err)
			return updatedDeployment.Status == "deployed" || updatedDeployment.Status == "failed"
		}, 1*time.Second, 50*time.Millisecond)

		require.EqualValues(t, "deployed", updatedDeployment.Status)

		// Verify the associated config suite was activated
		configSuite, err := accessor.Querier().ConfigSuiteGetById(ctx, accessor.Source(), *updatedDeployment.ConfigSuiteID)
		require.NoError(t, err)
		require.True(t, configSuite.Active)
	})

	t.Run("Valid admin token but non-draft deployment", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, accessor.Source(), "admin")
		fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", actor.ID, 0, []string{"admin"})

		deployment, err := accessor.Querier().DeploymentInsertWithConfigSuite(ctx, accessor.Source(), &dbsqlc.DeploymentInsertWithConfigSuiteParams{
			CreatedBy: "admin",
			Name:      "test-deployment",
			Reviewers: []string{"reviewer1", "reviewer2"},
		})
		require.NoError(t, err)

		_, err = accessor.Source().Exec(ctx, "UPDATE deployments SET status = 'rejected' WHERE id = $1", deployment.ID)
		require.NoError(t, err)

		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/admin/deployments/%d/publish", server.URL, deployment.ID), `{"user":"admin"}`, "admin-token")
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Valid admin token but non-existent deployment", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, accessor.Source(), "admin")
		fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", actor.ID, 0, []string{"admin"})

		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/admin/deployments/%d/publish", server.URL, 999), `{"user":"admin"}`, "admin-token")
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Non-admin token", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, accessor.Source(), "user")
		fixture.InsertToken(t, ctx, accessor.Source(), "user-token", actor.ID, 0, []string{"user"})

		deployment, err := accessor.Querier().DeploymentInsertWithConfigSuite(ctx, accessor.Source(), &dbsqlc.DeploymentInsertWithConfigSuiteParams{
			CreatedBy: "test-user",
			Name:      "test-deployment",
			Reviewers: []string{"reviewer1", "reviewer2"},
		})
		require.NoError(t, err)

		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/admin/deployments/%d/publish", server.URL, deployment.ID), `{"user":"user"}`, "user-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid token", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		deployment, err := accessor.Querier().DeploymentInsertWithConfigSuite(ctx, accessor.Source(), &dbsqlc.DeploymentInsertWithConfigSuiteParams{
			CreatedBy: "test-user",
			Name:      "test-deployment",
			Reviewers: []string{"reviewer1", "reviewer2"},
		})
		require.NoError(t, err)

		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/admin/deployments/%d/publish", server.URL, deployment.ID), `{"user":"admin"}`, "invalid_token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
