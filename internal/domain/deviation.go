package domain

import (
	"strings"
	"time"
)

type DeviationCase struct {
	DeviationID             string          `json:"deviation_id"`
	BatchID                 string          `json:"batch_id"`
	RuleCode                string          `json:"rule_code"`
	DetectedAt              time.Time       `json:"detected_at"`
	Status                  DeviationStatus `json:"status"`
	Cause                   string          `json:"cause,omitempty"`
	CorrectiveAction        string          `json:"corrective_action,omitempty"`
	OwnerID                 string          `json:"owner_id,omitempty"`
	RecoveryStartedAt       *time.Time      `json:"recovery_started_at,omitempty"`
	QualifiedObservationIDs []string        `json:"qualified_observation_ids"`
	ClosedAt                *time.Time      `json:"closed_at,omitempty"`
}

func (d *DeviationCase) RegisterCorrection(cause, action, owner string) error {
	if d.Status != DeviationOpen {
		return NewRuleError("deviation_not_open", "偏差不处于开放状态")
	}
	if strings.TrimSpace(cause) == "" || strings.TrimSpace(action) == "" || strings.TrimSpace(owner) == "" {
		return NewRuleError("incomplete_correction", "原因、纠正措施和责任人均不能为空")
	}
	d.Cause, d.CorrectiveAction, d.OwnerID = strings.TrimSpace(cause), strings.TrimSpace(action), strings.TrimSpace(owner)
	d.Status = DeviationCorrected
	return nil
}

func (d *DeviationCase) StartRecovery(at time.Time) error {
	if d.Status != DeviationCorrected {
		return NewRuleError("correction_not_ready", "偏差尚未完成纠正登记")
	}
	t := utc(at)
	d.RecoveryStartedAt = &t
	d.QualifiedObservationIDs = nil
	return nil
}

func (d *DeviationCase) AddQualified(observationID string) {
	d.QualifiedObservationIDs = appendUnique(d.QualifiedObservationIDs, observationID)
}

func (d *DeviationCase) ResetRecovery() {
	d.QualifiedObservationIDs = nil
	d.RecoveryStartedAt = nil
	d.Status = DeviationOpen
}

func (d *DeviationCase) Close(at time.Time) {
	t := utc(at)
	d.Status = DeviationClosed
	d.ClosedAt = &t
}
