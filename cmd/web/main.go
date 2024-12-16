package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
)

type config = struct {
	mockOpenAI bool
	addr       string
	appUri     string
}

type application struct {
	logger *slog.Logger
	config config
}

func main() {
	var cfg config

	flag.StringVar(&cfg.addr, "addr", ":4000", "HTTP network address")
	flag.StringVar(&cfg.appUri, "appUri", "http://localhost:4000", "The application URI")
	flag.BoolVar(&cfg.mockOpenAI, "mockOpenAI", true, "Mock OpenAI requests with local file outputs")

	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}))

	app := &application{
		logger: logger,
		config: cfg,
	}

	logger.Info("starting server", slog.String("addr", app.config.addr))
	logger.Info("Mocking OpenAI Requests: ", slog.Bool("mockOpenAI", app.config.mockOpenAI))
	logger.Info("Application URL: ", slog.String("appUri", app.config.appUri))

	err := http.ListenAndServe(app.config.addr, app.routes())

	logger.Error(err.Error())
	os.Exit(1)
}
