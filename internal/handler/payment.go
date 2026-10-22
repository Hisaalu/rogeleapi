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

type PaymentHandler struct {
	svc *service.PaymentService
	log *slog.Logger
}

func NewPaymentHandler(svc *service.PaymentService, log *slog.Logger) *PaymentHandler {
	return &PaymentHandler{svc: svc, log: log}
}

func (h *PaymentHandler) Initiate(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var req dto.InitiatePaymentRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	result, err := h.svc.Initiate(r.Context(), actor, service.InitiatePaymentInput{
		PlanID:      req.PlanID,
		Method:      req.Method,
		PhoneNumber: req.PhoneNumber,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusAccepted, dto.InitiatePaymentResponse{
		Payment:           dto.PaymentFromDomain(result.Payment),
		ProviderReference: result.ProviderReference,
		Status:            result.Status,
	})
}

func (h *PaymentHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	p, err := h.svc.GetStatus(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.PaymentFromDomain(p))
}

func (h *PaymentHandler) MyList(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	items, err := h.svc.ListForUser(r.Context(), actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.PaymentsFromDomain(items))
}

func (h *PaymentHandler) ListAll(w http.ResponseWriter, r *http.Request) {
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

	f := repository.PaymentFilter{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}
	if v := r.URL.Query().Get("user_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.UserID = &id
		}
	}
	if v := r.URL.Query().Get("status"); v != "" {
		s := domain.PaymentStatus(v)
		f.Status = &s
	}

	items, err := h.svc.ListAll(r.Context(), actor, f)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.PaymentsFromDomain(items))
}

func (h *PaymentHandler) MyHistory(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	items, err := h.svc.ListHistory(r.Context(), actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.HistoryFromDomain(items))
}

func (h *PaymentHandler) actor(w http.ResponseWriter, r *http.Request) (service.ActorContext, bool) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return service.ActorContext{}, false
	}
	return service.ActorContext{UserID: claims.UserID, Role: claims.Role}, true
}

type PreviewPaymentRequest struct {
	PlanID int64 `json:"plan_id"`
}

func (h *PaymentHandler) Preview(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var req PreviewPaymentRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	amount, err := h.svc.PreviewPayment(r.Context(), actor, req.PlanID)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]float64{"amount": amount})
}
