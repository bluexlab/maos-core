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
		expectedContent map[string]string
	}{
		{
			name:    "Valid admin token and existing config",
			agentID: 1,
			token:   "admin-token",
			setupFunc: func(ctx context.Context, t *testing.T, accessor dbaccess.Accessor, agentID int64) {
				fixture.InsertConfig(t, ctx, accessor.Source(), agentID, map[string]string{"key": "value"})
			},
			expectedStatus:  http.StatusOK,
			expectedContent: map[string]string{"key": "value"},
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
