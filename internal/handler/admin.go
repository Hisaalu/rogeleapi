package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/dto"
	"github.com/Hisaalu/rogele/internal/httputil"
	"github.com/Hisaalu/rogele/internal/repository/postgres"
	"github.com/Hisaalu/rogele/internal/server/middleware"
)

type AdminHandler struct {
	repo *postgres.AdminStatsRepo
	log  *slog.Logger
}

func NewAdminHandler(repo *postgres.AdminStatsRepo, log *slog.Logger) *AdminHandler {
	return &AdminHandler{repo: repo, log: log}
}

func (h *AdminHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
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

func (h *AdminHandler) Overview(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	stats, err := h.repo.GetOverview(r.Context())
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.AdminOverviewFromDomain(stats))
}

func (h *AdminHandler) RecentActivity(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.repo.ListRecentActivity(r.Context(), limit)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.ActivityItemsFromDomain(items))
}

func (h *AdminHandler) RecentSignups(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.repo.ListRecentSignups(r.Context(), limit)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.RecentUsersFromDomain(items))
}

func (h *AdminHandler) RecentPayments(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.repo.ListRecentPayments(r.Context(), limit)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.RecentPaymentsFromDomain(items))
}

func (h *AdminHandler) LearnersByClass(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	items, err := h.repo.LearnersByClass(r.Context())
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.ClassLearnerCountsFromDomain(items))
}
