// Package middleware holds HTTP middleware for the gateway.
package middleware

import (
	"context"
	"log/slog"
	"net/http"
)

// contextKey avoids collisions with other packages' context values.
type contextKey string

const userContextKey contextKey = "auth_user"

// AuthenticatedUser is the identity attached to the request context by
// Authenticate. The stub implementation below always injects an
// anonymous user; a real implementation (JWT/RBAC) fills this in from
// a verified token instead.
type AuthenticatedUser struct {
	ID    string
	Roles []string
}

// Authenticator verifies a request and returns the identity making it.
// Swap StubAuthenticator for a real JWT/RBAC implementation later
// without changing any handler or route wiring.
type Authenticator interface {
	Authenticate(r *http.Request) (*AuthenticatedUser, error)
}

// StubAuthenticator performs no verification. It exists so the
// middleware chain, route wiring, and downstream handlers are already
// shaped for auth before real JWT/RBAC logic is implemented.
type StubAuthenticator struct {
	logger *slog.Logger
}

// NewStubAuthenticator constructs a no-op Authenticator.
func NewStubAuthenticator(logger *slog.Logger) *StubAuthenticator {
	return &StubAuthenticator{logger: logger}
}

// Authenticate always succeeds as an anonymous user.
func (a *StubAuthenticator) Authenticate(r *http.Request) (*AuthenticatedUser, error) {
	return &AuthenticatedUser{ID: "anonymous", Roles: []string{"guest"}}, nil
}

// Auth returns middleware that authenticates each request via the
// given Authenticator and rejects it with 401 on failure.
func Auth(authenticator Authenticator, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := authenticator.Authenticate(r)
			if err != nil {
				logger.Warn("authentication failed", "error", err, "path", r.URL.Path)
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext retrieves the AuthenticatedUser injected by Auth.
func UserFromContext(ctx context.Context) (*AuthenticatedUser, bool) {
	user, ok := ctx.Value(userContextKey).(*AuthenticatedUser)
	return user, ok
}
