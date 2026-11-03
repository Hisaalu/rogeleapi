package handler

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
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

func (h *AdminHandler) requireAuth(w http.ResponseWriter, r *http.Request) bool {
	_, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return false
	}
	return true
}

func (h *AdminHandler) Overview(w http.ResponseWriter, r *http.Request) {
	if !h.requireAuth(w, r) {
		return
	}
	stats, err := h.repo.GetOverview(r.Context())
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.AdminOverviewFromDomain(stats))
}

func (h *AdminHandler) LearnersByClass(w http.ResponseWriter, r *http.Request) {
	if !h.requireAuth(w, r) {
		return
	}
	items, err := h.repo.LearnersByClass(r.Context())
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.ClassLearnerCountsFromDomain(items))
}

func (h *AdminHandler) RecentActivity(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))

	var cursor *domain.ActivityCursor
	if c := q.Get("cursor"); c != "" {
		parsed, err := decodeActivityCursor(c)
		if err != nil {
			httputil.Error(w, http.StatusBadRequest, "INVALID_INPUT", "invalid cursor")
			return
		}
		cursor = parsed
	}

	items, hasMore, err := h.repo.ListRecentActivity(r.Context(), limit, cursor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	var nextCursor *string
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		encoded := encodeActivityCursor(domain.ActivityCursor{
			CreatedAt: last.CreatedAt,
			ID:        last.ID,
		})
		nextCursor = &encoded
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"data":        dto.ActivityItemsFromDomain(items),
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

func encodeActivityCursor(c domain.ActivityCursor) string {
	raw := fmt.Sprintf("%d:%d", c.CreatedAt.UTC().UnixNano(), c.ID)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeActivityCursor(s string) (*domain.ActivityCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed cursor")
	}
	nano, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, err
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, err
	}
	return &domain.ActivityCursor{
		CreatedAt: time.Unix(0, nano).UTC(),
		ID:        id,
	}, nil
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
