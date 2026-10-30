package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/dto"
	"github.com/Hisaalu/rogele/internal/httputil"
	"github.com/Hisaalu/rogele/internal/repository"
	"github.com/Hisaalu/rogele/internal/server/middleware"
	"github.com/Hisaalu/rogele/internal/service"
)

type QuizHandler struct {
	svc *service.QuizService
	db  *sql.DB
	log *slog.Logger
}

func NewQuizHandler(svc *service.QuizService, db *sql.DB, log *slog.Logger) *QuizHandler {
	return &QuizHandler{
		svc: svc,
		db:  db,
		log: log,
	}
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
		Search: r.URL.Query().Get("q"),
	}

	if v := r.URL.Query().Get("teacher_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.TeacherID = &id
		}
	} else if actor.Role == domain.RoleTeacher {
		f.TeacherID = &actor.UserID
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

	total, _ := h.svc.CountList(r.Context(), actor, f)

	ids := make([]int64, 0, len(list))
	for _, q := range list {
		ids = append(ids, q.ID)
	}

	questionCounts := map[int64]int{}
	attemptCounts := map[int64]int{}
	classNames := map[int64]string{}
	subjectNames := map[int64]string{}

	if len(ids) > 0 {
		if rows, err := h.db.QueryContext(r.Context(),
			`SELECT quiz_id, COUNT(*)::int FROM quiz_questions WHERE quiz_id = ANY($1) GROUP BY quiz_id`,
			pq.Array(ids),
		); err == nil {
			for rows.Next() {
				var id int64
				var n int
				if rows.Scan(&id, &n) == nil {
					questionCounts[id] = n
				}
			}
			rows.Close()
		}

		if rows, err := h.db.QueryContext(r.Context(),
			`SELECT quiz_id, COUNT(*)::int FROM quiz_attempts WHERE quiz_id = ANY($1) GROUP BY quiz_id`,
			pq.Array(ids),
		); err == nil {
			for rows.Next() {
				var id int64
				var n int
				if rows.Scan(&id, &n) == nil {
					attemptCounts[id] = n
				}
			}
			rows.Close()
		}
	}

	classIDs := make([]int64, 0)
	seenClass := map[int64]bool{}
	subjectIDs := make([]int64, 0)
	seenSubject := map[int64]bool{}

	for _, q := range list {
		if q.ClassID != nil && !seenClass[*q.ClassID] {
			seenClass[*q.ClassID] = true
			classIDs = append(classIDs, *q.ClassID)
		}
		if q.SubjectID != nil && !seenSubject[*q.SubjectID] {
			seenSubject[*q.SubjectID] = true
			subjectIDs = append(subjectIDs, *q.SubjectID)
		}
	}

	if len(classIDs) > 0 {
		if rows, err := h.db.QueryContext(r.Context(),
			`SELECT id, name FROM classes WHERE id = ANY($1)`,
			pq.Array(classIDs),
		); err == nil {
			for rows.Next() {
				var id int64
				var name string
				if rows.Scan(&id, &name) == nil {
					classNames[id] = name
				}
			}
			rows.Close()
		}
	}

	if len(subjectIDs) > 0 {
		if rows, err := h.db.QueryContext(r.Context(),
			`SELECT id, name FROM subjects WHERE id = ANY($1)`,
			pq.Array(subjectIDs),
		); err == nil {
			for rows.Next() {
				var id int64
				var name string
				if rows.Scan(&id, &name) == nil {
					subjectNames[id] = name
				}
			}
			rows.Close()
		}
	}

	out := make([]dto.QuizResponse, 0, len(list))
	for _, q := range list {
		var className, subjectName *string
		if q.ClassID != nil {
			if name, ok := classNames[*q.ClassID]; ok {
				className = &name
			}
		}
		if q.SubjectID != nil {
			if name, ok := subjectNames[*q.SubjectID]; ok {
				subjectName = &name
			}
		}
		out = append(out, dto.QuizWithExtras(
			q, className, subjectName,
			questionCounts[q.ID], attemptCounts[q.ID],
		))
	}

	lastPage := 1
	if perPage > 0 && total > 0 {
		lastPage = (total + perPage - 1) / perPage
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"data": out,
		"meta": map[string]int{
			"page":      page,
			"per_page":  perPage,
			"total":     total,
			"last_page": lastPage,
		},
	})
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
		IsPublished:  req.IsPublished,
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

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	var bulkReq struct {
		Questions []dto.CreateQuestionRequest `json:"questions"`
	}
	var singleReq dto.CreateQuestionRequest

	if err := json.Unmarshal(bodyBytes, &bulkReq); err == nil && len(bulkReq.Questions) > 0 {
		for _, q := range bulkReq.Questions {
			_, err := h.svc.AddQuestion(r.Context(), id, actor, service.CreateQuestionInput{
				Question:      q.Question,
				OptionA:       q.OptionA,
				OptionB:       q.OptionB,
				OptionC:       q.OptionC,
				OptionD:       q.OptionD,
				CorrectAnswer: q.CorrectAnswer,
				Points:        q.Points,
				Explanation:   q.Explanation,
			})
			if err != nil {
				httputil.DomainError(w, err, h.log)
				return
			}
		}
		httputil.JSON(w, http.StatusCreated, map[string]string{"message": "Questions added successfully"})
		return
	}

	if err := json.Unmarshal(bodyBytes, &singleReq); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	q, err := h.svc.AddQuestion(r.Context(), id, actor, service.CreateQuestionInput{
		Question:      singleReq.Question,
		OptionA:       singleReq.OptionA,
		OptionB:       singleReq.OptionB,
		OptionC:       singleReq.OptionC,
		OptionD:       singleReq.OptionD,
		CorrectAnswer: singleReq.CorrectAnswer,
		Points:        singleReq.Points,
		Explanation:   singleReq.Explanation,
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

	answers, _ := h.svc.ListAttemptAnswers(r.Context(), attempt.ID)
	saved := make(map[string]int, len(answers))
	for _, a := range answers {
		if a.SelectedAnswer != nil {
			saved[fmt.Sprintf("%d", a.QuestionID)] = *a.SelectedAnswer
		}
	}

	resp := dto.AttemptFromDomain(attempt)
	resp.SavedAnswers = saved
	httputil.JSON(w, http.StatusOK, resp)
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

	if attempt.Status == domain.QuizAttemptCompleted && attempt.QuizID != nil {
		questions, _ := h.svc.ListRawQuestions(r.Context(), *attempt.QuizID)
		answers, _ := h.svc.ListAttemptAnswers(r.Context(), attempt.ID)
		quiz, _ := h.svc.GetByID(r.Context(), *attempt.QuizID, actor)

		passingScore := 70
		if quiz != nil {
			passingScore = quiz.PassingScore
		}

		userAnswers := make(map[string]int)
		for _, a := range answers {
			if a.SelectedAnswer != nil {
				userAnswers[fmt.Sprintf("%d", a.QuestionID)] = *a.SelectedAnswer
			}
		}

		res := dto.AttemptResultDetailsResponse{
			ID: attempt.ID, QuizID: attempt.QuizID, UserID: attempt.UserID,
			Score: attempt.Score, TotalQuestions: attempt.TotalQuestions, CorrectAnswers: attempt.CorrectAnswers,
			TimeTaken: attempt.TimeTaken, Status: string(attempt.Status), StartedAt: attempt.StartedAt, CompletedAt: attempt.CompletedAt,
			PassingScore: passingScore,
			Questions:    dto.QuestionsFromDomain(questions),
			UserAnswers:  userAnswers,
		}
		httputil.JSON(w, http.StatusOK, res)
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

func (h *QuizHandler) ListAttempts(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	attempts, err := h.svc.ListAttemptsForQuiz(r.Context(), id, actor)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, dto.AttemptsFromDomain(attempts))
}

func (h *QuizHandler) DeleteAllAttempts(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}
	if err := h.svc.DeleteAllAttempts(r.Context(), id, actor); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.NoContent(w)
}

func (h *QuizHandler) AddBulkQuestions(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	quizID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "quiz id must be an integer")
		return
	}
	var req struct {
		Questions []dto.CreateQuestionRequest `json:"questions"`
	}
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	for _, q := range req.Questions {
		_, err := h.svc.AddQuestion(r.Context(), quizID, actor, service.CreateQuestionInput{
			Question:      q.Question,
			OptionA:       q.OptionA,
			OptionB:       q.OptionB,
			OptionC:       q.OptionC,
			OptionD:       q.OptionD,
			CorrectAnswer: q.CorrectAnswer,
			Points:        q.Points,
			Explanation:   q.Explanation,
		})
		if err != nil {
			httputil.DomainError(w, err, h.log)
			return
		}
	}
	httputil.JSON(w, http.StatusCreated, map[string]string{"message": "Questions added successfully"})
}

func (h *QuizHandler) GetGlobalStats(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	if actor.Role != domain.RoleAdmin && actor.Role != domain.RoleTeacher {
		httputil.Error(w, http.StatusForbidden, "FORBIDDEN", "only teachers and admins can view global stats")
		return
	}
	avg, err := h.svc.GetGlobalAverageScore(r.Context())
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]int{"average_score": avg})
}

func (h *QuizHandler) GetPerformanceChart(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	if actor.Role != domain.RoleAdmin && actor.Role != domain.RoleTeacher {
		httputil.Error(w, http.StatusForbidden, "FORBIDDEN", "only teachers and admins can view analytics")
		return
	}
	daysStr := r.URL.Query().Get("days")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 30
	}
	perf, err := h.svc.GetQuizPerformanceByDay(r.Context(), days)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, perf)
}

func (h *QuizHandler) SaveAttemptAnswer(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	attemptID, err := strconv.ParseInt(chi.URLParam(r, "attemptId"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "attempt id must be an integer")
		return
	}
	var req dto.AnswerItem
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	err = h.svc.SaveSingleAnswer(r.Context(), attemptID, actor, req.QuestionID, req.SelectedAnswer)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "saved"})
}
