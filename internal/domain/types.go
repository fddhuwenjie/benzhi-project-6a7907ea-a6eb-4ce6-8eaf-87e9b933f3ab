package domain

import "time"

type BatchStatus string

const (
	StatusDraft      BatchStatus = "draft"
	StatusMonitoring BatchStatus = "monitoring"
	StatusPaused     BatchStatus = "paused"
	StatusRecovering BatchStatus = "recovering"
	StatusReview     BatchStatus = "pending_review"
	StatusApproved   BatchStatus = "approved_sealed"
	StatusRejected   BatchStatus = "rejected_sealed"
)

func (s BatchStatus) Terminal() bool {
	return s == StatusApproved || s == StatusRejected
}

type Evaluation string

const (
	EvaluationQualified Evaluation = "qualified"
	EvaluationDeviation Evaluation = "deviation"
)

type DeviationStatus string

const (
	DeviationOpen      DeviationStatus = "open"
	DeviationCorrected DeviationStatus = "corrected"
	DeviationClosed    DeviationStatus = "closed"
)

type ReviewDecision string

const (
	DecisionApprove ReviewDecision = "approve"
	DecisionReject  ReviewDecision = "reject"
)

type ChecklistResult struct {
	Item   string `json:"item"`
	Passed bool   `json:"passed"`
	Note   string `json:"note,omitempty"`
}

type ParticipantRoles struct {
	Baseline   []string `json:"baseline"`
	Observers  []string `json:"observers"`
	Correctors []string `json:"correctors"`
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func appendUnique(values []string, value string) []string {
	if value != "" && !contains(values, value) {
		return append(values, value)
	}
	return values
}

func utc(t time.Time) time.Time { return t.UTC().Truncate(time.Microsecond) }
