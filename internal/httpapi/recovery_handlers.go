package httpapi

import (
	"net/http"
	"timber-stage-qualifier/internal/application"
)

type correctDeviationRequest struct {
	writeMeta
	Cause            string `json:"cause"`
	CorrectiveAction string `json:"corrective_action"`
	OwnerID          string `json:"owner_id"`
}

func (a *API) CorrectDeviationHandler(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r, "batch_id")
	if err != nil {
		badRequest(w, err)
		return
	}
	deviationID, err := pathID(r, "deviation_id")
	if err != nil {
		badRequest(w, err)
		return
	}
	var request correctDeviationRequest
	if err := decodeJSON(w, r, &request); err != nil {
		badRequest(w, err)
		return
	}
	meta, err := commandMeta(r, request.writeMeta)
	if err != nil {
		badRequest(w, err)
		return
	}
	outcome, err := a.service.CorrectDeviation(r.Context(), application.CorrectDeviationCommand{Meta: meta, BatchID: batchID, DeviationID: deviationID, Cause: request.Cause, CorrectiveAction: request.CorrectiveAction, OwnerID: request.OwnerID})
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeOutcome(w, http.StatusOK, outcome)
}

type approveRecoveryRequest struct{ writeMeta }

func (a *API) ApproveRecoveryHandler(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r, "batch_id")
	if err != nil {
		badRequest(w, err)
		return
	}
	var request approveRecoveryRequest
	if err := decodeJSON(w, r, &request); err != nil {
		badRequest(w, err)
		return
	}
	meta, err := commandMeta(r, request.writeMeta)
	if err != nil {
		badRequest(w, err)
		return
	}
	outcome, err := a.service.ApproveRecovery(r.Context(), application.ApproveRecoveryCommand{Meta: meta, BatchID: batchID})
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeOutcome(w, http.StatusOK, outcome)
}
