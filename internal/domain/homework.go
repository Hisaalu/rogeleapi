package domain

import "time"

type HomeworkStatus string

const (
	HomeworkStatusPending   HomeworkStatus = "pending"
	HomeworkStatusSubmitted HomeworkStatus = "submitted"
	HomeworkStatusLate      HomeworkStatus = "late"
	HomeworkStatusGraded    HomeworkStatus = "graded"
)

type Homework struct {
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

type HomeworkAttachment struct {
	ID         int64     `json:"id"`
	HomeworkID int64     `json:"homework_id"`
	FileName   string    `json:"file_name"`
	FilePath   string    `json:"file_path"`
	FileType   *string   `json:"file_type,omitempty"`
	FileSize   *int      `json:"file_size,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type HomeworkSubmission struct {
	ID          int64          `json:"id"`
	HomeworkID  int64          `json:"homework_id"`
	StudentID   int64          `json:"student_id"`
	TextAnswer  *string        `json:"text_answer,omitempty"`
	Grade       *float64       `json:"grade,omitempty"`
	Feedback    *string        `json:"feedback,omitempty"`
	Status      HomeworkStatus `json:"status"`
	SubmittedAt *time.Time     `json:"submitted_at,omitempty"`
	GradedAt    *time.Time     `json:"graded_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   *time.Time     `json:"updated_at,omitempty"`
}

type HomeworkSubmissionFile struct {
	ID           int64     `json:"id"`
	SubmissionID int64     `json:"submission_id"`
	FileName     string    `json:"file_name"`
	FilePath     string    `json:"file_path"`
	FileType     *string   `json:"file_type,omitempty"`
	FileSize     *int      `json:"file_size,omitempty"`
	UploadedAt   time.Time `json:"uploaded_at"`
}
