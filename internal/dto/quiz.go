package dto

import (
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
)

type CreateQuizRequest struct {
	Title        string     `json:"title"`
	Description  *string    `json:"description,omitempty"`
	SubjectID    *int64     `json:"subject_id,omitempty"`
	ClassID      *int64     `json:"class_id,omitempty"`
	TimeLimit    *int       `json:"time_limit,omitempty"`
	PassingScore int        `json:"passing_score,omitempty"`
	MaxAttempts  int        `json:"max_attempts,omitempty"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
}

type UpdateQuizRequest struct {
	Title        *string    `json:"title,omitempty"`
	Description  *string    `json:"description,omitempty"`
	SubjectID    *int64     `json:"subject_id,omitempty"`
	ClassID      *int64     `json:"class_id,omitempty"`
	TimeLimit    *int       `json:"time_limit,omitempty"`
	PassingScore *int       `json:"passing_score,omitempty"`
	MaxAttempts  *int       `json:"max_attempts,omitempty"`
	IsPublished  *bool      `json:"is_published,omitempty"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
}

type CreateQuestionRequest struct {
	Question      string  `json:"question"`
	OptionA       *string `json:"option_a,omitempty"`
	OptionB       *string `json:"option_b,omitempty"`
	OptionC       *string `json:"option_c,omitempty"`
	OptionD       *string `json:"option_d,omitempty"`
	CorrectAnswer *string `json:"correct_answer,omitempty"`
	Points        int     `json:"points,omitempty"`
	Explanation   *string `json:"explanation,omitempty"`
}

type SubmitAttemptRequest struct {
	Answers []AnswerItem `json:"answers"`
}

type AnswerItem struct {
	QuestionID     int64 `json:"question_id"`
	SelectedAnswer *int  `json:"selected_answer,omitempty"`
}

type QuizResponse struct {
	ID           int64      `json:"id"`
	Title        string     `json:"title"`
	Description  *string    `json:"description,omitempty"`
	SubjectID    *int64     `json:"subject_id,omitempty"`
	ClassID      *int64     `json:"class_id,omitempty"`
	TeacherID    *int64     `json:"teacher_id,omitempty"`
	TimeLimit    *int       `json:"time_limit,omitempty"`
	PassingScore int        `json:"passing_score"`
	MaxAttempts  int        `json:"max_attempts"`
	IsPublished  bool       `json:"is_published"`
	IsApproved   bool       `json:"is_approved"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type QuestionResponse struct {
	ID            int64     `json:"id"`
	QuizID        *int64    `json:"quiz_id,omitempty"`
	Question      string    `json:"question"`
	OptionA       *string   `json:"option_a,omitempty"`
	OptionB       *string   `json:"option_b,omitempty"`
	OptionC       *string   `json:"option_c,omitempty"`
	OptionD       *string   `json:"option_d,omitempty"`
	CorrectAnswer *string   `json:"correct_answer,omitempty"`
	Points        int       `json:"points"`
	Explanation   *string   `json:"explanation,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type AttemptResponse struct {
	ID             int64      `json:"id"`
	QuizID         *int64     `json:"quiz_id,omitempty"`
	UserID         *int64     `json:"user_id,omitempty"`
	Score          *int       `json:"score,omitempty"`
	TotalQuestions *int       `json:"total_questions,omitempty"`
	CorrectAnswers *int       `json:"correct_answers,omitempty"`
	TimeTaken      *int       `json:"time_taken,omitempty"`
	Status         string     `json:"status"`
	StartedAt      time.Time  `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

func QuizFromDomain(q *domain.Quiz) QuizResponse {
	return QuizResponse{
		ID: q.ID, Title: q.Title, Description: q.Description,
		SubjectID: q.SubjectID, ClassID: q.ClassID, TeacherID: q.TeacherID,
		TimeLimit: q.TimeLimit, PassingScore: q.PassingScore,
		MaxAttempts: q.MaxAttempts, IsPublished: q.IsPublished,
		IsApproved: q.IsApproved, StartDate: q.StartDate, EndDate: q.EndDate,
		Status: q.Status, CreatedAt: q.CreatedAt, UpdatedAt: q.UpdatedAt,
	}
}

func QuizListFromDomain(qs []*domain.Quiz) []QuizResponse {
	out := make([]QuizResponse, 0, len(qs))
	for _, q := range qs {
		out = append(out, QuizFromDomain(q))
	}
	return out
}

func QuestionFromDomain(q *domain.QuizQuestion) QuestionResponse {
	return QuestionResponse{
		ID: q.ID, QuizID: q.QuizID, Question: q.Question,
		OptionA: q.OptionA, OptionB: q.OptionB, OptionC: q.OptionC, OptionD: q.OptionD,
		CorrectAnswer: q.CorrectAnswer, Points: q.Points,
		Explanation: q.Explanation, CreatedAt: q.CreatedAt,
	}
}

func QuestionsFromDomain(qs []*domain.QuizQuestion) []QuestionResponse {
	out := make([]QuestionResponse, 0, len(qs))
	for _, q := range qs {
		out = append(out, QuestionFromDomain(q))
	}
	return out
}

func AttemptFromDomain(a *domain.QuizAttempt) AttemptResponse {
	return AttemptResponse{
		ID: a.ID, QuizID: a.QuizID, UserID: a.UserID,
		Score: a.Score, TotalQuestions: a.TotalQuestions,
		CorrectAnswers: a.CorrectAnswers, TimeTaken: a.TimeTaken,
		Status: string(a.Status), StartedAt: a.StartedAt, CompletedAt: a.CompletedAt,
	}
}

func AttemptsFromDomain(as []*domain.QuizAttempt) []AttemptResponse {
	out := make([]AttemptResponse, 0, len(as))
	for _, a := range as {
		out = append(out, AttemptFromDomain(a))
	}
	return out
}
