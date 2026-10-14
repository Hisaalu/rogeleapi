package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/Hisaalu/rogele/internal/httputil"
	"github.com/Hisaalu/rogele/internal/service"
)

type WebhookHandler struct {
	svc *service.PaymentService
	log *slog.Logger
}

func NewWebhookHandler(svc *service.PaymentService, log *slog.Logger) *WebhookHandler {
	return &WebhookHandler{svc: svc, log: log}
}

func (h *WebhookHandler) MTN(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, "mtn")
}

func (h *WebhookHandler) Airtel(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, "airtel")
}

func (h *WebhookHandler) handle(w http.ResponseWriter, r *http.Request, provider string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "READ_ERROR", "could not read body")
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	h.log.Info("payment webhook received", "provider", provider, "payload", payload)

	if err := h.svc.HandleWebhook(r.Context(), provider, payload); err != nil {
		h.log.Error("webhook handling failed", "provider", provider, "error", err)
		httputil.DomainError(w, err, h.log)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
