package repository

import (
	"context"

	"github.com/Hisaalu/rogele/internal/domain"
)

type UserFilter struct {
	Role    *domain.UserRole
	ClassID *int64
	Search  string
	Limit   int
	Offset  int
}

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) (*domain.User, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, u *domain.User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, f UserFilter) ([]*domain.User, error)
	UpdateLastLogin(ctx context.Context, id int64) error
}

type LessonFilter struct {
	TeacherID   *int64
	SubjectID   *int64
	ClassID     *int64
	PublishedAt *bool
	ApprovedAt  *bool
	Search      string
	Limit       int
	Offset      int
}

type LessonRepository interface {
	Create(ctx context.Context, l *domain.Lesson) (*domain.Lesson, error)
	GetByID(ctx context.Context, id int64) (*domain.Lesson, error)
	Update(ctx context.Context, l *domain.Lesson) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, f LessonFilter) ([]*domain.Lesson, error)
	IncrementViews(ctx context.Context, id int64) error

	ListMaterials(ctx context.Context, lessonID int64) ([]*domain.LessonMaterial, error)
	CreateMaterial(ctx context.Context, m *domain.LessonMaterial) (*domain.LessonMaterial, error)
	DeleteMaterial(ctx context.Context, id int64) error

	IsBookmarked(ctx context.Context, userID, lessonID int64) (bool, error)
	AddBookmark(ctx context.Context, b *domain.Bookmark) error
	RemoveBookmark(ctx context.Context, userID, lessonID int64) error
	ListBookmarks(ctx context.Context, userID int64) ([]*domain.Lesson, error)
}

type Repositories struct {
	User   UserRepository
	Lesson LessonRepository
}
