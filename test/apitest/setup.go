package apitest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/handler"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
	"gitlab.com/navyx/ai/maos/maos-core/k8s"
	"gitlab.com/navyx/ai/maos/maos-core/middleware"
)

var querier = dbsqlc.New()

type MockK8sController struct {
	mock.Mock
}

func (m *MockK8sController) UpdateDeploymentSet(ctx context.Context, deploymentSet []k8s.DeploymentParams) error {
	args := m.Called(ctx, deploymentSet)
	return args.Error(0)
}

func (m *MockK8sController) TriggerRollingRestart(ctx context.Context, deploymentName string) error {
	args := m.Called(ctx, deploymentName)
	return args.Error(0)
}

func (m *MockK8sController) ListSecrets(ctx context.Context) ([]k8s.Secret, error) {
	args := m.Called(ctx)
	return args.Get(0).([]k8s.Secret), args.Error(1)
}

func (m *MockK8sController) UpdateSecret(ctx context.Context, secretName string, secretData map[string]string) error {
	args := m.Called(ctx, secretName, secretData)
	return args.Error(0)
}

func (m *MockK8sController) DeleteSecret(ctx context.Context, secretName string) error {
	args := m.Called(ctx, secretName)
	return args.Error(0)
}

func (m *MockK8sController) ListRunningPodsWithMetrics(ctx context.Context) ([]k8s.PodWithMetrics, error) {
	args := m.Called(ctx)
	return args.Get(0).([]k8s.PodWithMetrics), args.Error(1)
}

func (m *MockK8sController) RunMigrations(ctx context.Context, params []k8s.MigrationParams) (map[string]interface{}, error) {
	args := m.Called(ctx, params)
	res1 := args.Get(0)
	if res1 == nil {
		return nil, args.Error(1)
	}
	return res1.(map[string]interface{}), args.Error(1)
}

// SetupHttpTestWithDb sets up two test servers and database accessors.
// It can simulate two running services in HA mode.
func SetupHttpTestWithDb(t *testing.T, ctx context.Context) (*httptest.Server, dbaccess.DataSource, *httptest.Server) {
	dbPool := testhelper.TestDB(ctx, t)
	pool2, err := pgxpool.NewWithConfig(ctx, dbPool.Config())
	require.NoError(t, err)

	t.Cleanup(func() {
		dbPool.Close()
		pool2.Close()
	})

	s, ds, _, _ := builder(t, ctx, dbPool)
	s2, _, _, _ := builder(t, ctx, pool2)

	return s, ds, s2
}

func SetupHttpTestWithDbAndSuiteStore(t *testing.T, ctx context.Context) (*httptest.Server, dbaccess.DataSource, *httptest.Server, *testhelper.MockSuiteStore) {
	dbPool := testhelper.TestDB(ctx, t)
	pool2, err := pgxpool.NewWithConfig(ctx, dbPool.Config())
	require.NoError(t, err)

	t.Cleanup(func() {
		dbPool.Close()
		pool2.Close()
	})

	s, ds, suiteStore, _ := builder(t, ctx, dbPool)
	s2, _, _, _ := builder(t, ctx, pool2)

	return s, ds, s2, suiteStore
}

func SetupHttpTestWithDbAndK8s(t *testing.T, ctx context.Context) (*httptest.Server, dbaccess.DataSource, *MockK8sController) {
	dbPool := testhelper.TestDB(ctx, t)
	pool2, err := pgxpool.NewWithConfig(ctx, dbPool.Config())
	require.NoError(t, err)

	t.Cleanup(func() {
		dbPool.Close()
		pool2.Close()
	})

	s, ds, _, k8sController := builder(t, ctx, dbPool)

	return s, ds, k8sController
}

func builder(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (*httptest.Server, dbaccess.DataSource, *testhelper.MockSuiteStore, *MockK8sController) {
	suiteStore := testhelper.NewMockSuiteStore()
	logger := testhelper.Logger(t)

	mockK8sController := new(MockK8sController)

	apiHandler := handler.NewAPIHandler(handler.NewAPIHandlerParams{
		Logger:          logger.WithGroup("APIHandler"),
		SourcePool:      pool,
		SuiteStore:      suiteStore,
		K8sController:   mockK8sController,
		AOAIEndpoint:    "--AOAI_ENDPOINT--",
		AOAIAPIKey:      "--AOAI_API_KEY--",
		AnthropicAPIKey: "--ANTHROPIC_API_KEY--",
	})
	err := apiHandler.Start(ctx)
	require.NoError(t, err)

	router := mux.NewRouter()
	middleware, cacheCloser := middleware.NewBearerAuthMiddleware(
		middleware.NewDatabaseApiTokenFetch(pool, ""),
		10*time.Second,
	)

	middlewares := []api.StrictMiddlewareFunc{middleware}
	options := api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			message, _ := json.Marshal(err.Error())
			http.Error(w, fmt.Sprintf(`{"error":%s}`, message), http.StatusBadRequest)
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			message, _ := json.Marshal(err.Error())
			http.Error(w, fmt.Sprintf(`{"error":%s}`, message), http.StatusInternalServerError)
		},
	}
	api.HandlerFromMux(api.NewStrictHandlerWithOptions(apiHandler, middlewares, options), router)

	server := httptest.NewServer(router)

	t.Cleanup(func() {
		apiHandler.Close(ctx)
		server.Close()
		cacheCloser()
	})

	return server, pool, suiteStore, mockK8sController
}
