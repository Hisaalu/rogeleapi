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

type HomeworkHandler struct {
	svc *service.HomeworkService
	log *slog.Logger
}

func NewHomeworkHandler(svc *service.HomeworkService, log *slog.Logger) *HomeworkHandler {
	return &HomeworkHandler{svc: svc, log: log}
}

func (h *HomeworkHandler) List(w http.ResponseWriter, r *http.Request) {
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

	f := repository.HomeworkFilter{
		Search: r.URL.Query().Get("q"),
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}
	if v := r.URL.Query().Get("subject_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.SubjectID = &id
		}
	}
	if v := r.URL.Query().Get("class_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.ClassID = &id
		}
	}

	list, err := h.svc.List(r.Context(), actor, f)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.HomeworkListFromDomain(list))
}

func (h *HomeworkHandler) Get(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	hw, err := h.svc.GetByID(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.HomeworkFromDomain(hw))
}

func (h *HomeworkHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	if actor.Role != domain.RoleAdmin && actor.Role != domain.RoleTeacher {
		httputil.Error(w, http.StatusForbidden, "FORBIDDEN", "only teachers and admins can create homework")
		return
	}
	var req dto.CreateHomeworkRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	hw, err := h.svc.Create(r.Context(), actor.UserID, service.CreateHomeworkInput{
		ClassID:     req.ClassID,
		SubjectID:   req.SubjectID,
		Title:       req.Title,
		Description: req.Description,
		DueDate:     req.DueDate,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusCreated, dto.HomeworkFromDomain(hw))
}

func (h *HomeworkHandler) Update(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	var req dto.UpdateHomeworkRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	hw, err := h.svc.Update(r.Context(), id, actor, service.UpdateHomeworkInput{
		ClassID:     req.ClassID,
		SubjectID:   req.SubjectID,
		Title:       req.Title,
		Description: req.Description,
		DueDate:     req.DueDate,
		IsActive:    req.IsActive,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.HomeworkFromDomain(hw))
}

func (h *HomeworkHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func (h *HomeworkHandler) Submit(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	var req dto.SubmitHomeworkRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	sub, err := h.svc.Submit(r.Context(), id, actor, req.TextAnswer)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusCreated, dto.SubmissionFromDomain(sub))
}

func (h *HomeworkHandler) ListSubmissions(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	subs, err := h.svc.ListSubmissions(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.SubmissionsFromDomain(subs))
}

func (h *HomeworkHandler) MySubmissions(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	subs, err := h.svc.MySubmissions(r.Context(), actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.SubmissionsFromDomain(subs))
}

func (h *HomeworkHandler) Grade(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "submissionId"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "submission id must be an integer")
		return
	}
	var req dto.GradeSubmissionRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	sub, err := h.svc.Grade(r.Context(), id, actor, service.GradeInput{
		Grade:    req.Grade,
		Feedback: req.Feedback,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.SubmissionFromDomain(sub))
}

func (h *HomeworkHandler) actor(w http.ResponseWriter, r *http.Request) (service.ActorContext, bool) {
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

func (h *HomeworkHandler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	items, err := h.svc.ListAttachments(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.HomeworkAttachmentsFromDomain(items))
}

func (h *HomeworkHandler) AddAttachment(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	var req dto.CreateHomeworkAttachmentRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	a, err := h.svc.AddAttachment(r.Context(), id, actor, &domain.HomeworkAttachment{
		FileName: req.FileName,
		FilePath: req.FilePath,
		FileType: req.FileType,
		FileSize: req.FileSize,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusCreated, dto.HomeworkAttachmentFromDomain(a))
}

func (h *HomeworkHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	hwID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "homework id must be an integer")
		return
	}
	attachmentID, err := strconv.ParseInt(chi.URLParam(r, "attachmentId"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "attachment id must be an integer")
		return
	}
	if err := h.svc.DeleteAttachment(r.Context(), hwID, attachmentID, actor); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.NoContent(w)
}

func (h *HomeworkHandler) ListSubmissionFiles(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "submissionId"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "submission id must be an integer")
		return
	}
	items, err := h.svc.ListSubmissionFiles(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.SubmissionFilesFromDomain(items))
}

func (h *HomeworkHandler) AddSubmissionFile(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "submissionId"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "submission id must be an integer")
		return
	}
	var req dto.CreateSubmissionFileRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	f, err := h.svc.AddSubmissionFile(r.Context(), id, actor, &domain.HomeworkSubmissionFile{
		FileName: req.FileName,
		FilePath: req.FilePath,
		FileType: req.FileType,
		FileSize: req.FileSize,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusCreated, dto.SubmissionFileFromDomain(f))
}

func (h *HomeworkHandler) DeleteSubmissionFile(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	subID, err := strconv.ParseInt(chi.URLParam(r, "submissionId"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "submission id must be an integer")
		return
	}
	fileID, err := strconv.ParseInt(chi.URLParam(r, "fileId"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "file id must be an integer")
		return
	}
	if err := h.svc.DeleteSubmissionFile(r.Context(), subID, fileID, actor); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.NoContent(w)
}
