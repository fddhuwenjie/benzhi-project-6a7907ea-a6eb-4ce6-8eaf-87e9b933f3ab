package repository

import "time"

type EventRecord struct {
	SequenceNo     int64     `json:"sequence_no"`
	BatchID        string    `json:"batch_id"`
	EventType      string    `json:"event_type"`
	ActorID        string    `json:"actor_id"`
	OccurredAt     time.Time `json:"occurred_at"`
	Revision       int64     `json:"revision"`
	Payload        []byte    `json:"payload"`
	PreviousDigest string    `json:"previous_digest"`
	Digest         string    `json:"digest"`
}

type CertificateRecord struct {
	CertificateID    string     `json:"certificate_id"`
	BatchID          string     `json:"batch_id"`
	DocumentVersion  string     `json:"document_version"`
	CanonicalPayload []byte     `json:"canonical_payload"`
	PayloadSHA256    string     `json:"payload_sha256"`
	AuditRootDigest  string     `json:"audit_root_digest"`
	IssuedAt         time.Time  `json:"issued_at"`
	VerifiedAt       *time.Time `json:"verified_at,omitempty"`
}

type AuditVerification struct {
	Valid      bool   `json:"valid"`
	EventCount int    `json:"event_count"`
	RootDigest string `json:"root_digest"`
	Failure    string `json:"failure,omitempty"`
}

type CommandResult struct {
	Body     []byte
	Replayed bool
}

type EventInput struct {
	Type    string
	ActorID string
	At      time.Time
	Payload any
}
