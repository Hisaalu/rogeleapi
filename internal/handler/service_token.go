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
	if r.ContentLength > 0 {
		_ = httputil.DecodeJSON(r, &req)
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "analytics"
	}

	token, err := h.tokens.GenerateLongLivedService(name)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "TOKEN_FAILED", "could not generate token")
		return
	}

	expiresAt := time.Now().UTC().Add(auth.ServiceTokenTTL)

	h.log.Info("service token issued", "name", name, "expires_at", expiresAt)
	httputil.JSON(w, http.StatusCreated, map[string]any{
		"name":       name,
		"token":      token,
		"expires_at": expiresAt,
		"type":       "access",
	})
}