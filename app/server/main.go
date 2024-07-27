package main

import (
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	app := &App{logger}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		app.Migrate()
	} else {
		app.Run()
	}
}
