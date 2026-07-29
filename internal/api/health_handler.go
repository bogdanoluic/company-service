package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type DatabasePinger interface {
	Ping(context.Context) error
}

type HealthHandler struct {
	database DatabasePinger
	logger   *slog.Logger
}

func NewHealthHandler(
	database DatabasePinger,
	logger *slog.Logger,
) *HealthHandler {
	return &HealthHandler{
		database: database,
		logger:   logger,
	}
}

type healthResponse struct {
	Status string `json:"status"`
}

func (h *HealthHandler) Live(
	w http.ResponseWriter,
	r *http.Request,
) {
	if err := writeJSON(
		w,
		http.StatusOK,
		healthResponse{Status: "ok"},
	); err != nil {
		h.logger.Error(
			"failed to write liveness response",
			"error", err,
		)
	}
}

func (h *HealthHandler) Ready(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx, cancel := context.WithTimeout(
		r.Context(),
		2*time.Second,
	)
	defer cancel()

	if err := h.database.Ping(ctx); err != nil {
		h.logger.Error(
			"readiness check failed",
			"error", err,
		)

		if writeErr := writeJSON(
			w,
			http.StatusServiceUnavailable,
			healthResponse{Status: "unavailable"},
		); writeErr != nil {
			h.logger.Error(
				"failed to write readiness response",
				"error", writeErr,
			)
		}

		return
	}

	if err := writeJSON(
		w,
		http.StatusOK,
		healthResponse{Status: "ok"},
	); err != nil {
		h.logger.Error(
			"failed to write readiness response",
			"error", err,
		)
	}
}
