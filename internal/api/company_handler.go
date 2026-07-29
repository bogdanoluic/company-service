package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bogdanoluic/company-service/internal/company"
	"github.com/google/uuid"
)

type CompanyService interface {
	Create(context.Context, company.CreateInput) (company.Company, error)
	GetByID(context.Context, uuid.UUID) (company.Company, error)
	Patch(context.Context, uuid.UUID, company.PatchInput) (company.Company, error)
	Delete(context.Context, uuid.UUID) error
}

var _ CompanyService = (*company.Service)(nil)

type CompanyHandler struct {
	service CompanyService
	logger  *slog.Logger
}

func NewCompanyHandler(
	service CompanyService,
	logger *slog.Logger,
) *CompanyHandler {
	return &CompanyHandler{
		service: service,
		logger:  logger,
	}
}

type createCompanyRequest struct {
	Name              *string       `json:"name"`
	Description       *string       `json:"description"`
	AmountOfEmployees *int          `json:"amount_of_employees"`
	Registered        *bool         `json:"registered"`
	Type              *company.Type `json:"type"`
}

type companyResponse struct {
	ID                uuid.UUID    `json:"id"`
	Name              string       `json:"name"`
	Description       *string      `json:"description"`
	AmountOfEmployees int          `json:"amount_of_employees"`
	Registered        bool         `json:"registered"`
	Type              company.Type `json:"type"`
}

func (h *CompanyHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request createCompanyRequest

	if err := decodeJSON(w, r, &request); err != nil {
		h.logger.Debug(
			"invalid create company request body",
			"error", err,
		)

		h.respondError(
			w,
			r,
			http.StatusBadRequest,
			"invalid request body",
			"",
		)

		return
	}

	input, validationErr := request.input()
	if validationErr != nil {
		h.respondError(
			w,
			r,
			http.StatusBadRequest,
			validationErr.Message,
			validationErr.Field,
		)

		return
	}

	created, err := h.service.Create(r.Context(), input)
	if err != nil {
		h.respondCompanyError(w, r, err)
		return
	}

	w.Header().Set(
		"Location",
		"/companies/"+created.ID.String(),
	)

	h.respondJSON(
		w,
		r,
		http.StatusCreated,
		newCompanyResponse(created),
	)
}

func (r createCompanyRequest) input() (
	company.CreateInput,
	*company.ValidationError,
) {
	if r.Name == nil {
		return company.CreateInput{}, &company.ValidationError{
			Field:   "name",
			Message: "is required",
		}
	}

	if r.AmountOfEmployees == nil {
		return company.CreateInput{}, &company.ValidationError{
			Field:   "amount_of_employees",
			Message: "is required",
		}
	}

	if r.Registered == nil {
		return company.CreateInput{}, &company.ValidationError{
			Field:   "registered",
			Message: "is required",
		}
	}

	if r.Type == nil {
		return company.CreateInput{}, &company.ValidationError{
			Field:   "type",
			Message: "is required",
		}
	}

	return company.CreateInput{
		Name:              *r.Name,
		Description:       r.Description,
		AmountOfEmployees: *r.AmountOfEmployees,
		Registered:        *r.Registered,
		Type:              *r.Type,
	}, nil
}

func newCompanyResponse(c company.Company) companyResponse {
	return companyResponse{
		ID:                c.ID,
		Name:              c.Name,
		Description:       c.Description,
		AmountOfEmployees: c.AmountOfEmployees,
		Registered:        c.Registered,
		Type:              c.Type,
	}
}

func (h *CompanyHandler) respondCompanyError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	var validationErr *company.ValidationError

	switch {
	case errors.As(err, &validationErr):
		h.respondError(
			w,
			r,
			http.StatusBadRequest,
			validationErr.Message,
			validationErr.Field,
		)

	case errors.Is(err, company.ErrNameAlreadyExists):
		h.respondError(
			w,
			r,
			http.StatusConflict,
			"company name already exists",
			"name",
		)

	case errors.Is(err, company.ErrNotFound):
		h.respondError(
			w,
			r,
			http.StatusNotFound,
			"company not found",
			"",
		)

	default:
		h.logger.Error(
			"company request failed",
			"method", r.Method,
			"path", r.URL.Path,
			"error", err,
		)

		h.respondError(
			w,
			r,
			http.StatusInternalServerError,
			"internal server error",
			"",
		)
	}
}

func (h *CompanyHandler) respondJSON(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	value any,
) {
	if err := writeJSON(w, status, value); err != nil {
		h.logger.Error(
			"failed to write JSON response",
			"method", r.Method,
			"path", r.URL.Path,
			"error", err,
		)
	}
}

func (h *CompanyHandler) respondError(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	message string,
	field string,
) {
	if err := writeError(w, status, message, field); err != nil {
		h.logger.Error(
			"failed to write error response",
			"method", r.Method,
			"path", r.URL.Path,
			"error", err,
		)
	}
}
