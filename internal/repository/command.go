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

type Mutation func(batch *domain.TreatmentBatch, auditRoot string) (response any, event EventInput, certificate *CertificateRecord, err error)
type Creation func() (*domain.TreatmentBatch, any, EventInput, error)

func (r *SQLiteRepository) Create(ctx context.Context, batchID, requestID, fingerprint string, expectedRevision int64, create Creation) (CommandResult, error) {
	unlock := r.locks.lock(ctx, batchID)
	defer unlock()
	return r.execute(ctx, batchID, requestID, fingerprint, func(tx *sql.Tx) ([]byte, error) {
		if expectedRevision != 0 {
			return nil, domain.ErrConflict
		}
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM batches WHERE batch_id=?`, batchID).Scan(&exists); err != nil {
			return nil, err
		}
		if exists != 0 {
			return nil, domain.ErrConflict
		}
		batch, response, event, err := create()
		if err != nil {
			return nil, err
		}
		now := event.At.UTC()
		if err := insertBatchTx(ctx, tx, batch, now); err != nil {
			return nil, err
		}
		if _, err := appendEventTx(ctx, tx, batch, event); err != nil {
			return nil, err
		}
		return json.Marshal(response)
	})
}

func (r *SQLiteRepository) Mutate(ctx context.Context, batchID, requestID, fingerprint string, expectedRevision int64, mutate Mutation) (CommandResult, error) {
	unlock := r.locks.lock(ctx, batchID)
	defer unlock()
	return r.execute(ctx, batchID, requestID, fingerprint, func(tx *sql.Tx) ([]byte, error) {
		batch, err := loadBatchTx(ctx, tx, batchID)
		if err != nil {
			return nil, err
		}
		if batch.Revision != expectedRevision {
			return nil, domain.ErrConflict
		}
		root, err := auditRootTx(ctx, tx, batchID)
		if err != nil {
			return nil, err
		}
		response, event, certificate, err := mutate(batch, root)
		if err != nil {
			return nil, err
		}
		if err := saveBatchTx(ctx, tx, batch, event.At); err != nil {
			return nil, err
		}
		newRoot, err := appendEventTx(ctx, tx, batch, event)
		if err != nil {
			return nil, err
		}
		if certificate != nil {
			certificate.AuditRootDigest = newRoot
			if err := insertCertificateTx(ctx, tx, certificate); err != nil {
				return nil, err
			}
		}
		return json.Marshal(response)
	})
}

func (r *SQLiteRepository) execute(ctx context.Context, batchID, requestID, fingerprint string, operation func(*sql.Tx) ([]byte, error)) (CommandResult, error) {
	if requestID == "" || fingerprint == "" {
		return CommandResult{}, domain.ErrValidation
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return CommandResult{}, err
	}
	defer tx.Rollback()
	var storedFingerprint string
	var body []byte
	err = tx.QueryRowContext(ctx, `SELECT fingerprint,response_body FROM idempotency WHERE request_id=?`, requestID).Scan(&storedFingerprint, &body)
	if err == nil {
		if storedFingerprint != fingerprint {
			return CommandResult{}, domain.ErrIdempotency
		}
		return CommandResult{Body: body, Replayed: true}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return CommandResult{}, err
	}
	body, err = operation(tx)
	if err != nil {
		return CommandResult{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO idempotency(request_id,batch_id,fingerprint,response_body,committed_at) VALUES(?,?,?,?,?)`, requestID, batchID, fingerprint, body, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return CommandResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return CommandResult{}, err
	}
	return CommandResult{Body: body}, nil
}

func appendEventTx(ctx context.Context, tx *sql.Tx, batch *domain.TreatmentBatch, event EventInput) (string, error) {
	previous, err := auditRootTx(ctx, tx, batch.BatchID)
	if err != nil {
		return "", err
	}
	var sequence int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_no),0)+1 FROM audit_events WHERE batch_id=?`, batch.BatchID).Scan(&sequence); err != nil {
		return "", err
	}
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return "", err
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
	}{batch.BatchID, sequence, event.Type, event.ActorID, event.At.UTC().Format(time.RFC3339Nano), batch.Revision, payload, previous}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	digest := hex.EncodeToString(sum[:])
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_events(batch_id,sequence_no,event_type,actor_id,occurred_at,revision,payload,previous_digest,digest) VALUES(?,?,?,?,?,?,?,?,?)`, batch.BatchID, sequence, event.Type, event.ActorID, event.At.UTC().Format(time.RFC3339Nano), batch.Revision, payload, previous, digest)
	if err != nil {
		return "", fmt.Errorf("追加审计事件: %w", err)
	}
	return digest, nil
}

func auditRootTx(ctx context.Context, tx *sql.Tx, batchID string) (string, error) {
	var digest string
	err := tx.QueryRowContext(ctx, `SELECT digest FROM audit_events WHERE batch_id=? ORDER BY sequence_no DESC LIMIT 1`, batchID).Scan(&digest)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return digest, err
}

func insertCertificateTx(ctx context.Context, tx *sql.Tx, c *CertificateRecord) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO certificates(certificate_id,batch_id,document_version,canonical_payload,payload_sha256,audit_root_digest,issued_at,verified_at) VALUES(?,?,?,?,?,?,?,NULL)`, c.CertificateID, c.BatchID, c.DocumentVersion, c.CanonicalPayload, c.PayloadSHA256, c.AuditRootDigest, c.IssuedAt.UTC().Format(time.RFC3339Nano))
	return err
}
