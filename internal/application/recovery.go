package application

import (
	"context"
	"timber-stage-qualifier/internal/domain"
	"timber-stage-qualifier/internal/repository"
)

func (s *Service) CorrectDeviation(ctx context.Context, command CorrectDeviationCommand) (Outcome[BatchResult], error) {
	fp, err := fingerprint(command)
	if err != nil {
		return Outcome[BatchResult]{}, err
	}
	now := s.now().UTC()
	result, err := s.repo.Mutate(ctx, command.BatchID, command.Meta.RequestID, fp, command.Meta.ExpectedRevision, func(batch *domain.TreatmentBatch, _ string) (any, repository.EventInput, *repository.CertificateRecord, error) {
		if err := batch.RegisterCorrection(command.DeviationID, command.Cause, command.CorrectiveAction, command.OwnerID); err != nil {
			return nil, repository.EventInput{}, nil, err
		}
		response := BatchResult{BatchID: batch.BatchID, Status: batch.Status, Revision: batch.Revision, BaselineDigest: batch.BaselineDigest}
		event := repository.EventInput{Type: "DEVIATION_CORRECTED", ActorID: command.Meta.ActorID, At: now, Payload: map[string]any{"deviation_id": command.DeviationID, "owner_id": command.OwnerID}}
		return response, event, nil, nil
	})
	if err != nil {
		return Outcome[BatchResult]{}, err
	}
	return decodeResult[BatchResult](result)
}

func (s *Service) ApproveRecovery(ctx context.Context, command ApproveRecoveryCommand) (Outcome[BatchResult], error) {
	fp, err := fingerprint(command)
	if err != nil {
		return Outcome[BatchResult]{}, err
	}
	now := s.now().UTC()
	result, err := s.repo.Mutate(ctx, command.BatchID, command.Meta.RequestID, fp, command.Meta.ExpectedRevision, func(batch *domain.TreatmentBatch, _ string) (any, repository.EventInput, *repository.CertificateRecord, error) {
		if err := batch.ApproveRecovery(command.Meta.ActorID, now); err != nil {
			return nil, repository.EventInput{}, nil, err
		}
		response := BatchResult{BatchID: batch.BatchID, Status: batch.Status, Revision: batch.Revision, BaselineDigest: batch.BaselineDigest}
		event := repository.EventInput{Type: "RECOVERY_APPROVED", ActorID: command.Meta.ActorID, At: now, Payload: map[string]any{"recovery_window_count": batch.Protocol.RecoveryWindowCount}}
		return response, event, nil, nil
	})
	if err != nil {
		return Outcome[BatchResult]{}, err
	}
	return decodeResult[BatchResult](result)
}
