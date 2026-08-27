package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
	"timber-stage-qualifier/internal/domain"
)

type SQLiteRepository struct {
	db    *sql.DB
	locks *keyedLocks
}

func Open(ctx context.Context, dsn string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite: %w", err)
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys=ON`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `PRAGMA busy_timeout=5000`); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	r := &SQLiteRepository{db: db, locks: newKeyedLocks()}
	if err := r.VerifyAllAuditChains(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return r, nil
}

func (r *SQLiteRepository) Close() error { return r.db.Close() }

func (r *SQLiteRepository) Ready(ctx context.Context) error {
	var version int
	if err := r.db.QueryRowContext(ctx, `SELECT version FROM schema_version LIMIT 1`).Scan(&version); err != nil {
		return err
	}
	if version != schemaVersion {
		return fmt.Errorf("数据库版本不匹配")
	}
	return r.db.PingContext(ctx)
}

func (r *SQLiteRepository) GetBatch(ctx context.Context, batchID string) (*domain.TreatmentBatch, error) {
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT aggregate_json FROM batches WHERE batch_id=?`, batchID).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var batch domain.TreatmentBatch
	if err := json.Unmarshal(data, &batch); err != nil {
		return nil, fmt.Errorf("解码批次: %w", err)
	}
	if err := batch.ValidateLoaded(); err != nil {
		return nil, err
	}
	return &batch, nil
}

func loadBatchTx(ctx context.Context, tx *sql.Tx, batchID string) (*domain.TreatmentBatch, error) {
	var data []byte
	err := tx.QueryRowContext(ctx, `SELECT aggregate_json FROM batches WHERE batch_id=?`, batchID).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var batch domain.TreatmentBatch
	if err := json.Unmarshal(data, &batch); err != nil {
		return nil, fmt.Errorf("解码批次: %w", err)
	}
	return &batch, nil
}

func saveBatchTx(ctx context.Context, tx *sql.Tx, batch *domain.TreatmentBatch, now time.Time) error {
	data, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE batches SET revision=?, status=?, aggregate_json=?, updated_at=? WHERE batch_id=?`, batch.Revision, string(batch.Status), data, now.UTC().Format(time.RFC3339Nano), batch.BatchID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return domain.ErrNotFound
	}
	return syncChildrenTx(ctx, tx, batch)
}

func insertBatchTx(ctx context.Context, tx *sql.Tx, batch *domain.TreatmentBatch, now time.Time) error {
	data, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO batches(batch_id,revision,status,aggregate_json,created_at,updated_at) VALUES(?,?,?,?,?,?)`, batch.BatchID, batch.Revision, string(batch.Status), data, batch.CreatedAt.UTC().Format(time.RFC3339Nano), now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("插入批次: %w", err)
	}
	return syncChildrenTx(ctx, tx, batch)
}

func syncChildrenTx(ctx context.Context, tx *sql.Tx, batch *domain.TreatmentBatch) error {
	for _, observation := range batch.Observations {
		payload, err := json.Marshal(observation)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO observations(batch_id,sequence_no,observation_id,captured_at,payload_json) VALUES(?,?,?,?,?) ON CONFLICT(batch_id,sequence_no) DO NOTHING`, batch.BatchID, observation.SequenceNo, observation.ObservationID, observation.CapturedAt.UTC().Format(time.RFC3339Nano), payload)
		if err != nil {
			return err
		}
	}
	for _, deviation := range batch.Deviations {
		payload, err := json.Marshal(deviation)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO deviations(deviation_id,batch_id,status,payload_json) VALUES(?,?,?,?) ON CONFLICT(deviation_id) DO UPDATE SET status=excluded.status,payload_json=excluded.payload_json`, deviation.DeviationID, batch.BatchID, string(deviation.Status), payload)
		if err != nil {
			return err
		}
	}
	if batch.Review != nil {
		payload, err := json.Marshal(batch.Review)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO reviews(review_id,batch_id,decision,payload_json) VALUES(?,?,?,?) ON CONFLICT(batch_id) DO UPDATE SET review_id=excluded.review_id,decision=excluded.decision,payload_json=excluded.payload_json`, batch.Review.ReviewID, batch.BatchID, string(batch.Review.Decision), payload)
		if err != nil {
			return err
		}
	}
	return nil
}
