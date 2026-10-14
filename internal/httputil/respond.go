package httputil

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Hisaalu/rogele/internal/domain"
)

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ListMeta struct {
	Page     int   `json:"page"`
	PerPage  int   `json:"per_page"`
	Total    int64 `json:"total"`
	LastPage int64 `json:"last_page"`
}

type ListResponse struct {
	Data any      `json:"data"`
	Meta ListMeta `json:"meta"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, ErrorResponse{Error: ErrorBody{Code: code, Message: message}})
}

func DomainError(w http.ResponseWriter, err error, log *slog.Logger) {
	type mapping struct {
		target error
		status int
		code   string
	}
	table := []mapping{
		{domain.ErrInvalidInput, http.StatusBadRequest, "INVALID_INPUT"},
		{domain.ErrUnauthorized, http.StatusUnauthorized, "UNAUTHORIZED"},
		{domain.ErrForbidden, http.StatusForbidden, "FORBIDDEN"},
		{domain.ErrNotFound, http.StatusNotFound, "NOT_FOUND"},
		{domain.ErrConflict, http.StatusConflict, "CONFLICT"},
	}

	for _, m := range table {
		if errors.Is(err, m.target) {
			Error(w, m.status, m.code, err.Error())
			return
		}
	}

	log.Error("unhandled error", "error", err)
	Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

func DecodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func List(w http.ResponseWriter, data any, meta ListMeta) {
	if meta.PerPage > 0 && meta.Total >= 0 {
		meta.LastPage = (meta.Total + int64(meta.PerPage) - 1) / int64(meta.PerPage)
	}
	JSON(w, http.StatusOK, ListResponse{Data: data, Meta: meta})
}
