package evidence

import "time"

const DocumentVersion = "timber-stage-certificate/v1"

type Certificate struct {
	CertificateID    string     `json:"certificate_id"`
	BatchID          string     `json:"batch_id"`
	DocumentVersion  string     `json:"document_version"`
	CanonicalPayload []byte     `json:"canonical_payload"`
	PayloadSHA256    string     `json:"payload_sha256"`
	AuditRootDigest  string     `json:"audit_root_digest"`
	IssuedAt         time.Time  `json:"issued_at"`
	VerifiedAt       *time.Time `json:"verified_at,omitempty"`
}

type Verification struct {
	Valid              bool      `json:"valid"`
	PayloadDigestValid bool      `json:"payload_digest_valid"`
	BatchStateValid    bool      `json:"batch_state_valid"`
	AuditRootValid     bool      `json:"audit_root_valid"`
	Failures           []string  `json:"failures"`
	VerifiedAt         time.Time `json:"verified_at"`
}

type canonicalObservationRange struct {
	FirstSequence int64  `json:"first_sequence"`
	LastSequence  int64  `json:"last_sequence"`
	Count         int    `json:"count"`
	FirstCaptured string `json:"first_captured_at"`
	LastCaptured  string `json:"last_captured_at"`
}

type canonicalDeviation struct {
	DeviationID             string   `json:"deviation_id"`
	RuleCode                string   `json:"rule_code"`
	Status                  string   `json:"status"`
	OwnerID                 string   `json:"owner_id"`
	QualifiedObservationIDs []string `json:"qualified_observation_ids"`
	ClosedAt                string   `json:"closed_at"`
}

type canonicalReview struct {
	ReviewID           string `json:"review_id"`
	ReviewerID         string `json:"reviewer_id"`
	Decision           string `json:"decision"`
	Rationale          string `json:"rationale"`
	EvidenceRootDigest string `json:"evidence_root_digest"`
	DecidedAt          string `json:"decided_at"`
}

type canonicalPayload struct {
	DocumentVersion           string                    `json:"document_version"`
	CertificateID             string                    `json:"certificate_id"`
	BatchID                   string                    `json:"batch_id"`
	SpecimenCode              string                    `json:"specimen_code"`
	WoodSpecies               string                    `json:"wood_species"`
	CurrentStage              string                    `json:"current_stage"`
	TargetStage               string                    `json:"target_stage"`
	FinalStatus               string                    `json:"final_status"`
	FinalRevision             int64                     `json:"final_revision"`
	BaselineDigest            string                    `json:"baseline_digest"`
	TargetConcentrationPct    string                    `json:"target_concentration_pct"`
	ConcentrationTolerancePct string                    `json:"concentration_tolerance_pct"`
	TemperatureMinC           string                    `json:"temperature_min_c"`
	TemperatureMaxC           string                    `json:"temperature_max_c"`
	MassChangeLimitPct        string                    `json:"mass_change_limit_pct"`
	ObservationIntervalHours  int                       `json:"observation_interval_hours"`
	RecoveryWindowCount       int                       `json:"recovery_window_count"`
	ObservationRange          canonicalObservationRange `json:"observation_range"`
	DeviationClosures         []canonicalDeviation      `json:"deviation_closures"`
	Review                    canonicalReview           `json:"review"`
	IssuedAt                  string                    `json:"issued_at"`
}
