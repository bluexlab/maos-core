package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-playground/validator"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/handler"
)

const appName string = "maos-core-server"

type App struct {
	logger *slog.Logger
}

func (a *App) Run() {
	a.runServer()
}

func (a *App) runServer() {
	ctx := context.Background()

	// Load environment variables from .env file if exists
	if _, err := os.Stat(".env"); err == nil {
		a.logger.Info("Load environment variables from .env file")
		if err := godotenv.Load(".env"); err != nil {
			a.logger.Error("Error loading .env file: %v", err)
			os.Exit(1)
		}
	}

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

	// Connect to the database and create a new accessor
	pool, err := pgxpool.New(ctx, config.DatabaseUrl)
	if err != nil {
		a.logger.Error("Failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	accessor := dbaccess.New(pool)
	a.logger.Info("Connected to database", "database", config.DatabaseUrl)

	// Init Mux router and API handler
	router := mux.NewRouter()

	middlewares := []api.StrictMiddlewareFunc{}
	handler := handler.NewAPIHandler(a.logger.WithGroup("APIHandler"), accessor)
	err = handler.Start(ctx)
	if err != nil {
		a.logger.Error("Failed to initialize handler", "err", err)
		os.Exit(1)
	}

	defer handler.Close(ctx)

	api.HandlerFromMux(api.NewStrictHandler(handler, middlewares), router)

	a.logger.Info("Starting server", "port", config.Port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", config.Port), router)
	if err != nil {
		a.logger.Error("Server running error", "err", err)
		os.Exit(1)
	}
}
