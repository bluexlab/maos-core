package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-playground/validator"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelseyhightower/envconfig"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/handler"
	"gitlab.com/navyx/ai/maos/maos-core/internal/suitestore"
	"gitlab.com/navyx/ai/maos/maos-core/middleware"
	"gitlab.com/navyx/ai/maos/maos-core/migrate"
)

const appName string = "maos-core-server"
const bootstrapApiToken string = "bootstrap-token"

type LoggingQueryTracer struct {
	logger *slog.Logger
}

func (t *LoggingQueryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	traceId := ctx.Value("TRACEID")
	if strings.Contains(data.SQL, "api_tokens") {
		t.logger.Debug("Query started", "sql", data.SQL, "TraceId", traceId)
	} else {
		t.logger.Debug("Query started", "sql", data.SQL, "args", data.Args, "TraceId", traceId)
	}
	return ctx
}

func (t *LoggingQueryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	traceId := ctx.Value("TRACEID")
	t.logger.Debug("Query ended", "CommandTag", data.CommandTag, "duration", data.CommandTag, "TraceId", traceId)
}

type App struct {
	logger *slog.Logger
}

func (a *App) Run() {
	ctx := context.Background()

	config := a.loadConfig()

	// Connect to the database and create a new accessor
	dbConfig, err := pgxpool.ParseConfig(config.DatabaseUrl)
	if err != nil {
		a.logger.Error("Failed to parse connection string", "err", err)
		os.Exit(1)
	}

	dbConfig.ConnConfig.Tracer = &LoggingQueryTracer{a.logger}
	pool, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		a.logger.Error("Failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	accessor := dbaccess.New(pool)
	a.logger.Info("Connected to database", "database", config.DatabaseUrl)

	// Init Mux router and API handler
	router := mux.NewRouter()

	// Init auth middleware and token cache
	middleware, cacheCloser := middleware.NewBearerAuthMiddleware(
		middleware.NewDatabaseApiTokenFetch(accessor, bootstrapApiToken),
		10*time.Second,
	)
	defer cacheCloser()

	middlewares := []api.StrictMiddlewareFunc{middleware}
	options := api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			message, _ := json.Marshal(err.Error())
			a.logger.Debug("Bad request error", "url", r.RequestURI, "err", message)
			http.Error(w, fmt.Sprintf(`{"error":%s}`, message), http.StatusBadRequest)
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			message, _ := json.Marshal(err.Error())
			a.logger.Debug("Internal server error", "url", r.RequestURI, "err", message)
			http.Error(w, fmt.Sprintf(`{"error":%s}`, message), http.StatusInternalServerError)
		},
	}

	// Create S3 client with AWS credentials
	s3Client, err := a.createS3Client(config)
	if err != nil {
		a.logger.Error("Failed to create S3 client", "err", err)
		os.Exit(1)
	}
	suiteStore := suitestore.NewS3SuiteStore(a.logger.WithGroup("SuiteStore"), s3Client, config.SuiteStoreBucket, config.SuiteStorePrefix, config.ClusterName, accessor, 10*time.Second)
	suiteStore.StartBackgroundScanner(ctx)
	defer suiteStore.StopAndWaitForScannerToStop(10 * time.Second)

	apiHandler := handler.NewAPIHandler(a.logger.WithGroup("APIHandler"), accessor, suiteStore)
	err = apiHandler.Start(ctx)
	if err != nil {
		a.logger.Error("Failed to initialize handler", "err", err)
		os.Exit(1)
	}

	defer apiHandler.Close(ctx)

	api.HandlerWithOptions(
		api.NewStrictHandlerWithOptions(apiHandler, middlewares, options),
		api.GorillaServerOptions{
			BaseRouter: router,
			ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
				message, _ := json.Marshal(err.Error())
				a.logger.Debug("Bad request error", "url", r.RequestURI, "err", err.Error())
				http.Error(w, fmt.Sprintf(`{"error":%s}`, message), http.StatusBadRequest)
			},
		},
	)
	api.HandlerFromMux(api.NewStrictHandlerWithOptions(apiHandler, middlewares, options), router)

	a.logger.Info("Starting server", "port", config.Port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", config.Port), router)
	if err != nil {
		a.logger.Error("Server running error", "err", err)
		os.Exit(1)
	}
}

func (a *App) Migrate() {
	ctx := context.Background()

	config := a.loadConfig()

	// Connect to the database and create a new accessor
	dbConfig, err := pgxpool.ParseConfig(config.DatabaseUrl)
	if err != nil {
		a.logger.Error("Failed to parse connection string", "err", err)
		os.Exit(1)
	}

	dbConfig.ConnConfig.Tracer = &LoggingQueryTracer{a.logger}
	pool, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		a.logger.Error("Failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	accessor := dbaccess.New(pool)

	_, err = migrate.New(accessor, nil).Migrate(ctx, migrate.DirectionUp, &migrate.MigrateOpts{})
	if err != nil {
		a.logger.Error("Failed to migrate db", "url", config.DatabaseUrl, "error", err.Error())
		os.Exit(2)
	}

	a.logger.Info("maos-core Database migrated")
}

func (a *App) loadConfig() Config {
	// Load environment variables into the struct
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		a.logger.Error("Failed to process environment variables.", "err", err)
		os.Exit(1)
	}

	// Validate the struct
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		a.logger.Error("Validation failed: %v", err)
		os.Exit(1)
	}

	// construct database url from the environment variables
	if config.DatabaseUrl == "" {
		config.DatabaseUrl = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s",
			config.DatabaseUser,
			config.DatabasePassword,
			config.DatabaseHost,
			config.DatabasePort,
			config.DatabaseName)
	}

	return config
}

func (a *App) createS3Client(systemConfig Config) (*s3.Client, error) {
	creds := credentials.NewStaticCredentialsProvider(systemConfig.AWSAccessKeyID, systemConfig.AWSSecretAccessKey, "")
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(systemConfig.AWSRegion),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return s3.NewFromConfig(cfg), nil
}
