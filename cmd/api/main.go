package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Hisaalu/rogele/internal/auth"
	"github.com/Hisaalu/rogele/internal/config"
	"github.com/Hisaalu/rogele/internal/db"
	"github.com/Hisaalu/rogele/internal/logger"
	"github.com/Hisaalu/rogele/internal/repository"
	"github.com/Hisaalu/rogele/internal/repository/postgres"
	"github.com/Hisaalu/rogele/internal/server"
	"github.com/Hisaalu/rogele/internal/service"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg.App.Env)
	log.Info("starting rogele api", "env", cfg.App.Env, "version", version)

	ctx := context.Background()

	database, err := db.NewPostgres(ctx, cfg.Database, log)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Error("closing db", "error", err)
		}
	}()

	repos := &repository.Repositories{
		User: postgres.NewUserRepo(database),
	}

	tokenManager := auth.NewTokenManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
	)

	authSvc := service.NewAuthService(repos.User, tokenManager)

	router := server.NewRouter(server.RouterDeps{
		Log:     log,
		Version: version,
		DB:      database,
		Repos:   repos,
		Tokens:  tokenManager,
		AuthSvc: authSvc,
	})

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	srv := server.New(addr, router, log)

	return srv.Run(ctx)
}
