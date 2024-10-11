package apitest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
)

func TestUpdateConfigEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("Valid update", func(t *testing.T) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "actor1")
		fixture.InsertToken(t, ctx, ds, "admin-token", actor.ID, 0, []string{"admin"})

		setupActor := fixture.InsertActor(t, ctx, ds, "TestActor")
		configSuite := fixture.InsertConfigSuite(t, ctx, ds)
		config := fixture.InsertConfig2(t, ctx, ds, setupActor.ID, &configSuite.ID, "testuser", map[string]string{"key": "value"})

		body := `{"content":{"key":"newValue"},"min_actor_version":"2.0.0","user":"testuser"}`
		resp, resBody := PatchHttp(t, fmt.Sprintf("%s/v1/admin/configs/%d", server.URL, config.ID), body, "admin-token")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.AdminUpdateConfig200JSONResponse
		err := json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)

		expectedBody := fmt.Sprintf(`{"data":{"id":%d,"actor_id":%d,"content":{"key":"newValue"},"created_by":"testuser"}}`, config.ID, response.Data.ActorId)
		var expectedResponse api.AdminUpdateConfig200JSONResponse
		err = json.Unmarshal([]byte(expectedBody), &expectedResponse)
		require.NoError(t, err)

		require.Equal(t, expectedResponse.Data.Id, response.Data.Id)
		require.Equal(t, expectedResponse.Data.ActorId, response.Data.ActorId)
		require.Equal(t, expectedResponse.Data.Content, response.Data.Content)
		require.Equal(t, expectedResponse.Data.CreatedBy, response.Data.CreatedBy)
		require.NotZero(t, response.Data.CreatedAt)
	})

	t.Run("Config not found", func(t *testing.T) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "actor1")
		fixture.InsertToken(t, ctx, ds, "admin-token", actor.ID, 0, []string{"admin"})

		body := `{"content":{"key":"value"},"user":"testuser"}`
		resp, resBody := PatchHttp(t, fmt.Sprintf("%s/v1/admin/configs/%d", server.URL, 999999), body, "admin-token")
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Empty(t, resBody)
	})

	t.Run("Config suite deployed", func(t *testing.T) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "actor1")
		fixture.InsertToken(t, ctx, ds, "admin-token", actor.ID, 0, []string{"admin"})

		setupActor := fixture.InsertActor(t, ctx, ds, "TestActor")
		configSuite := fixture.InsertConfigSuite(t, ctx, ds)
		_, err := ds.Exec(ctx, "UPDATE config_suites SET deployed_at = 16888 WHERE id = $1", configSuite.ID)
		require.NoError(t, err)
		config := fixture.InsertConfig2(t, ctx, ds, setupActor.ID, &configSuite.ID, "testuser", map[string]string{"key": "value"})

		body := `{"content":{"key":"newValue"},"user":"testuser"}`
		resp, resBody := PatchHttp(t, fmt.Sprintf("%s/v1/admin/configs/%d", server.URL, config.ID), body, "admin-token")
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Empty(t, resBody)
	})

	t.Run("Invalid token", func(t *testing.T) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "actor1")
		fixture.InsertToken(t, ctx, ds, "admin-token", actor.ID, 0, []string{"admin"})

		body := `{"content":{"key":"value"},"user":"testuser"}`
		resp, _ := PatchHttp(t, fmt.Sprintf("%s/v1/admin/configs/%d", server.URL, 1), body, "invalid-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
