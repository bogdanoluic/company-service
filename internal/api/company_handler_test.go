package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bogdanoluic/company-service/internal/company"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type fakeCompanyService struct {
	createFn func(
		context.Context,
		company.CreateInput,
	) (company.Company, error)

	getByIDFn func(
		context.Context,
		uuid.UUID,
	) (company.Company, error)

	patchFn func(
		context.Context,
		uuid.UUID,
		company.PatchInput,
	) (company.Company, error)

	deleteFn func(
		context.Context,
		uuid.UUID,
	) error
}

var _ CompanyService = (*fakeCompanyService)(nil)

func (s *fakeCompanyService) Create(
	ctx context.Context,
	input company.CreateInput,
) (company.Company, error) {
	if s.createFn == nil {
		panic("unexpected call to Create")
	}

	return s.createFn(ctx, input)
}

func (s *fakeCompanyService) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (company.Company, error) {
	if s.getByIDFn == nil {
		panic("unexpected call to GetByID")
	}

	return s.getByIDFn(ctx, id)
}

func (s *fakeCompanyService) Patch(
	ctx context.Context,
	id uuid.UUID,
	input company.PatchInput,
) (company.Company, error) {
	if s.patchFn == nil {
		panic("unexpected call to Patch")
	}

	return s.patchFn(ctx, id, input)
}

func (s *fakeCompanyService) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	if s.deleteFn == nil {
		panic("unexpected call to Delete")
	}

	return s.deleteFn(ctx, id)
}

