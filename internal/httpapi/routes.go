package httpapi

func (a *API) routes() {
	a.mux.HandleFunc("GET /readyz", a.ReadinessHandler)
	a.mux.HandleFunc("POST /api/v1/treatment-batches", a.CreateBatchHandler)
	a.mux.HandleFunc("GET /api/v1/treatment-batches/{batch_id}", a.GetBatchHandler)
	a.mux.HandleFunc("POST /api/v1/treatment-batches/{batch_id}/baseline", a.FreezeBaselineHandler)
	a.mux.HandleFunc("POST /api/v1/treatment-batches/{batch_id}/observations", a.SubmitObservationHandler)
	a.mux.HandleFunc("POST /api/v1/treatment-batches/{batch_id}/deviations/{deviation_id}/correction", a.CorrectDeviationHandler)
	a.mux.HandleFunc("POST /api/v1/treatment-batches/{batch_id}/recovery", a.ApproveRecoveryHandler)
	a.mux.HandleFunc("POST /api/v1/treatment-batches/{batch_id}/reviews", a.DecideReviewHandler)
	a.mux.HandleFunc("GET /api/v1/treatment-batches/{batch_id}/certificate", a.GetCertificateHandler)
	a.mux.HandleFunc("GET /api/v1/treatment-batches/{batch_id}/certificate/verify", a.VerifyCertificateHandler)
	a.mux.HandleFunc("GET /api/v1/treatment-batches/{batch_id}/audit", a.GetAuditHandler)
}
