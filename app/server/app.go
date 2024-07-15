package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-playground/validator"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"gitlab.com/navyx/ai/maos/maos-core/pkg/api"
	"gitlab.com/navyx/ai/maos/maos-core/pkg/handler"
)

const appName string = "maos-core-server"

type App struct{}

func (a *App) Run() {
	a.runServer()
}

func (a *App) runServer() {
	// ctx := context.Background()

	// Load environment variables from .env file if exists
	if _, err := os.Stat(".env"); err == nil {
		logrus.Infof("Load environment variables from .env file")
		if err := godotenv.Load(".env"); err != nil {
			logrus.Fatalf("Error loading .env file: %v", err)
		}
	}

	// Load environment variables into the struct
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		logrus.Fatalf("Failed to process environment variables: %v", err)
	}

	// Validate the struct
	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		logrus.Fatalf("Validation failed: %v", err)
	}

	// Init Mux router and API handler
	router := mux.NewRouter()

	handler := &handler.APIHandler{}
	api.HandlerFromMux(api.NewStrictHandler(handler, nil), router)

	logrus.Infof("Starting %s server on pot %d", appName, cfg.Port)
	logrus.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), router))
}
