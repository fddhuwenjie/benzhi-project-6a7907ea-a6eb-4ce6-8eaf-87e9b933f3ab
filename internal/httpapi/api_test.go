package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"timber-stage-qualifier/internal/application"
	"timber-stage-qualifier/internal/evidence"
	"timber-stage-qualifier/internal/repository"
)

func TestAPIRejectsUnknownFieldsAndRequiresIdentity(t *testing.T) {
	repo, err := repository.Open(context.Background(), filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	handler := New(application.NewService(repo, evidence.NewGenerator())).Handler()
	payload := `{"request_id":"r1","expected_revision":0,"batch_id":"b1","specimen_code":"s","wood_species":"oak","current_stage":"one","target_stage":"two","unexpected":true}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/treatment-batches", strings.NewReader(payload))
	request.Header.Set("X-Actor-ID", "actor")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("未知字段返回 %d", response.Code)
	}
	payload = `{"request_id":"r2","expected_revision":0,"batch_id":"b2","specimen_code":"s","wood_species":"oak","current_stage":"one","target_stage":"two"}`
	request = httptest.NewRequest(http.MethodPost, "/api/v1/treatment-batches", strings.NewReader(payload))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("缺失身份返回 %d", response.Code)
	}
}
