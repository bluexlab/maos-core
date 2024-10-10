package admin_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/admin"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/k8s"
)

type mockK8sController4Secret struct {
	mock.Mock
}

func (m *mockK8sController4Secret) ListSecrets(ctx context.Context) ([]k8s.Secret, error) {
	args := m.Called(ctx)
	return args.Get(0).([]k8s.Secret), args.Error(1)
}

func (m *mockK8sController4Secret) UpdateSecret(ctx context.Context, name string, data map[string]string) error {
	args := m.Called(ctx, name, data)
	return args.Error(0)
}

func (m *mockK8sController4Secret) DeleteSecret(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *mockK8sController4Secret) TriggerRollingRestart(ctx context.Context, deploymentName string) error {
	args := m.Called(ctx, deploymentName)
	return args.Error(0)
}

func (m *mockK8sController4Secret) UpdateDeploymentSet(ctx context.Context, deploymentSet []k8s.DeploymentParams) error {
	args := m.Called(ctx, deploymentSet)
	return args.Error(0)
}

func (m *mockK8sController4Secret) ListRunningPodsWithMetrics(ctx context.Context) ([]k8s.PodWithMetrics, error) {
	args := m.Called(ctx)
	return args.Get(0).([]k8s.PodWithMetrics), args.Error(1)
}

func (m *mockK8sController4Secret) RunMigrations(ctx context.Context, migrations []k8s.MigrationParams) (map[string][]string, error) {
	args := m.Called(ctx, migrations)
	return args.Get(0).(map[string][]string), args.Error(1)
}

func TestListSecrets(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Successfully list secrets", func(t *testing.T) {
		t.Parallel()
		mockController := new(mockK8sController4Secret)
		secrets := []k8s.Secret{
			{Name: "secret1", Keys: []string{"key1", "key2"}},
			{Name: "secret2", Keys: []string{"key3"}},
		}
		mockController.On("ListSecrets", ctx).Return(secrets, nil)

		response, err := admin.ListSecrets(ctx, mockController)
		require.NoError(t, err)
		require.IsType(t, &api.AdminListSecrets200JSONResponse{}, response)

		jsonResponse := response.(*api.AdminListSecrets200JSONResponse)
		require.Len(t, jsonResponse.Data, 2)
		require.Equal(t, "secret1", jsonResponse.Data[0].Name)
		require.ElementsMatch(t, []string{"key1", "key2"}, jsonResponse.Data[0].Keys)
		require.Equal(t, "secret2", jsonResponse.Data[1].Name)
		require.ElementsMatch(t, []string{"key3"}, jsonResponse.Data[1].Keys)

		mockController.AssertExpectations(t)
	})

	t.Run("Empty secrets list", func(t *testing.T) {
		t.Parallel()
		mockController := new(mockK8sController4Secret)
		mockController.On("ListSecrets", ctx).Return([]k8s.Secret{}, nil)

		response, err := admin.ListSecrets(ctx, mockController)
		require.NoError(t, err)
		require.IsType(t, &api.AdminListSecrets200JSONResponse{}, response)

		jsonResponse := response.(*api.AdminListSecrets200JSONResponse)
		require.Empty(t, jsonResponse.Data)

		mockController.AssertExpectations(t)
	})

	t.Run("K8s controller error", func(t *testing.T) {
		t.Parallel()
		mockController := new(mockK8sController4Secret)
		mockController.On("ListSecrets", ctx).Return([]k8s.Secret{}, assert.AnError)

		response, err := admin.ListSecrets(ctx, mockController)
		require.NoError(t, err)
		require.IsType(t, &api.AdminListSecrets500JSONResponse{}, response)

		errorResponse := response.(*api.AdminListSecrets500JSONResponse)
		require.Contains(t, errorResponse.Error, "Failed to list secrets")

		mockController.AssertExpectations(t)
	})
}

func TestUpdateSecret(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Successfully update secret", func(t *testing.T) {
		t.Parallel()
		mockController := new(mockK8sController4Secret)
		secretName := "test-secret"
		secretData := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		mockController.On("UpdateSecret", ctx, secretName, secretData).Return(nil)

		body := api.AdminUpdateSecretJSONRequestBody(secretData)
		request := api.AdminUpdateSecretRequestObject{
			Name: secretName,
			Body: &body,
		}

		response, err := admin.UpdateSecret(ctx, mockController, request)
		require.NoError(t, err)
		require.IsType(t, &api.AdminUpdateSecret200Response{}, response)

		mockController.AssertExpectations(t)
	})

	t.Run("K8s controller error", func(t *testing.T) {
		t.Parallel()
		mockController := new(mockK8sController4Secret)
		secretName := "test-secret"
		secretData := map[string]string{
			"key1": "value1",
		}
		mockController.On("UpdateSecret", ctx, secretName, secretData).Return(assert.AnError)

		body := api.AdminUpdateSecretJSONRequestBody(secretData)
		request := api.AdminUpdateSecretRequestObject{
			Name: secretName,
			Body: &body,
		}

		response, err := admin.UpdateSecret(ctx, mockController, request)
		require.NoError(t, err)
		require.IsType(t, &api.AdminUpdateSecret500JSONResponse{}, response)

		errorResponse := response.(*api.AdminUpdateSecret500JSONResponse)
		require.Contains(t, errorResponse.Error, "Failed to update secret")

		mockController.AssertExpectations(t)
	})
}

func TestDeleteSecret(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Successfully delete secret", func(t *testing.T) {
		t.Parallel()
		mockController := new(mockK8sController4Secret)
		secretName := "test-secret"
		mockController.On("DeleteSecret", ctx, secretName).Return(nil)

		request := api.AdminDeleteSecretRequestObject{
			Name: secretName,
		}

		response, err := admin.DeleteSecret(ctx, mockController, request)
		require.NoError(t, err)
		require.IsType(t, &api.AdminDeleteSecret200Response{}, response)

		mockController.AssertExpectations(t)
	})

	t.Run("K8s controller error", func(t *testing.T) {
		t.Parallel()
		mockController := new(mockK8sController4Secret)
		secretName := "test-secret"
		mockController.On("DeleteSecret", ctx, secretName).Return(assert.AnError)

		request := api.AdminDeleteSecretRequestObject{
			Name: secretName,
		}

		response, err := admin.DeleteSecret(ctx, mockController, request)
		require.NoError(t, err)
		require.IsType(t, &api.AdminDeleteSecret500JSONResponse{}, response)

		errorResponse := response.(*api.AdminDeleteSecret500JSONResponse)
		require.Contains(t, errorResponse.Error, "Failed to delete secret")

		mockController.AssertExpectations(t)
	})
}
