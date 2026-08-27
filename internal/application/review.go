package application

import (
	"context"
	"timber-stage-qualifier/internal/domain"
	"timber-stage-qualifier/internal/evidence"
	"timber-stage-qualifier/internal/repository"
)

func (s *Service) DecideReview(ctx context.Context, command DecideReviewCommand) (Outcome[ReviewResult], error) {
	fp, err := fingerprint(command)
	if err != nil {
		return Outcome[ReviewResult]{}, err
	}
	now := s.now().UTC()
	result, err := s.repo.Mutate(ctx, command.BatchID, command.Meta.RequestID, fp, command.Meta.ExpectedRevision, func(batch *domain.TreatmentBatch, auditRoot string) (any, repository.EventInput, *repository.CertificateRecord, error) {
		if command.EvidenceRootDigest != "" && command.EvidenceRootDigest != auditRoot {
			return nil, repository.EventInput{}, nil, domain.NewRuleError("evidence_root_mismatch", "提交的证据根与当前审计根不匹配")
		}
		review := domain.QualificationReview{ReviewID: command.ReviewID, BatchID: command.BatchID, ReviewerID: command.Meta.ActorID, Decision: command.Decision, ChecklistResults: command.ChecklistResults, Rationale: command.Rationale, EvidenceRootDigest: auditRoot, DecidedAt: now}
		if err := batch.DecideReview(review); err != nil {
			return nil, repository.EventInput{}, nil, err
		}
		response := ReviewResult{BatchResult: BatchResult{BatchID: batch.BatchID, Status: batch.Status, Revision: batch.Revision, BaselineDigest: batch.BaselineDigest}, ReviewID: review.ReviewID, Decision: review.Decision}
		var record *repository.CertificateRecord
		if review.Decision == domain.DecisionApprove {
			certificate, err := s.certificates.Generate(batch, s.newID("cert"), auditRoot, now)
			if err != nil {
				return nil, repository.EventInput{}, nil, err
			}
			record = certificateRecord(certificate)
			response.CertificateID = certificate.CertificateID
			response.CertificateSHA256 = certificate.PayloadSHA256
		}
		event := repository.EventInput{Type: "REVIEW_DECIDED", ActorID: command.Meta.ActorID, At: now, Payload: map[string]any{"review_id": review.ReviewID, "decision": review.Decision, "certificate_id": response.CertificateID}}
		return response, event, record, nil
	})
	if err != nil {
		return Outcome[ReviewResult]{}, err
	}
	return decodeResult[ReviewResult](result)
}

func certificateRecord(c *evidence.Certificate) *repository.CertificateRecord {
	return &repository.CertificateRecord{CertificateID: c.CertificateID, BatchID: c.BatchID, DocumentVersion: c.DocumentVersion, CanonicalPayload: c.CanonicalPayload, PayloadSHA256: c.PayloadSHA256, AuditRootDigest: c.AuditRootDigest, IssuedAt: c.IssuedAt, VerifiedAt: c.VerifiedAt}
}

func evidenceCertificate(c *repository.CertificateRecord) *evidence.Certificate {
	return &evidence.Certificate{CertificateID: c.CertificateID, BatchID: c.BatchID, DocumentVersion: c.DocumentVersion, CanonicalPayload: c.CanonicalPayload, PayloadSHA256: c.PayloadSHA256, AuditRootDigest: c.AuditRootDigest, IssuedAt: c.IssuedAt, VerifiedAt: c.VerifiedAt}
}
