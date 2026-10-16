package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/httputil"
	"github.com/Hisaalu/rogele/internal/server/middleware"
	"github.com/Hisaalu/rogele/internal/service"
)

type AnalyticsHandler struct {
	svc *service.AnalyticsService
	log *slog.Logger
}

func NewAnalyticsHandler(svc *service.AnalyticsService, log *slog.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc, log: log}
}

func (h *AnalyticsHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return false
	}
	if claims.Role != domain.RoleAdmin {
		httputil.Error(w, http.StatusForbidden, "FORBIDDEN", "admin only")
		return false
	}
	return true
}

func (h *AnalyticsHandler) parseRange(w http.ResponseWriter, r *http.Request) (service.DateRange, bool) {
	dr := service.DefaultRange()

	if v := r.URL.Query().Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			if t2, err2 := time.Parse("2006-01-02", v); err2 == nil {
				t = t2
			} else {
				httputil.Error(w, http.StatusBadRequest, "INVALID_FROM", "from must be RFC3339 or YYYY-MM-DD")
				return dr, false
			}
		}
		dr.From = t.UTC()
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			if t2, err2 := time.Parse("2006-01-02", v); err2 == nil {
				t = t2.Add(24 * time.Hour)
			} else {
				httputil.Error(w, http.StatusBadRequest, "INVALID_TO", "to must be RFC3339 or YYYY-MM-DD")
				return dr, false
			}
		}
		dr.To = t.UTC()
	}
	return dr, true
}

func (h *AnalyticsHandler) EnrollmentTrend(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "month"
	}
	dr, ok := h.parseRange(w, r)
	if !ok {
		return
	}

	result, err := h.svc.EnrollmentTrend(r.Context(), interval, dr)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, result)
}

func (h *AnalyticsHandler) RevenueSeries(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "month"
	}
	dr, ok := h.parseRange(w, r)
	if !ok {
		return
	}

	result, err := h.svc.RevenueSeries(r.Context(), interval, dr)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, result)
}

func (h *AnalyticsHandler) ScoreDistribution(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	result, err := h.svc.ScoreDistribution(r.Context())
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, result)
}
