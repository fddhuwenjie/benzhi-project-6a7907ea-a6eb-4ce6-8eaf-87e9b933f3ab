package certificaterevisioncache_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"timber-stage-qualifier/internal/application"
	"timber-stage-qualifier/internal/domain"
	"timber-stage-qualifier/internal/evidence"
	"timber-stage-qualifier/internal/repository"
)

func TestCertificatePayloadCacheIsolatedByBatch(t *testing.T) {
	ctx := context.Background()
	repo, err := repository.Open(ctx, filepath.Join(t.TempDir(), "certificate-cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	now := time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC)
	nextID := 0
	service := application.NewServiceWithDependencies(
		repo,
		evidence.NewGenerator(),
		func() time.Time { return now },
		func(prefix string) string {
			nextID++
			return fmt.Sprintf("%s-shared-service-%d", prefix, nextID)
		},
	)

	approveBatch(t, ctx, service, "batch-cache-a", now)
	approveBatch(t, ctx, service, "batch-cache-b", now.Add(time.Hour))

	view, err := service.VerifyCertificate(ctx, "batch-cache-b")
	if err != nil {
		t.Fatal(err)
	}
	if !view.Verification.Valid {
		t.Fatalf("第二批次复用了第一批次的证书载荷: %+v", view.Verification)
	}
}

func approveBatch(t *testing.T, ctx context.Context, service *application.Service, batchID string, capturedAt time.Time) {
	t.Helper()
	create, err := service.CreateBatch(ctx, application.CreateBatchCommand{
		Meta:         meta("create-"+batchID, 0, "creator-"+batchID),
		BatchID:      batchID,
		SpecimenCode: "specimen-" + batchID,
		WoodSpecies:  "oak",
		CurrentStage: "PEG-10",
		TargetStage:  "PEG-20",
	})
	if err != nil {
		t.Fatal(err)
	}

	freeze, err := service.FreezeProtocol(ctx, application.FreezeProtocolCommand{
		Meta:                      meta("freeze-"+batchID, create.Value.Revision, "baseline-"+batchID),
		BatchID:                   batchID,
		ProtocolID:                "protocol-" + batchID,
		TargetConcentrationPct:    20,
		ConcentrationTolerancePct: 1,
		TemperatureMinC:           18,
		TemperatureMaxC:           24,
		MassChangeLimitPct:        3,
		ObservationIntervalHours:  4,
		RecoveryWindowCount:       1,
	})
	if err != nil {
		t.Fatal(err)
	}

	deviation, err := service.SubmitObservation(ctx, application.SubmitObservationCommand{
		Meta:             meta("deviation-"+batchID, freeze.Value.Revision, "observer-"+batchID),
		BatchID:          batchID,
		ObservationID:    "observation-deviation-" + batchID,
		SequenceNo:       1,
		CapturedAt:       capturedAt,
		ConcentrationPct: 25,
		TemperatureC:     20,
		TimberMassG:      1000,
		EvidenceDigest:   "evidence-deviation-" + batchID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(deviation.Value.OpenDeviationIDs) != 1 {
		t.Fatalf("预期一个开放偏差，实际 %+v", deviation.Value.OpenDeviationIDs)
	}

	corrected, err := service.CorrectDeviation(ctx, application.CorrectDeviationCommand{
		Meta:             meta("correct-"+batchID, deviation.Value.Revision, "operator-"+batchID),
		BatchID:          batchID,
		DeviationID:      deviation.Value.OpenDeviationIDs[0],
		Cause:            "浓度配制偏差",
		CorrectiveAction: "重新配制溶液",
		OwnerID:          "owner-" + batchID,
	})
	if err != nil {
		t.Fatal(err)
	}

	recovery, err := service.ApproveRecovery(ctx, application.ApproveRecoveryCommand{
		Meta:    meta("recover-"+batchID, corrected.Value.Revision, "supervisor-"+batchID),
		BatchID: batchID,
	})
	if err != nil {
		t.Fatal(err)
	}

	qualified, err := service.SubmitObservation(ctx, application.SubmitObservationCommand{
		Meta:             meta("qualified-"+batchID, recovery.Value.Revision, "observer-"+batchID),
		BatchID:          batchID,
		ObservationID:    "observation-qualified-" + batchID,
		SequenceNo:       2,
		CapturedAt:       capturedAt.Add(time.Hour),
		ConcentrationPct: 20,
		TemperatureC:     20,
		TimberMassG:      1000,
		EvidenceDigest:   "evidence-qualified-" + batchID,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.DecideReview(ctx, application.DecideReviewCommand{
		Meta:             meta("review-"+batchID, qualified.Value.Revision, "reviewer-"+batchID),
		BatchID:          batchID,
		ReviewID:         "review-" + batchID,
		Decision:         domain.DecisionApprove,
		ChecklistResults: []domain.ChecklistResult{{Item: "证据完整性", Passed: true}},
		Rationale:        "全部资格条件满足",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func meta(requestID string, revision int64, actorID string) application.CommandMeta {
	return application.CommandMeta{RequestID: requestID, ExpectedRevision: revision, ActorID: actorID}
}
