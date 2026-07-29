package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testJWTSecret = "01234567890123456789012345678901"

func TestAuthMiddleware(t *testing.T) {
	t.Run("accepts valid token", func(t *testing.T) {
		token := signTestToken(
			t,
			jwt.SigningMethodHS256,
			testJWTSecret,
			jwt.RegisteredClaims{
				Subject: "user-123",
				ExpiresAt: jwt.NewNumericDate(
					time.Now().Add(time.Hour),
				),
			},
		)

		middleware := NewAuthMiddleware(testJWTSecret)

		handlerCalled := false

		next := http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			handlerCalled = true

			user, ok := AuthenticatedUserFromContext(
				r.Context(),
			)
			if !ok {
				t.Fatal(
					"authenticated user missing from context",
				)
			}

			if user.Subject != "user-123" {
				t.Errorf(
					"subject = %q, want %q",
					user.Subject,
					"user-123",
				)
			}

			w.WriteHeader(http.StatusNoContent)
		})

		request := httptest.NewRequest(
			http.MethodPost,
			"/companies",
			nil,
		)
		request.Header.Set(
			"Authorization",
			"Bearer "+token,
		)

		recorder := httptest.NewRecorder()

		middleware.Authenticate(next).
			ServeHTTP(recorder, request)

		if !handlerCalled {
			t.Fatal("next handler was not called")
		}

		if recorder.Code != http.StatusNoContent {
			t.Errorf(
				"status = %d, want %d",
				recorder.Code,
				http.StatusNoContent,
			)
		}
	})

	tests := []struct {
		name   string
		header func(t *testing.T) string
	}{
		{
			name: "missing authorization header",
			header: func(*testing.T) string {
				return ""
			},
		},
		{
			name: "malformed authorization header",
			header: func(*testing.T) string {
				return "Basic credentials"
			},
		},
		{
			name: "missing bearer token",
			header: func(*testing.T) string {
				return "Bearer"
			},
		},
		{
			name: "invalid signature",
			header: func(t *testing.T) string {
				token := signTestToken(
					t,
					jwt.SigningMethodHS256,
					"different-secret-that-is-long-enough",
					validTestClaims(),
				)

				return "Bearer " + token
			},
		},
		{
			name: "expired token",
			header: func(t *testing.T) string {
				token := signTestToken(
					t,
					jwt.SigningMethodHS256,
					testJWTSecret,
					jwt.RegisteredClaims{
						Subject: "user-123",
						ExpiresAt: jwt.NewNumericDate(
							time.Now().Add(-time.Hour),
						),
					},
				)

				return "Bearer " + token
			},
		},
		{
			name: "missing expiration",
			header: func(t *testing.T) string {
				token := signTestToken(
					t,
					jwt.SigningMethodHS256,
					testJWTSecret,
					jwt.RegisteredClaims{
						Subject: "user-123",
					},
				)

				return "Bearer " + token
			},
		},
		{
			name: "missing subject",
			header: func(t *testing.T) string {
				token := signTestToken(
					t,
					jwt.SigningMethodHS256,
					testJWTSecret,
					jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(
							time.Now().Add(time.Hour),
						),
					},
				)

				return "Bearer " + token
			},
		},
		{
			name: "wrong signing algorithm",
			header: func(t *testing.T) string {
				token := signTestToken(
					t,
					jwt.SigningMethodHS384,
					testJWTSecret,
					validTestClaims(),
				)

				return "Bearer " + token
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewAuthMiddleware(testJWTSecret)

			next := http.HandlerFunc(func(
				http.ResponseWriter,
				*http.Request,
			) {
				t.Fatal(
					"next handler called for invalid token",
				)
			})

			request := httptest.NewRequest(
				http.MethodPost,
				"/companies",
				nil,
			)

			if header := tt.header(t); header != "" {
				request.Header.Set(
					"Authorization",
					header,
				)
			}

			recorder := httptest.NewRecorder()

			middleware.Authenticate(next).
				ServeHTTP(recorder, request)

			assertUnauthorized(t, recorder)
		})
	}
}

func validTestClaims() jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		Subject: "user-123",
		ExpiresAt: jwt.NewNumericDate(
			time.Now().Add(time.Hour),
		),
	}
}

func signTestToken(
	t *testing.T,
	method jwt.SigningMethod,
	secret string,
	claims jwt.RegisteredClaims,
) string {
	t.Helper()

	token := jwt.NewWithClaims(method, claims)

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return signed
}

func assertUnauthorized(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
) {
	t.Helper()

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"status = %d, want %d; body = %s",
			recorder.Code,
			http.StatusUnauthorized,
			recorder.Body.String(),
		)
	}

	if got := recorder.Header().Get(
		"WWW-Authenticate",
	); got != `Bearer realm="company-service"` {
		t.Errorf(
			"WWW-Authenticate = %q, want %q",
			got,
			`Bearer realm="company-service"`,
		)
	}

	var response errorResponse

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Error != "authentication required" {
		t.Errorf(
			"error = %q, want %q",
			response.Error,
			"authentication required",
		)
	}
}
