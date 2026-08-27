package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"timber-stage-qualifier/internal/application"
)

var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)

type writeMeta struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision int64  `json:"expected_revision"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("请求 JSON 无效: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("请求体只能包含一个 JSON 对象")
	}
	return nil
}

func commandMeta(r *http.Request, meta writeMeta) (application.CommandMeta, error) {
	actor := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
	if !idPattern.MatchString(actor) {
		return application.CommandMeta{}, fmt.Errorf("X-Actor-ID 缺失或格式无效")
	}
	if !idPattern.MatchString(meta.RequestID) {
		return application.CommandMeta{}, fmt.Errorf("request_id 缺失或格式无效")
	}
	if meta.ExpectedRevision < 0 {
		return application.CommandMeta{}, fmt.Errorf("expected_revision 不能为负数")
	}
	return application.CommandMeta{RequestID: meta.RequestID, ExpectedRevision: meta.ExpectedRevision, ActorID: actor}, nil
}

func pathID(r *http.Request, name string) (string, error) {
	value := strings.TrimSpace(r.PathValue(name))
	if !idPattern.MatchString(value) {
		return "", fmt.Errorf("路径参数 %s 格式无效", name)
	}
	return value, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeOutcome[T any](w http.ResponseWriter, status int, outcome application.Outcome[T]) {
	if outcome.Replayed {
		w.Header().Set("Idempotent-Replay", "true")
	}
	writeJSON(w, status, outcome.Value)
}
