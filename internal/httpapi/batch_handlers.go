package httpapi

import (
	"net/http"
	"timber-stage-qualifier/internal/application"
)

type createBatchRequest struct {
	writeMeta
	BatchID      string `json:"batch_id"`
	SpecimenCode string `json:"specimen_code"`
	WoodSpecies  string `json:"wood_species"`
	CurrentStage string `json:"current_stage"`
	TargetStage  string `json:"target_stage"`
}

func (a *API) CreateBatchHandler(w http.ResponseWriter, r *http.Request) {
	var request createBatchRequest
	if err := decodeJSON(w, r, &request); err != nil {
		badRequest(w, err)
		return
	}
	meta, err := commandMeta(r, request.writeMeta)
	if err != nil {
		badRequest(w, err)
		return
	}
	if !idPattern.MatchString(request.BatchID) {
		badRequest(w, errText("batch_id 格式无效"))
		return
	}
	outcome, err := a.service.CreateBatch(r.Context(), application.CreateBatchCommand{Meta: meta, BatchID: request.BatchID, SpecimenCode: request.SpecimenCode, WoodSpecies: request.WoodSpecies, CurrentStage: request.CurrentStage, TargetStage: request.TargetStage})
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeOutcome(w, http.StatusCreated, outcome)
}

type freezeBaselineRequest struct {
	writeMeta
	ProtocolID                string  `json:"protocol_id"`
	TargetConcentrationPct    float64 `json:"target_concentration_pct"`
	ConcentrationTolerancePct float64 `json:"concentration_tolerance_pct"`
	TemperatureMinC           float64 `json:"temperature_min_c"`
	TemperatureMaxC           float64 `json:"temperature_max_c"`
	MassChangeLimitPct        float64 `json:"mass_change_limit_pct"`
	ObservationIntervalHours  int     `json:"observation_interval_hours"`
	RecoveryWindowCount       int     `json:"recovery_window_count"`
}

func (a *API) FreezeBaselineHandler(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r, "batch_id")
	if err != nil {
		badRequest(w, err)
		return
	}
	var request freezeBaselineRequest
	if err := decodeJSON(w, r, &request); err != nil {
		badRequest(w, err)
		return
	}
	meta, err := commandMeta(r, request.writeMeta)
	if err != nil {
		badRequest(w, err)
		return
	}
	command := application.FreezeProtocolCommand{Meta: meta, BatchID: batchID, ProtocolID: request.ProtocolID, TargetConcentrationPct: request.TargetConcentrationPct, ConcentrationTolerancePct: request.ConcentrationTolerancePct, TemperatureMinC: request.TemperatureMinC, TemperatureMaxC: request.TemperatureMaxC, MassChangeLimitPct: request.MassChangeLimitPct, ObservationIntervalHours: request.ObservationIntervalHours, RecoveryWindowCount: request.RecoveryWindowCount}
	outcome, err := a.service.FreezeProtocol(r.Context(), command)
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeOutcome(w, http.StatusOK, outcome)
}

type errText string

func (e errText) Error() string { return string(e) }
