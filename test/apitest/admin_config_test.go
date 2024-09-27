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

func TestUpdateConfigEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name           string
		setupFunc      func(context.Context, *testing.T, dbaccess.Accessor) int64
		configID       int64
		body           string
		token          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Valid update",
			setupFunc: func(ctx context.Context, t *testing.T, accessor dbaccess.Accessor) int64 {
				actor := fixture.InsertActor(t, ctx, accessor.Source(), "TestActor")
				configSuite := fixture.InsertConfigSuite(t, ctx, accessor.Source())
				config := fixture.InsertConfig2(t, ctx, accessor.Source(), actor.ID, &configSuite.ID, "testuser", map[string]string{"key": "value"})
				return config.ID
			},
			body:           `{"content":{"key":"newValue"},"min_actor_version":"2.0.0","user":"testuser"}`,
			token:          "admin-token",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":{"id":%d,"actor_id":%d,"content":{"key":"newValue"},"min_actor_version":"2.0.0","created_by":"testuser"}}`,
		},
		{
			name:           "Config not found",
			configID:       999999,
			body:           `{"content":{"key":"value"},"user":"testuser"}`,
			token:          "admin-token",
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Config suite deployed",
			setupFunc: func(ctx context.Context, t *testing.T, accessor dbaccess.Accessor) int64 {
				actor := fixture.InsertActor(t, ctx, accessor.Source(), "TestActor")
				configSuite := fixture.InsertConfigSuite(t, ctx, accessor.Source())
				_, err := accessor.Source().Exec(ctx, "UPDATE config_suites SET deployed_at = 16888 WHERE id = $1", configSuite.ID)
				require.NoError(t, err)
				config := fixture.InsertConfig2(t, ctx, accessor.Source(), actor.ID, &configSuite.ID, "testuser", map[string]string{"key": "value"})
				return config.ID
			},
			body:           `{"content":{"key":"newValue"},"user":"testuser"}`,
			token:          "admin-token",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid token",
			configID:       1,
			body:           `{"content":{"key":"value"},"user":"testuser"}`,
			token:          "invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, _ := SetupHttpTestWithDb(t, ctx)

			actor := fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", actor.ID, 0, []string{"admin"})

			if tt.setupFunc != nil {
				tt.configID = tt.setupFunc(ctx, t, accessor)
			}

			resp, resBody := PatchHttp(t, fmt.Sprintf("%s/v1/admin/configs/%d", server.URL, tt.configID), tt.body, tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			switch tt.expectedStatus {
			case http.StatusOK:
				var response api.AdminUpdateConfig200JSONResponse
				err := json.Unmarshal([]byte(resBody), &response)
				require.NoError(t, err)

				expectedBody := fmt.Sprintf(tt.expectedBody, tt.configID, response.Data.ActorId)
				var expectedResponse api.AdminUpdateConfig200JSONResponse
				err = json.Unmarshal([]byte(expectedBody), &expectedResponse)
				require.NoError(t, err)

				require.Equal(t, expectedResponse.Data.Id, response.Data.Id)
				require.Equal(t, expectedResponse.Data.ActorId, response.Data.ActorId)
				require.Equal(t, expectedResponse.Data.Content, response.Data.Content)
				require.Equal(t, expectedResponse.Data.MinActorVersion, response.Data.MinActorVersion)
				require.Equal(t, expectedResponse.Data.CreatedBy, response.Data.CreatedBy)
				require.NotZero(t, response.Data.CreatedAt)

			case http.StatusNotFound:
				require.Empty(t, resBody)

			case http.StatusUnauthorized:

			default:
				t.Fatalf("Unexpected status code: %d", tt.expectedStatus)
			}
		})
	}
}
