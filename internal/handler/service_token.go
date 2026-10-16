package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Hisaalu/rogele/internal/auth"
	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/httputil"
	"github.com/Hisaalu/rogele/internal/server/middleware"
)

type ServiceTokenHandler struct {
	tokens *auth.TokenManager
	log    *slog.Logger
}

func NewServiceTokenHandler(tokens *auth.TokenManager, log *slog.Logger) *ServiceTokenHandler {
	return &ServiceTokenHandler{tokens: tokens, log: log}
}

type issueServiceTokenRequest struct {
	Name string `json:"name"`
}

func (h *ServiceTokenHandler) Issue(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	if claims.Role != domain.RoleAdmin {
		httputil.Error(w, http.StatusForbidden, "FORBIDDEN", "admin only")
		return
	}

	var req issueServiceTokenRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		httputil.Error(w, http.StatusBadRequest, "INVALID_INPUT", "name is required")
		return
	}

	token, err := h.tokens.GenerateService(name, 365*24*time.Hour)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "TOKEN_FAILED", "could not generate token")
		return
	}

	h.log.Info("service token issued", "name", name)
	httputil.JSON(w, http.StatusCreated, map[string]any{
		"name":       name,
		"token":      token,
		"expires_in": 365 * 24 * 60 * 60,
	})
}
