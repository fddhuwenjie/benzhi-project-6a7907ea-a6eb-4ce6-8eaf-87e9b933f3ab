package repository

import (
	"encoding/json"
	"testing"
	"time"
)

func TestVerifyEventsDetectsTampering(t *testing.T) {
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	events := []EventRecord{{SequenceNo: 1, BatchID: "b", EventType: "CREATED", ActorID: "a", OccurredAt: at, Revision: 1, Payload: json.RawMessage(`{"value":1}`), PreviousDigest: ""}}
	canonical := struct {
		BatchID  string          `json:"batch_id"`
		Sequence int64           `json:"sequence_no"`
		Type     string          `json:"event_type"`
		Actor    string          `json:"actor_id"`
		At       string          `json:"occurred_at"`
		Revision int64           `json:"revision"`
		Payload  json.RawMessage `json:"payload"`
		Previous string          `json:"previous_digest"`
	}{"b", 1, "CREATED", "a", at.Format(time.RFC3339Nano), 1, events[0].Payload, ""}
	encoded, _ := json.Marshal(canonical)
	events[0].Digest = digestBytes(encoded)
	if result := VerifyEvents(events); !result.Valid {
		t.Fatalf("有效审计事件被拒绝: %+v", result)
	}
	events[0].Payload = json.RawMessage(`{"value":2}`)
	if result := VerifyEvents(events); result.Valid {
		t.Fatal("被篡改的事件未被识别")
	}
}
