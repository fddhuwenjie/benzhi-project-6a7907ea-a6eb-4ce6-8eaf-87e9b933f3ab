package auditcachealias_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"timber-stage-qualifier/internal/domain"
	"timber-stage-qualifier/internal/repository"
)

func TestAuditTimelineCacheIsolatedFromCallerMutation(t *testing.T) {
	ctx := context.Background()
	repo, err := repository.Open(ctx, filepath.Join(t.TempDir(), "audit-cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	at := time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC)
	_, err = repo.Create(ctx, "batch-cache", "request-create", "fingerprint-create", 0, func() (*domain.TreatmentBatch, any, repository.EventInput, error) {
		batch, createErr := domain.NewTreatmentBatch("batch-cache", "specimen-cache", "oak", "PEG-10", "PEG-20", "operator", at)
		return batch, map[string]string{"batch_id": "batch-cache"}, repository.EventInput{
			Type: "BATCH_CREATED", ActorID: "operator", At: at,
			Payload: map[string]string{"specimen_code": "specimen-cache"},
		}, createErr
	})
	if err != nil {
		t.Fatal(err)
	}

	first, err := repo.AuditTimeline(ctx, "batch-cache")
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || !repository.VerifyEvents(first).Valid {
		t.Fatalf("初次读取应得到有效审计时间线: %+v", first)
	}
	first[0].EventType = "CALLER_MUTATED"
	payloadIndex := bytes.Index(first[0].Payload, []byte("specimen-cache"))
	if payloadIndex < 0 {
		t.Fatalf("审计载荷缺少预期标本编码: %q", first[0].Payload)
	}
	first[0].Payload[payloadIndex] = 'X'

	second, err := repo.AuditTimeline(ctx, "batch-cache")
	if err != nil {
		t.Fatal(err)
	}
	verification, err := repo.VerifyAudit(ctx, "batch-cache")
	if err != nil {
		t.Fatal(err)
	}
	if second[0].EventType != "BATCH_CREATED" || bytes.Contains(second[0].Payload, []byte("Xpecimen-cache")) || !verification.Valid {
		t.Fatalf("调用方修改查询结果污染了后续审计读取: event_type=%q payload=%q verification=%+v", second[0].EventType, second[0].Payload, verification)
	}
}
