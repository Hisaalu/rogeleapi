package domain

import "time"

type Lesson struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Content     *string    `json:"content,omitempty"`
	SubjectID   *int64     `json:"subject_id,omitempty"`
	ClassID     *int64     `json:"class_id,omitempty"`
	TeacherID   *int64     `json:"teacher_id,omitempty"`
	VideoURL    *string    `json:"video_url,omitempty"`
	Duration    *int       `json:"duration,omitempty"`
	IsPublished bool       `json:"is_published"`
	IsApproved  bool       `json:"is_approved"`
	Views       int        `json:"views"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type LessonMaterial struct {
	ID            int64     `json:"id"`
	LessonID      *int64    `json:"lesson_id,omitempty"`
	FileName      *string   `json:"file_name,omitempty"`
	FilePath      *string   `json:"file_path,omitempty"`
	FileType      *string   `json:"file_type,omitempty"`
	FileSize      *int      `json:"file_size,omitempty"`
	DownloadCount int       `json:"download_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type Bookmark struct {
	ID        int64     `json:"id"`
	UserID    *int64    `json:"user_id,omitempty"`
	LessonID  *int64    `json:"lesson_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
