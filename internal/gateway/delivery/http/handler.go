// Package http contains the gateway's HTTP delivery layer: routing and
// request/response handling for the triage API.
package http

import (
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

// TriageHandler serves the diagnostic HTTP endpoints.
type TriageHandler struct {
	usecase *usecase.TriageUsecase
	logger  *slog.Logger
}

// NewTriageHandler constructs a TriageHandler.
func NewTriageHandler(u *usecase.TriageUsecase, logger *slog.Logger) *TriageHandler {
	return &TriageHandler{usecase: u, logger: logger}
}

// DiagnoseBreastCancer handles POST /v1/diagnose/breast-cancer.
// It expects a multipart/form-data body with the image under the
// "image" field.
func (h *TriageHandler) DiagnoseBreastCancer(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.usecase.DiagnoseBreastCancer(r.Context(), imageData, contentType, requestID)
	if err != nil {
		var validationErr *usecase.ValidationError
		if errors.As(err, &validationErr) {
			h.writeError(w, http.StatusBadRequest, validationErr.Message)
			return
		}
		h.logger.Error("diagnose breast cancer failed", "request_id", requestID, "error", err)
		h.writeError(w, http.StatusBadGateway, "inference service unavailable")
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

// Health handles GET /healthz.
func (h *TriageHandler) Health(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
