package apitest

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
)

func TestGetActorConfigEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("Valid actor token with active config", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, accessor.Source(), "test-actor")
		configSuite := fixture.InsertConfigSuite(t, ctx, accessor.Source())
		fixture.InsertConfig2(t, ctx, accessor.Source(), actor.ID, &configSuite.ID, "test-user", map[string]string{"key": "value"})
		_, err := accessor.Source().Exec(ctx, "UPDATE config_suites SET deployed_at = EXTRACT(EPOCH FROM NOW()), active = TRUE WHERE id = $1", configSuite.ID)
		require.NoError(t, err)
		fixture.InsertToken(t, ctx, accessor.Source(), "actor-token", actor.ID, 0, []string{"actor"})

		resp, resBody := GetHttp(t, server.URL+"/v1/config", "actor-token")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.GetCallerConfig200JSONResponse
		err = json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)

		expectedResponse := api.GetCallerConfig200JSONResponse{}
		err = json.Unmarshal([]byte(`{"data":{"key":"value"}}`), &expectedResponse)
		require.NoError(t, err)

		require.Equal(t, expectedResponse.Data, response.Data)
	})

	t.Run("Valid actor token without active config", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, accessor.Source(), "test-actor")
		fixture.InsertToken(t, ctx, accessor.Source(), "actor-token", actor.ID, 0, []string{"actor"})

		resp, resBody := GetHttp(t, server.URL+"/v1/config", "actor-token")
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Empty(t, resBody)
	})

	t.Run("Invalid token", func(t *testing.T) {
		server, _, _ := SetupHttpTestWithDb(t, ctx)

		resp, _ := GetHttp(t, server.URL+"/v1/config", "invalid-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Valid actor token with version check", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, accessor.Source(), "test-actor")
		configSuite := fixture.InsertConfigSuite(t, ctx, accessor.Source())
		fixture.InsertConfig2(t, ctx, accessor.Source(), actor.ID, &configSuite.ID, "test-user", map[string]string{"key": "value"})
		_, err := accessor.Source().Exec(ctx, "UPDATE config_suites SET deployed_at = EXTRACT(EPOCH FROM NOW()), active = TRUE WHERE id = $1", configSuite.ID)
		require.NoError(t, err)
		_, err = accessor.Source().Exec(ctx, "UPDATE configs SET min_actor_version = ARRAY[1, 2, 3] WHERE actor_id = $1", actor.ID)
		require.NoError(t, err)
		fixture.InsertToken(t, ctx, accessor.Source(), "actor-token", actor.ID, 0, []string{"actor"})

		// Test with a compatible version
		resp, resBody := GetHttpWithHeader(t, server.URL+"/v1/config", "actor-token", map[string]string{"X-Actor-Version": "1.2.3"})
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.GetCallerConfig200JSONResponse
		err = json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)
		require.EqualValues(t, map[string]string{"key": "value"}, response.Data)

		// Test with an incompatible version
		resp, _ = GetHttpWithHeader(t, server.URL+"/v1/config", "actor-token", map[string]string{"X-Actor-Version": "1.1.0"})
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
