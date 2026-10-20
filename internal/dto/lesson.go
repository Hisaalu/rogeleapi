package dto

import (
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
)

type CreateLessonRequest struct {
	Title       string  `json:"title"`
	Content     *string `json:"content,omitempty"`
	SubjectID   *int64  `json:"subject_id,omitempty"`
	ClassID     *int64  `json:"class_id,omitempty"`
	VideoURL    *string `json:"video_url,omitempty"`
	Duration    *int    `json:"duration,omitempty"`
	IsPublished bool    `json:"is_published,omitempty"`
}

type UpdateLessonRequest struct {
	Title       *string `json:"title,omitempty"`
	Content     *string `json:"content,omitempty"`
	SubjectID   *int64  `json:"subject_id,omitempty"`
	ClassID     *int64  `json:"class_id,omitempty"`
	VideoURL    *string `json:"video_url,omitempty"`
	Duration    *int    `json:"duration,omitempty"`
	IsPublished *bool   `json:"is_published,omitempty"`
}

type PublishRequest struct {
	Published bool `json:"published"`
}

type ApproveRequest struct {
	Approved bool `json:"approved"`
}

type CreateMaterialRequest struct {
	FileName *string `json:"file_name,omitempty"`
	FilePath *string `json:"file_path,omitempty"`
	FileType *string `json:"file_type,omitempty"`
	FileSize *int    `json:"file_size,omitempty"`
}

type LessonResponse struct {
	ID             int64      `json:"id"`
	Title          string     `json:"title"`
	Content        *string    `json:"content,omitempty"`
	SubjectID      *int64     `json:"subject_id,omitempty"`
	SubjectName    *string    `json:"subject_name,omitempty"`
	ClassID        *int64     `json:"class_id,omitempty"`
	ClassName      *string    `json:"class_name,omitempty"`
	TeacherID      *int64     `json:"teacher_id,omitempty"`
	TeacherName    *string    `json:"teacher_name,omitempty"`
	VideoURL       *string    `json:"video_url,omitempty"`
	Duration       *int       `json:"duration,omitempty"`
	IsPublished    bool       `json:"is_published"`
	IsApproved     bool       `json:"is_approved"`
	Views          int        `json:"views"`
	MaterialsCount int        `json:"materials_count"`
	IsBookmarked   bool       `json:"is_bookmarked"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

type MaterialResponse struct {
	ID            int64     `json:"id"`
	LessonID      *int64    `json:"lesson_id,omitempty"`
	FileName      *string   `json:"file_name,omitempty"`
	FilePath      *string   `json:"file_path,omitempty"`
	FileType      *string   `json:"file_type,omitempty"`
	FileSize      *int      `json:"file_size,omitempty"`
	DownloadCount int       `json:"download_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type BookmarkToggleResponse struct {
	Bookmarked bool `json:"bookmarked"`
}

type LessonExtras struct {
	ClassName      *string
	SubjectName    *string
	TeacherName    *string
	MaterialsCount int
	IsBookmarked   bool
}

func LessonFromDomain(l *domain.Lesson) LessonResponse {
	return LessonResponse{
		ID:          l.ID,
		Title:       l.Title,
		Content:     l.Content,
		SubjectID:   l.SubjectID,
		ClassID:     l.ClassID,
		TeacherID:   l.TeacherID,
		VideoURL:    l.VideoURL,
		Duration:    l.Duration,
		IsPublished: l.IsPublished,
		IsApproved:  l.IsApproved,
		Views:       l.Views,
		CreatedAt:   l.CreatedAt,
		UpdatedAt:   l.UpdatedAt,
	}
}

func LessonWithJoins(l *domain.Lesson, e LessonExtras) LessonResponse {
	return LessonResponse{
		ID:             l.ID,
		Title:          l.Title,
		Content:        l.Content,
		SubjectID:      l.SubjectID,
		SubjectName:    e.SubjectName,
		ClassID:        l.ClassID,
		ClassName:      e.ClassName,
		TeacherID:      l.TeacherID,
		TeacherName:    e.TeacherName,
		VideoURL:       l.VideoURL,
		Duration:       l.Duration,
		IsPublished:    l.IsPublished,
		IsApproved:     l.IsApproved,
		Views:          l.Views,
		MaterialsCount: e.MaterialsCount,
		IsBookmarked:   e.IsBookmarked,
		CreatedAt:      l.CreatedAt,
		UpdatedAt:      l.UpdatedAt,
	}
}

func LessonsFromDomain(ls []*domain.Lesson) []LessonResponse {
	out := make([]LessonResponse, 0, len(ls))
	for _, l := range ls {
		out = append(out, LessonFromDomain(l))
	}
	return out
}

func MaterialFromDomain(m *domain.LessonMaterial) MaterialResponse {
	return MaterialResponse{
		ID:            m.ID,
		LessonID:      m.LessonID,
		FileName:      m.FileName,
		FilePath:      m.FilePath,
		FileType:      m.FileType,
		FileSize:      m.FileSize,
		DownloadCount: m.DownloadCount,
		CreatedAt:     m.CreatedAt,
	}
}

func MaterialsFromDomain(ms []*domain.LessonMaterial) []MaterialResponse {
	out := make([]MaterialResponse, 0, len(ms))
	for _, m := range ms {
		out = append(out, MaterialFromDomain(m))
	}
	return out
}
