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

type QuizHandler struct {
	svc *service.QuizService
	log *slog.Logger
}

func NewQuizHandler(svc *service.QuizService, log *slog.Logger) *QuizHandler {
	return &QuizHandler{svc: svc, log: log}
}

func (h *QuizHandler) List(w http.ResponseWriter, r *http.Request) {
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

	f := repository.QuizFilter{
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
	httputil.JSON(w, http.StatusOK, dto.QuizListFromDomain(list))
}

func (h *QuizHandler) Get(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	q, err := h.svc.GetByID(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.QuizFromDomain(q))
}

func (h *QuizHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	if actor.Role != domain.RoleAdmin && actor.Role != domain.RoleTeacher {
		httputil.Error(w, http.StatusForbidden, "FORBIDDEN", "only teachers and admins can create quizzes")
		return
	}
	var req dto.CreateQuizRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	q, err := h.svc.Create(r.Context(), actor.UserID, service.CreateQuizInput{
		Title:        req.Title,
		Description:  req.Description,
		SubjectID:    req.SubjectID,
		ClassID:      req.ClassID,
		TimeLimit:    req.TimeLimit,
		PassingScore: req.PassingScore,
		MaxAttempts:  req.MaxAttempts,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusCreated, dto.QuizFromDomain(q))
}

func (h *QuizHandler) Update(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	var req dto.UpdateQuizRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	q, err := h.svc.Update(r.Context(), id, actor, service.UpdateQuizInput{
		Title:        req.Title,
		Description:  req.Description,
		SubjectID:    req.SubjectID,
		ClassID:      req.ClassID,
		TimeLimit:    req.TimeLimit,
		PassingScore: req.PassingScore,
		MaxAttempts:  req.MaxAttempts,
		IsPublished:  req.IsPublished,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.QuizFromDomain(q))
}

func (h *QuizHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func (h *QuizHandler) Approve(w http.ResponseWriter, r *http.Request) {
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
	q, err := h.svc.Approve(r.Context(), id, actor, req.Approved)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.QuizFromDomain(q))
}

func (h *QuizHandler) ListQuestions(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	questions, err := h.svc.ListQuestions(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.QuestionsFromDomain(questions))
}

func (h *QuizHandler) AddQuestion(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	var req dto.CreateQuestionRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	q, err := h.svc.AddQuestion(r.Context(), id, actor, service.CreateQuestionInput{
		Question:      req.Question,
		OptionA:       req.OptionA,
		OptionB:       req.OptionB,
		OptionC:       req.OptionC,
		OptionD:       req.OptionD,
		CorrectAnswer: req.CorrectAnswer,
		Points:        req.Points,
		Explanation:   req.Explanation,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusCreated, dto.QuestionFromDomain(q))
}

func (h *QuizHandler) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	quizID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "quiz id must be an integer")
		return
	}
	questionID, err := strconv.ParseInt(chi.URLParam(r, "questionId"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "question id must be an integer")
		return
	}
	if err := h.svc.DeleteQuestion(r.Context(), quizID, questionID, actor); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.NoContent(w)
}

func (h *QuizHandler) StartAttempt(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	attempt, err := h.svc.StartAttempt(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.AttemptFromDomain(attempt))
}

func (h *QuizHandler) SubmitAttempt(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "attemptId"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "attempt id must be an integer")
		return
	}
	var req dto.SubmitAttemptRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	answers := make([]service.AnswerInput, 0, len(req.Answers))
	for _, a := range req.Answers {
		answers = append(answers, service.AnswerInput{
			QuestionID:     a.QuestionID,
			SelectedAnswer: a.SelectedAnswer,
		})
	}
	attempt, err := h.svc.SubmitAttempt(r.Context(), id, actor, answers)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.AttemptFromDomain(attempt))
}

func (h *QuizHandler) GetAttempt(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "attemptId"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "attempt id must be an integer")
		return
	}
	attempt, err := h.svc.GetAttempt(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.AttemptFromDomain(attempt))
}

func (h *QuizHandler) ListMyAttempts(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	attempts, err := h.svc.ListMyAttempts(r.Context(), actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.AttemptsFromDomain(attempts))
}

func (h *QuizHandler) actor(w http.ResponseWriter, r *http.Request) (service.ActorContext, bool) {
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
