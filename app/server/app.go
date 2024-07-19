package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-playground/validator"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/handler"
)

const appName string = "maos-core-server"

type App struct{}

func (a *App) Run() {
	a.runServer()
}

func (a *App) runServer() {
	ctx := context.Background()

	// Load environment variables from .env file if exists
	if _, err := os.Stat(".env"); err == nil {
		logrus.Infof("Load environment variables from .env file")
		if err := godotenv.Load(".env"); err != nil {
			logrus.Fatalf("Error loading .env file: %v", err)
		}
	}

	// Load environment variables into the struct
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		logrus.Fatalf("Failed to process environment variables: %v", err)
	}

	// Validate the struct
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		logrus.Fatalf("Validation failed: %v", err)
	}

	// Connect to the database and create a new accessor
	db, err := pgx.Connect(ctx, config.DatabaseUrl)
	if err != nil {
		logrus.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close(ctx)

	accessor := dbaccess.New(db)
	logrus.Infof("Connected to database: %v", accessor)

	// Init Mux router and API handler
	router := mux.NewRouter()

	middlewares := []api.StrictMiddlewareFunc{}
	handler := handler.NewAPIHandler(accessor)
	api.HandlerFromMux(api.NewStrictHandler(handler, middlewares), router)

	logrus.Infof("Starting %s server on pot %d", appName, config.Port)
	logrus.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), router))
}
