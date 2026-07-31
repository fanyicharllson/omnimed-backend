// Package http contains the gateway's HTTP delivery layer: routing and
// request/response handling for the triage API.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/fanyicharllson/omnimed-backend/internal/gateway/usecase"
	"github.com/google/uuid"
)

// maxUploadBytes bounds the multipart form parsed into memory.
const maxUploadBytes = 10 << 20

// TriageHandler serves the diagnostic HTTP endpoints. Image upload and
// diagnosis are the only things served over HTTP — everything else
// client-facing (auth, session, medical logs) lives on the gateway's
// gRPC server (internal/gateway/delivery/grpc) as it's added.
type TriageHandler struct {
	usecase *usecase.TriageUsecase
	logger  *slog.Logger
}

// NewTriageHandler constructs a TriageHandler.
func NewTriageHandler(u *usecase.TriageUsecase, logger *slog.Logger) *TriageHandler {
	return &TriageHandler{usecase: u, logger: logger}
}

// diagnoseFunc is the usecase-layer signature every per-modality
// diagnosis method shares (DiagnoseBreastCancer today; DiagnoseSkin,
// DiagnoseOral later).
type diagnoseFunc func(ctx context.Context, imageData []byte, contentType, requestID string) (*usecase.DiagnosisResult, error)

// handleDiagnose is the shared thin-handler skeleton for every
// diagnosis endpoint: parse the multipart upload, forward it to the
// given usecase method, and write the resulting DiagnosisResult as
// JSON. It holds no business logic itself — validation lives in the
// usecase layer, risk classification lives in DecisionPolicy. Adding a
// new modality is just a new one-line wrapper around this (see
// DiagnoseBreastCancer below) plus a route registration in router.go.
func (h *TriageHandler) handleDiagnose(w http.ResponseWriter, r *http.Request, modality string, diagnose diagnoseFunc) {
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.NewString()
	}

	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid multipart form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "missing \"image\" form field: "+err.Error())
		return
	}
	defer file.Close()

	imageData, err := io.ReadAll(file)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to read uploaded image")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	result, err := diagnose(r.Context(), imageData, contentType, requestID)
	if err != nil {
		var validationErr *usecase.ValidationError
		if errors.As(err, &validationErr) {
			h.writeError(w, http.StatusBadRequest, validationErr.Message)
			return
		}
		h.logger.Error("diagnose failed", "modality", modality, "request_id", requestID, "error", err)
		h.writeError(w, http.StatusBadGateway, "inference service unavailable")
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

// DiagnoseBreastCancer handles POST /api/v1/diagnose/breast-cancer.
// It expects a multipart/form-data body with the image under the
// "image" field.
func (h *TriageHandler) DiagnoseBreastCancer(w http.ResponseWriter, r *http.Request) {
	h.handleDiagnose(w, r, "breast_cancer", h.usecase.DiagnoseBreastCancer)
}

// Health handles GET /healthz. It only reports that the gateway process
// itself is alive — it does not check any downstream dependency.
func (h *TriageHandler) Health(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readiness handles GET /readyz: whether the gateway and everything it
// depends on to actually serve a diagnosis (currently just the AI
// inference service) are up. Use this to confirm the whole stack is
// running together — whether started via `docker compose up` or as
// separately launched local processes — not just that the gateway
// process started.
func (h *TriageHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	dependencies := map[string]string{"ai_inference": "ok"}
	ready := true

	if err := h.usecase.CheckReadiness(r.Context()); err != nil {
		ready = false
		dependencies["ai_inference"] = err.Error()
		h.logger.Warn("readiness check failed", "dependency", "ai_inference", "error", err)
	}

	status := http.StatusOK
	overall := "ready"
	if !ready {
		status = http.StatusServiceUnavailable
		overall = "not_ready"
	}

	h.writeJSON(w, status, map[string]any{
		"status":       overall,
		"dependencies": dependencies,
	})
}

func (h *TriageHandler) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

func (h *TriageHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
