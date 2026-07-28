package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/bogdanoluic/company-service/internal/api"
	"github.com/bogdanoluic/company-service/internal/config"
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

	router := api.NewRouter(appLogger)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	appLogger.Info(
		"starting HTTP server",
		"address", addr,
		"configured_log_level", cfg.Log.Level, // debug or info :)
	)

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}

	return nil
}
