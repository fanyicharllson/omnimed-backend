// Package usecase contains the gateway's application logic: validating
// incoming requests and orchestrating calls to the AI inference service.
package usecase

import (
	"context"
	"fmt"
	"log/slog"

	triagepb "github.com/fanyicharllson/omnimed-backend/pb/triage"
)

// maxImageBytes caps uploads accepted for diagnosis (10 MiB).
const maxImageBytes = 10 << 20

// allowedContentTypes is the set of image formats forwarded to the
// inference service.
var allowedContentTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/jpg":  true,
}

// TriageClient is the subset of the gRPC client the usecase depends on.
// Defined here (consumer side) so it can be faked in tests without
// depending on the concrete gRPC implementation.
type TriageClient interface {
	DiagnoseBreastCancer(ctx context.Context, req *triagepb.ImageRequest) (*triagepb.DiagnosisResponse, error)
}

// ClassConfidence mirrors the proto message as a plain Go type so the
// delivery layer doesn't need to depend on generated pb types directly.
type ClassConfidence struct {
	Label      string  `json:"label"`
	Confidence float32 `json:"confidence"`
}

// DiagnosisResult is the usecase-level result returned to the delivery
// layer for JSON serialization.
type DiagnosisResult struct {
	PredictedClass   string            `json:"predicted_class"`
	ClassConfidences []ClassConfidence `json:"class_confidences"`
	ModelVersion     string            `json:"model_version"`
	RequestID        string            `json:"request_id"`
}

// TriageUsecase validates uploads and orchestrates diagnosis requests
// against the AI inference service.
type TriageUsecase struct {
	client TriageClient
	logger *slog.Logger
}

// NewTriageUsecase constructs a TriageUsecase.
func NewTriageUsecase(client TriageClient, logger *slog.Logger) *TriageUsecase {
	return &TriageUsecase{client: client, logger: logger}
}

// DiagnoseBreastCancer validates the uploaded image and forwards it to
// the inference service, mapping its response into a DiagnosisResult.
func (u *TriageUsecase) DiagnoseBreastCancer(ctx context.Context, imageData []byte, contentType, requestID string) (*DiagnosisResult, error) {
	if err := validateImage(imageData, contentType); err != nil {
		return nil, err
	}

	resp, err := u.client.DiagnoseBreastCancer(ctx, &triagepb.ImageRequest{
		ImageData:   imageData,
		ContentType: contentType,
		RequestId:   requestID,
	})
	if err != nil {
		u.logger.Error("inference call failed", "request_id", requestID, "error", err)
		return nil, fmt.Errorf("diagnose breast cancer: %w", err)
	}

	result := &DiagnosisResult{
		PredictedClass: resp.GetPredictedClass(),
		ModelVersion:   resp.GetModelVersion(),
		RequestID:      resp.GetRequestId(),
	}
	for _, c := range resp.GetClassConfidences() {
		result.ClassConfidences = append(result.ClassConfidences, ClassConfidence{
			Label:      c.GetLabel(),
			Confidence: c.GetConfidence(),
		})
	}

	u.logger.Info("diagnosis completed",
		"request_id", requestID,
		"predicted_class", result.PredictedClass,
		"model_version", result.ModelVersion,
	)

	return result, nil
}

// ValidationError indicates the uploaded image failed input validation
// and should be surfaced to the client as a 4xx response.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func validateImage(imageData []byte, contentType string) error {
	if len(imageData) == 0 {
		return &ValidationError{Message: "image data is empty"}
	}
	if len(imageData) > maxImageBytes {
		return &ValidationError{Message: fmt.Sprintf("image exceeds maximum size of %d bytes", maxImageBytes)}
	}
	if !allowedContentTypes[contentType] {
		return &ValidationError{Message: fmt.Sprintf("unsupported content type %q", contentType)}
	}
	return nil
}
