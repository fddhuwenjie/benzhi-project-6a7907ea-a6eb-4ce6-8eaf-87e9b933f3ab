package domain

import (
	"math"
	"strings"
	"time"
)

type ProcessObservation struct {
	ObservationID    string     `json:"observation_id"`
	BatchID          string     `json:"batch_id"`
	SequenceNo       int64      `json:"sequence_no"`
	CapturedAt       time.Time  `json:"captured_at"`
	ConcentrationPct float64    `json:"concentration_pct"`
	TemperatureC     float64    `json:"temperature_c"`
	TimberMassG      float64    `json:"timber_mass_g"`
	EvidenceDigest   string     `json:"evidence_digest"`
	RecordedBy       string     `json:"recorded_by"`
	Evaluation       Evaluation `json:"evaluation"`
	RuleCodes        []string   `json:"rule_codes,omitempty"`
}

func (o ProcessObservation) ValidateShape() error {
	if strings.TrimSpace(o.ObservationID) == "" || strings.TrimSpace(o.BatchID) == "" || strings.TrimSpace(o.RecordedBy) == "" {
		return NewRuleError("invalid_observation_identity", "观测标识、批次标识和记录人不能为空")
	}
	if o.SequenceNo < 1 || o.CapturedAt.IsZero() {
		return NewRuleError("invalid_observation_order", "观测序号和采集时间无效")
	}
	if math.IsNaN(o.ConcentrationPct) || math.IsInf(o.ConcentrationPct, 0) || o.ConcentrationPct < 0 || o.ConcentrationPct > 100 {
		return NewRuleError("metric_outside_baseline", "溶液浓度必须在 0 到 100 之间")
	}
	if math.IsNaN(o.TemperatureC) || math.IsInf(o.TemperatureC, 0) || o.TemperatureC < -20 || o.TemperatureC > 100 {
		return NewRuleError("metric_outside_baseline", "温度超出基线可接受量程")
	}
	if math.IsNaN(o.TimberMassG) || math.IsInf(o.TimberMassG, 0) || o.TimberMassG <= 0 {
		return NewRuleError("metric_outside_baseline", "木材质量必须为正数")
	}
	if len(strings.TrimSpace(o.EvidenceDigest)) < 16 || len(o.EvidenceDigest) > 256 {
		return NewRuleError("invalid_evidence_digest", "证据摘要长度无效")
	}
	return nil
}

type EvaluationResult struct {
	Qualified bool     `json:"qualified"`
	RuleCodes []string `json:"rule_codes"`
}

func EvaluateObservation(protocol FrozenProtocol, previous *ProcessObservation, current ProcessObservation) EvaluationResult {
	codes := make([]string, 0, 4)
	if math.Abs(current.ConcentrationPct-protocol.TargetConcentrationPct) > protocol.ConcentrationTolerancePct {
		codes = append(codes, "CONCENTRATION_OUT_OF_RANGE")
	}
	if current.TemperatureC < protocol.TemperatureMinC || current.TemperatureC > protocol.TemperatureMaxC {
		codes = append(codes, "TEMPERATURE_OUT_OF_RANGE")
	}
	if previous != nil {
		massChange := math.Abs(current.TimberMassG-previous.TimberMassG) / previous.TimberMassG * 100
		if massChange > protocol.MassChangeLimitPct {
			codes = append(codes, "MASS_CHANGE_EXCEEDED")
		}
		allowed := time.Duration(protocol.ObservationIntervalHours) * time.Hour
		if current.CapturedAt.Sub(previous.CapturedAt) > allowed {
			codes = append(codes, "OBSERVATION_MISSED")
		}
	}
	return EvaluationResult{Qualified: len(codes) == 0, RuleCodes: codes}
}
