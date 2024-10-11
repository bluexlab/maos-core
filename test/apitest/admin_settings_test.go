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

func TestAdminGetSettingsEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server, ds, _ := SetupHttpTestWithDb(t, ctx)

	actor1 := fixture.InsertActor(t, ctx, ds, "actor1")
	fixture.InsertToken(t, ctx, ds, "admin-token", actor1.ID, []string{"admin"})
	fixture.InsertExpiredToken(t, ctx, ds, "expired-token", actor1.ID, []string{"admin"})

	t.Run("Valid admin token", func(t *testing.T) {
		_, err := querier.SettingUpdateSystem(ctx, ds, json.RawMessage(`{"display_name":"test-maos", "deployment_approve_required":true}`))
		require.NoError(t, err)

		resp, resBody := GetHttp(t, server.URL+"/v1/admin/setting", "admin-token")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.AdminGetSetting200JSONResponse
		err = json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)

		expectedResponse := api.AdminGetSetting200JSONResponse{}
		err = json.Unmarshal([]byte(`{"display_name":"test-maos", "deployment_approve_required":true}`), &expectedResponse)
		require.NoError(t, err)

		require.Equal(t, expectedResponse, response)
	})

	t.Run("Non-admin token", func(t *testing.T) {
		resp, _ := GetHttp(t, server.URL+"/v1/admin/setting", "actor-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Expired token", func(t *testing.T) {
		resp, _ := GetHttp(t, server.URL+"/v1/admin/setting", "expired-token")
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Invalid token", func(t *testing.T) {
		resp, _ := GetHttp(t, server.URL+"/v1/admin/setting", "invalid_token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestAdminUpdateSettingsEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server, ds, _ := SetupHttpTestWithDb(t, ctx)
	actor1 := fixture.InsertActor(t, ctx, ds, "actor1")
	fixture.InsertToken(t, ctx, ds, "admin-token", actor1.ID, []string{"admin"})

	t.Run("Valid admin token", func(t *testing.T) {
		updateBody := `{"display_name":"updated-maos", "deployment_approve_required":false}`
		resp, resBody := PatchHttp(t, server.URL+"/v1/admin/setting", updateBody, "admin-token")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.AdminUpdateSetting200JSONResponse
		err := json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)

		expectedResponse := api.AdminUpdateSetting200JSONResponse{}
		err = json.Unmarshal([]byte(updateBody), &expectedResponse)
		require.NoError(t, err)
		require.Equal(t, expectedResponse, response)

		// Verify the setting were actually updated in the database
		getResp, getResBody := GetHttp(t, server.URL+"/v1/admin/setting", "admin-token")
		require.Equal(t, http.StatusOK, getResp.StatusCode)

		var getResponse api.AdminGetSetting200JSONResponse
		err = json.Unmarshal([]byte(getResBody), &getResponse)
		require.NoError(t, err)
		require.Equal(t, "updated-maos", getResponse.DisplayName)
		require.Equal(t, false, getResponse.DeploymentApproveRequired)
	})

	t.Run("Non-admin token", func(t *testing.T) {
		updateBody := `{"display_name":"unauthorized-update", "deployment_approve_required":true}`
		resp, _ := PatchHttp(t, server.URL+"/v1/admin/setting", updateBody, "actor-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid token", func(t *testing.T) {
		updateBody := `{"display_name":"invalid-update", "deployment_approve_required":true}`
		resp, _ := PatchHttp(t, server.URL+"/v1/admin/setting", updateBody, "invalid_token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		updateBody := `{"display_name":"invalid-json", "deployment_approve_required":true`
		resp, _ := PatchHttp(t, server.URL+"/v1/admin/setting", updateBody, "admin-token")
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
