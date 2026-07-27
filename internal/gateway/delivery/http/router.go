package http

import (
	"log/slog"
	"net/http"

	"github.com/fanyicharllson/omnimed-backend/internal/gateway/delivery/http/middleware"
)

// NewRouter wires the gateway's HTTP routes and middleware chain.
func NewRouter(handler *TriageHandler, authenticator middleware.Authenticator, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handler.Health)

	protected := middleware.Auth(authenticator, logger)(
		http.HandlerFunc(handler.DiagnoseBreastCancer),
	)
	mux.Handle("POST /v1/diagnose/breast-cancer", protected)

	return requestLogger(logger)(mux)
}

// requestLogger logs each request's method, path, and status code.
func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			logger.Info("request handled",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
			)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
