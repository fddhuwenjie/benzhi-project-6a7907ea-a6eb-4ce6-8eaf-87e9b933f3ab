package application

import (
	"timber-stage-qualifier/internal/domain"
	"time"
)

type CommandMeta struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision int64  `json:"expected_revision"`
	ActorID          string `json:"actor_id"`
}

type Outcome[T any] struct {
	Value    T
	Replayed bool
}

type BatchResult struct {
	BatchID        string             `json:"batch_id"`
	Status         domain.BatchStatus `json:"status"`
	Revision       int64              `json:"revision"`
	BaselineDigest string             `json:"baseline_digest,omitempty"`
}

type ObservationResult struct {
	BatchResult
	ObservationID    string            `json:"observation_id"`
	Evaluation       domain.Evaluation `json:"evaluation"`
	RuleCodes        []string          `json:"rule_codes"`
	OpenDeviationIDs []string          `json:"open_deviation_ids"`
}

type ReviewResult struct {
	BatchResult
	ReviewID          string                `json:"review_id"`
	Decision          domain.ReviewDecision `json:"decision"`
	CertificateID     string                `json:"certificate_id,omitempty"`
	CertificateSHA256 string                `json:"certificate_sha256,omitempty"`
}

type CreateBatchCommand struct {
	Meta         CommandMeta `json:"meta"`
	BatchID      string      `json:"batch_id"`
	SpecimenCode string      `json:"specimen_code"`
	WoodSpecies  string      `json:"wood_species"`
	CurrentStage string      `json:"current_stage"`
	TargetStage  string      `json:"target_stage"`
}

type FreezeProtocolCommand struct {
	Meta                      CommandMeta `json:"meta"`
	BatchID                   string      `json:"batch_id"`
	ProtocolID                string      `json:"protocol_id"`
	TargetConcentrationPct    float64     `json:"target_concentration_pct"`
	ConcentrationTolerancePct float64     `json:"concentration_tolerance_pct"`
	TemperatureMinC           float64     `json:"temperature_min_c"`
	TemperatureMaxC           float64     `json:"temperature_max_c"`
	MassChangeLimitPct        float64     `json:"mass_change_limit_pct"`
	ObservationIntervalHours  int         `json:"observation_interval_hours"`
	RecoveryWindowCount       int         `json:"recovery_window_count"`
}

type SubmitObservationCommand struct {
	Meta             CommandMeta `json:"meta"`
	BatchID          string      `json:"batch_id"`
	ObservationID    string      `json:"observation_id"`
	SequenceNo       int64       `json:"sequence_no"`
	CapturedAt       time.Time   `json:"captured_at"`
	ConcentrationPct float64     `json:"concentration_pct"`
	TemperatureC     float64     `json:"temperature_c"`
	TimberMassG      float64     `json:"timber_mass_g"`
	EvidenceDigest   string      `json:"evidence_digest"`
}

type CorrectDeviationCommand struct {
	Meta             CommandMeta `json:"meta"`
	BatchID          string      `json:"batch_id"`
	DeviationID      string      `json:"deviation_id"`
	Cause            string      `json:"cause"`
	CorrectiveAction string      `json:"corrective_action"`
	OwnerID          string      `json:"owner_id"`
}

type ApproveRecoveryCommand struct {
	Meta    CommandMeta `json:"meta"`
	BatchID string      `json:"batch_id"`
}

type DecideReviewCommand struct {
	Meta               CommandMeta              `json:"meta"`
	BatchID            string                   `json:"batch_id"`
	ReviewID           string                   `json:"review_id"`
	Decision           domain.ReviewDecision    `json:"decision"`
	ChecklistResults   []domain.ChecklistResult `json:"checklist_results"`
	Rationale          string                   `json:"rationale"`
	EvidenceRootDigest string                   `json:"evidence_root_digest,omitempty"`
}
