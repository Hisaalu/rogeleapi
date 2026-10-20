package handler

import (
	"context"
	"database/sql"
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
	db  *sql.DB
	log *slog.Logger
}

func NewHomeworkHandler(svc *service.HomeworkService, db *sql.DB, log *slog.Logger) *HomeworkHandler {
	return &HomeworkHandler{
		svc: svc,
		db:  db,
		log: log,
	}
}

func (h *HomeworkHandler) enrichHomework(ctx context.Context, hw *domain.Homework) dto.HomeworkResponse {
	var className, subjectName *string
	var submissionsCount int

	if h.db != nil {
		var cName string
		if err := h.db.QueryRowContext(ctx, `SELECT name FROM classes WHERE id = $1`, hw.ClassID).Scan(&cName); err == nil {
			className = &cName
		}

		var sName string
		if err := h.db.QueryRowContext(ctx, `SELECT name FROM subjects WHERE id = $1`, hw.SubjectID).Scan(&sName); err == nil {
			subjectName = &sName
		}

		_ = h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM homework_submissions WHERE homework_id = $1`, hw.ID).Scan(&submissionsCount)
	}

	res := dto.HomeworkFromDomain(hw)
	res.ClassName = className
	res.SubjectName = subjectName
	res.SubmissionsCount = submissionsCount
	return res
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

	if v := r.URL.Query().Get("class_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.ClassID = &id
		}
	}
	if v := r.URL.Query().Get("subject_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.SubjectID = &id
		}
	}

	homeworks, err := h.svc.List(r.Context(), actor, f)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	out := make([]dto.HomeworkResponse, 0, len(homeworks))
	for _, hw := range homeworks {
		out = append(out, h.enrichHomework(r.Context(), hw))
	}
	httputil.JSON(w, http.StatusOK, out)
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

	httputil.JSON(w, http.StatusOK, h.enrichHomework(r.Context(), hw))
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
		Title:       req.Title,
		Description: req.Description,
		ClassID:     req.ClassID,
		SubjectID:   req.SubjectID,
		DueDate:     req.DueDate,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusCreated, h.enrichHomework(r.Context(), hw))
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
		Title:       req.Title,
		Description: req.Description,
		ClassID:     req.ClassID,
		SubjectID:   req.SubjectID,
		DueDate:     req.DueDate,
		IsActive:    req.IsActive,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusOK, h.enrichHomework(r.Context(), hw))
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
	attachments, err := h.svc.ListAttachments(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.AttachmentsFromDomain(attachments))
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
	var req dto.CreateAttachmentRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	fileName := ""
	if req.FileName != nil {
		fileName = *req.FileName
	}
	filePath := ""
	if req.FilePath != nil {
		filePath = *req.FilePath
	}

	a, err := h.svc.AddAttachment(r.Context(), id, actor, &domain.HomeworkAttachment{
		FileName: fileName,
		FilePath: filePath,
		FileSize: req.FileSize,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusCreated, dto.AttachmentFromDomain(a))
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
	attachID, err := strconv.ParseInt(chi.URLParam(r, "attachmentId"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "attachment id must be an integer")
		return
	}
	if err := h.svc.DeleteAttachment(r.Context(), hwID, attachID, actor); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.NoContent(w)
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
	submissions, err := h.svc.ListSubmissions(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	filesMap := make(map[int64][]*domain.HomeworkSubmissionFile)
	for _, s := range submissions {
		files, _ := h.svc.ListSubmissionFiles(r.Context(), s.ID, actor)
		filesMap[s.ID] = files
	}

	httputil.JSON(w, http.StatusOK, dto.SubmissionsFromDomain(submissions, filesMap))
}

func (h *HomeworkHandler) MySubmissions(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}

	f := repository.HomeworkFilter{
		Limit: 50,
	}

	_ = f
	_ = actor
	httputil.JSON(w, http.StatusOK, []dto.SubmissionResponse{})
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
	httputil.JSON(w, http.StatusCreated, dto.SubmissionFromDomain(sub, nil))
}

func (h *HomeworkHandler) Grade(w http.ResponseWriter, r *http.Request) {
	h.GradeSubmission(w, r)
}

func (h *HomeworkHandler) GradeSubmission(w http.ResponseWriter, r *http.Request) {
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
	sub, err := h.svc.GradeSubmission(r.Context(), id, actor, req.Grade, req.Feedback)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.SubmissionFromDomain(sub, nil))
}

func (h *HomeworkHandler) DeleteSubmission(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "submissionId"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "submission id must be an integer")
		return
	}
	if err := h.svc.DeleteSubmission(r.Context(), id, actor); err != nil {
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
	files, err := h.svc.ListSubmissionFiles(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	out := make([]dto.SubmissionFileResponse, 0, len(files))
	for _, f := range files {
		out = append(out, dto.SubmissionFileResponse{
			ID:           f.ID,
			SubmissionID: f.SubmissionID,
			FileName:     f.FileName,
			FilePath:     f.FilePath,
			FileSize:     f.FileSize,
		})
	}
	httputil.JSON(w, http.StatusOK, out)
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

	fileName := ""
	if req.FileName != nil {
		fileName = *req.FileName
	}
	filePath := ""
	if req.FilePath != nil {
		filePath = *req.FilePath
	}

	f, err := h.svc.AddSubmissionFile(r.Context(), id, actor, &domain.HomeworkSubmissionFile{
		FileName: fileName,
		FilePath: filePath,
		FileSize: req.FileSize,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusCreated, dto.SubmissionFileResponse{
		ID:           f.ID,
		SubmissionID: f.SubmissionID,
		FileName:     f.FileName,
		FilePath:     f.FilePath,
		FileSize:     f.FileSize,
	})
}

func (h *HomeworkHandler) DeleteSubmissionFile(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "fileId"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "file id must be an integer")
		return
	}
	if err := h.svc.DeleteSubmissionFile(r.Context(), id, actor); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.NoContent(w)
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
