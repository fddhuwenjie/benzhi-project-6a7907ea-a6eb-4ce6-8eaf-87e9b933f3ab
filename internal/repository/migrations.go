package repository

import (
	"context"
	"database/sql"
	"fmt"
)

const schemaVersion = 1

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL);`,
	`CREATE TABLE IF NOT EXISTS batches (
		batch_id TEXT PRIMARY KEY, revision INTEGER NOT NULL, status TEXT NOT NULL,
		aggregate_json BLOB NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS observations (
		batch_id TEXT NOT NULL, sequence_no INTEGER NOT NULL, observation_id TEXT NOT NULL,
		captured_at TEXT NOT NULL, payload_json BLOB NOT NULL,
		PRIMARY KEY(batch_id, sequence_no), UNIQUE(observation_id),
		FOREIGN KEY(batch_id) REFERENCES batches(batch_id) ON DELETE CASCADE
	);`,
	`CREATE TABLE IF NOT EXISTS deviations (
		deviation_id TEXT PRIMARY KEY, batch_id TEXT NOT NULL, status TEXT NOT NULL,
		payload_json BLOB NOT NULL, FOREIGN KEY(batch_id) REFERENCES batches(batch_id) ON DELETE CASCADE
	);`,
	`CREATE TABLE IF NOT EXISTS reviews (
		review_id TEXT PRIMARY KEY, batch_id TEXT NOT NULL UNIQUE, decision TEXT NOT NULL,
		payload_json BLOB NOT NULL, FOREIGN KEY(batch_id) REFERENCES batches(batch_id) ON DELETE CASCADE
	);`,
	`CREATE TABLE IF NOT EXISTS certificates (
		certificate_id TEXT PRIMARY KEY, batch_id TEXT NOT NULL UNIQUE, document_version TEXT NOT NULL,
		canonical_payload BLOB NOT NULL, payload_sha256 TEXT NOT NULL, audit_root_digest TEXT NOT NULL,
		issued_at TEXT NOT NULL, verified_at TEXT,
		FOREIGN KEY(batch_id) REFERENCES batches(batch_id) ON DELETE CASCADE
	);`,
	`CREATE TABLE IF NOT EXISTS audit_events (
		batch_id TEXT NOT NULL, sequence_no INTEGER NOT NULL, event_type TEXT NOT NULL,
		actor_id TEXT NOT NULL, occurred_at TEXT NOT NULL, revision INTEGER NOT NULL,
		payload BLOB NOT NULL, previous_digest TEXT NOT NULL, digest TEXT NOT NULL,
		PRIMARY KEY(batch_id, sequence_no), UNIQUE(batch_id, digest),
		FOREIGN KEY(batch_id) REFERENCES batches(batch_id) ON DELETE CASCADE
	);`,
	`CREATE TABLE IF NOT EXISTS idempotency (
		request_id TEXT PRIMARY KEY, batch_id TEXT NOT NULL, fingerprint TEXT NOT NULL,
		response_body BLOB NOT NULL, committed_at TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_observations_batch ON observations(batch_id, sequence_no);`,
	`CREATE INDEX IF NOT EXISTS idx_deviations_batch ON deviations(batch_id, status);`,
	`CREATE INDEX IF NOT EXISTS idx_audit_batch ON audit_events(batch_id, sequence_no);`,
}

func migrate(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始迁移事务: %w", err)
	}
	defer tx.Rollback()
	for _, statement := range migrations {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("执行迁移: %w", err)
		}
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_version`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_version(version) VALUES (?)`, schemaVersion); err != nil {
			return err
		}
	}
	var version int
	if err := tx.QueryRowContext(ctx, `SELECT version FROM schema_version LIMIT 1`).Scan(&version); err != nil {
		return err
	}
	if version != schemaVersion {
		return fmt.Errorf("不支持的数据库版本 %d，期望 %d", version, schemaVersion)
	}
	return tx.Commit()
}