func TestCompanyHandlerCreate(t *testing.T) {
	t.Run("creates company", func(t *testing.T) {
		id := uuid.MustParse(
			"5ec4a25a-793f-43a1-8912-f9d1ea163117",
		)
		description := "Software company"

		serviceCalled := false

		service := &fakeCompanyService{
			createFn: func(
				_ context.Context,
				input company.CreateInput,
			) (company.Company, error) {
				serviceCalled = true

				if input.Name != "XM" {
					t.Errorf(
						"Create() name = %q, want %q",
						input.Name,
						"XM",
					)
				}

				if input.Description == nil {
					t.Fatal("Create() description = nil")
				}

				if *input.Description != description {
					t.Errorf(
						"Create() description = %q, want %q",
						*input.Description,
						description,
					)
				}

				// These assertions prove that explicitly provided
				// zero and false values are not treated as missing.
				if input.AmountOfEmployees != 0 {
					t.Errorf(
						"Create() amount of employees = %d, want 0",
						input.AmountOfEmployees,
					)
				}

				if input.Registered {
					t.Error("Create() registered = true, want false")
				}

				if input.Type != company.TypeCorporations {
					t.Errorf(
						"Create() type = %q, want %q",
						input.Type,
						company.TypeCorporations,
					)
				}

				return company.Company{
					ID:                id,
					Name:              input.Name,
					Description:       input.Description,
					AmountOfEmployees: input.AmountOfEmployees,
					Registered:        input.Registered,
					Type:              input.Type,
				}, nil
			},
		}

		handler := newTestCompanyHandler(service)

		request := httptest.NewRequest(
			http.MethodPost,
			"/companies",
			strings.NewReader(`{
				"name": "XM",
				"description": "Software company",
				"amount_of_employees": 0,
				"registered": false,
				"type": "Corporations"
			}`),
		)
		recorder := httptest.NewRecorder()

		handler.Create(recorder, request)

		if !serviceCalled {
			t.Fatal("service Create() was not called")
		}

		if recorder.Code != http.StatusCreated {
			t.Fatalf(
				"status = %d, want %d",
				recorder.Code,
				http.StatusCreated,
			)
		}

		if got := recorder.Header().Get("Location"); got != "/companies/"+id.String() {
			t.Errorf(
				"Location = %q, want %q",
				got,
				"/companies/"+id.String(),
			)
		}

		if got := recorder.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf(
				"Content-Type = %q, want %q",
				got,
				"application/json",
			)
		}

		var response companyResponse

		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if response.ID != id {
			t.Errorf("response ID = %s, want %s", response.ID, id)
		}

		if response.Name != "XM" {
			t.Errorf(
				"response name = %q, want %q",
				response.Name,
				"XM",
			)
		}

		if response.Description == nil {
			t.Fatal("response description = nil")
		}

		if *response.Description != description {
			t.Errorf(
				"response description = %q, want %q",
				*response.Description,
				description,
			)
		}

		if response.AmountOfEmployees != 0 {
			t.Errorf(
				"response amount of employees = %d, want 0",
				response.AmountOfEmployees,
			)
		}

		if response.Registered {
			t.Error("response registered = true, want false")
		}

		if response.Type != company.TypeCorporations {
			t.Errorf(
				"response type = %q, want %q",
				response.Type,
				company.TypeCorporations,
			)
		}
	})

	t.Run("rejects invalid JSON bodies", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{
				name: "malformed JSON",
				body: `{"name":`,
			},
			{
				name: "unknown field",
				body: `{
					"name": "XM",
					"amount_of_employees": 10,
					"registered": true,
					"type": "Corporations",
					"unexpected": true
				}`,
			},
			{
				name: "multiple JSON values",
				body: `{
					"name": "XM",
					"amount_of_employees": 10,
					"registered": true,
					"type": "Corporations"
				}
				{
					"name": "Trading.com"
				}`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				handler := newTestCompanyHandler(
					&fakeCompanyService{},
				)

				request := httptest.NewRequest(
					http.MethodPost,
					"/companies",
					strings.NewReader(tt.body),
				)
				recorder := httptest.NewRecorder()

				handler.Create(recorder, request)

				assertErrorResponse(
					t,
					recorder,
					http.StatusBadRequest,
					"invalid request body",
					"",
				)
			})
		}
	})

	t.Run("rejects missing required fields", func(t *testing.T) {
		tests := []struct {
			name      string
			body      string
			wantField string
		}{
			{
				name: "missing name",
				body: `{
					"amount_of_employees": 10,
					"registered": true,
					"type": "Corporations"
				}`,
				wantField: "name",
			},
			{
				name: "missing amount of employees",
				body: `{
					"name": "XM",
					"registered": true,
					"type": "Corporations"
				}`,
				wantField: "amount_of_employees",
			},
			{
				name: "missing registered",
				body: `{
					"name": "XM",
					"amount_of_employees": 10,
					"type": "Corporations"
				}`,
				wantField: "registered",
			},
			{
				name: "missing type",
				body: `{
					"name": "XM",
					"amount_of_employees": 10,
					"registered": true
				}`,
				wantField: "type",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				handler := newTestCompanyHandler(
					&fakeCompanyService{},
				)

				request := httptest.NewRequest(
					http.MethodPost,
					"/companies",
					strings.NewReader(tt.body),
				)
				recorder := httptest.NewRecorder()

				handler.Create(recorder, request)

				assertErrorResponse(
					t,
					recorder,
					http.StatusBadRequest,
					"is required",
					tt.wantField,
				)
			})
		}
	})

	t.Run("maps service errors", func(t *testing.T) {
		tests := []struct {
			name        string
			serviceErr  error
			wantStatus  int
			wantMessage string
			wantField   string
		}{
			{
				name: "validation error",
				serviceErr: &company.ValidationError{
					Field:   "name",
					Message: "must not exceed 15 characters",
				},
				wantStatus:  http.StatusBadRequest,
				wantMessage: "must not exceed 15 characters",
				wantField:   "name",
			},
			{
				name:        "duplicate name",
				serviceErr:  company.ErrNameAlreadyExists,
				wantStatus:  http.StatusConflict,
				wantMessage: "company name already exists",
				wantField:   "name",
			},
			{
				name:        "unexpected error",
				serviceErr:  errors.New("database unavailable"),
				wantStatus:  http.StatusInternalServerError,
				wantMessage: "internal server error",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				service := &fakeCompanyService{
					createFn: func(
						context.Context,
						company.CreateInput,
					) (company.Company, error) {
						return company.Company{}, tt.serviceErr
					},
				}

				handler := newTestCompanyHandler(service)

				request := httptest.NewRequest(
					http.MethodPost,
					"/companies",
					strings.NewReader(`{
						"name": "XM",
						"amount_of_employees": 10,
						"registered": true,
						"type": "Corporations"
					}`),
				)
				recorder := httptest.NewRecorder()

				handler.Create(recorder, request)

				assertErrorResponse(
					t,
					recorder,
					tt.wantStatus,
					tt.wantMessage,
					tt.wantField,
				)
			})
		}
	})
}
func TestCompanyHandlerGetByID(t *testing.T) {
	t.Run("returns company", func(t *testing.T) {
		id := uuid.New()

		want := company.Company{
			ID:                id,
			Name:              "Acme",
			AmountOfEmployees: 10,
			Registered:        true,
			Type:              company.TypeCorporations,
		}

		service := &fakeCompanyService{
			getByIDFn: func(
				_ context.Context,
				gotID uuid.UUID,
			) (company.Company, error) {
				if gotID != id {
					t.Errorf("ID = %s, want %s", gotID, id)
				}

				return want, nil
			},
		}

		handler := newTestCompanyHandler(service)

		request := httptest.NewRequest(
			http.MethodGet,
			"/companies/"+id.String(),
			nil,
		)

		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("id", id.String())

		request = request.WithContext(
			context.WithValue(
				request.Context(),
				chi.RouteCtxKey,
				routeContext,
			),
		)

		recorder := httptest.NewRecorder()

		handler.GetByID(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf(
				"status = %d, want %d",
				recorder.Code,
				http.StatusOK,
			)
		}

		var response companyResponse

		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if response.ID != id {
			t.Errorf("ID = %s, want %s", response.ID, id)
		}
	})

	t.Run("rejects invalid ID", func(t *testing.T) {
		handler := newTestCompanyHandler(&fakeCompanyService{})

		request := requestWithCompanyID(
			http.MethodGet,
			"/companies/not-a-uuid",
			"not-a-uuid",
			nil,
		)
		recorder := httptest.NewRecorder()

		handler.GetByID(recorder, request)

		assertErrorResponse(
			t,
			recorder,
			http.StatusBadRequest,
			"invalid company ID",
			"id",
		)
	})

	t.Run("returns not found", func(t *testing.T) {
		id := uuid.New()

		service := &fakeCompanyService{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (company.Company, error) {
				return company.Company{}, company.ErrNotFound
			},
		}

		handler := newTestCompanyHandler(service)

		request := requestWithCompanyID(
			http.MethodGet,
			"/companies/"+id.String(),
			id.String(),
			nil,
		)
		recorder := httptest.NewRecorder()

		handler.GetByID(recorder, request)

		assertErrorResponse(
			t,
			recorder,
			http.StatusNotFound,
			"company not found",
			"",
		)
	})
}

