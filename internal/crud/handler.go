package crud

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/httputil"
	"github.com/Hisaalu/rogele/internal/server/middleware"
)

type Handler struct {
	res  *Resource
	repo *Repository
	log  *slog.Logger
}

func NewHandler(res *Resource, repo *Repository, log *slog.Logger) *Handler {
	return &Handler{res: res, repo: repo, log: log}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireReadRole(w, r); !ok {
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	filters := make(map[string]string)
	for _, f := range h.res.Filterable {
		if v := r.URL.Query().Get(f); v != "" {
			filters[f] = v
		}
	}

	var sort []string
	if s := r.URL.Query().Get("sort"); s != "" {
		sort = strings.Split(s, ",")
	}

	result, err := h.repo.List(r.Context(), h.res, ListOptions{
		Page:     page,
		PerPage:  perPage,
		Search:   r.URL.Query().Get("q"),
		Filters:  filters,
		Sort:     sort,
		MaxLimit: 100,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	if result.Rows == nil {
		result.Rows = []map[string]any{}
	}

	httputil.List(w, result.Rows, httputil.ListMeta{
		Page:    result.Page,
		PerPage: result.PerPage,
		Total:   result.Total,
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireReadRole(w, r); !ok {
		return
	}

	id, err := h.parseID(w, r)
	if err != nil {
		return
	}

	row, err := h.repo.Get(r.Context(), h.res, id)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, row)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireWriteRole(w, r); !ok {
		return
	}

	var body map[string]any
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	clean, err := h.validateBody(body, true)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	row, err := h.repo.Create(r.Context(), h.res, clean)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusCreated, row)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireWriteRole(w, r); !ok {
		return
	}

	id, err := h.parseID(w, r)
	if err != nil {
		return
	}

	var body map[string]any
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	clean, err := h.validateBody(body, false)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	row, err := h.repo.Update(r.Context(), h.res, id, clean)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, row)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireWriteRole(w, r); !ok {
		return
	}

	id, err := h.parseID(w, r)
	if err != nil {
		return
	}

	if err := h.repo.Delete(r.Context(), h.res, id); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.NoContent(w)
}

func (h *Handler) parseID(w http.ResponseWriter, r *http.Request) (int64, error) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return 0, err
	}
	return id, nil
}

func (h *Handler) requireReadRole(w http.ResponseWriter, r *http.Request) (domain.UserRole, bool) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return "", false
	}
	if !h.res.CanRead(claims.Role) {
		httputil.Error(w, http.StatusForbidden, "FORBIDDEN", "insufficient role to read "+h.res.Name)
		return "", false
	}
	return claims.Role, true
}

func (h *Handler) requireWriteRole(w http.ResponseWriter, r *http.Request) (domain.UserRole, bool) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return "", false
	}
	if !h.res.CanWrite(claims.Role) {
		httputil.Error(w, http.StatusForbidden, "FORBIDDEN", "insufficient role to write "+h.res.Name)
		return "", false
	}
	return claims.Role, true
}

func (h *Handler) validateBody(body map[string]any, isCreate bool) (map[string]any, error) {
	clean := make(map[string]any, len(body))

	for k, v := range body {
		col, ok := h.res.Columns[k]
		if !ok {
			return nil, fmt.Errorf("%w: field not allowed: %s", domain.ErrInvalidInput, k)
		}
		if !col.Writable {
			return nil, fmt.Errorf("%w: field is read-only: %s", domain.ErrInvalidInput, k)
		}
		if err := checkType(v, col.Type); err != nil {
			return nil, fmt.Errorf("%w: field %s: %s", domain.ErrInvalidInput, k, err.Error())
		}
		clean[k] = v
	}

	if isCreate {
		for name, col := range h.res.Columns {
			if !col.Required || !col.Writable {
				continue
			}
			if _, ok := clean[name]; !ok {
				return nil, fmt.Errorf("%w: missing required field: %s", domain.ErrInvalidInput, name)
			}
		}
	}

	return clean, nil
}

func checkType(v any, t ColumnType) error {
	if v == nil {
		return nil
	}
	switch t {
	case TypeString:
		if _, ok := v.(string); !ok {
			return errors.New("expected string")
		}
	case TypeInteger:
		switch n := v.(type) {
		case float64:
			if n != float64(int64(n)) {
				return errors.New("expected integer")
			}
		case int, int64:
		default:
			return errors.New("expected integer")
		}
	case TypeNumber:
		switch v.(type) {
		case float64, int, int64:
		default:
			return errors.New("expected number")
		}
	case TypeBoolean:
		if _, ok := v.(bool); !ok {
			return errors.New("expected boolean")
		}
	}
	return nil
}
