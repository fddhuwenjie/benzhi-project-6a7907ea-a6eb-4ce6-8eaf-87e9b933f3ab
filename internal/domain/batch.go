package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type TreatmentBatch struct {
	BatchID        string               `json:"batch_id"`
	SpecimenCode   string               `json:"specimen_code"`
	WoodSpecies    string               `json:"wood_species"`
	CurrentStage   string               `json:"current_stage"`
	TargetStage    string               `json:"target_stage"`
	Status         BatchStatus          `json:"status"`
	Revision       int64                `json:"revision"`
	BaselineDigest string               `json:"baseline_digest,omitempty"`
	CreatedBy      string               `json:"created_by"`
	CreatedAt      time.Time            `json:"created_at"`
	SealedAt       *time.Time           `json:"sealed_at,omitempty"`
	Protocol       *FrozenProtocol      `json:"protocol,omitempty"`
	Observations   []ProcessObservation `json:"observations"`
	Deviations     []DeviationCase      `json:"deviations"`
	Review         *QualificationReview `json:"review,omitempty"`
	Participants   ParticipantRoles     `json:"participants"`
}

func NewTreatmentBatch(id, specimen, species, currentStage, targetStage, actor string, now time.Time) (*TreatmentBatch, error) {
	values := []string{id, specimen, species, currentStage, targetStage, actor}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return nil, NewRuleError("missing_batch_field", "批次建档字段不能为空")
		}
	}
	if currentStage == targetStage {
		return nil, NewRuleError("same_stage", "当前阶段和目标阶段不能相同")
	}
	if now.IsZero() {
		return nil, NewRuleError("invalid_created_at", "建档时间不能为空")
	}
	return &TreatmentBatch{BatchID: id, SpecimenCode: specimen, WoodSpecies: species, CurrentStage: currentStage, TargetStage: targetStage, Status: StatusDraft, Revision: 1, CreatedBy: actor, CreatedAt: utc(now), Observations: []ProcessObservation{}, Deviations: []DeviationCase{}}, nil
}

func (b *TreatmentBatch) ensureMutable() error {
	if b.Status.Terminal() {
		return ErrSealed
	}
	return nil
}

func (b *TreatmentBatch) FreezeProtocol(protocol FrozenProtocol) error {
	if err := b.ensureMutable(); err != nil {
		return err
	}
	if b.Status != StatusDraft || b.Protocol != nil {
		return NewRuleError("baseline_already_frozen", "基线已冻结或批次不在草稿状态")
	}
	if protocol.BatchID != b.BatchID {
		return NewRuleError("protocol_batch_mismatch", "协议批次标识不匹配")
	}
	if err := protocol.Validate(); err != nil {
		return err
	}
	digest, err := protocol.Digest()
	if err != nil {
		return err
	}
	protocol.FrozenAt = utc(protocol.FrozenAt)
	b.Protocol, b.BaselineDigest, b.Status = &protocol, digest, StatusMonitoring
	b.Participants.Baseline = appendUnique(b.Participants.Baseline, protocol.FrozenBy)
	b.Revision++
	return nil
}

func (b *TreatmentBatch) AddObservation(observation ProcessObservation, now time.Time, newDeviationID func(string) string) (EvaluationResult, error) {
	if err := b.ensureMutable(); err != nil {
		return EvaluationResult{}, err
	}
	if b.Protocol == nil {
		return EvaluationResult{}, NewRuleError("baseline_not_frozen", "必须先冻结基线")
	}
	if b.Status != StatusMonitoring && b.Status != StatusRecovering {
		return EvaluationResult{}, ErrInvalidState
	}
	if observation.BatchID != b.BatchID {
		return EvaluationResult{}, NewRuleError("observation_batch_mismatch", "观测批次标识不匹配")
	}
	if err := observation.ValidateShape(); err != nil {
		return EvaluationResult{}, err
	}
	var previous *ProcessObservation
	if len(b.Observations) > 0 {
		last := &b.Observations[len(b.Observations)-1]
		if observation.SequenceNo <= last.SequenceNo {
			if observation.SequenceNo == last.SequenceNo {
				return EvaluationResult{}, ErrDuplicateSequence
			}
			return EvaluationResult{}, ErrTimeRegression
		}
		if !observation.CapturedAt.After(last.CapturedAt) {
			return EvaluationResult{}, ErrTimeRegression
		}
		previous = last
	}
	result := EvaluateObservation(*b.Protocol, previous, observation)
	observation.CapturedAt = utc(observation.CapturedAt)
	observation.RuleCodes = append([]string(nil), result.RuleCodes...)
	if result.Qualified {
		observation.Evaluation = EvaluationQualified
	} else {
		observation.Evaluation = EvaluationDeviation
	}
	b.Observations = append(b.Observations, observation)
	b.Participants.Observers = appendUnique(b.Participants.Observers, observation.RecordedBy)
	if !result.Qualified {
		for _, code := range result.RuleCodes {
			b.Deviations = append(b.Deviations, DeviationCase{DeviationID: newDeviationID(code), BatchID: b.BatchID, RuleCode: code, DetectedAt: utc(now), Status: DeviationOpen, QualifiedObservationIDs: []string{}})
		}
		if b.Status == StatusRecovering {
			for i := range b.Deviations {
				if b.Deviations[i].Status != DeviationClosed {
					b.Deviations[i].ResetRecovery()
				}
			}
		}
		b.Status = StatusPaused
	} else if b.Status == StatusRecovering {
		complete := true
		for i := range b.Deviations {
			d := &b.Deviations[i]
			if d.Status == DeviationCorrected {
				d.AddQualified(observation.ObservationID)
				if len(d.QualifiedObservationIDs) >= b.Protocol.RecoveryWindowCount {
					d.Close(now)
				} else {
					complete = false
				}
			}
			if d.Status != DeviationClosed {
				complete = false
			}
		}
		if complete {
			b.Status = StatusReview
		}
	}
	b.Revision++
	return result, nil
}

