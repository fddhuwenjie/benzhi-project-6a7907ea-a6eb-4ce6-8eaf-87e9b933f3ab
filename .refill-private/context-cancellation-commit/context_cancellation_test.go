package contextcancellationcommit

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"timber-stage-qualifier/internal/domain"
	"timber-stage-qualifier/internal/repository"
)

func TestCanceledMutationDoesNotCommit(t *testing.T) {
	ctx := context.Background()
	repo, err := repository.Open(ctx, filepath.Join(t.TempDir(), "cancel.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	now := time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC)
	batch, err := domain.NewTreatmentBatch("batch-cancel", "wood-cancel", "oak", "PEG-10", "PEG-20", "operator", now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.Create(ctx, batch.BatchID, "request-create-cancel", "fingerprint-create-cancel", 0, func() (*domain.TreatmentBatch, any, repository.EventInput, error) {
		return batch, map[string]any{"batch_id": batch.BatchID}, repository.EventInput{Type: "BATCH_CREATED", ActorID: "operator", At: now, Payload: map[string]any{"batch_id": batch.BatchID}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = repo.Mutate(canceled, batch.BatchID, "request-cancel-mutate", "fingerprint-cancel-mutate", 1, func(b *domain.TreatmentBatch, _ string) (any, repository.EventInput, *repository.CertificateRecord, error) {
		b.Revision++
		return map[string]any{"revision": b.Revision}, repository.EventInput{Type: "CANCELED_MUTATION", ActorID: "operator", At: now, Payload: map[string]any{"revision": b.Revision}}, nil, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("已取消请求应返回 context.Canceled，实际 %v", err)
	}

	loaded, err := repo.GetBatch(ctx, batch.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Revision != 1 {
		t.Fatalf("已取消请求不应提交修订，实际 revision=%d", loaded.Revision)
	}
}
