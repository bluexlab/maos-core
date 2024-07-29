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
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/handler"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
	"gitlab.com/navyx/ai/maos/maos-core/middleware"
)

// SetupHttpTestWithDb sets up two test servers and database accessors.
// It can simulate two running services in HA mode.
func SetupHttpTestWithDb(t *testing.T, ctx context.Context) (*httptest.Server, dbaccess.Accessor, *httptest.Server) {
	builder := func(pool *pgxpool.Pool) (*httptest.Server, *dbaccess.PgAccessor) {
		accessor := dbaccess.New(pool)

		logger := testhelper.Logger(t)
		apiHandler := handler.NewAPIHandler(logger.WithGroup("APIHandler"), accessor)
		err := apiHandler.Start(ctx)
		require.NoError(t, err)

		router := mux.NewRouter()
		middleware, cacheCloser := middleware.NewBearerAuthMiddleware(
			middleware.NewDatabaseApiTokenFetch(accessor, ""),
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

		return server, accessor
	}

	dbPool := testhelper.TestDB(ctx, t)
	pool2, err := pgxpool.NewWithConfig(ctx, dbPool.Config())
	require.NoError(t, err)

	t.Cleanup(func() {
		dbPool.Close()
		pool2.Close()
	})

	s, a := builder(dbPool)
	s2, _ := builder(pool2)

	return s, a, s2
}
