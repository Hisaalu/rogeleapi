package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Hisaalu/rogele/internal/httputil"
	"github.com/Hisaalu/rogele/internal/repository/postgres"
)

type SettingsHandler struct {
	repo *postgres.SettingsRepo
	log  *slog.Logger
}

func NewSettingsHandler(repo *postgres.SettingsRepo, log *slog.Logger) *SettingsHandler {
	return &SettingsHandler{repo: repo, log: log}
}

func (h *SettingsHandler) UpdateTrial(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key   string `json:"setting_key"`
		Value string `json:"setting_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if req.Key == "" {
		httputil.Error(w, http.StatusBadRequest, "INVALID_INPUT", "setting_key is required")
		return
	}

	if err := h.repo.Upsert(r.Context(), req.Key, req.Value); err != nil {
		h.log.Error("failed to upsert setting", "key", req.Key, "error", err)
		httputil.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update setting")
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{"status": "success"})
}