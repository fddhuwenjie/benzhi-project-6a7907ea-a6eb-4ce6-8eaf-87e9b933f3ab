package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

type FrozenProtocol struct {
	ProtocolID                string    `json:"protocol_id"`
	BatchID                   string    `json:"batch_id"`
	TargetConcentrationPct    float64   `json:"target_concentration_pct"`
	ConcentrationTolerancePct float64   `json:"concentration_tolerance_pct"`
	TemperatureMinC           float64   `json:"temperature_min_c"`
	TemperatureMaxC           float64   `json:"temperature_max_c"`
	MassChangeLimitPct        float64   `json:"mass_change_limit_pct"`
	ObservationIntervalHours  int       `json:"observation_interval_hours"`
	RecoveryWindowCount       int       `json:"recovery_window_count"`
	FrozenBy                  string    `json:"frozen_by"`
	FrozenAt                  time.Time `json:"frozen_at"`
}

func (p FrozenProtocol) Validate() error {
	if strings.TrimSpace(p.ProtocolID) == "" || strings.TrimSpace(p.BatchID) == "" || strings.TrimSpace(p.FrozenBy) == "" {
		return NewRuleError("invalid_protocol_identity", "协议标识、批次标识和冻结人不能为空")
	}
	if p.TargetConcentrationPct < 0 || p.TargetConcentrationPct > 100 || p.ConcentrationTolerancePct < 0 || p.ConcentrationTolerancePct > 100 {
		return NewRuleError("invalid_concentration_rule", "浓度目标或容差超出允许范围")
	}
	if p.TemperatureMinC < -20 || p.TemperatureMaxC > 100 || p.TemperatureMinC >= p.TemperatureMaxC {
		return NewRuleError("invalid_temperature_rule", "温度区间无效")
	}
	if p.MassChangeLimitPct <= 0 || p.MassChangeLimitPct > 100 {
		return NewRuleError("invalid_mass_rule", "质量变化阈值必须在 0 到 100 之间")
	}
	if p.ObservationIntervalHours <= 0 || p.ObservationIntervalHours > 720 {
		return NewRuleError("invalid_interval", "观测间隔必须在 1 到 720 小时之间")
	}
	if p.RecoveryWindowCount < 1 || p.RecoveryWindowCount > 100 {
		return NewRuleError("invalid_recovery_window", "恢复窗口数量必须在 1 到 100 之间")
	}
	if p.FrozenAt.IsZero() {
		return NewRuleError("invalid_frozen_at", "冻结时间不能为空")
	}
	return nil
}

func (p FrozenProtocol) Digest() (string, error) {
	view := struct {
		ProtocolID     string `json:"protocol_id"`
		BatchID        string `json:"batch_id"`
		Target         string `json:"target_concentration_pct"`
		Tolerance      string `json:"concentration_tolerance_pct"`
		TemperatureMin string `json:"temperature_min_c"`
		TemperatureMax string `json:"temperature_max_c"`
		MassLimit      string `json:"mass_change_limit_pct"`
		Interval       int    `json:"observation_interval_hours"`
		Recovery       int    `json:"recovery_window_count"`
		FrozenBy       string `json:"frozen_by"`
		FrozenAt       string `json:"frozen_at"`
	}{p.ProtocolID, p.BatchID, formatNumber(p.TargetConcentrationPct), formatNumber(p.ConcentrationTolerancePct), formatNumber(p.TemperatureMinC), formatNumber(p.TemperatureMaxC), formatNumber(p.MassChangeLimitPct), p.ObservationIntervalHours, p.RecoveryWindowCount, p.FrozenBy, utc(p.FrozenAt).Format(time.RFC3339Nano)}
	b, err := json.Marshal(view)
	if err != nil {
		return "", fmt.Errorf("编码协议摘要: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func formatNumber(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "invalid"
	}
	return fmt.Sprintf("%.6f", v)
}
