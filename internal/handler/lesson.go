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

type LessonHandler struct {
	svc *service.LessonService
	log *slog.Logger
}

func NewLessonHandler(svc *service.LessonService, log *slog.Logger) *LessonHandler {
	return &LessonHandler{svc: svc, log: log}
}

func (h *LessonHandler) List(w http.ResponseWriter, r *http.Request) {
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
	offset := (page - 1) * perPage

	filter := repository.LessonFilter{
		Search: r.URL.Query().Get("q"),
		Limit:  perPage,
		Offset: offset,
	}

	if v := r.URL.Query().Get("subject_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			filter.SubjectID = &id
		}
	}
	if v := r.URL.Query().Get("class_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			filter.ClassID = &id
		}
	}

	lessons, err := h.svc.List(r.Context(), actor, filter)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusOK, dto.LessonsFromDomain(lessons))
}

func (h *LessonHandler) Get(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	l, err := h.svc.GetByID(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.LessonFromDomain(l))
}

func (h *LessonHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	if actor.Role != domain.RoleAdmin && actor.Role != domain.RoleTeacher {
		httputil.Error(w, http.StatusForbidden, "FORBIDDEN", "only teachers and admins can create lessons")
		return
	}

	var req dto.CreateLessonRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	l, err := h.svc.Create(r.Context(), actor.UserID, service.CreateLessonInput{
		Title:       req.Title,
		Content:     req.Content,
		SubjectID:   req.SubjectID,
		ClassID:     req.ClassID,
		VideoURL:    req.VideoURL,
		Duration:    req.Duration,
		IsPublished: req.IsPublished,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusCreated, dto.LessonFromDomain(l))
}

func (h *LessonHandler) Update(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	var req dto.UpdateLessonRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	l, err := h.svc.Update(r.Context(), id, actor, service.UpdateLessonInput{
		Title:       req.Title,
		Content:     req.Content,
		SubjectID:   req.SubjectID,
		ClassID:     req.ClassID,
		VideoURL:    req.VideoURL,
		Duration:    req.Duration,
		IsPublished: req.IsPublished,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.LessonFromDomain(l))
}

func (h *LessonHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	if err := h.svc.Delete(r.Context(), id, actor); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.NoContent(w)
}

func (h *LessonHandler) Publish(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	var req dto.PublishRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	l, err := h.svc.Publish(r.Context(), id, actor, req.Published)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.LessonFromDomain(l))
}

func (h *LessonHandler) Approve(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	var req dto.ApproveRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	l, err := h.svc.Approve(r.Context(), id, actor, req.Approved)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.LessonFromDomain(l))
}

func (h *LessonHandler) ToggleBookmark(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	bookmarked, err := h.svc.ToggleBookmark(r.Context(), actor.UserID, id)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.BookmarkToggleResponse{Bookmarked: bookmarked})
}

func (h *LessonHandler) ListBookmarks(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	lessons, err := h.svc.ListBookmarks(r.Context(), actor.UserID)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.LessonsFromDomain(lessons))
}

func (h *LessonHandler) ListMaterials(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	materials, err := h.svc.ListMaterials(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.MaterialsFromDomain(materials))
}

func (h *LessonHandler) AddMaterial(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	var req dto.CreateMaterialRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	m, err := h.svc.AddMaterial(r.Context(), id, actor, &domain.LessonMaterial{
		FileName: req.FileName,
		FilePath: req.FilePath,
		FileType: req.FileType,
		FileSize: req.FileSize,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusCreated, dto.MaterialFromDomain(m))
}

func (h *LessonHandler) DeleteMaterial(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "materialId"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "material id must be an integer")
		return
	}
	if err := h.svc.DeleteMaterial(r.Context(), id, actor); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.NoContent(w)
}

func (h *LessonHandler) actor(w http.ResponseWriter, r *http.Request) (service.ActorContext, bool) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return service.ActorContext{}, false
	}
	return service.ActorContext{
		UserID: claims.UserID,
		Role:   claims.Role,
	}, true
}
