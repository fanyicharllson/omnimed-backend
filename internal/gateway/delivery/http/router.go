package http

import (
	"log/slog"
	"net/http"

	"github.com/fanyicharllson/omnimed-backend/internal/gateway/delivery/http/middleware"
)

// NewRouter wires the gateway's HTTP routes and middleware chain.
//
// HTTP is deliberately kept to image-upload diagnosis endpoints (which
// need multipart/form-data) plus health/readiness checks. Everything
// else client-facing (auth, session, medical logs) belongs on the
// gateway's gRPC server (internal/gateway/delivery/grpc) instead.
//
// Adding a new modality is one line here plus a thin wrapper handler in
// handler.go — see DiagnoseBreastCancer / /api/v1/diagnose/breast-cancer
// as the template for /api/v1/diagnose/skin, /api/v1/diagnose/oral, etc.
func NewRouter(handler *TriageHandler, authenticator middleware.Authenticator, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handler.Health)
	mux.HandleFunc("GET /readyz", handler.Readiness)

	protected := middleware.Auth(authenticator, logger)(
		http.HandlerFunc(handler.DiagnoseBreastCancer),
	)
	mux.Handle("POST /api/v1/diagnose/breast-cancer", protected)

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
