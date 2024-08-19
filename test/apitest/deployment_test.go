package apitest

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func TestAdminListDeploymentsEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name           string
		token          string
		queryParams    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid admin token",
			token:          "admin-token",
			queryParams:    "?page=1&page_size=15",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[{"id":2,"name":"deployment2"},{"id":1,"name":"deployment1"}],"meta":{"total":2,"page":1,"page_size":15}}`,
		},
		{
			name:           "Valid admin token with pagination",
			token:          "admin-token",
			queryParams:    "?page=1&page_size=1",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[{"id":2,"name":"deployment2"}],"meta":{"total":2,"page":1,"page_size":1}}`,
		},
		{
			name:           "Non-admin token",
			token:          "user-token",
			queryParams:    "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid token",
			token:          "invalid_token",
			queryParams:    "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, _ := SetupHttpTestWithDb(t, ctx)

			fixture.InsertDeployment(t, ctx, accessor.Source(), "deployment1", []string{"user1"})
			fixture.InsertDeployment(t, ctx, accessor.Source(), "deployment2", []string{"user2"})
			agent1 := fixture.InsertAgent(t, ctx, accessor.Source(), "agent1")
			agent2 := fixture.InsertAgent(t, ctx, accessor.Source(), "agent2")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", agent1.ID, 0, []string{"admin"})
			fixture.InsertToken(t, ctx, accessor.Source(), "user-token", agent2.ID, 0, []string{"user"})

			resp, resBody := GetHttp(t, server.URL+"/v1/admin/deployments"+tt.queryParams, tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var response api.AdminListDeployments200JSONResponse
				err := json.Unmarshal([]byte(resBody), &response)
				require.NoError(t, err)

				expectedResponse := api.AdminListDeployments200JSONResponse{}
				err = json.Unmarshal([]byte(tt.expectedBody), &expectedResponse)
				require.NoError(t, err)

				require.Equal(t, len(expectedResponse.Data), len(response.Data))
				require.Equal(t, expectedResponse.Meta.Total, response.Meta.Total)
				require.Equal(t, expectedResponse.Meta.Page, response.Meta.Page)
				require.Equal(t, expectedResponse.Meta.PageSize, response.Meta.PageSize)

				for i, expectedDeployment := range expectedResponse.Data {
					require.Equal(t, expectedDeployment.Name, response.Data[i].Name)
					require.NotZero(t, response.Data[i].Id)
					require.NotZero(t, response.Data[i].CreatedAt)
				}
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

			agent1 := fixture.InsertAgent(t, ctx, accessor.Source(), "agent1")
			agent2 := fixture.InsertAgent(t, ctx, accessor.Source(), "agent2")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", agent1.ID, 0, []string{"admin"})
			fixture.InsertToken(t, ctx, accessor.Source(), "user-token", agent2.ID, 0, []string{"user"})

			resp, resBody := PostHttp(t, server.URL+"/v1/admin/deployments", tt.body, tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusCreated {
				var response api.AdminCreateDeployment201JSONResponse
				err := json.Unmarshal([]byte(resBody), &response)
				require.NoError(t, err)

				require.NotZero(t, response.Id)
				require.Equal(t, "new_deployment", response.Name)
				require.Equal(t, api.DeploymentStatusDraft, response.Status)
				require.Equal(t, "admin", response.CreatedBy)
				require.NotZero(t, response.CreatedAt)

				// Verify the deployment was actually created in the database
				deployments, err := accessor.Querier().DeploymentListPaginated(ctx, accessor.Source(), &dbsqlc.DeploymentListPaginatedParams{})
				require.NoError(t, err)
				require.Len(t, deployments, 1)
				createdDeployment := deployments[0]
				require.Equal(t, response.Id, createdDeployment.ID)
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
