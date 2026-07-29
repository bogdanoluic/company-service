package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDatabasePinger struct {
	pingFn func(context.Context) error
}

func (p *fakeDatabasePinger) Ping(
	ctx context.Context,
) error {
	if p.pingFn == nil {
		panic("unexpected call to Ping")
	}

	return p.pingFn(ctx)
}

func TestHealthHandlerLive(t *testing.T) {
	database := &fakeDatabasePinger{}

	handler := newTestHealthHandler(database)

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/live",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.Live(recorder, request)

	assertHealthResponse(
		t,
		recorder,
		http.StatusOK,
		"ok",
	)
}

func TestHealthHandlerReady(t *testing.T) {
	t.Run("ready when database responds", func(t *testing.T) {
		pingCalled := false

		database := &fakeDatabasePinger{
			pingFn: func(ctx context.Context) error {
				pingCalled = true

				if ctx.Err() != nil {
					t.Errorf(
						"Ping() context error = %v",
						ctx.Err(),
					)
				}

				return nil
			},
		}

		handler := newTestHealthHandler(database)

		request := httptest.NewRequest(
			http.MethodGet,
			"/health/ready",
			nil,
		)
		recorder := httptest.NewRecorder()

		handler.Ready(recorder, request)

		if !pingCalled {
			t.Fatal("database Ping() was not called")
		}

		assertHealthResponse(
			t,
			recorder,
			http.StatusOK,
			"ok",
		)
	})

	t.Run("unavailable when database ping fails", func(t *testing.T) {
		database := &fakeDatabasePinger{
			pingFn: func(context.Context) error {
				return errors.New("database unavailable")
			},
		}

		handler := newTestHealthHandler(database)

		request := httptest.NewRequest(
			http.MethodGet,
			"/health/ready",
			nil,
		)
		recorder := httptest.NewRecorder()

		handler.Ready(recorder, request)

		assertHealthResponse(
			t,
			recorder,
			http.StatusServiceUnavailable,
			"unavailable",
		)
	})
}

func newTestHealthHandler(
	database DatabasePinger,
) *HealthHandler {
	logger := slog.New(
		slog.NewTextHandler(io.Discard, nil),
	)

	return NewHealthHandler(database, logger)
}

func assertHealthResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	wantStatus int,
	wantHealthStatus string,
) {
	t.Helper()

	if recorder.Code != wantStatus {
		t.Fatalf(
			"HTTP status = %d, want %d; body = %s",
			recorder.Code,
			wantStatus,
			recorder.Body.String(),
		)
	}

	if got := recorder.Header().Get(
		"Content-Type",
	); got != "application/json" {
		t.Errorf(
			"Content-Type = %q, want %q",
			got,
			"application/json",
		)
	}

	var response healthResponse

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"decode health response: %v",
			err,
		)
	}

	if response.Status != wantHealthStatus {
		t.Errorf(
			"status = %q, want %q",
			response.Status,
			wantHealthStatus,
		)
	}
}
