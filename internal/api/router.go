package api

import (
	"log/slog"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

func NewRouter(
	logger *slog.Logger,
	healthHandler *HealthHandler,
	companyHandler *CompanyHandler,
	authMiddleware *AuthMiddleware,
) chi.Router {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)

	router.Get("/health", healthHandler.Live)
	router.Get("/health/live", healthHandler.Live)
	router.Get("/health/ready", healthHandler.Ready)

	router.Get("/companies/{id}", companyHandler.GetByID)

	router.Group(func(router chi.Router) {
		router.Use(authMiddleware.Authenticate)

		router.Post("/companies", companyHandler.Create)
		router.Patch("/companies/{id}", companyHandler.Patch)
		router.Delete("/companies/{id}", companyHandler.Delete)
	})

	return router
}
