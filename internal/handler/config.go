package handler

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Hisaalu/rogele/internal/httputil"
)

type PublicSettingsReader interface {
	ListPublicSettings(ctx context.Context) (map[string]string, error)
}

type ConfigHandler struct {
	repo PublicSettingsReader
	db   *sql.DB
	log  *slog.Logger
}

func NewConfigHandler(repo PublicSettingsReader, db *sql.DB, log *slog.Logger) *ConfigHandler {
	return &ConfigHandler{repo: repo, db: db, log: log}
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

	custom := make(map[string]string)
	for k, v := range values {
		if strings.HasPrefix(k, "branding.") {
			continue
		}
		lower := strings.ToLower(k)
		if strings.Contains(lower, "secret") ||
			strings.Contains(lower, "password") ||
			strings.Contains(lower, "token") ||
			strings.Contains(lower, "api_key") ||
			strings.Contains(lower, "apikey") {
			continue
		}
		custom[k] = v
	}

	w.Header().Set("Cache-Control", "public, max-age=60")
	httputil.JSON(w, http.StatusOK, map[string]any{
		"branding": branding,
		"settings": custom,
	})
}

func (h *ConfigHandler) PublicClasses(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT id, name FROM classes
		WHERE is_active = true
		ORDER BY name ASC
	`)
	if err != nil {
		h.log.Error("query classes", "error", err)
		httputil.Error(w, http.StatusInternalServerError, "DB_ERROR", "could not fetch classes")
		return
	}
	defer rows.Close()

	type Class struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	classes := make([]Class, 0)
	for rows.Next() {
		var c Class
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			h.log.Error("scan class", "error", err)
			httputil.Error(w, http.StatusInternalServerError, "SCAN_ERROR", "could not parse classes")
			return
		}
		classes = append(classes, c)
	}
	if err := rows.Err(); err != nil {
		h.log.Error("iterate classes", "error", err)
		httputil.Error(w, http.StatusInternalServerError, "DB_ERROR", "could not read classes")
		return
	}

	httputil.JSON(w, http.StatusOK, classes)
}
