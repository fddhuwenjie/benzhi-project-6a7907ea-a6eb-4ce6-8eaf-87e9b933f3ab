package failed_idempotency_replay_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"timber-stage-qualifier/internal/application"
	"timber-stage-qualifier/internal/domain"
	"timber-stage-qualifier/internal/evidence"
	"timber-stage-qualifier/internal/repository"
)

func TestFailedCommandDoesNotPoisonIdempotency(t *testing.T) {
	ctx := context.Background()
	repo, err := repository.Open(ctx, filepath.Join(t.TempDir(), "failed-replay.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	now := time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC)
	service := application.NewServiceWithDependencies(
		repo,
		evidence.NewGenerator(),
		func() time.Time { return now },
		func(prefix string) string { return prefix + "-fixed" },
	)
	freeze := application.FreezeProtocolCommand{
		Meta: application.CommandMeta{
			RequestID:        "request-freeze-before-create",
			ExpectedRevision: 1,
			ActorID:          "conservator-freeze",
		},
		BatchID:                   "batch-late-create",
		ProtocolID:                "protocol-late-create",
		TargetConcentrationPct:    20,
		ConcentrationTolerancePct: 1,
		TemperatureMinC:           18,
		TemperatureMaxC:           24,
		MassChangeLimitPct:        3,
		ObservationIntervalHours:  4,
		RecoveryWindowCount:       2,
	}

	if _, err := service.FreezeProtocol(ctx, freeze); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("首次冻结应因批次不存在失败，实际错误为 %v", err)
	}
	create := application.CreateBatchCommand{
		Meta: application.CommandMeta{
			RequestID:        "request-create-after-failure",
			ExpectedRevision: 0,
			ActorID:          "conservator-create",
		},
		BatchID:      freeze.BatchID,
		SpecimenCode: "timber-late-create",
		WoodSpecies:  "oak",
		CurrentStage: "PEG-10",
		TargetStage:  "PEG-20",
	}
	if _, err := service.CreateBatch(ctx, create); err != nil {
		t.Fatalf("后续建档失败: %v", err)
	}

	result, err := service.FreezeProtocol(ctx, freeze)
	if err != nil {
		t.Fatalf("批次建立后重试冻结应成功，实际错误为 %v", err)
	}
	detail, err := service.GetBatch(ctx, freeze.BatchID)
	if err != nil {
		t.Fatalf("读取重试后的批次失败: %v", err)
	}
	if result.Replayed || result.Value.BatchID != freeze.BatchID || result.Value.Status != domain.StatusMonitoring || result.Value.Revision != 2 || detail.Batch.Status != domain.StatusMonitoring || detail.Batch.Revision != 2 || detail.Batch.Protocol == nil {
		t.Fatalf("首次失败的命令被错误重放，重试未执行真实冻结事务: result=%+v batch=%+v", result, detail.Batch)
	}
}
