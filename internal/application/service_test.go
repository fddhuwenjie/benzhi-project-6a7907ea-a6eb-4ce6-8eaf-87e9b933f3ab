package application

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"timber-stage-qualifier/internal/domain"
	"timber-stage-qualifier/internal/evidence"
	"timber-stage-qualifier/internal/repository"
)

func TestPersistentIdempotencyAndRevisionConflict(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "application.db")
	repo, err := repository.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)
	service := NewServiceWithDependencies(repo, evidence.NewGenerator(), func() time.Time { return now }, func(prefix string) string { return prefix + "-fixed" })
	command := CreateBatchCommand{Meta: CommandMeta{RequestID: "request-create", ExpectedRevision: 0, ActorID: "creator"}, BatchID: "batch-app", SpecimenCode: "wood-app", WoodSpecies: "oak", CurrentStage: "PEG-10", TargetStage: "PEG-20"}
	first, err := service.CreateBatch(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed {
		t.Fatal("首次命令不应标记重放")
	}
	if err := repo.Close(); err != nil {
		t.Fatal(err)
	}
	repo, err = repository.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	service = NewServiceWithDependencies(repo, evidence.NewGenerator(), func() time.Time { return now }, func(prefix string) string { return prefix + "-fixed" })
	second, err := service.CreateBatch(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Replayed || second.Value.Revision != first.Value.Revision {
		t.Fatalf("持久幂等重放错误: %+v", second)
	}
	changed := command
	changed.SpecimenCode = "different"
	if _, err := service.CreateBatch(ctx, changed); !errors.Is(err, domain.ErrIdempotency) {
		t.Fatalf("不同指纹应冲突，实际 %v", err)
	}
	freeze := FreezeProtocolCommand{Meta: CommandMeta{RequestID: "freeze-stale", ExpectedRevision: 99, ActorID: "operator"}, BatchID: command.BatchID, ProtocolID: "p", TargetConcentrationPct: 20, ConcentrationTolerancePct: 1, TemperatureMinC: 18, TemperatureMaxC: 24, MassChangeLimitPct: 3, ObservationIntervalHours: 4, RecoveryWindowCount: 2}
	if _, err := service.FreezeProtocol(ctx, freeze); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("陈旧修订应冲突，实际 %v", err)
	}
}
