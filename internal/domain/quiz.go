package domain

import "time"

type QuizAttemptStatus string

const (
	QuizAttemptInProgress QuizAttemptStatus = "in_progress"
	QuizAttemptCompleted  QuizAttemptStatus = "completed"
	QuizAttemptExpired    QuizAttemptStatus = "expired"
)

type Quiz struct {
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
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	ApprovedBy   *int64     `json:"approved_by,omitempty"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type QuizQuestion struct {
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

type QuizAttempt struct {
	ID             int64             `json:"id"`
	QuizID         *int64            `json:"quiz_id,omitempty"`
	UserID         *int64            `json:"user_id,omitempty"`
	Score          *int              `json:"score,omitempty"`
	TotalQuestions *int              `json:"total_questions,omitempty"`
	CorrectAnswers *int              `json:"correct_answers,omitempty"`
	TimeTaken      *int              `json:"time_taken,omitempty"`
	Status         QuizAttemptStatus `json:"status"`
	StartedAt      time.Time         `json:"started_at"`
	CompletedAt    *time.Time        `json:"completed_at,omitempty"`
}

type QuizAttemptAnswer struct {
	ID             int64     `json:"id"`
	AttemptID      int64     `json:"attempt_id"`
	QuestionID     int64     `json:"question_id"`
	SelectedAnswer *int      `json:"selected_answer,omitempty"`
	IsCorrect      bool      `json:"is_correct"`
	CreatedAt      time.Time `json:"created_at"`
}
