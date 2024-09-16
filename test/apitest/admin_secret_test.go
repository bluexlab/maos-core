package apitest

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
	"gitlab.com/navyx/ai/maos/maos-core/k8s"
)

func TestAdminListSecretsEndpoint(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Valid admin token", func(t *testing.T) {
		server, accessor, mockController := SetupHttpTestWithDbAndK8s(t, ctx)

		adminAgent := fixture.InsertAgent(t, ctx, accessor.Source(), "admin")
		fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", adminAgent.ID, 0, []string{"admin"})

		mockController.On("ListSecrets", mock.Anything).Return([]k8s.Secret{
			{Name: "secret1", Keys: []string{"key1", "key2"}},
			{Name: "secret2", Keys: []string{"key3", "key4"}},
		}, nil)

		resp, resBody := GetHttp(t, server.URL+"/v1/admin/secrets", "admin-token")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.AdminListSecrets200JSONResponse
		err := json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)

		require.Len(t, response.Data, 2)
		require.Equal(t, "secret1", response.Data[0].Name)
		require.Equal(t, []string{"key1", "key2"}, response.Data[0].Keys)
		require.Equal(t, "secret2", response.Data[1].Name)
		require.Equal(t, []string{"key3", "key4"}, response.Data[1].Keys)

		mockController.AssertExpectations(t)
	})

	t.Run("Non-admin token", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		adminAgent := fixture.InsertAgent(t, ctx, accessor.Source(), "admin")
		fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", adminAgent.ID, 0, []string{"user"})

		resp, _ := GetHttp(t, server.URL+"/v1/admin/secrets", "agent-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid token", func(t *testing.T) {
		server, _, _ := SetupHttpTestWithDb(t, ctx)

		resp, _ := GetHttp(t, server.URL+"/v1/admin/secrets", "invalid_token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestAdminUpdateSecretEndpoint(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Valid admin token", func(t *testing.T) {
		server, accessor, mockController := SetupHttpTestWithDbAndK8s(t, ctx)

		adminAgent := fixture.InsertAgent(t, ctx, accessor.Source(), "admin")
		fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", adminAgent.ID, 0, []string{"admin"})

		mockController.On("UpdateSecret", mock.Anything, "test-secret", map[string]string{"test-key": "new-value"}).Return(nil)

		body := `{"test-key":"new-value"}`
		resp, _ := PatchHttp(t, server.URL+"/v1/admin/secrets/test-secret", body, "admin-token")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		mockController.AssertExpectations(t)
	})

	t.Run("Non-admin token", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDbAndK8s(t, ctx)

		adminAgent := fixture.InsertAgent(t, ctx, accessor.Source(), "admin")
		fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", adminAgent.ID, 0, []string{"user"})

		body := `{"test-key":"new-value"}`
		resp, _ := PatchHttp(t, server.URL+"/v1/admin/secrets/test-secret", body, "agent-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid token", func(t *testing.T) {
		server, _, _ := SetupHttpTestWithDb(t, ctx)

		body := `{"test-key":"new-value"}`
		resp, _ := PatchHttp(t, server.URL+"/v1/admin/secrets/test-secret", body, "invalid_token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid body", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDbAndK8s(t, ctx)

		adminAgent := fixture.InsertAgent(t, ctx, accessor.Source(), "admin")
		fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", adminAgent.ID, 0, []string{"admin"})

		body := `{"invalid_json"}`
		resp, resBody := PatchHttp(t, server.URL+"/v1/admin/secrets/test-secret", body, "admin-token")
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		resJson := testhelper.JsonToMap(t, resBody)
		require.Contains(t, resJson, "error")
		require.Contains(t, resJson["error"], "invalid character")
	})
}

func TestAdminDeleteSecretEndpoint(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Valid admin token", func(t *testing.T) {
		server, accessor, mockController := SetupHttpTestWithDbAndK8s(t, ctx)

		adminAgent := fixture.InsertAgent(t, ctx, accessor.Source(), "admin")
		fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", adminAgent.ID, 0, []string{"admin"})

		mockController.On("DeleteSecret", mock.Anything, "test-secret").Return(nil)

		resp, _ := DeleteHttp(t, server.URL+"/v1/admin/secrets/test-secret", "admin-token")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		mockController.AssertExpectations(t)
	})

	t.Run("Non-admin token", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDbAndK8s(t, ctx)

		adminAgent := fixture.InsertAgent(t, ctx, accessor.Source(), "admin")
		fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", adminAgent.ID, 0, []string{"user"})

		resp, _ := DeleteHttp(t, server.URL+"/v1/admin/secrets/test-secret", "agent-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid token", func(t *testing.T) {
		server, _, _ := SetupHttpTestWithDb(t, ctx)

		resp, _ := DeleteHttp(t, server.URL+"/v1/admin/secrets/test-secret", "invalid_token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("K8s controller error", func(t *testing.T) {
		server, accessor, mockController := SetupHttpTestWithDbAndK8s(t, ctx)

		adminAgent := fixture.InsertAgent(t, ctx, accessor.Source(), "admin")
		fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", adminAgent.ID, 0, []string{"admin"})

		mockController.On("DeleteSecret", mock.Anything, "test-secret").Return(assert.AnError)

		resp, resBody := DeleteHttp(t, server.URL+"/v1/admin/secrets/test-secret", "admin-token")
		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		var errorResponse api.AdminDeleteSecret500JSONResponse
		err := json.Unmarshal([]byte(resBody), &errorResponse)
		require.NoError(t, err)
		require.Contains(t, errorResponse.Error, "Failed to delete secret")

		mockController.AssertExpectations(t)
	})
}
