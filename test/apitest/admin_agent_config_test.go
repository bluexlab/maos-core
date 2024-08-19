package apitest

import (
	"context"
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

func TestAdminGetAgentConfigEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name           string
		agentID        int64
		token          string
		setupFunc      func(context.Context, *testing.T, dbaccess.Accessor, int64)
		expectedStatus int
		// expectedBody   string
		expectedContent map[string]interface{}
	}{
		{
			name:    "Valid admin token and existing config",
			agentID: 1,
			token:   "admin-token",
			setupFunc: func(ctx context.Context, t *testing.T, accessor dbaccess.Accessor, agentID int64) {
				fixture.InsertConfig(t, ctx, accessor.Source(), agentID, map[string]interface{}{"key": "value"})
			},
			expectedStatus:  http.StatusOK,
			expectedContent: map[string]interface{}{"key": "value"},
		},
		{
			name:           "Valid admin token but non-existent config",
			agentID:        999,
			token:          "admin-token",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Non-admin token",
			agentID:        1,
			token:          "agent-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token",
			agentID:        1,
			token:          "invalid_token",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, _ := SetupHttpTestWithDb(t, ctx)

			agent1 := fixture.InsertAgent(t, ctx, accessor.Source(), "agent1")
			agent2 := fixture.InsertAgent(t, ctx, accessor.Source(), "agent2")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", agent1.ID, 0, []string{"admin"})
			fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", agent2.ID, 0, []string{"user"})

			if tt.setupFunc != nil {
				tt.setupFunc(ctx, t, accessor, tt.agentID)
			}

			resp, resBody := GetHttp(t, fmt.Sprintf("%s/v1/admin/agents/%d/config", server.URL, tt.agentID), tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			switch tt.expectedStatus {
			case http.StatusOK:
				var response api.AdminGetAgentConfig200JSONResponse
				err := json.Unmarshal([]byte(resBody), &response)
				require.NoError(t, err)

				require.Equal(t, tt.agentID, response.Data.AgentId)
				require.Equal(t, tt.expectedContent, response.Data.Content)
				require.Empty(t, response.Data.MinAgentVersion)
				require.NotZero(t, response.Data.CreatedAt)
				require.Equal(t, "test", response.Data.CreatedBy)

			case http.StatusNotFound:
				require.Empty(t, resBody)

			case http.StatusUnauthorized:

			default:
				t.Fatalf("Unexpected status code: %d", tt.expectedStatus)
			}
		})
	}
}

func TestAdminUpdateAgentConfigEndpoint(t *testing.T) {
	t.Parallel()

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
			body:           `{"content":{"key":"updated_value"},"min_agent_version":"1.1.0","user":"admin"}`,
			token:          "admin-token",
			expectedStatus: http.StatusCreated,
			expectedBody:   "",
		},
		{
			name:           "Valid admin token but non-existent agent",
			agentID:        999,
			body:           `{"content":{"key":"value"},"min_agent_version":"1.0.0","user":"admin"}`,
			token:          "admin-token",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Cannot update agent config",
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
			body:           `{"content":{"key":"value"},"min_agent_version":"1.0.0","user":"admin"}`,
			token:          "agent-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid token",
			agentID:        1,
			body:           `{"content":{"key":"value"},"min_agent_version":"1.0.0","user":"admin"}`,
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

			resp, resBody := PostHttp(t, fmt.Sprintf("%s/v1/admin/agents/%d/config", server.URL, tt.agentID), tt.body, tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			switch tt.expectedStatus {
			case http.StatusCreated:
				require.Empty(t, resBody)

				// Verify the config was actually updated in the database
				config, err := accessor.Querier().ConfigFindByAgentId(ctx, accessor.Source(), tt.agentID)
				require.NoError(t, err)
				require.NotNil(t, config)

				var content map[string]interface{}
				err = json.Unmarshal(config.Content, &content)
				require.NoError(t, err)
				require.Equal(t, "updated_value", content["key"])
				require.Equal(t, "1.1.0", *config.MinAgentVersion)
				require.Equal(t, "admin", config.CreatedBy)

			case http.StatusInternalServerError, http.StatusBadRequest, http.StatusUnauthorized:
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
