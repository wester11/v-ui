package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/voidwg/control/internal/domain"
	"github.com/voidwg/control/internal/interfaces/http/dto"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, dto.ErrorResponse{Error: "not_found"})
	case errors.Is(err, domain.ErrAlreadyExists):
		writeJSON(w, http.StatusConflict, dto.ErrorResponse{Error: "already_exists"})
	case errors.Is(err, domain.ErrInvalidCredential):
		writeJSON(w, http.StatusUnauthorized, dto.ErrorResponse{Error: "invalid_credentials"})
	case errors.Is(err, domain.ErrForbidden):
		writeJSON(w, http.StatusForbidden, dto.ErrorResponse{Error: "forbidden"})
	case errors.Is(err, domain.ErrValidation):
		writeJSON(w, http.StatusBadRequest, dto.ErrorResponse{Error: "validation"})
	case errors.Is(err, domain.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, dto.ErrorResponse{Error: "invalid_input"})
	case errors.Is(err, domain.ErrPoolExhausted):
		writeJSON(w, http.StatusConflict, dto.ErrorResponse{Error: "ip_pool_exhausted"})
	default:
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{Error: "internal"})
	}
}

func decode(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return domain.ErrValidation
	}
	return nil
}
