package httpapi

import (
	"net/http"
	"timber-stage-qualifier/internal/application"
	"timber-stage-qualifier/internal/domain"
)

type decideReviewRequest struct {
	writeMeta
	ReviewID           string                   `json:"review_id"`
	Decision           domain.ReviewDecision    `json:"decision"`
	ChecklistResults   []domain.ChecklistResult `json:"checklist_results"`
	Rationale          string                   `json:"rationale"`
	EvidenceRootDigest string                   `json:"evidence_root_digest,omitempty"`
}

func (a *API) DecideReviewHandler(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r, "batch_id")
	if err != nil {
		badRequest(w, err)
		return
	}
	var request decideReviewRequest
	if err := decodeJSON(w, r, &request); err != nil {
		badRequest(w, err)
		return
	}
	meta, err := commandMeta(r, request.writeMeta)
	if err != nil {
		badRequest(w, err)
		return
	}
	command := application.DecideReviewCommand{Meta: meta, BatchID: batchID, ReviewID: request.ReviewID, Decision: request.Decision, ChecklistResults: request.ChecklistResults, Rationale: request.Rationale, EvidenceRootDigest: request.EvidenceRootDigest}
	outcome, err := a.service.DecideReview(r.Context(), command)
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeOutcome(w, http.StatusCreated, outcome)
}
