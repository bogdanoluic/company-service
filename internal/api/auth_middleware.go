package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type authContextKey struct{}

type AuthenticatedUser struct {
	Subject string
}

type AuthMiddleware struct {
	secret []byte
}

func NewAuthMiddleware(secret string) *AuthMiddleware {
	return &AuthMiddleware{
		secret: []byte(secret),
	}
}

func (m *AuthMiddleware) Authenticate(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		tokenValue, ok := bearerToken(
			r.Header.Get("Authorization"),
		)
		if !ok {
			respondUnauthorized(w)
			return
		}

		claims := &jwt.RegisteredClaims{}

		token, err := jwt.ParseWithClaims(
			tokenValue,
			claims,
			func(*jwt.Token) (any, error) {
				return m.secret, nil
			},
			jwt.WithValidMethods(
				[]string{jwt.SigningMethodHS256.Alg()},
			),
			jwt.WithExpirationRequired(),
		)
		if err != nil || !token.Valid || claims.Subject == "" {
			respondUnauthorized(w)
			return
		}

		user := AuthenticatedUser{
			Subject: claims.Subject,
		}

		ctx := context.WithValue(
			r.Context(),
			authContextKey{},
			user,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AuthenticatedUserFromContext(
	ctx context.Context,
) (AuthenticatedUser, bool) {
	user, ok := ctx.Value(authContextKey{}).(AuthenticatedUser)

	return user, ok
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)

	if len(parts) != 2 {
		return "", false
	}

	if !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	if parts[1] == "" {
		return "", false
	}

	return parts[1], true
}

func respondUnauthorized(w http.ResponseWriter) {
	w.Header().Set(
		"WWW-Authenticate",
		`Bearer realm="company-service"`,
	)

	_ = writeError(
		w,
		http.StatusUnauthorized,
		"authentication required",
		"",
	)
}
