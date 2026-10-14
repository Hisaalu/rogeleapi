package handler

import (
	"log/slog"
	"net/http"

	"github.com/Hisaalu/rogele/internal/dto"
	"github.com/Hisaalu/rogele/internal/httputil"
	"github.com/Hisaalu/rogele/internal/storage"
)

type UploadHandler struct {
	r2  *storage.R2Client
	log *slog.Logger
}

func NewUploadHandler(r2 *storage.R2Client, log *slog.Logger) *UploadHandler {
	return &UploadHandler{r2: r2, log: log}
}

func (h *UploadHandler) Presign(w http.ResponseWriter, r *http.Request) {
	var req dto.PresignUploadRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	result, err := h.r2.PresignUpload(r.Context(), storage.PresignRequest{
		FileName:    req.FileName,
		ContentType: req.ContentType,
		Size:        req.Size,
		Prefix:      req.Prefix,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusOK, dto.PresignUploadResponse{
		UploadURL: result.UploadURL,
		FileKey:   result.FileKey,
		PublicURL: result.PublicURL,
		ExpiresIn: result.ExpiresIn,
	})
}
