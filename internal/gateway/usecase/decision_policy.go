package usecase

// RiskTier is the coarse-grained clinical risk classification derived
// from the model's raw per-class probabilities. It intentionally
// collapses "benign" vs "normal" into a single non-concerning tier —
// that distinction is still visible in DiagnosisResult.RawProbabilities
// for anyone who wants it, but the tier itself only needs to answer
// "does this need a clinician's attention or not".
type RiskTier string

const (
	RiskTierLikelyBenign     RiskTier = "likely_benign"
	RiskTierFlaggedForReview RiskTier = "flagged_for_review"
	RiskTierInconclusive     RiskTier = "inconclusive"
)

// Class labels as returned by the AI inference service.
const (
	LabelBenign    = "benign"
	LabelMalignant = "malignant"
	LabelNormal    = "normal"
)

// ScreeningDisclaimer is attached to every diagnosis response. The model
// is a screening aid — it does not, and must never appear to, replace a
// clinician's judgment.
const ScreeningDisclaimer = "This is an AI screening aid, not a medical diagnosis. All results must be reviewed and confirmed by a qualified clinician before any clinical decision is made."

// DecisionPolicy turns a raw per-class probability map into a
// DiagnosisResult.
//
// It deliberately does not just take argmax: a malignant probability
// that clears MalignantFlagThreshold is flagged for clinician review
// even when it isn't the single highest-probability class, because in
// a screening context under-calling malignancy is far costlier than a
// false positive. Only once malignancy risk is ruled low does
// MinConfidenceFloor guard against reporting a low-confidence guess as
// if it were a firm result.
type DecisionPolicy struct {
	// MalignantFlagThreshold: a malignant probability at or above this
	// value always yields RiskTierFlaggedForReview, regardless of
	// whether malignant is the argmax class.
	MalignantFlagThreshold float32

	// MinConfidenceFloor: once malignancy risk is ruled low, the top
	// remaining probability must be at or above this value or the
	// result is RiskTierInconclusive.
	MinConfidenceFloor float32
}

// NewDecisionPolicy constructs a DecisionPolicy from configured
// thresholds.
func NewDecisionPolicy(malignantFlagThreshold, minConfidenceFloor float32) *DecisionPolicy {
	return &DecisionPolicy{
		MalignantFlagThreshold: malignantFlagThreshold,
		MinConfidenceFloor:     minConfidenceFloor,
	}
}

// Evaluate applies the policy to a raw label -> probability map (as
// returned by the AI inference service) and produces the client-facing
// DiagnosisResult.
func (p *DecisionPolicy) Evaluate(rawProbabilities map[string]float32, modelVersion, requestID string) *DiagnosisResult {
	malignantProb := rawProbabilities[LabelMalignant]

	var tier RiskTier
	var confidence float32

	if malignantProb >= p.MalignantFlagThreshold {
		tier = RiskTierFlaggedForReview
		confidence = malignantProb
	} else {
		_, topProb := argmaxExcluding(rawProbabilities, LabelMalignant)
		confidence = topProb

		if topProb < p.MinConfidenceFloor {
			tier = RiskTierInconclusive
		} else {
			tier = RiskTierLikelyBenign
		}
	}

	return &DiagnosisResult{
		RiskTier:         tier,
		RawProbabilities: rawProbabilities,
		Confidence:       confidence,
		ModelVersion:     modelVersion,
		Disclaimer:       ScreeningDisclaimer,
		RequestID:        requestID,
	}
}

// argmaxExcluding returns the label and probability of the
// highest-probability entry in probabilities, ignoring excluded.
func argmaxExcluding(probabilities map[string]float32, excluded string) (label string, prob float32) {
	best := float32(-1)
	for l, p := range probabilities {
		if l == excluded {
			continue
		}
		if p > best {
			label, best = l, p
		}
	}
	return label, best
}
