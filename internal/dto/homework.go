package dto

import (
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
)

type CreateHomeworkRequest struct {
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	ClassID     int64     `json:"class_id"`
	SubjectID   int64     `json:"subject_id"`
	DueDate     time.Time `json:"due_date"`
	IsActive    bool      `json:"is_active,omitempty"`
}

type UpdateHomeworkRequest struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	ClassID     *int64     `json:"class_id,omitempty"`
	SubjectID   *int64     `json:"subject_id,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
}

type CreateAttachmentRequest struct {
	FileName *string `json:"file_name,omitempty"`
	FilePath *string `json:"file_path,omitempty"`
	FileSize *int    `json:"file_size,omitempty"`
}

type SubmitHomeworkRequest struct {
	TextAnswer *string `json:"text_answer,omitempty"`
}

type GradeSubmissionRequest struct {
	Grade    float64 `json:"grade"`
	Feedback string  `json:"feedback"`
}

type CreateSubmissionFileRequest struct {
	FileName *string `json:"file_name,omitempty"`
	FilePath *string `json:"file_path,omitempty"`
	FileSize *int    `json:"file_size,omitempty"`
}

type HomeworkResponse struct {
	ID               int64     `json:"id"`
	Title            string    `json:"title"`
	Description      *string   `json:"description,omitempty"`
	ClassID          int64     `json:"class_id"`
	ClassName        *string   `json:"class_name,omitempty"`
	SubjectID        int64     `json:"subject_id"`
	SubjectName      *string   `json:"subject_name,omitempty"`
	TeacherID        int64     `json:"teacher_id"`
	DueDate          time.Time `json:"due_date"`
	IsActive         bool      `json:"is_active"`
	SubmissionsCount int       `json:"submissions_count"`
	CreatedAt        time.Time `json:"created_at"`
}

type AttachmentResponse struct {
	ID         int64  `json:"id"`
	HomeworkID int64  `json:"homework_id"`
	FileName   string `json:"file_name"`
	FilePath   string `json:"file_path"`
	FileSize   *int   `json:"file_size,omitempty"` // Changed to *int
}

type SubmissionResponse struct {
	ID               int64                    `json:"id"`
	HomeworkID       int64                    `json:"homework_id"`
	StudentID        int64                    `json:"student_id"`
	StudentFirstName *string                  `json:"student_first_name,omitempty"`
	StudentLastName  *string                  `json:"student_last_name,omitempty"`
	StudentEmail     *string                  `json:"student_email,omitempty"`
	TextAnswer       *string                  `json:"text_answer,omitempty"`
	Grade            *float64                 `json:"grade,omitempty"`
	Feedback         *string                  `json:"feedback,omitempty"`
	Status           string                   `json:"status"`
	SubmittedAt      *time.Time               `json:"submitted_at,omitempty"` // Changed to *time.Time
	Files            []SubmissionFileResponse `json:"files,omitempty"`
}

type SubmissionFileResponse struct {
	ID           int64  `json:"id"`
	SubmissionID int64  `json:"submission_id"`
	FileName     string `json:"file_name"`
	FilePath     string `json:"file_path"`
	FileSize     *int   `json:"file_size,omitempty"` // Changed to *int
}

func HomeworkFromDomain(hw *domain.Homework) HomeworkResponse {
	return HomeworkResponse{
		ID:          hw.ID,
		Title:       hw.Title,
		Description: hw.Description,
		ClassID:     hw.ClassID,
		SubjectID:   hw.SubjectID,
		TeacherID:   hw.TeacherID,
		DueDate:     hw.DueDate,
		IsActive:    hw.IsActive,
		CreatedAt:   hw.CreatedAt,
	}
}

func HomeworksFromDomain(hws []*domain.Homework) []HomeworkResponse {
	out := make([]HomeworkResponse, 0, len(hws))
	for _, hw := range hws {
		out = append(out, HomeworkFromDomain(hw))
	}
	return out
}

func AttachmentFromDomain(a *domain.HomeworkAttachment) AttachmentResponse {
	return AttachmentResponse{
		ID:         a.ID,
		HomeworkID: a.HomeworkID,
		FileName:   a.FileName,
		FilePath:   a.FilePath,
		FileSize:   a.FileSize,
	}
}

func AttachmentsFromDomain(as []*domain.HomeworkAttachment) []AttachmentResponse {
	out := make([]AttachmentResponse, 0, len(as))
	for _, a := range as {
		out = append(out, AttachmentFromDomain(a))
	}
	return out
}

func SubmissionFromDomain(s *domain.HomeworkSubmission, files []*domain.HomeworkSubmissionFile) SubmissionResponse {
	res := SubmissionResponse{
		ID:          s.ID,
		HomeworkID:  s.HomeworkID,
		StudentID:   s.StudentID,
		TextAnswer:  s.TextAnswer,
		Grade:       s.Grade,
		Feedback:    s.Feedback,
		Status:      string(s.Status),
		SubmittedAt: s.SubmittedAt,
	}

	if len(files) > 0 {
		res.Files = make([]SubmissionFileResponse, 0, len(files))
		for _, f := range files {
			res.Files = append(res.Files, SubmissionFileResponse{
				ID:           f.ID,
				SubmissionID: f.SubmissionID,
				FileName:     f.FileName,
				FilePath:     f.FilePath,
				FileSize:     f.FileSize,
			})
		}
	}

	return res
}

func SubmissionsFromDomain(ss []*domain.HomeworkSubmission, filesMap map[int64][]*domain.HomeworkSubmissionFile) []SubmissionResponse {
	out := make([]SubmissionResponse, 0, len(ss))
	for _, s := range ss {
		out = append(out, SubmissionFromDomain(s, filesMap[s.ID]))
	}
	return out
}
