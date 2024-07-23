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
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/handler"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
	"gitlab.com/navyx/ai/maos/maos-core/middleware"
)

type ServerCloser func()

// SetupHttpTestWithDb sets up a test server and database accessor.
// user has to call the returned ServerCloser to stop the server after the test.
func SetupHttpTestWithDb(t *testing.T, ctx context.Context) (*httptest.Server, dbaccess.Accessor, ServerCloser) {
	dbPool := testhelper.TestDB(ctx, t)
	accessor := dbaccess.New(dbPool)

	apiHandler := handler.NewAPIHandler(accessor)
	router := mux.NewRouter()
	middleware, cacheCloser := middleware.NewBearerAuthMiddleware(
		middleware.NewDatabaseApiTokenFetch(accessor),
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

	closer := func() {
		server.Close()
		dbPool.Close()
		cacheCloser()
	}

	return server, accessor, closer
}
