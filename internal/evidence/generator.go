package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"timber-stage-qualifier/internal/domain"
)

type Generator struct{}

func NewGenerator() *Generator { return &Generator{} }

func (g *Generator) Generate(batch *domain.TreatmentBatch, certificateID, evidenceRoot string, issuedAt time.Time) (*Certificate, error) {
	if batch == nil || batch.Status != domain.StatusApproved || batch.Review == nil || batch.Protocol == nil {
		return nil, domain.NewRuleError("certificate_not_eligible", "只有已批准且材料完整的批次可生成证书")
	}
	if certificateID == "" || issuedAt.IsZero() {
		return nil, domain.NewRuleError("invalid_certificate_identity", "证书标识和签发时间不能为空")
	}
	if len(batch.Observations) == 0 {
		return nil, domain.NewRuleError("missing_observations", "证书缺少观测范围")
	}
	observations := append([]domain.ProcessObservation(nil), batch.Observations...)
	sort.Slice(observations, func(i, j int) bool { return observations[i].SequenceNo < observations[j].SequenceNo })
	deviations := append([]domain.DeviationCase(nil), batch.Deviations...)
	sort.Slice(deviations, func(i, j int) bool { return deviations[i].DeviationID < deviations[j].DeviationID })
	closures := make([]canonicalDeviation, 0, len(deviations))
	for _, d := range deviations {
		closed := ""
		if d.ClosedAt != nil {
			closed = d.ClosedAt.UTC().Format(time.RFC3339Nano)
		}
		ids := append([]string(nil), d.QualifiedObservationIDs...)
		sort.Strings(ids)
		closures = append(closures, canonicalDeviation{d.DeviationID, d.RuleCode, string(d.Status), d.OwnerID, ids, closed})
	}
	first, last := observations[0], observations[len(observations)-1]
	p := batch.Protocol
	r := batch.Review
	payload := canonicalPayload{
		DocumentVersion: DocumentVersion, CertificateID: certificateID, BatchID: batch.BatchID, SpecimenCode: batch.SpecimenCode, WoodSpecies: batch.WoodSpecies,
		CurrentStage: batch.CurrentStage, TargetStage: batch.TargetStage, FinalStatus: string(batch.Status), FinalRevision: batch.Revision, BaselineDigest: batch.BaselineDigest,
		TargetConcentrationPct: format(p.TargetConcentrationPct), ConcentrationTolerancePct: format(p.ConcentrationTolerancePct), TemperatureMinC: format(p.TemperatureMinC), TemperatureMaxC: format(p.TemperatureMaxC), MassChangeLimitPct: format(p.MassChangeLimitPct),
		ObservationIntervalHours: p.ObservationIntervalHours, RecoveryWindowCount: p.RecoveryWindowCount,
		ObservationRange:  canonicalObservationRange{first.SequenceNo, last.SequenceNo, len(observations), first.CapturedAt.UTC().Format(time.RFC3339Nano), last.CapturedAt.UTC().Format(time.RFC3339Nano)},
		DeviationClosures: closures, Review: canonicalReview{r.ReviewID, r.ReviewerID, string(r.Decision), r.Rationale, evidenceRoot, r.DecidedAt.UTC().Format(time.RFC3339Nano)}, IssuedAt: issuedAt.UTC().Format(time.RFC3339Nano),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("编码规范证书: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return &Certificate{CertificateID: certificateID, BatchID: batch.BatchID, DocumentVersion: DocumentVersion, CanonicalPayload: encoded, PayloadSHA256: hex.EncodeToString(sum[:]), IssuedAt: issuedAt.UTC()}, nil
}

func format(value float64) string { return fmt.Sprintf("%.6f", value) }
