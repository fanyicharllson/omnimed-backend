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
	CheckHealth(ctx context.Context) error
}

// DiagnosisResult is the usecase-level result returned to the delivery
// layer for JSON serialization. It is produced by DecisionPolicy.Evaluate
// rather than by trusting the inference service's raw argmax.
type DiagnosisResult struct {
	RiskTier         RiskTier           `json:"risk_tier"`
	RawProbabilities map[string]float32 `json:"raw_probabilities"`
	Confidence       float32            `json:"confidence"`
	ModelVersion     string             `json:"model_version"`
	Disclaimer       string             `json:"disclaimer"`
	RequestID        string             `json:"request_id"`
}

// TriageUsecase validates uploads and orchestrates diagnosis requests
// against the AI inference service.
type TriageUsecase struct {
	client TriageClient
	policy *DecisionPolicy
	logger *slog.Logger
}

// NewTriageUsecase constructs a TriageUsecase.
func NewTriageUsecase(client TriageClient, policy *DecisionPolicy, logger *slog.Logger) *TriageUsecase {
	return &TriageUsecase{client: client, policy: policy, logger: logger}
}

// CheckReadiness reports whether the AI inference service is reachable
// and serving, via the standard gRPC health checking protocol.
func (u *TriageUsecase) CheckReadiness(ctx context.Context) error {
	return u.client.CheckHealth(ctx)
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

	rawProbabilities := make(map[string]float32, len(resp.GetClassConfidences()))
	for _, c := range resp.GetClassConfidences() {
		rawProbabilities[c.GetLabel()] = c.GetConfidence()
	}

	result := u.policy.Evaluate(rawProbabilities, resp.GetModelVersion(), requestID)

	u.logger.Info("diagnosis completed",
		"request_id", requestID,
		"risk_tier", result.RiskTier,
		"confidence", result.Confidence,
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
