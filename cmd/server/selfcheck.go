package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type checkClient struct {
	baseURL string
	client  *http.Client
}

func (c checkClient) request(ctx context.Context, method, path, actor string, payload any, destination any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if actor != "" {
		request.Header.Set("X-Actor-ID", actor)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%s %s 返回 %d: %s", method, path, response.StatusCode, string(data))
	}
	if destination != nil && len(data) > 0 {
		if err := json.Unmarshal(data, destination); err != nil {
			return fmt.Errorf("解码 %s: %w", path, err)
		}
	}
	return nil
}

func runSelfCheck(ctx context.Context, c config) error {
	directory, err := os.MkdirTemp("", "timber-stage-selfcheck-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	databasePath := filepath.Join(directory, "check.db")
	instance, err := buildRuntime(ctx, c.Address, databasePath)
	if err != nil {
		return err
	}
	defer instance.close()
	listener, err := net.Listen("tcp", c.Address)
	if err != nil {
		return fmt.Errorf("自检监听失败: %w", err)
	}
	serveErr := make(chan error, 1)
	go func() {
		e := instance.server.Serve(listener)
		if e == http.ErrServerClosed {
			e = nil
		}
		serveErr <- e
	}()
	shutdown := func() error {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := instance.server.Shutdown(closeCtx); err != nil {
			return err
		}
		return <-serveErr
	}
	client := checkClient{baseURL: "http://" + listener.Addr().String(), client: &http.Client{Timeout: 5 * time.Second}}
	if err := executeSelfCheck(ctx, client); err != nil {
		_ = shutdown()
		return err
	}
	if err := shutdown(); err != nil {
		return err
	}
	fmt.Println("自检通过：异常暂停、纠正恢复、独立批准、证书封存和完整性校验均已完成")
	return nil
}

func executeSelfCheck(ctx context.Context, client checkClient) error {
	const batchID = "selfcheck-batch"
	const operator = "conservator-a"
	baseTime := time.Date(2026, 1, 10, 8, 0, 0, 0, time.UTC)
	var batchResult struct {
		Revision int64  `json:"revision"`
		Status   string `json:"status"`
	}
	create := map[string]any{"request_id": "self-create", "expected_revision": 0, "batch_id": batchID, "specimen_code": "SC-WOOD-001", "wood_species": "oak", "current_stage": "PEG-20", "target_stage": "PEG-30"}
	if err := client.request(ctx, http.MethodPost, "/api/v1/treatment-batches", operator, create, &batchResult); err != nil {
		return err
	}
	freeze := map[string]any{"request_id": "self-freeze", "expected_revision": batchResult.Revision, "protocol_id": "self-protocol", "target_concentration_pct": 30.0, "concentration_tolerance_pct": 1.0, "temperature_min_c": 18.0, "temperature_max_c": 24.0, "mass_change_limit_pct": 3.0, "observation_interval_hours": 4, "recovery_window_count": 2}
	if err := client.request(ctx, http.MethodPost, "/api/v1/treatment-batches/"+batchID+"/baseline", operator, freeze, &batchResult); err != nil {
		return err
	}
	qualified := map[string]any{"request_id": "self-observation-1", "expected_revision": batchResult.Revision, "observation_id": "self-obs-1", "sequence_no": 1, "captured_at": baseTime.Format(time.RFC3339), "concentration_pct": 30.0, "temperature_c": 20.0, "timber_mass_g": 1000.0, "evidence_digest": "sha256:selfcheck-evidence-0001"}
	if err := client.request(ctx, http.MethodPost, "/api/v1/treatment-batches/"+batchID+"/observations", operator, qualified, &batchResult); err != nil {
		return err
	}
	abnormal := map[string]any{"request_id": "self-observation-2", "expected_revision": batchResult.Revision, "observation_id": "self-obs-2", "sequence_no": 2, "captured_at": baseTime.Add(2 * time.Hour).Format(time.RFC3339), "concentration_pct": 35.0, "temperature_c": 20.0, "timber_mass_g": 1000.0, "evidence_digest": "sha256:selfcheck-evidence-0002"}
	if err := client.request(ctx, http.MethodPost, "/api/v1/treatment-batches/"+batchID+"/observations", operator, abnormal, &batchResult); err != nil {
		return err
	}
	if batchResult.Status != "paused" {
		return fmt.Errorf("异常观测未暂停批次")
	}
	var detail struct {
		OpenDeviations []struct {
			DeviationID string `json:"deviation_id"`
		} `json:"open_deviations"`
	}
	if err := client.request(ctx, http.MethodGet, "/api/v1/treatment-batches/"+batchID, "", nil, &detail); err != nil {
		return err
	}
	if len(detail.OpenDeviations) != 1 {
		return fmt.Errorf("自检期望一个开放偏差，实际为 %d", len(detail.OpenDeviations))
	}
	deviationID := detail.OpenDeviations[0].DeviationID
	correction := map[string]any{"request_id": "self-correction", "expected_revision": batchResult.Revision, "cause": "配液补加量记录错误", "corrective_action": "复核计量器具并重新配液", "owner_id": operator}
	if err := client.request(ctx, http.MethodPost, "/api/v1/treatment-batches/"+batchID+"/deviations/"+deviationID+"/correction", operator, correction, &batchResult); err != nil {
		return err
	}
	recovery := map[string]any{"request_id": "self-recovery", "expected_revision": batchResult.Revision}
	if err := client.request(ctx, http.MethodPost, "/api/v1/treatment-batches/"+batchID+"/recovery", operator, recovery, &batchResult); err != nil {
		return err
	}
	for sequence := 3; sequence <= 4; sequence++ {
		payload := map[string]any{"request_id": fmt.Sprintf("self-observation-%d", sequence), "expected_revision": batchResult.Revision, "observation_id": fmt.Sprintf("self-obs-%d", sequence), "sequence_no": sequence, "captured_at": baseTime.Add(time.Duration(sequence) * time.Hour).Format(time.RFC3339), "concentration_pct": 30.0, "temperature_c": 20.0, "timber_mass_g": 1000.0, "evidence_digest": fmt.Sprintf("sha256:selfcheck-evidence-%04d", sequence)}
		if err := client.request(ctx, http.MethodPost, "/api/v1/treatment-batches/"+batchID+"/observations", operator, payload, &batchResult); err != nil {
			return err
		}
	}
	if batchResult.Status != "pending_review" {
		return fmt.Errorf("恢复窗口完成后未进入待复核")
	}
	review := map[string]any{"request_id": "self-review", "expected_revision": batchResult.Revision, "review_id": "self-review-1", "decision": "approve", "checklist_results": []map[string]any{{"item": "基线完整", "passed": true}, {"item": "偏差闭环", "passed": true}, {"item": "恢复窗口", "passed": true}}, "rationale": "材料完整且连续恢复观测符合冻结规则"}
	if err := client.request(ctx, http.MethodPost, "/api/v1/treatment-batches/"+batchID+"/reviews", "reviewer-independent", review, &batchResult); err != nil {
		return err
	}
	if batchResult.Status != "approved_sealed" {
		return fmt.Errorf("独立批准后未封存")
	}
	var certificate map[string]any
	if err := client.request(ctx, http.MethodGet, "/api/v1/treatment-batches/"+batchID+"/certificate", "", nil, &certificate); err != nil {
		return err
	}
	var verification struct {
		Verification struct {
			Valid bool `json:"valid"`
		} `json:"verification"`
		AuditVerification struct {
			Valid bool `json:"valid"`
		} `json:"audit_verification"`
	}
	if err := client.request(ctx, http.MethodGet, "/api/v1/treatment-batches/"+batchID+"/certificate/verify", "", nil, &verification); err != nil {
		return err
	}
	if !verification.Verification.Valid || !verification.AuditVerification.Valid {
		return fmt.Errorf("证书或审计链完整性校验失败")
	}
	return nil
}
