package httpapi

import (
	"errors"
	"net/http"
	"timber-stage-qualifier/internal/domain"
)

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	var response errorResponse
	response.Error.Code = code
	response.Error.Message = message
	writeJSON(w, status, response)
}

func writeApplicationError(w http.ResponseWriter, err error) {
	var rule *domain.RuleError
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusConflict, "revision_conflict", err.Error())
	case errors.Is(err, domain.ErrIdempotency):
		writeError(w, http.StatusConflict, "idempotency_conflict", err.Error())
	case errors.Is(err, domain.ErrDuplicateSequence):
		writeError(w, http.StatusConflict, "duplicate_sequence", err.Error())
	case errors.Is(err, domain.ErrTimeRegression):
		writeError(w, http.StatusUnprocessableEntity, "observation_order_invalid", err.Error())
	case errors.Is(err, domain.ErrDutySeparation):
		writeError(w, http.StatusUnprocessableEntity, "duty_separation_failed", err.Error())
	case errors.Is(err, domain.ErrInvalidState), errors.Is(err, domain.ErrSealed):
		writeError(w, http.StatusUnprocessableEntity, "invalid_batch_state", err.Error())
	case errors.As(err, &rule):
		writeError(w, http.StatusUnprocessableEntity, rule.Code, rule.Message)
	case errors.Is(err, domain.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
	}
}

func badRequest(w http.ResponseWriter, err error) {
	writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
}