func TestCompanyHandlerPatch(t *testing.T) {
	t.Run("patches provided fields", func(t *testing.T) {
		id := uuid.New()

		service := &fakeCompanyService{
			patchFn: func(
				_ context.Context,
				gotID uuid.UUID,
				input company.PatchInput,
			) (company.Company, error) {
				if gotID != id {
					t.Errorf("ID = %s, want %s", gotID, id)
				}

				if input.Name.Set {
					t.Error("Name.Set = true, want false")
				}

				if !input.Description.Set {
					t.Error("Description.Set = false, want true")
				}

				if input.Description.Value != nil {
					t.Error("Description.Value != nil, want nil")
				}

				if !input.AmountOfEmployees.Set {
					t.Error(
						"AmountOfEmployees.Set = false, want true",
					)
				}

				if input.AmountOfEmployees.Value != 0 {
					t.Errorf(
						"AmountOfEmployees.Value = %d, want 0",
						input.AmountOfEmployees.Value,
					)
				}

				if !input.Registered.Set {
					t.Error("Registered.Set = false, want true")
				}

				if input.Registered.Value {
					t.Error("Registered.Value = true, want false")
				}

				return company.Company{
					ID:                id,
					Name:              "Acme",
					Description:       nil,
					AmountOfEmployees: 0,
					Registered:        false,
					Type:              company.TypeCorporations,
				}, nil
			},
		}

		handler := newTestCompanyHandler(service)

		request := requestWithCompanyID(
			http.MethodPatch,
			"/companies/"+id.String(),
			id.String(),
			strings.NewReader(`{
				"description": null,
				"amount_of_employees": 0,
				"registered": false
			}`),
		)
		recorder := httptest.NewRecorder()

		handler.Patch(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf(
				"status = %d, want %d; body = %s",
				recorder.Code,
				http.StatusOK,
				recorder.Body.String(),
			)
		}
	})

	t.Run("maps empty patch validation error", func(t *testing.T) {
		id := uuid.New()

		service := &fakeCompanyService{
			patchFn: func(
				context.Context,
				uuid.UUID,
				company.PatchInput,
			) (company.Company, error) {
				return company.Company{}, &company.ValidationError{
					Field:   "body",
					Message: "at least one field must be provided",
				}
			},
		}

		handler := newTestCompanyHandler(service)

		request := requestWithCompanyID(
			http.MethodPatch,
			"/companies/"+id.String(),
			id.String(),
			strings.NewReader(`{}`),
		)
		recorder := httptest.NewRecorder()

		handler.Patch(recorder, request)

		assertErrorResponse(
			t,
			recorder,
			http.StatusBadRequest,
			"at least one field must be provided",
			"body",
		)
	})
}

