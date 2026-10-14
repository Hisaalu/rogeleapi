package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/dto"
	"github.com/Hisaalu/rogele/internal/httputil"
	"github.com/Hisaalu/rogele/internal/server/middleware"
	"github.com/Hisaalu/rogele/internal/service"
)

type CampaignHandler struct {
	svc *service.CampaignService
	log *slog.Logger
}

func NewCampaignHandler(svc *service.CampaignService, log *slog.Logger) *CampaignHandler {
	return &CampaignHandler{svc: svc, log: log}
}

func (h *CampaignHandler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	items, err := h.svc.List(r.Context(), perPage, (page-1)*perPage)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.CampaignsFromDomain(items))
}

func (h *CampaignHandler) Get(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	c, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.CampaignFromDomain(c))
}

func (h *CampaignHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	var req dto.CreateCampaignRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	c, err := h.svc.Create(r.Context(), claims.UserID, service.CreateCampaignInput{
		Subject:       req.Subject,
		Body:          req.Body,
		RecipientType: req.RecipientType,
		RecipientIDs:  req.RecipientIDs,
		CustomEmails:  req.CustomEmails,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusCreated, dto.CampaignFromDomain(c))
}

func (h *CampaignHandler) Update(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	var req dto.UpdateCampaignRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	c, err := h.svc.Update(r.Context(), id, service.UpdateCampaignInput{
		Subject:       req.Subject,
		Body:          req.Body,
		RecipientType: req.RecipientType,
		RecipientIDs:  req.RecipientIDs,
		CustomEmails:  req.CustomEmails,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.CampaignFromDomain(c))
}

func (h *CampaignHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.NoContent(w)
}

func (h *CampaignHandler) Preview(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	var req dto.CampaignPreviewRequest
	_ = httputil.DecodeJSON(r, &req)

	subject, body, count, err := h.svc.Preview(r.Context(), id, service.PreviewInput{
		SampleEmail: req.SampleEmail,
		SampleName:  req.SampleName,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.CampaignPreviewResponse{
		Subject:        subject,
		HTML:           body,
		RecipientCount: count,
	})
}

func (h *CampaignHandler) Send(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	queued, err := h.svc.Send(r.Context(), id)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusAccepted, dto.CampaignSendResponse{
		CampaignID:  id,
		QueuedCount: queued,
		Status:      "sending",
	})
}

func (h *CampaignHandler) Logs(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 200 {
		perPage = 50
	}

	logs, err := h.svc.Logs(r.Context(), id, perPage, (page-1)*perPage)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.LogsFromDomain(logs))
}

func (h *CampaignHandler) requireAdmin(w http.ResponseWriter, r *http.Request) (*authClaims, bool) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return nil, false
	}
	if claims.Role != domain.RoleAdmin {
		httputil.Error(w, http.StatusForbidden, "FORBIDDEN", "admin only")
		return nil, false
	}
	return &authClaims{UserID: claims.UserID, Role: claims.Role}, true
}

type authClaims struct {
	UserID int64
	Role   domain.UserRole
}
