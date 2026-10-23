package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/encourage13/url-short/db/migrations"
	"github.com/encourage13/url-short/internal/config"
	"github.com/encourage13/url-short/internal/handler"
	"github.com/encourage13/url-short/internal/repository"
	"github.com/encourage13/url-short/internal/service"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if cfg.AdminToken == "" {
		log.Warn("ADMIN_TOKEN не задан: GET /urls отключен")
	}

	repo, closeRepo, err := newRepository(ctx, cfg, log)
	if err != nil {
		return err
	}
	defer closeRepo()

	urlService := service.NewURLService(repo, cfg.CodeLength)
	urlHandler := handler.NewURLHandler(urlService, cfg.BaseURL, cfg.AdminToken, log)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           urlHandler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("server started", "addr", srv.Addr, "base_url", cfg.BaseURL, "db", cfg.DatabaseType)
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-serverErr; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func newRepository(ctx context.Context, cfg config.Config, log *slog.Logger) (repository.URLRepository, func(), error) {
	if cfg.DatabaseType != "postgres" {
		log.Info("using in-memory storage")
		return repository.NewInMemoryRepository(), func() {}, nil
	}

	db, err := gorm.Open(postgres.Open(cfg.Postgres.DSN()), &gorm.Config{
		TranslateError: true,
		Logger:         logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("connect to postgres: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	closeDB := func() { sqlDB.Close() }

	migrator, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		closeDB()
		return nil, nil, fmt.Errorf("migrations: %w", err)
	}
	results, err := migrator.Up(ctx)
	if err != nil {
		closeDB()
		return nil, nil, fmt.Errorf("apply migrations: %w", err)
	}
	for _, r := range results {
		log.Info("migration applied", "source", r.Source.Path, "duration", r.Duration)
	}

	log.Info("connected to postgres", "host", cfg.Postgres.Host, "db", cfg.Postgres.DBName)
	return repository.NewGormRepository(db), closeDB, nil
}
