package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Hisaalu/rogele/internal/httputil"
)

type PublicSettingsReader interface {
	ListPublicSettings(ctx context.Context) (map[string]string, error)
}

type ConfigHandler struct {
	repo PublicSettingsReader
	log  *slog.Logger
}

func NewConfigHandler(repo PublicSettingsReader, log *slog.Logger) *ConfigHandler {
	return &ConfigHandler{repo: repo, log: log}
}

func (h *ConfigHandler) PublicConfig(w http.ResponseWriter, r *http.Request) {
	values, err := h.repo.ListPublicSettings(r.Context())
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	branding := map[string]string{
		"school_name":   values["branding.school_name"],
		"tagline":       values["branding.tagline"],
		"logo_url":      values["branding.logo_url"],
		"favicon_url":   values["branding.favicon_url"],
		"color_primary": values["branding.color_primary"],
		"color_accent":  values["branding.color_accent"],
		"support_email": values["branding.support_email"],
		"website_url":   values["branding.website_url"],
	}

	w.Header().Set("Cache-Control", "public, max-age=60")
	httputil.JSON(w, http.StatusOK, map[string]any{
		"branding": branding,
	})
}
