package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"pull_req/pull_req/adapters/rest"
	"pull_req/pull_req/config"
	"pull_req/pull_req/core"
	"pull_req/pull_req/adapters/db"
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

	storage, err := db.NewDB(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to create db: %v", err)
	}
	teamDB := db.NewTeamDB(storage)
	teamService := core.NewTeamService(log, teamDB)

	userDB := db.NewUserDB(storage)
	userService := core.NewUserService(log, userDB)

	prDB := db.NewPRDB(storage)
	prService := core.NewPRService(log, prDB)

	mux := http.NewServeMux()
	mux.Handle("POST /team/add", rest.NewAddTeamHandler(log, teamService))
	mux.Handle("GET /team/get", rest.NewGetTeamHandler(log, teamService))

	mux.Handle("POST /users/setIsActive", rest.NewSetIsActiveHandler(log, userService))

	mux.Handle("POST /pullRequest/create", rest.NewCreatePRHandler(log, prService))
	mux.Handle("POST /pullRequest/merge", rest.NewMergePRHandler(log, prService))
	mux.Handle("POST /pullRequest/reassign", rest.NewReassignPRHandler(log, prService))
	mux.Handle("GET /users/getReview", rest.NewGetReviewHandler(log, prService))

	server := http.Server{
		Addr:        cfg.HTTPConfig.Address,
		ReadTimeout: cfg.HTTPConfig.Timeout,
		Handler:     mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error("erroneous shutdown", "error", err)
		}
	}()

	log.Info("Running HTTP server", "address", cfg.HTTPConfig.Address)
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server closed unexpectedly: %v", err)
		}
	}

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
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level, AddSource: true})
	return slog.New(handler)
}
