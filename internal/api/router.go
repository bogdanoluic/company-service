package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(logger *slog.Logger) chi.Router {
	router := chi.NewRouter()

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("health check requested")

		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte("OK")); err != nil {
			logger.Error(
				"write health response",
				"error", err,
			)
		}
	})

	return router
}
