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

	"github.com/bogdanoluic/company-service/internal/api"
	"github.com/bogdanoluic/company-service/internal/config"
	"github.com/bogdanoluic/company-service/internal/database"
	"github.com/bogdanoluic/company-service/internal/logger"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	appLogger, err := logger.New(cfg.Log.Level)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	slog.SetDefault(appLogger)

	dbPool, err := database.NewPool(context.Background())
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	defer dbPool.Close()

	appLogger.Info("connected to PostgreSQL")

	router := api.NewRouter(appLogger)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErr := make(chan error, 1)

	appLogger.Info(
		"starting HTTP server",
		"address", addr,
		"configured_log_level", cfg.Log.Level,
	)

	go func() {
		err := server.ListenAndServe()

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("serve HTTP: %w", err)
			return
		}

		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		// Server is already finished.
		return err

	case <-signalCtx.Done():
		// Server is still running.
		// Restore the default signal behaviour.
		// A second Ctrl+C can forcefully terminate the application.
		stop()

		appLogger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	if err := <-serverErr; err != nil {
		return err
	}

	appLogger.Info("HTTP server stopped gracefully")

	return nil
}
