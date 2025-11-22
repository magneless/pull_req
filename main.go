package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"pull_req/config"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()
	cfg := config.MustLoad(configPath)

	log := mustMakeLogger(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting server")
	log.Debug("debug messages are enabled")

	mux := http.NewServeMux()
	mux.Handle("POST /team/add", )
	mux.Handle("GET /team/get", )

	mux.Handle("POST /users/selectIsActive", )
	
	mux.Handle("POST /pullRequest/create", )
	mux.Handle("POST /pullRequest/merge", )
	mux.Handle("POST /pullRequest/reassign", )
	mux.Handle("GET /users/getReview", )
	return nil
}

func mustMakeLogger(logLevel string) *slog.Logger {
	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown log level: " + logLevel)
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
