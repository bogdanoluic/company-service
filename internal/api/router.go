package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(
	logger *slog.Logger,
	companyHandler *CompanyHandler,
) chi.Router {
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

	router.Post("/companies", companyHandler.Create)
	router.Get("/companies/{id}", companyHandler.GetByID)
	router.Patch("/companies/{id}", companyHandler.Patch)
	router.Delete("/companies/{id}", companyHandler.Delete)

	return router
}
