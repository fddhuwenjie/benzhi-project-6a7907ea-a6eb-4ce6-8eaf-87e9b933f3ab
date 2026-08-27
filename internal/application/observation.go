package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"timber-stage-qualifier/internal/domain"
	"timber-stage-qualifier/internal/repository"
)

func deviationID(observationID, code string) string {
	sum := sha256.Sum256([]byte(observationID + ":" + code))
	return "dev_" + hex.EncodeToString(sum[:8])
}

func (s *Service) SubmitObservation(ctx context.Context, command SubmitObservationCommand) (Outcome[ObservationResult], error) {
	fp, err := fingerprint(command)
	if err != nil {
		return Outcome[ObservationResult]{}, err
	}
	now := s.now().UTC()
	result, err := s.repo.Mutate(ctx, command.BatchID, command.Meta.RequestID, fp, command.Meta.ExpectedRevision, func(batch *domain.TreatmentBatch, _ string) (any, repository.EventInput, *repository.CertificateRecord, error) {
		observation := domain.ProcessObservation{ObservationID: command.ObservationID, BatchID: command.BatchID, SequenceNo: command.SequenceNo, CapturedAt: command.CapturedAt, ConcentrationPct: command.ConcentrationPct, TemperatureC: command.TemperatureC, TimberMassG: command.TimberMassG, EvidenceDigest: command.EvidenceDigest, RecordedBy: command.Meta.ActorID}
		evaluation, err := batch.AddObservation(observation, now, func(code string) string { return deviationID(command.ObservationID, code) })
		if err != nil {
			return nil, repository.EventInput{}, nil, err
		}
		open := batch.OpenDeviations()
		ids := make([]string, 0, len(open))
		for _, d := range open {
			ids = append(ids, d.DeviationID)
		}
		response := ObservationResult{BatchResult: BatchResult{BatchID: batch.BatchID, Status: batch.Status, Revision: batch.Revision, BaselineDigest: batch.BaselineDigest}, ObservationID: command.ObservationID, Evaluation: domain.EvaluationQualified, RuleCodes: evaluation.RuleCodes, OpenDeviationIDs: ids}
		if !evaluation.Qualified {
			response.Evaluation = domain.EvaluationDeviation
		}
		event := repository.EventInput{Type: "OBSERVATION_RECORDED", ActorID: command.Meta.ActorID, At: now, Payload: map[string]any{"observation_id": command.ObservationID, "sequence_no": command.SequenceNo, "evaluation": response.Evaluation, "rule_codes": evaluation.RuleCodes}}
		return response, event, nil, nil
	})
	if err != nil {
		return Outcome[ObservationResult]{}, err
	}
	return decodeResult[ObservationResult](result)
}
