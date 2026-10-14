package dto

import (
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
)

type CreateHomeworkRequest struct {
	ClassID     int64     `json:"class_id"`
	SubjectID   int64     `json:"subject_id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	DueDate     time.Time `json:"due_date"`
}

type UpdateHomeworkRequest struct {
	ClassID     *int64     `json:"class_id,omitempty"`
	SubjectID   *int64     `json:"subject_id,omitempty"`
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
}

type SubmitHomeworkRequest struct {
	TextAnswer *string `json:"text_answer,omitempty"`
}

type GradeSubmissionRequest struct {
	Grade    float64 `json:"grade"`
	Feedback *string `json:"feedback,omitempty"`
}

type HomeworkResponse struct {
	ID          int64      `json:"id"`
	TeacherID   int64      `json:"teacher_id"`
	ClassID     int64      `json:"class_id"`
	SubjectID   int64      `json:"subject_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	DueDate     time.Time  `json:"due_date"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type SubmissionResponse struct {
	ID          int64      `json:"id"`
	HomeworkID  int64      `json:"homework_id"`
	StudentID   int64      `json:"student_id"`
	TextAnswer  *string    `json:"text_answer,omitempty"`
	Grade       *float64   `json:"grade,omitempty"`
	Feedback    *string    `json:"feedback,omitempty"`
	Status      string     `json:"status"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	GradedAt    *time.Time `json:"graded_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

func HomeworkFromDomain(hw *domain.Homework) HomeworkResponse {
	return HomeworkResponse{
		ID: hw.ID, TeacherID: hw.TeacherID, ClassID: hw.ClassID, SubjectID: hw.SubjectID,
		Title: hw.Title, Description: hw.Description, DueDate: hw.DueDate,
		IsActive: hw.IsActive, CreatedAt: hw.CreatedAt, UpdatedAt: hw.UpdatedAt,
	}
}

func HomeworkListFromDomain(hws []*domain.Homework) []HomeworkResponse {
	out := make([]HomeworkResponse, 0, len(hws))
	for _, hw := range hws {
		out = append(out, HomeworkFromDomain(hw))
	}
	return out
}

func SubmissionFromDomain(s *domain.HomeworkSubmission) SubmissionResponse {
	return SubmissionResponse{
		ID: s.ID, HomeworkID: s.HomeworkID, StudentID: s.StudentID,
		TextAnswer: s.TextAnswer, Grade: s.Grade, Feedback: s.Feedback,
		Status: string(s.Status), SubmittedAt: s.SubmittedAt, GradedAt: s.GradedAt,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func SubmissionsFromDomain(ss []*domain.HomeworkSubmission) []SubmissionResponse {
	out := make([]SubmissionResponse, 0, len(ss))
	for _, s := range ss {
		out = append(out, SubmissionFromDomain(s))
	}
	return out
}