func TestCompanyHandlerDelete(t *testing.T) {
	t.Run("deletes company", func(t *testing.T) {
		id := uuid.New()

		service := &fakeCompanyService{
			deleteFn: func(
				_ context.Context,
				gotID uuid.UUID,
			) error {
				if gotID != id {
					t.Errorf("ID = %s, want %s", gotID, id)
				}

				return nil
			},
		}

		handler := newTestCompanyHandler(service)

		request := requestWithCompanyID(
			http.MethodDelete,
			"/companies/"+id.String(),
			id.String(),
			nil,
		)
		recorder := httptest.NewRecorder()

		handler.Delete(recorder, request)

		if recorder.Code != http.StatusNoContent {
			t.Fatalf(
				"status = %d, want %d",
				recorder.Code,
				http.StatusNoContent,
			)
		}

		if recorder.Body.Len() != 0 {
			t.Errorf(
				"body = %q, want empty body",
				recorder.Body.String(),
			)
		}
	})

	t.Run("returns not found", func(t *testing.T) {
		id := uuid.New()

		service := &fakeCompanyService{
			deleteFn: func(
				context.Context,
				uuid.UUID,
			) error {
				return company.ErrNotFound
			},
		}

		handler := newTestCompanyHandler(service)

		request := requestWithCompanyID(
			http.MethodDelete,
			"/companies/"+id.String(),
			id.String(),
			nil,
		)
		recorder := httptest.NewRecorder()

		handler.Delete(recorder, request)

		assertErrorResponse(
			t,
			recorder,
			http.StatusNotFound,
			"company not found",
			"",
		)
	})
}

func newTestCompanyHandler(
	service CompanyService,
) *CompanyHandler {
	logger := slog.New(
		slog.NewTextHandler(io.Discard, nil),
	)

	return NewCompanyHandler(service, logger)
}

func assertErrorResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	wantStatus int,
	wantMessage string,
	wantField string,
) {
	t.Helper()

	if recorder.Code != wantStatus {
		t.Fatalf(
			"status = %d, want %d; body = %s",
			recorder.Code,
			wantStatus,
			recorder.Body.String(),
		)
	}

	var response errorResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if response.Error != wantMessage {
		t.Errorf(
			"error = %q, want %q",
			response.Error,
			wantMessage,
		)
	}

	if response.Field != wantField {
		t.Errorf(
			"field = %q, want %q",
			response.Field,
			wantField,
		)
	}
}

func requestWithCompanyID(
	method string,
	target string,
	id string,
	body io.Reader,
) *http.Request {
	request := httptest.NewRequest(method, target, body)

	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", id)

	return request.WithContext(
		context.WithValue(
			request.Context(),
			chi.RouteCtxKey,
			routeContext,
		),
	)
}
