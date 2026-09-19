// Command server is the entrypoint of the MockHub API Mock service.
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

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/mockhub/mockhub/internal/config"
	"github.com/mockhub/mockhub/internal/handler"
	"github.com/mockhub/mockhub/internal/logger"
	"github.com/mockhub/mockhub/internal/model"
	"github.com/mockhub/mockhub/internal/repository"
	"github.com/mockhub/mockhub/internal/router"
	"github.com/mockhub/mockhub/internal/service"
)

func main() {
	logger := logger.New(os.Getenv("APP_ENV"))
	if err := run(logger); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := openDB(cfg, logger)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql db: %w", err)
	}
	defer sqlDB.Close()

	if err := db.AutoMigrate(
		&model.User{},
		&model.Project{},
		&model.MockAPI{},
		&model.MockAPIVersion{},
		&model.ResponseTemplate{},
		&model.RequestLog{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	seedSvc := service.NewSeedService(db, cfg.BaseURL, logger)
	if err := seedSvc.EnsureSeedData(context.Background()); err != nil {
		return fmt.Errorf("seed data: %w", err)
	}

	h := buildHandlers(cfg, db, logger)
	engine := router.New(cfg, logger, h)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "port", cfg.ServerPort, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		logger.Info("shutting down", "signal", sig.String())
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}
	return nil
}

func openDB(cfg *config.Config, logger *slog.Logger) (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	retries := cfg.DBConnectRetries
	if retries < 0 {
		retries = 0
	}
	for attempt := 0; ; attempt++ {
		db, err = gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Warn),
		})
		if err == nil {
			sqlDB, pingErr := db.DB()
			if pingErr == nil {
				if pingErr = sqlDB.Ping(); pingErr == nil {
					break
				}
			}
			err = pingErr
		}
		if attempt >= retries {
			return nil, fmt.Errorf("connect mysql after %d retries: %w", retries, err)
		}
		logger.Warn("database not ready, retrying", "attempt", attempt+1, "error", err)
		time.Sleep(time.Duration(cfg.DBConnectRetryIntervalSec) * time.Second)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxLifetimeMin) * time.Minute)
	return db, nil
}

func buildHandlers(cfg *config.Config, db *gorm.DB, logger *slog.Logger) *router.Handlers {
	userRepo := repository.NewUserRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	endpointRepo := repository.NewEndpointRepository(db)
	versionRepo := repository.NewEndpointVersionRepository(db)
	logRepo := repository.NewRequestLogRepository(db)

	authSvc := service.NewAuthService(cfg, userRepo, logger)
	projectSvc := service.NewProjectService(projectRepo, logger)
	endpointSvc := service.NewEndpointService(projectRepo, endpointRepo, versionRepo, userRepo, logger)
	logSvc := service.NewRequestLogService(projectRepo, logRepo, logger)
	mockEngine := service.NewMockEngine(endpointRepo, logRepo, logger)

	return &router.Handlers{
		Health:     handler.NewHealthHandler(db, logger),
		Auth:       handler.NewAuthHandler(authSvc, logger),
		Project:    handler.NewProjectHandler(projectSvc, cfg, logger),
		Endpoint:   handler.NewEndpointHandler(endpointSvc, logger),
		Mock:       handler.NewMockHandler(mockEngine, logger),
		RequestLog: handler.NewRequestLogHandler(logSvc, logger),
	}
}
