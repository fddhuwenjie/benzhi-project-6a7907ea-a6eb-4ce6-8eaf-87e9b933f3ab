package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"timber-stage-qualifier/internal/domain"
)

func (r *SQLiteRepository) invalidateAuditCache(batchID string) {
	r.auditMu.Lock()
	delete(r.auditCache, batchID)
	r.auditMu.Unlock()
}

func (r *SQLiteRepository) AuditTimeline(ctx context.Context, batchID string) ([]EventRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.auditMu.RLock()
	cached, ok := r.auditCache[batchID]
	r.auditMu.RUnlock()
	if ok {
		return cloneEventRecords(cached), nil
	}

	rows, err := r.db.QueryContext(ctx, `SELECT sequence_no,event_type,actor_id,occurred_at,revision,payload,previous_digest,digest FROM audit_events WHERE batch_id=? ORDER BY sequence_no`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]EventRecord, 0)
	for rows.Next() {
		var e EventRecord
		var at string
		if err := rows.Scan(&e.SequenceNo, &e.EventType, &e.ActorID, &at, &e.Revision, &e.Payload, &e.PreviousDigest, &e.Digest); err != nil {
			return nil, err
		}
		e.BatchID = batchID
		e.OccurredAt, err = time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(events) == 0 {
		var n int
		err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM batches WHERE batch_id=?`, batchID).Scan(&n)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, domain.ErrNotFound
		}
	}
	r.auditMu.Lock()
	r.auditCache[batchID] = events
	r.auditMu.Unlock()
	return cloneEventRecords(events), nil
}

// cloneEventRecords returns an independent deep copy of the audit event slice
// so that callers mutating the returned EventRecord fields or the Payload
// backing byte slice cannot corrupt the cached records consumed by later
// AuditTimeline or VerifyAudit calls.
func cloneEventRecords(events []EventRecord) []EventRecord {
	if len(events) == 0 {
		return make([]EventRecord, 0)
	}
	clone := make([]EventRecord, len(events))
	for i, e := range events {
		clone[i] = e
		if e.Payload != nil {
			payload := make([]byte, len(e.Payload))
			copy(payload, e.Payload)
			clone[i].Payload = payload
		}
	}
	return clone
}

func VerifyEvents(events []EventRecord) AuditVerification {
	previous := ""
	for i, e := range events {
		if e.SequenceNo != int64(i+1) {
			return AuditVerification{Valid: false, EventCount: len(events), RootDigest: previous, Failure: "事件序号不连续"}
		}
		if e.PreviousDigest != previous {
			return AuditVerification{Valid: false, EventCount: len(events), RootDigest: previous, Failure: "前序摘要不连续"}
		}
		canonical := struct {
			BatchID  string          `json:"batch_id"`
			Sequence int64           `json:"sequence_no"`
			Type     string          `json:"event_type"`
			Actor    string          `json:"actor_id"`
			At       string          `json:"occurred_at"`
			Revision int64           `json:"revision"`
			Payload  json.RawMessage `json:"payload"`
			Previous string          `json:"previous_digest"`
		}{e.BatchID, e.SequenceNo, e.EventType, e.ActorID, e.OccurredAt.UTC().Format(time.RFC3339Nano), e.Revision, e.Payload, e.PreviousDigest}
		encoded, err := json.Marshal(canonical)
		if err != nil {
			return AuditVerification{Valid: false, EventCount: len(events), RootDigest: previous, Failure: err.Error()}
		}
		sum := sha256.Sum256(encoded)
		expected := hex.EncodeToString(sum[:])
		if expected != e.Digest {
			return AuditVerification{Valid: false, EventCount: len(events), RootDigest: previous, Failure: "事件摘要不匹配"}
		}
		previous = e.Digest
	}
	return AuditVerification{Valid: true, EventCount: len(events), RootDigest: previous}
}

func (r *SQLiteRepository) VerifyAudit(ctx context.Context, batchID string) (AuditVerification, error) {
	events, err := r.AuditTimeline(ctx, batchID)
	if err != nil {
		return AuditVerification{}, err
	}
	return VerifyEvents(events), nil
}

func (r *SQLiteRepository) VerifyAllAuditChains(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT batch_id FROM audit_events ORDER BY batch_id`)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		result, err := r.VerifyAudit(ctx, id)
		if err != nil {
			return err
		}
		if !result.Valid {
			return fmt.Errorf("批次 %s 审计链损坏: %s", id, result.Failure)
		}
	}
	return nil
}

func (r *SQLiteRepository) GetCertificate(ctx context.Context, batchID string) (*CertificateRecord, error) {
	var c CertificateRecord
	var issued string
	var verified sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT certificate_id,document_version,canonical_payload,payload_sha256,audit_root_digest,issued_at,verified_at FROM certificates WHERE batch_id=?`, batchID).Scan(&c.CertificateID, &c.DocumentVersion, &c.CanonicalPayload, &c.PayloadSHA256, &c.AuditRootDigest, &issued, &verified)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.BatchID = batchID
	c.IssuedAt, err = time.Parse(time.RFC3339Nano, issued)
	if err != nil {
		return nil, err
	}
	if verified.Valid {
		t, e := time.Parse(time.RFC3339Nano, verified.String)
		if e != nil {
			return nil, e
		}
		c.VerifiedAt = &t
	}
	return &c, nil
}
