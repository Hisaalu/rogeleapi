package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/dto"
	"github.com/Hisaalu/rogele/internal/httputil"
	"github.com/Hisaalu/rogele/internal/repository"
	"github.com/Hisaalu/rogele/internal/server/middleware"
	"github.com/Hisaalu/rogele/internal/service"
)

type SubscriptionHandler struct {
	svc *service.SubscriptionService
	log *slog.Logger
}

func NewSubscriptionHandler(svc *service.SubscriptionService, log *slog.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc, log: log}
}

func (h *SubscriptionHandler) MyActive(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	sub, err := h.svc.GetMyActive(r.Context(), actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.SubscriptionFromDomain(sub))
}

func (h *SubscriptionHandler) MyList(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	subs, err := h.svc.ListForUser(r.Context(), actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.SubscriptionsFromDomain(subs))
}

func (h *SubscriptionHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	if page < 1 {
		page = 1
	}

	f := repository.SubscriptionFilter{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}
	if v := r.URL.Query().Get("user_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.UserID = &id
		}
	}
	if v := r.URL.Query().Get("status"); v != "" {
		s := domain.SubscriptionStatus(v)
		f.Status = &s
	}

	subs, err := h.svc.ListAll(r.Context(), actor, f)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.SubscriptionsFromDomain(subs))
}

func (h *SubscriptionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	sub, err := h.svc.Cancel(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.SubscriptionFromDomain(sub))
}

func (h *SubscriptionHandler) StartTrial(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	trial, err := h.svc.StartTrial(r.Context(), actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusCreated, dto.TrialFromDomain(trial))
}

func (h *SubscriptionHandler) MyTrial(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	trial, err := h.svc.GetMyTrial(r.Context(), actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusOK, dto.TrialFromDomain(trial))
}

func (h *SubscriptionHandler) actor(w http.ResponseWriter, r *http.Request) (service.ActorContext, bool) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return service.ActorContext{}, false
	}
	return service.ActorContext{UserID: claims.UserID, Role: claims.Role}, true
}
