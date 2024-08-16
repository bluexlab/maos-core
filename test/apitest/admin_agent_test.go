package apitest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func TestAdminListAgentEndpoint(t *testing.T) {
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
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[{"id":1,"name":"agent1"},{"id":2,"name":"agent2"}],"meta":{"total_pages":1}}`,
		},
		{
			name:           "Valid admin token with pagination",
			token:          "admin-token",
			queryParams:    "?page=1&page_size=1",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[{"id":1,"name":"agent1"}],"meta":{"total_pages":2}}`,
		},
		{
			name:           "Non-admin token",
			token:          "agent-token",
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

			agent1 := fixture.InsertAgent(t, ctx, accessor.Source(), "agent1")
			agent2 := fixture.InsertAgent(t, ctx, accessor.Source(), "agent2")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", agent1.ID, 0, []string{"admin"})
			fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", agent2.ID, 0, []string{"user"})

			resp, resBody := GetHttp(t, server.URL+"/v1/admin/agents"+tt.queryParams, tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var response api.AdminListAgents200JSONResponse
				err := json.Unmarshal([]byte(resBody), &response)
				require.NoError(t, err)

				expectedResponse := api.AdminListAgents200JSONResponse{}
				err = json.Unmarshal([]byte(tt.expectedBody), &expectedResponse)
				require.NoError(t, err)

				require.Equal(t, len(expectedResponse.Data), len(response.Data))
				require.Equal(t, expectedResponse.Meta.TotalPages, response.Meta.TotalPages)

				for i, expectedAgent := range expectedResponse.Data {
					require.Equal(t, expectedAgent.Name, response.Data[i].Name)
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

func TestAdminCreateAgentEndpoint(t *testing.T) {
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
			body:           `{"name":"new_agent"}`,
			token:          "admin-token",
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"id":3,"name":"new_agent"}`,
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
			body:           `{}`,
			token:          "admin-token",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing required field: name",
		},
		{
			name:           "Non-admin token",
			body:           `{"name":"new_agent"}`,
			token:          "agent-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid token",
			body:           `{"name":"new_agent"}`,
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
			fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", agent2.ID, 0, []string{"user"})

			resp, resBody := PostHttp(t, server.URL+"/v1/admin/agents", tt.body, tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusCreated {
				var response api.AdminCreateAgent201JSONResponse
				err := json.Unmarshal([]byte(resBody), &response)
				require.NoError(t, err)

				require.NotZero(t, response.Id)
				require.Equal(t, "new_agent", response.Name)
				require.NotZero(t, response.CreatedAt)

				// Verify the agent was actually created in the database
				createdAgent, err := accessor.Querier().AgentFindById(ctx, accessor.Source(), response.Id)
				require.NoError(t, err)
				require.NotNil(t, createdAgent)
				require.Equal(t, "new_agent", createdAgent.Name)

				// Verify the associated queue was created
				queue, err := accessor.Querier().QueueFindById(ctx, accessor.Source(), createdAgent.QueueID)
				require.NoError(t, err)
				require.NotNil(t, queue)
				require.Equal(t, "new_agent", queue.Name)
				require.Equal(t, []byte(`{"type": "agent"}`), queue.Metadata)
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

func TestAdminGetAgentEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name           string
		agentID        int64
		token          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid admin token and existing agent",
			agentID:        1,
			token:          "admin-token",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":{"id":1,"name":"agent1"}}`,
		},
		{
			name:           "Valid admin token but non-existent agent",
			agentID:        999,
			token:          "admin-token",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
		{
			name:           "Non-admin token",
			agentID:        1,
			token:          "agent-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid token",
			agentID:        1,
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
			fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", agent2.ID, 0, []string{"user"})

			resp, resBody := GetHttp(t, fmt.Sprintf("%s/v1/admin/agents/%d", server.URL, tt.agentID), tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			switch tt.expectedStatus {
			case http.StatusOK:
				var response api.AdminGetAgent200JSONResponse
				err := json.Unmarshal([]byte(resBody), &response)
				require.NoError(t, err)

				require.Equal(t, tt.agentID, response.Data.Id)
				require.Equal(t, "agent1", response.Data.Name)
				require.NotZero(t, response.Data.CreatedAt)

			case http.StatusNotFound:
				require.Empty(t, resBody)

			case http.StatusUnauthorized:
				if tt.expectedBody != "" {
					resJson := testhelper.JsonToMap(t, resBody)
					require.Contains(t, resJson, "error")
					require.Contains(t, resJson["error"], tt.expectedBody)
				}

			default:
				t.Fatalf("Unexpected status code: %d", tt.expectedStatus)
			}
		})
	}
}

func TestAdminUpdateAgentEndpoint(t *testing.T) {
	// t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name           string
		agentID        int64
		body           string
		token          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid admin token and existing agent",
			agentID:        1,
			body:           `{"name":"updated_agent"}`,
			token:          "admin-token",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":{"id":1,"name":"updated_agent"}}`,
		},
		{
			name:           "Valid admin token but non-existent agent",
			agentID:        999,
			body:           `{"name":"updated_agent"}`,
			token:          "admin-token",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid body",
			agentID:        1,
			body:           `{"invalid_json"}`,
			token:          "admin-token",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid character",
		},
		{
			name:           "Non-admin token",
			agentID:        1,
			body:           `{"name":"updated_agent"}`,
			token:          "agent-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid token",
			agentID:        1,
			body:           `{"name":"updated_agent"}`,
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
			fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", agent2.ID, 0, []string{"user"})

			resp, resBody := PatchHttp(t, fmt.Sprintf("%s/v1/admin/agents/%d", server.URL, tt.agentID), tt.body, tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			switch tt.expectedStatus {
			case http.StatusOK:
				var response api.AdminUpdateAgent200JSONResponse
				err := json.Unmarshal([]byte(resBody), &response)
				require.NoError(t, err)

				require.Equal(t, tt.agentID, response.Data.Id)
				require.Equal(t, "updated_agent", response.Data.Name)
				require.NotZero(t, response.Data.CreatedAt)

				// Verify the agent was actually updated in the database
				updatedAgent, err := accessor.Querier().AgentFindById(ctx, accessor.Source(), tt.agentID)
				require.NoError(t, err)
				require.NotNil(t, updatedAgent)
				require.Equal(t, "updated_agent", updatedAgent.Name)

			case http.StatusNotFound, http.StatusBadRequest, http.StatusUnauthorized:
				if tt.expectedBody != "" {
					resJson := testhelper.JsonToMap(t, resBody)
					require.Contains(t, resJson, "error")
					require.Contains(t, resJson["error"], tt.expectedBody)
				}

			default:
				t.Fatalf("Unexpected status code: %d", tt.expectedStatus)
			}
		})
	}
}

func TestAdminDeleteAgentEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name           string
		agentID        int64
		token          string
		setupFunc      func(context.Context, *testing.T, dbaccess.Accessor, int64)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid admin token and existing agent",
			agentID:        1,
			token:          "admin-token",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":{}}`,
		},
		{
			name:           "Valid admin token but non-existent agent",
			agentID:        999,
			token:          "admin-token",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
		{
			name:           "Non-admin token",
			agentID:        1,
			token:          "agent-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid token",
			agentID:        1,
			token:          "invalid_token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:    "Agent with associated configs",
			agentID: 1,
			token:   "admin-token",
			setupFunc: func(ctx context.Context, t *testing.T, accessor dbaccess.Accessor, agentID int64) {
				fixture.InsertConfig(t, ctx, accessor.Source(), agentID, map[string]interface{}{"key": "value"})
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   "Agent has associated configs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, _ := SetupHttpTestWithDb(t, ctx)

			fixture.InsertAgent(t, ctx, accessor.Source(), "agent1")
			adminAgent := fixture.InsertAgent(t, ctx, accessor.Source(), "agent2")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", adminAgent.ID, 0, []string{"admin"})

			if tt.setupFunc != nil {
				tt.setupFunc(ctx, t, accessor, tt.agentID)
			}

			resp, resBody := DeleteHttp(t, fmt.Sprintf("%s/v1/admin/agents/%d", server.URL, tt.agentID), tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			switch tt.expectedStatus {
			case http.StatusOK:
				// Verify the agent was actually deleted from the database
				_, err := accessor.Querier().AgentFindById(ctx, accessor.Source(), tt.agentID)
				require.Error(t, err)
				require.IsType(t, err, sql.ErrNoRows)

			case http.StatusNotFound:
				require.Empty(t, resBody)

			case http.StatusConflict:
				require.Empty(t, resBody)

			case http.StatusUnauthorized:
				if tt.expectedBody != "" {
					resJson := testhelper.JsonToMap(t, resBody)
					require.Contains(t, resJson, "error")
					require.Contains(t, resJson["error"], tt.expectedBody)
				}

			default:
				t.Fatalf("Unexpected status code: %d", tt.expectedStatus)
			}
		})
	}
}