func (b *TreatmentBatch) RegisterCorrection(deviationID, cause, action, owner string) error {
	if err := b.ensureMutable(); err != nil {
		return err
	}
	if b.Status != StatusPaused {
		return ErrInvalidState
	}
	for i := range b.Deviations {
		if b.Deviations[i].DeviationID == deviationID {
			if err := b.Deviations[i].RegisterCorrection(cause, action, owner); err != nil {
				return err
			}
			b.Participants.Correctors = appendUnique(b.Participants.Correctors, owner)
			b.Revision++
			return nil
		}
	}
	return ErrNotFound
}

func (b *TreatmentBatch) ApproveRecovery(actor string, now time.Time) error {
	if err := b.ensureMutable(); err != nil {
		return err
	}
	if b.Status != StatusPaused {
		return ErrInvalidState
	}
	open := 0
	for i := range b.Deviations {
		if b.Deviations[i].Status == DeviationOpen {
			return NewRuleError("uncorrected_deviation", "仍有偏差未登记纠正措施")
		}
		if b.Deviations[i].Status == DeviationCorrected {
			open++
		}
	}
	if open == 0 {
		return NewRuleError("no_recovery_required", "没有可恢复的开放偏差")
	}
	if strings.TrimSpace(actor) == "" {
		return NewRuleError("missing_actor", "批准恢复人不能为空")
	}
	for i := range b.Deviations {
		if b.Deviations[i].Status == DeviationCorrected {
			_ = b.Deviations[i].StartRecovery(now)
		}
	}
	b.Participants.Correctors = appendUnique(b.Participants.Correctors, actor)
	b.Status = StatusRecovering
	b.Revision++
	return nil
}

func (b *TreatmentBatch) DecideReview(review QualificationReview) error {
	if err := b.ensureMutable(); err != nil {
		return err
	}
	if b.Status != StatusReview {
		return ErrInvalidState
	}
	if review.BatchID != b.BatchID {
		return NewRuleError("review_batch_mismatch", "复核批次标识不匹配")
	}
	if err := review.Validate(); err != nil {
		return err
	}
	if review.ReviewerID == b.CreatedBy || contains(b.Participants.Baseline, review.ReviewerID) || contains(b.Participants.Observers, review.ReviewerID) || contains(b.Participants.Correctors, review.ReviewerID) {
		return ErrDutySeparation
	}
	if len(b.Observations) == 0 {
		return NewRuleError("missing_observations", "没有可供复核的观测")
	}
	for _, d := range b.Deviations {
		if d.Status != DeviationClosed {
			return NewRuleError("open_deviation", "仍有未关闭偏差")
		}
	}
	review.DecidedAt = utc(review.DecidedAt)
	b.Review = &review
	t := review.DecidedAt
	b.SealedAt = &t
	if review.Decision == DecisionApprove {
		b.Status = StatusApproved
	} else {
		b.Status = StatusRejected
	}
	b.Revision++
	return nil
}

func (b *TreatmentBatch) OpenDeviations() []DeviationCase {
	result := make([]DeviationCase, 0)
	for _, d := range b.Deviations {
		if d.Status != DeviationClosed {
			result = append(result, d)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].DetectedAt.Before(result[j].DetectedAt) })
	return result
}

func (b *TreatmentBatch) ValidateLoaded() error {
	if b.BatchID == "" || b.Revision < 1 {
		return fmt.Errorf("损坏的批次聚合")
	}
	if b.Status != StatusDraft && b.Protocol == nil {
		return fmt.Errorf("非草稿批次缺少冻结协议")
	}
	if b.Status.Terminal() && (b.Review == nil || b.SealedAt == nil) {
		return fmt.Errorf("封存批次缺少复核材料")
	}
	return nil
}
