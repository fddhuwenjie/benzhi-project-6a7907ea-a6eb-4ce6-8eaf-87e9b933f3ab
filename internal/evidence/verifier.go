package evidence

import (
	"encoding/hex"
	"encoding/json"
	"time"

	"timber-stage-qualifier/internal/domain"
)

func (g *Generator) Verify(batch *domain.TreatmentBatch, certificate *Certificate, currentAuditRoot string, now time.Time) Verification {
	result := Verification{Valid: true, PayloadDigestValid: true, BatchStateValid: true, AuditRootValid: true, Failures: []string{}, VerifiedAt: now.UTC()}
	if batch == nil || certificate == nil {
		result.Valid = false
		result.PayloadDigestValid = false
		result.BatchStateValid = false
		result.AuditRootValid = false
		result.Failures = append(result.Failures, "批次或证书不存在")
		return result
	}
	g.digest.Reset()
	_, _ = g.digest.Write(certificate.CanonicalPayload)
	sum := g.digest.Sum(nil)
	if hex.EncodeToString(sum) != certificate.PayloadSHA256 {
		result.PayloadDigestValid = false
		result.Failures = append(result.Failures, "证书载荷摘要不匹配")
	}
	var payload canonicalPayload
	if err := json.Unmarshal(certificate.CanonicalPayload, &payload); err != nil {
		result.PayloadDigestValid = false
		result.Failures = append(result.Failures, "证书载荷不是有效规范 JSON")
	}
	if batch.Status != domain.StatusApproved || batch.Review == nil || batch.BaselineDigest == "" || payload.BatchID != batch.BatchID || payload.FinalRevision != batch.Revision || payload.BaselineDigest != batch.BaselineDigest || payload.FinalStatus != string(batch.Status) {
		result.BatchStateValid = false
		result.Failures = append(result.Failures, "证书与批次批准终态不一致")
	}
	if certificate.AuditRootDigest == "" || currentAuditRoot != certificate.AuditRootDigest {
		result.AuditRootValid = false
		result.Failures = append(result.Failures, "证书封存审计根与当前审计链不一致")
	}
	result.Valid = result.PayloadDigestValid && result.BatchStateValid && result.AuditRootValid
	return result
}
