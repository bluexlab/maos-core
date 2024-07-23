package apitest

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func TestAdminTokenCreateEndpoint(t *testing.T) {
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
			name:           "Valid admin token creation",
			body:           `{"agent_id":1,"created_by":"admin","expire_at":2000,"permissions":["config:read","admin"]}`,
			token:          "admin-token",
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"agent_id":1, "id":"(ignore)", "created_at":2000, "created_by":"admin", "expire_at":2000, "permissions":["config:read", "admin"]}`,
		},
		{
			name:           "Invalid body",
			body:           `{"invalid_json"}`,
			token:          "admin-token",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid character ",
		},
		{
			name:           "Missing required fields",
			body:           `{"agent_id":1}`,
			token:          "admin-token",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing required fields",
		},
		{
			name:           "Non-admin token",
			body:           `{"agent_id":1,"created_by":"user","expire_at":2000,"permissions":["config:read"]}`,
			token:          "agent-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid token",
			body:           `{"agent_id":1,"created_by":"admin","expire_at":2000,"permissions":["config:read","admin"]}`,
			token:          "invalid_token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, closer := SetupHttpTestWithDb(t, ctx)
			defer closer()

			agent := fixture.InsertAgent(t, ctx, accessor.Source(), "agent1")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", agent.ID, 0, []string{"admin"})
			fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", agent.ID, 0, []string{"user"})

			resp := PostHttp(t, server.URL+"/v1/admin/api_tokens", tt.body, tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			resBody, err := testhelper.ReadBody(resp.Body)
			require.NoError(t, err)

			if tt.expectedStatus == http.StatusCreated {
				tokens, err := accessor.Querier().ApiTokenListByPage(ctx, accessor.Source(), &dbsqlc.ApiTokenListByPageParams{})
				require.NoError(t, err)
				require.GreaterOrEqual(t, len(tokens), 1)
				testhelper.AssertEqualIgnoringFields(t,
					testhelper.JsonToMap(t, tt.expectedBody),
					testhelper.JsonToMap(t, resBody),
					"id",
				)
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
