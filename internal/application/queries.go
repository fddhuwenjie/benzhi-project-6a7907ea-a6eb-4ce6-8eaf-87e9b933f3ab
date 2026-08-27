package application

import (
	"context"
	"encoding/json"
	"time"

	"timber-stage-qualifier/internal/domain"
	"timber-stage-qualifier/internal/evidence"
	"timber-stage-qualifier/internal/repository"
)

type BatchDetail struct {
	Batch          *domain.TreatmentBatch `json:"batch"`
	OpenDeviations []domain.DeviationCase `json:"open_deviations"`
}

type AuditView struct {
	Events       []repository.EventRecord     `json:"events"`
	Verification repository.AuditVerification `json:"verification"`
}

type CertificateView struct {
	CertificateID    string          `json:"certificate_id"`
	BatchID          string          `json:"batch_id"`
	DocumentVersion  string          `json:"document_version"`
	CanonicalPayload json.RawMessage `json:"canonical_payload"`
	PayloadSHA256    string          `json:"payload_sha256"`
	AuditRootDigest  string          `json:"audit_root_digest"`
	IssuedAt         time.Time       `json:"issued_at"`
}

type CertificateVerificationView struct {
	CertificateID     string                       `json:"certificate_id"`
	Verification      evidence.Verification        `json:"verification"`
	AuditVerification repository.AuditVerification `json:"audit_verification"`
}

func (s *Service) GetBatch(ctx context.Context, batchID string) (BatchDetail, error) {
	batch, err := s.repo.GetBatch(ctx, batchID)
	if err != nil {
		return BatchDetail{}, err
	}
	return BatchDetail{Batch: batch, OpenDeviations: batch.OpenDeviations()}, nil
}

func (s *Service) GetAudit(ctx context.Context, batchID string) (AuditView, error) {
	events, err := s.repo.AuditTimeline(ctx, batchID)
	if err != nil {
		return AuditView{}, err
	}
	return AuditView{Events: events, Verification: repository.VerifyEvents(events)}, nil
}

func (s *Service) GetCertificate(ctx context.Context, batchID string) (CertificateView, error) {
	c, err := s.repo.GetCertificate(ctx, batchID)
	if err != nil {
		return CertificateView{}, err
	}
	return CertificateView{CertificateID: c.CertificateID, BatchID: c.BatchID, DocumentVersion: c.DocumentVersion, CanonicalPayload: json.RawMessage(c.CanonicalPayload), PayloadSHA256: c.PayloadSHA256, AuditRootDigest: c.AuditRootDigest, IssuedAt: c.IssuedAt}, nil
}

func (s *Service) VerifyCertificate(ctx context.Context, batchID string) (CertificateVerificationView, error) {
	batch, err := s.repo.GetBatch(ctx, batchID)
	if err != nil {
		return CertificateVerificationView{}, err
	}
	certificate, err := s.repo.GetCertificate(ctx, batchID)
	if err != nil {
		return CertificateVerificationView{}, err
	}
	audit, err := s.repo.VerifyAudit(ctx, batchID)
	if err != nil {
		return CertificateVerificationView{}, err
	}
	now := s.now().UTC()
	verification := s.certificates.Verify(batch, evidenceCertificate(certificate), audit.RootDigest, now)
	return CertificateVerificationView{CertificateID: certificate.CertificateID, Verification: verification, AuditVerification: audit}, nil
}
