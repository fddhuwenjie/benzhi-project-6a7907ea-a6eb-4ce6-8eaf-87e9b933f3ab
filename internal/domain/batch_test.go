package domain

import (
	"errors"
	"testing"
	"time"
)

func testBatch(t *testing.T, recoveryCount int) *TreatmentBatch {
	t.Helper()
	now := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
	batch, err := NewTreatmentBatch("batch-1", "wood-1", "oak", "PEG-20", "PEG-30", "creator", now)
	if err != nil {
		t.Fatal(err)
	}
	err = batch.FreezeProtocol(FrozenProtocol{ProtocolID: "protocol-1", BatchID: batch.BatchID, TargetConcentrationPct: 30, ConcentrationTolerancePct: 1, TemperatureMinC: 18, TemperatureMaxC: 24, MassChangeLimitPct: 3, ObservationIntervalHours: 4, RecoveryWindowCount: recoveryCount, FrozenBy: "operator", FrozenAt: now})
	if err != nil {
		t.Fatal(err)
	}
	return batch
}

func observation(id string, sequence int64, at time.Time, concentration float64, actor string) ProcessObservation {
	return ProcessObservation{ObservationID: id, BatchID: "batch-1", SequenceNo: sequence, CapturedAt: at, ConcentrationPct: concentration, TemperatureC: 20, TimberMassG: 1000, EvidenceDigest: "sha256:domain-test-evidence", RecordedBy: actor}
}

func TestDeviationRecoveryAndIndependentApproval(t *testing.T) {
	batch := testBatch(t, 2)
	base := batch.CreatedAt
	if _, err := batch.AddObservation(observation("obs-1", 1, base.Add(time.Hour), 30, "observer"), base.Add(time.Hour), func(string) string { return "dev-unused" }); err != nil {
		t.Fatal(err)
	}
	result, err := batch.AddObservation(observation("obs-2", 2, base.Add(2*time.Hour), 35, "observer"), base.Add(2*time.Hour), func(string) string { return "dev-concentration" })
	if err != nil {
		t.Fatal(err)
	}
	if result.Qualified || batch.Status != StatusPaused || len(batch.OpenDeviations()) != 1 {
		t.Fatalf("异常观测状态错误: %+v, %s", result, batch.Status)
	}
	if err := batch.RegisterCorrection("dev-concentration", "计量错误", "重新配液", "corrector"); err != nil {
		t.Fatal(err)
	}
	if err := batch.ApproveRecovery("supervisor", base.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := batch.AddObservation(observation("obs-3", 3, base.Add(3*time.Hour), 30, "observer"), base.Add(3*time.Hour), func(string) string { return "dev-3" }); err != nil {
		t.Fatal(err)
	}
	if batch.Status != StatusRecovering {
		t.Fatalf("第一条恢复观测后状态为 %s", batch.Status)
	}
	if _, err := batch.AddObservation(observation("obs-4", 4, base.Add(4*time.Hour), 30, "observer"), base.Add(4*time.Hour), func(string) string { return "dev-4" }); err != nil {
		t.Fatal(err)
	}
	if batch.Status != StatusReview {
		t.Fatalf("恢复窗口后状态为 %s", batch.Status)
	}
	review := QualificationReview{ReviewID: "review-1", BatchID: batch.BatchID, ReviewerID: "reviewer", Decision: DecisionApprove, ChecklistResults: []ChecklistResult{{Item: "完整性", Passed: true}}, Rationale: "证据完整", EvidenceRootDigest: "root", DecidedAt: base.Add(5 * time.Hour)}
	if err := batch.DecideReview(review); err != nil {
		t.Fatal(err)
	}
	if batch.Status != StatusApproved || batch.SealedAt == nil {
		t.Fatalf("批准未封存: %s", batch.Status)
	}
	if _, err := batch.AddObservation(observation("obs-5", 5, base.Add(5*time.Hour), 30, "observer"), base.Add(5*time.Hour), func(string) string { return "dev-5" }); !errors.Is(err, ErrSealed) {
		t.Fatalf("终态写入应返回 ErrSealed，实际 %v", err)
	}
}

func TestRecoveryFailureClearsWindowAndPauses(t *testing.T) {
	batch := testBatch(t, 2)
	base := batch.CreatedAt
	_, _ = batch.AddObservation(observation("obs-1", 1, base.Add(time.Hour), 30, "observer"), base, func(string) string { return "unused" })
	_, _ = batch.AddObservation(observation("obs-2", 2, base.Add(2*time.Hour), 35, "observer"), base, func(string) string { return "dev-original" })
	if err := batch.RegisterCorrection("dev-original", "原因", "措施", "corrector"); err != nil {
		t.Fatal(err)
	}
	if err := batch.ApproveRecovery("supervisor", base.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	_, _ = batch.AddObservation(observation("obs-3", 3, base.Add(3*time.Hour), 30, "observer"), base, func(string) string { return "unused" })
	_, err := batch.AddObservation(observation("obs-4", 4, base.Add(4*time.Hour), 40, "observer"), base, func(code string) string { return "new-" + code })
	if err != nil {
		t.Fatal(err)
	}
	if batch.Status != StatusPaused {
		t.Fatalf("失败恢复观测应暂停，实际 %s", batch.Status)
	}
	for _, d := range batch.Deviations {
		if d.Status != DeviationClosed && len(d.QualifiedObservationIDs) != 0 {
			t.Fatalf("恢复窗口未清空: %+v", d)
		}
	}
}

func TestReviewerDutySeparation(t *testing.T) {
	batch := testBatch(t, 1)
	base := batch.CreatedAt
	_, _ = batch.AddObservation(observation("obs-1", 1, base.Add(time.Hour), 35, "operator"), base, func(string) string { return "dev" })
	_ = batch.RegisterCorrection("dev", "原因", "措施", "corrector")
	_ = batch.ApproveRecovery("supervisor", base)
	_, _ = batch.AddObservation(observation("obs-2", 2, base.Add(2*time.Hour), 30, "operator"), base, func(string) string { return "unused" })
	review := QualificationReview{ReviewID: "review", BatchID: batch.BatchID, ReviewerID: "operator", Decision: DecisionApprove, ChecklistResults: []ChecklistResult{{Item: "完整性", Passed: true}}, Rationale: "通过", DecidedAt: base.Add(3 * time.Hour)}
	if err := batch.DecideReview(review); !errors.Is(err, ErrDutySeparation) {
		t.Fatalf("参与者复核应被拒绝，实际 %v", err)
	}
}
