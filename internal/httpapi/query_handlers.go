package httpapi

import "net/http"

func (a *API) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.service.Ready(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready", "数据库尚未就绪")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (a *API) GetBatchHandler(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r, "batch_id")
	if err != nil {
		badRequest(w, err)
		return
	}
	view, err := a.service.GetBatch(r.Context(), batchID)
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (a *API) GetAuditHandler(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r, "batch_id")
	if err != nil {
		badRequest(w, err)
		return
	}
	view, err := a.service.GetAudit(r.Context(), batchID)
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (a *API) GetCertificateHandler(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r, "batch_id")
	if err != nil {
		badRequest(w, err)
		return
	}
	view, err := a.service.GetCertificate(r.Context(), batchID)
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (a *API) VerifyCertificateHandler(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r, "batch_id")
	if err != nil {
		badRequest(w, err)
		return
	}
	view, err := a.service.VerifyCertificate(r.Context(), batchID)
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}
