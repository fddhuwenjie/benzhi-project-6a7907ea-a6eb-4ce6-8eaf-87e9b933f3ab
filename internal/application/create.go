package application

import (
	"context"

	"timber-stage-qualifier/internal/domain"
	"timber-stage-qualifier/internal/repository"
)

func (s *Service) CreateBatch(ctx context.Context, command CreateBatchCommand) (Outcome[BatchResult], error) {
	fp, err := fingerprint(command)
	if err != nil {
		return Outcome[BatchResult]{}, err
	}
	now := s.now().UTC()
	result, err := s.repo.Create(ctx, command.BatchID, command.Meta.RequestID, fp, command.Meta.ExpectedRevision, func() (*domain.TreatmentBatch, any, repository.EventInput, error) {
		batch, err := domain.NewTreatmentBatch(command.BatchID, command.SpecimenCode, command.WoodSpecies, command.CurrentStage, command.TargetStage, command.Meta.ActorID, now)
		if err != nil {
			return nil, nil, repository.EventInput{}, err
		}
		response := BatchResult{BatchID: batch.BatchID, Status: batch.Status, Revision: batch.Revision}
		event := repository.EventInput{Type: "BATCH_CREATED", ActorID: command.Meta.ActorID, At: now, Payload: map[string]any{"specimen_code": batch.SpecimenCode, "target_stage": batch.TargetStage}}
		return batch, response, event, nil
	})
	if err != nil {
		return Outcome[BatchResult]{}, err
	}
	return decodeResult[BatchResult](result)
}

func (s *Service) FreezeProtocol(ctx context.Context, command FreezeProtocolCommand) (Outcome[BatchResult], error) {
	fp, err := fingerprint(command)
	if err != nil {
		return Outcome[BatchResult]{}, err
	}
	now := s.now().UTC()
	result, err := s.repo.Mutate(ctx, command.BatchID, command.Meta.RequestID, fp, command.Meta.ExpectedRevision, func(batch *domain.TreatmentBatch, _ string) (any, repository.EventInput, *repository.CertificateRecord, error) {
		protocol := domain.FrozenProtocol{ProtocolID: command.ProtocolID, BatchID: command.BatchID, TargetConcentrationPct: command.TargetConcentrationPct, ConcentrationTolerancePct: command.ConcentrationTolerancePct, TemperatureMinC: command.TemperatureMinC, TemperatureMaxC: command.TemperatureMaxC, MassChangeLimitPct: command.MassChangeLimitPct, ObservationIntervalHours: command.ObservationIntervalHours, RecoveryWindowCount: command.RecoveryWindowCount, FrozenBy: command.Meta.ActorID, FrozenAt: now}
		if err := batch.FreezeProtocol(protocol); err != nil {
			return nil, repository.EventInput{}, nil, err
		}
		response := BatchResult{BatchID: batch.BatchID, Status: batch.Status, Revision: batch.Revision, BaselineDigest: batch.BaselineDigest}
		event := repository.EventInput{Type: "BASELINE_FROZEN", ActorID: command.Meta.ActorID, At: now, Payload: map[string]any{"protocol_id": protocol.ProtocolID, "baseline_digest": batch.BaselineDigest}}
		return response, event, nil, nil
	})
	if err != nil {
		return Outcome[BatchResult]{}, err
	}
	return decodeResult[BatchResult](result)
}
