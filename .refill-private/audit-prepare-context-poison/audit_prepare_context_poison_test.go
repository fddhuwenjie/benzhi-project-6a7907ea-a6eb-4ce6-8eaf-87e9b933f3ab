package auditpreparecontextpoison_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"timber-stage-qualifier/internal/domain"
	"timber-stage-qualifier/internal/repository"
)

func TestCanceledAuditPreparationDoesNotPoisonRepository(t *testing.T) {
	ctx := context.Background()
	repo, err := repository.Open(ctx, filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatalf("打开仓储: %v", err)
	}
	defer repo.Close()

	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	batch, err := domain.NewTreatmentBatch("batch-audit-context", "specimen-a", "oak", "PEG-20", "PEG-30", "creator-a", now)
	if err != nil {
		t.Fatalf("建立批次: %v", err)
	}
	_, err = repo.Create(ctx, batch.BatchID, "request-create-audit-context", "fingerprint-create-audit-context", 0, func() (*domain.TreatmentBatch, any, repository.EventInput, error) {
		return batch, map[string]string{"batch_id": batch.BatchID}, repository.EventInput{
			Type: "BATCH_CREATED", ActorID: batch.CreatedBy, At: now,
			Payload: map[string]string{"batch_id": batch.BatchID},
		}, nil
	})
	if err != nil {
		t.Fatalf("保存批次: %v", err)
	}

	requestContext, cancelRequest := context.WithCancel(ctx)
	events, err := repo.AuditTimeline(requestContext, batch.BatchID)
	if err != nil {
		t.Fatalf("首次健康审计查询失败: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("首次健康审计查询返回错误事件: %+v", events)
	}
	cancelRequest()

	events, err = repo.AuditTimeline(ctx, batch.BatchID)
	if errors.Is(err, context.Canceled) {
		t.Fatalf("首次请求结束后的取消状态污染了后续健康审计查询: %v", err)
	}
	if err != nil {
		t.Fatalf("后续健康审计查询失败: %v", err)
	}
	if len(events) != 1 || events[0].BatchID != batch.BatchID {
		t.Fatalf("后续健康审计查询返回错误事件: %+v", events)
	}
}
