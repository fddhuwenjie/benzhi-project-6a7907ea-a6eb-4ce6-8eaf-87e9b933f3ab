package lockwaitcancellation_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"timber-stage-qualifier/internal/domain"
	"timber-stage-qualifier/internal/repository"
)

func TestCanceledWriterLeavesLockQueue(t *testing.T) {
	ctx := context.Background()
	repo, err := repository.Open(ctx, filepath.Join(t.TempDir(), "lock-wait.db"))
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	defer repo.Close()

	base := time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC)
	_, err = repo.Create(ctx, "batch-lock-wait", "request-create", "fingerprint-create", 0, func() (*domain.TreatmentBatch, any, repository.EventInput, error) {
		batch, createErr := domain.NewTreatmentBatch("batch-lock-wait", "specimen-lock", "oak", "stage-a", "stage-b", "keeper-a", base)
		return batch, map[string]string{"batch_id": "batch-lock-wait"}, repository.EventInput{Type: "BATCH_CREATED", ActorID: "keeper-a", At: base, Payload: map[string]string{"specimen_code": "specimen-lock"}}, createErr
	})
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		_, mutateErr := repo.Mutate(ctx, "batch-lock-wait", "request-first", "fingerprint-first", 1, func(batch *domain.TreatmentBatch, _ string) (any, repository.EventInput, *repository.CertificateRecord, error) {
			close(entered)
			<-release
			batch.Revision++
			return map[string]int64{"revision": batch.Revision}, repository.EventInput{Type: "FIRST_MUTATION", ActorID: "keeper-a", At: base.Add(time.Minute), Payload: map[string]string{"step": "first"}}, nil, nil
		})
		firstDone <- mutateErr
	}()
	<-entered

	secondCtx, cancelSecond := context.WithCancel(ctx)
	cancelSecond()
	secondDone := make(chan error, 1)
	go func() {
		_, mutateErr := repo.Mutate(secondCtx, "batch-lock-wait", "request-second", "fingerprint-second", 2, func(batch *domain.TreatmentBatch, _ string) (any, repository.EventInput, *repository.CertificateRecord, error) {
			batch.Revision++
			return map[string]int64{"revision": batch.Revision}, repository.EventInput{Type: "SECOND_MUTATION", ActorID: "keeper-b", At: base.Add(2 * time.Minute), Payload: map[string]string{"step": "second"}}, nil, nil
		})
		secondDone <- mutateErr
	}()

	secondReceived := false
	defer func() {
		close(release)
		if firstErr := <-firstDone; firstErr != nil {
			t.Errorf("first mutation cleanup: %v", firstErr)
		}
		if !secondReceived {
			<-secondDone
		}
	}()

	select {
	case secondErr := <-secondDone:
		secondReceived = true
		if !errors.Is(secondErr, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", secondErr)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("TestCanceledWriterLeavesLockQueue: canceled writer remained blocked behind the active batch mutation")
	}
}
