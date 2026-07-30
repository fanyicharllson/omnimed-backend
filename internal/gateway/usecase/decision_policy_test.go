package usecase

import "testing"

func TestDecisionPolicy_Evaluate(t *testing.T) {
	policy := NewDecisionPolicy(0.3, 0.4)

	tests := []struct {
		name           string
		probabilities  map[string]float32
		wantTier       RiskTier
		wantConfidence float32
	}{
		{
			name: "clear benign case",
			probabilities: map[string]float32{
				LabelBenign:    0.85,
				LabelMalignant: 0.05,
				LabelNormal:    0.10,
			},
			wantTier:       RiskTierLikelyBenign,
			wantConfidence: 0.85,
		},
		{
			name: "clear malignant case",
			probabilities: map[string]float32{
				LabelBenign:    0.05,
				LabelMalignant: 0.90,
				LabelNormal:    0.05,
			},
			wantTier:       RiskTierFlaggedForReview,
			wantConfidence: 0.90,
		},
		{
			// Normal is the argmax (0.45 > 0.35 > 0.20), but malignant
			// still clears the 0.3 threshold — the asymmetric policy
			// must flag this for review rather than trusting argmax.
			name: "malignant flagged despite not being argmax",
			probabilities: map[string]float32{
				LabelBenign:    0.20,
				LabelMalignant: 0.35,
				LabelNormal:    0.45,
			},
			wantTier:       RiskTierFlaggedForReview,
			wantConfidence: 0.35,
		},
		{
			// Malignant is below threshold, and the best of the
			// remaining classes (0.38) is below the 0.4 confidence
			// floor — the model is essentially guessing.
			name: "inconclusive when everything is under the confidence floor",
			probabilities: map[string]float32{
				LabelBenign:    0.38,
				LabelMalignant: 0.25,
				LabelNormal:    0.37,
			},
			wantTier:       RiskTierInconclusive,
			wantConfidence: 0.38,
		},
		{
			// Boundary check: malignant probability exactly equal to
			// the threshold must still flag (>=, not >).
			name: "malignant probability exactly at threshold flags for review",
			probabilities: map[string]float32{
				LabelBenign:    0.35,
				LabelMalignant: 0.30,
				LabelNormal:    0.35,
			},
			wantTier:       RiskTierFlaggedForReview,
			wantConfidence: 0.30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := policy.Evaluate(tt.probabilities, "test-model-v1", "req-123")

			if result.RiskTier != tt.wantTier {
				t.Errorf("RiskTier = %v, want %v", result.RiskTier, tt.wantTier)
			}
			if result.Confidence != tt.wantConfidence {
				t.Errorf("Confidence = %v, want %v", result.Confidence, tt.wantConfidence)
			}
			if result.ModelVersion != "test-model-v1" {
				t.Errorf("ModelVersion = %q, want %q", result.ModelVersion, "test-model-v1")
			}
			if result.RequestID != "req-123" {
				t.Errorf("RequestID = %q, want %q", result.RequestID, "req-123")
			}
			if result.Disclaimer != ScreeningDisclaimer {
				t.Errorf("Disclaimer = %q, want the standard ScreeningDisclaimer", result.Disclaimer)
			}
			for label, prob := range tt.probabilities {
				if result.RawProbabilities[label] != prob {
					t.Errorf("RawProbabilities[%q] = %v, want %v (raw probabilities must pass through unmodified)",
						label, result.RawProbabilities[label], prob)
				}
			}
		})
	}
}
