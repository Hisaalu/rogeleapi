package domain

import "time"

type EnrollmentPoint struct {
	Period time.Time `json:"period"`
	Count  int       `json:"count"`
}

type RevenuePoint struct {
	Period time.Time `json:"period"`
	Amount float64   `json:"amount"`
	Count  int       `json:"count"`
}

type ScoreBucket struct {
	Label string `json:"label"`
	Min   int    `json:"min"`
	Max   int    `json:"max"`
	Count int    `json:"count"`
}

type ScoreDistribution struct {
	Buckets    []ScoreBucket `json:"buckets"`
	Total      int           `json:"total"`
	Average    float64       `json:"average"`
	Median     int           `json:"median"`
	PassCount  int           `json:"pass_count"`
	FailCount  int           `json:"fail_count"`
	PassingPct float64       `json:"passing_pct"`
}

type EnrollmentTrend struct {
	Interval string             `json:"interval"`
	From     time.Time          `json:"from"`
	To       time.Time          `json:"to"`
	Points   []*EnrollmentPoint `json:"points"`
	Total    int                `json:"total"`
}

type RevenueSeries struct {
	Interval   string          `json:"interval"`
	From       time.Time       `json:"from"`
	To         time.Time       `json:"to"`
	Points     []*RevenuePoint `json:"points"`
	Total      float64         `json:"total"`
	TotalCount int             `json:"total_count"`
}

type UserEvent struct {
	UserID    int64     `json:"user_id"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

type SignupDate struct {
	UserID    int64      `json:"user_id"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	CreatedAt time.Time  `json:"created_at"`
	LastLogin *time.Time `json:"last_login,omitempty"`
}

type LearnerSignal struct {
	UserID                   int64      `json:"user_id"`
	Email                    string     `json:"email"`
	FirstName                *string    `json:"first_name,omitempty"`
	LastName                 *string    `json:"last_name,omitempty"`
	Role                     string     `json:"role"`
	ClassID                  *int64     `json:"class_id,omitempty"`
	ClassName                *string    `json:"class_name,omitempty"`
	LastLogin                *time.Time `json:"last_login,omitempty"`
	LastActivity             *time.Time `json:"last_activity,omitempty"`
	DaysSinceLastActivity    *int       `json:"days_since_last_activity,omitempty"`
	QuizAttempts             int        `json:"quiz_attempts"`
	QuizAverage              *float64   `json:"quiz_average,omitempty"`
	HomeworkSubmitted        int        `json:"homework_submitted"`
	HomeworkGraded           int        `json:"homework_graded"`
	HomeworkAverageGrade     *float64   `json:"homework_average_grade,omitempty"`
	ActiveAssignmentsForUser int        `json:"active_assignments"`
	MissingAssignments       int        `json:"missing_assignments"`
}

type LearnerRawData struct {
	LearnerID     int64                         `json:"learner_id"`
	From          time.Time                     `json:"from"`
	To            time.Time                     `json:"to"`
	Homeworks     []*LearnerHomeworkSubmission  `json:"homeworks"`
	QuizAttempts  []*LearnerQuizAttempt         `json:"quiz_attempts"`
}

type LearnerHomeworkSubmission struct {
	SubmissionID int64      `json:"submission_id"`
	HomeworkID   int64      `json:"homework_id"`
	SubjectID    *int64     `json:"subject_id,omitempty"`
	SubjectName  *string    `json:"subject_name,omitempty"`
	HomeworkTitle string    `json:"homework_title"`
	Grade        *float64   `json:"grade,omitempty"`
	Status       string     `json:"status"`
	SubmittedAt  *time.Time `json:"submitted_at,omitempty"`
	GradedAt     *time.Time `json:"graded_at,omitempty"`
}

type LearnerQuizAttempt struct {
	AttemptID     int64      `json:"attempt_id"`
	QuizID        int64      `json:"quiz_id"`
	SubjectID     *int64     `json:"subject_id,omitempty"`
	SubjectName   *string    `json:"subject_name,omitempty"`
	QuizTitle     string     `json:"quiz_title"`
	Score         *int       `json:"score,omitempty"`
	TotalQuestions *int      `json:"total_questions,omitempty"`
	CorrectAnswers *int      `json:"correct_answers,omitempty"`
	TimeTaken     *int       `json:"time_taken,omitempty"`
	Status        string     `json:"status"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}