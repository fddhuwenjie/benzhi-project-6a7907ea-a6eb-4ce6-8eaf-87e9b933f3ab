package httpapi

import (
	"net/http"
	"timber-stage-qualifier/internal/application"
	"time"
)

type submitObservationRequest struct {
	writeMeta
	ObservationID    string    `json:"observation_id"`
	SequenceNo       int64     `json:"sequence_no"`
	CapturedAt       time.Time `json:"captured_at"`
	ConcentrationPct float64   `json:"concentration_pct"`
	TemperatureC     float64   `json:"temperature_c"`
	TimberMassG      float64   `json:"timber_mass_g"`
	EvidenceDigest   string    `json:"evidence_digest"`
}

func (a *API) SubmitObservationHandler(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r, "batch_id")
	if err != nil {
		badRequest(w, err)
		return
	}
	var request submitObservationRequest
	if err := decodeJSON(w, r, &request); err != nil {
		badRequest(w, err)
		return
	}
	meta, err := commandMeta(r, request.writeMeta)
	if err != nil {
		badRequest(w, err)
		return
	}
	command := application.SubmitObservationCommand{Meta: meta, BatchID: batchID, ObservationID: request.ObservationID, SequenceNo: request.SequenceNo, CapturedAt: request.CapturedAt, ConcentrationPct: request.ConcentrationPct, TemperatureC: request.TemperatureC, TimberMassG: request.TimberMassG, EvidenceDigest: request.EvidenceDigest}
	outcome, err := a.service.SubmitObservation(r.Context(), command)
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeOutcome(w, http.StatusCreated, outcome)
}
