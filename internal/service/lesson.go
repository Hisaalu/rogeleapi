package service

import (
	"context"
	"fmt"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository"
)

type CreateLessonInput struct {
	Title       string
	Content     *string
	SubjectID   *int64
	ClassID     *int64
	VideoURL    *string
	Duration    *int
	IsPublished bool
}

type UpdateLessonInput struct {
	Title       *string
	Content     *string
	SubjectID   *int64
	ClassID     *int64
	VideoURL    *string
	Duration    *int
	IsPublished *bool
}

type LessonService struct {
	lessons repository.LessonRepository
}

func NewLessonService(lessons repository.LessonRepository) *LessonService {
	return &LessonService{lessons: lessons}
}

func (s *LessonService) Create(ctx context.Context, teacherID int64, in CreateLessonInput) (*domain.Lesson, error) {
	if in.Title == "" {
		return nil, fmt.Errorf("%w: title is required", domain.ErrInvalidInput)
	}

	l := &domain.Lesson{
		Title:       in.Title,
		Content:     in.Content,
		SubjectID:   in.SubjectID,
		ClassID:     in.ClassID,
		TeacherID:   &teacherID,
		VideoURL:    in.VideoURL,
		Duration:    in.Duration,
		IsPublished: in.IsPublished,
		IsApproved:  false,
	}
	return s.lessons.Create(ctx, l)
}

func (s *LessonService) GetByID(ctx context.Context, id int64, actor ActorContext) (*domain.Lesson, error) {
	l, err := s.lessons.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := canViewLesson(l, actor); err != nil {
		return nil, err
	}
	if actor.Role == domain.RoleLearner {
		_ = s.lessons.IncrementViews(ctx, id)
	}
	return l, nil
}

func (s *LessonService) List(ctx context.Context, actor ActorContext, f repository.LessonFilter) ([]*domain.Lesson, error) {
	switch actor.Role {
	case domain.RoleAdmin:
	case domain.RoleTeacher:
		uid := actor.UserID
		f.TeacherID = &uid
	case domain.RoleLearner, domain.RoleExternal:
		pub := true
		f.PublishedAt = &pub
	default:
		return nil, domain.ErrForbidden
	}
	return s.lessons.List(ctx, f)
}

func (s *LessonService) Update(ctx context.Context, id int64, actor ActorContext, in UpdateLessonInput) (*domain.Lesson, error) {
	l, err := s.lessons.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := canEditLesson(l, actor); err != nil {
		return nil, err
	}

	if in.Title != nil {
		l.Title = *in.Title
	}
	if in.Content != nil {
		l.Content = in.Content
	}
	if in.SubjectID != nil {
		l.SubjectID = in.SubjectID
	}
	if in.ClassID != nil {
		l.ClassID = in.ClassID
	}
	if in.VideoURL != nil {
		l.VideoURL = in.VideoURL
	}
	if in.Duration != nil {
		l.Duration = in.Duration
	}
	if in.IsPublished != nil {
		l.IsPublished = *in.IsPublished
	}

	if err := s.lessons.Update(ctx, l); err != nil {
		return nil, err
	}
	return s.lessons.GetByID(ctx, id)
}

func (s *LessonService) Delete(ctx context.Context, id int64, actor ActorContext) error {
	l, err := s.lessons.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := canEditLesson(l, actor); err != nil {
		return err
	}
	return s.lessons.Delete(ctx, id)
}

func (s *LessonService) Publish(ctx context.Context, id int64, actor ActorContext, published bool) (*domain.Lesson, error) {
	l, err := s.lessons.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := canEditLesson(l, actor); err != nil {
		return nil, err
	}
	l.IsPublished = published
	if err := s.lessons.Update(ctx, l); err != nil {
		return nil, err
	}
	return s.lessons.GetByID(ctx, id)
}

func (s *LessonService) Approve(ctx context.Context, id int64, actor ActorContext, approved bool) (*domain.Lesson, error) {
	if actor.Role != domain.RoleAdmin {
		return nil, fmt.Errorf("%w: only admins can approve", domain.ErrForbidden)
	}
	l, err := s.lessons.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	l.IsApproved = approved
	if err := s.lessons.Update(ctx, l); err != nil {
		return nil, err
	}
	return s.lessons.GetByID(ctx, id)
}

func (s *LessonService) ToggleBookmark(ctx context.Context, userID, lessonID int64) (bool, error) {
	exists, err := s.lessons.IsBookmarked(ctx, userID, lessonID)
	if err != nil {
		return false, err
	}
	if exists {
		if err := s.lessons.RemoveBookmark(ctx, userID, lessonID); err != nil {
			return false, err
		}
		return false, nil
	}
	if err := s.lessons.AddBookmark(ctx, &domain.Bookmark{
		UserID:   &userID,
		LessonID: &lessonID,
	}); err != nil {
		return false, err
	}
	return true, nil
}

func (s *LessonService) ListBookmarks(ctx context.Context, userID int64) ([]*domain.Lesson, error) {
	return s.lessons.ListBookmarks(ctx, userID)
}

func (s *LessonService) ListMaterials(ctx context.Context, lessonID int64, actor ActorContext) ([]*domain.LessonMaterial, error) {
	l, err := s.lessons.GetByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if err := canViewLesson(l, actor); err != nil {
		return nil, err
	}
	return s.lessons.ListMaterials(ctx, lessonID)
}

func (s *LessonService) AddMaterial(ctx context.Context, lessonID int64, actor ActorContext, m *domain.LessonMaterial) (*domain.LessonMaterial, error) {
	l, err := s.lessons.GetByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if err := canEditLesson(l, actor); err != nil {
		return nil, err
	}
	m.LessonID = &lessonID
	return s.lessons.CreateMaterial(ctx, m)
}

func (s *LessonService) DeleteMaterial(ctx context.Context, materialID int64, actor ActorContext) error {
	if actor.Role != domain.RoleAdmin {
		return fmt.Errorf("%w: only admins can delete materials directly", domain.ErrForbidden)
	}
	return s.lessons.DeleteMaterial(ctx, materialID)
}

type ActorContext struct {
	UserID int64
	Role   domain.UserRole
}

func canViewLesson(l *domain.Lesson, actor ActorContext) error {
	switch actor.Role {
	case domain.RoleAdmin:
		return nil
	case domain.RoleTeacher:
		if l.TeacherID != nil && *l.TeacherID == actor.UserID {
			return nil
		}
		if l.IsPublished {
			return nil
		}
		return fmt.Errorf("%w: teachers can only view their own unpublished lessons", domain.ErrForbidden)
	case domain.RoleLearner, domain.RoleExternal:
		if l.IsPublished {
			return nil
		}
		return fmt.Errorf("%w: lesson is not published", domain.ErrForbidden)
	}
	return domain.ErrForbidden
}

func canEditLesson(l *domain.Lesson, actor ActorContext) error {
	switch actor.Role {
	case domain.RoleAdmin:
		return nil
	case domain.RoleTeacher:
		if l.TeacherID != nil && *l.TeacherID == actor.UserID {
			return nil
		}
		return fmt.Errorf("%w: teachers can only modify their own lessons", domain.ErrForbidden)
	}
	return fmt.Errorf("%w: insufficient role", domain.ErrForbidden)
}
