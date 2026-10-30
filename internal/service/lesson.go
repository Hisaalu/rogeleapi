package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository"
	"github.com/Hisaalu/rogele/internal/storage"
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
	lessons       repository.LessonRepository
	users         repository.UserRepository
	notifications *NotificationService
	activity      *ActivityService
	r2            *storage.R2Client
	log           *slog.Logger
}

func NewLessonService(
	lessons repository.LessonRepository,
	users repository.UserRepository,
	notifications *NotificationService,
	activity *ActivityService,
	r2 *storage.R2Client,
	log *slog.Logger,
) *LessonService {
	return &LessonService{
		lessons:       lessons,
		users:         users,
		notifications: notifications,
		activity:      activity,
		r2:            r2,
		log:           log,
	}
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
	created, err := s.lessons.Create(ctx, l)
	if err != nil {
		return nil, err
	}
	if created.IsPublished {
		s.notifyNewLesson(ctx, created)
	}
	return created, nil
}

func (s *LessonService) GetByID(ctx context.Context, id int64, actor ActorContext) (*domain.Lesson, error) {
	l, err := s.lessons.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.canViewLesson(ctx, l, actor); err != nil {
		return nil, err
	}
	if actor.Role == domain.RoleLearner {
		_ = s.lessons.IncrementViews(ctx, id)
		s.activity.Log(ctx, actor.UserID, "lesson_view", l.Title)
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
		classID, err := s.learnerClassID(ctx, actor)
		if err != nil {
			return nil, err
		}
		if classID == nil {
			return []*domain.Lesson{}, nil
		}
		f.ClassID = classID
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

	materials, err := s.lessons.ListMaterials(ctx, id)
	if err == nil && s.r2 != nil {
		for _, m := range materials {
			if m.FilePath == nil || *m.FilePath == "" {
				continue
			}
			key := s.r2.ExtractKey(*m.FilePath)
			if key != "" {
				s.log.Info("deleting material on lesson delete", "key", key)
				if delErr := s.r2.DeleteObject(ctx, key); delErr != nil {
					s.log.Error("failed to delete r2 object", "key", key, "error", delErr)
				}
			}
		}
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
	wasPublished := l.IsPublished
	l.IsPublished = published
	if err := s.lessons.Update(ctx, l); err != nil {
		return nil, err
	}
	updated, err := s.lessons.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if published && !wasPublished {
		s.notifyNewLesson(ctx, updated)
	}
	return updated, nil
}

func (s *LessonService) notifyNewLesson(ctx context.Context, l *domain.Lesson) {
	if s.notifications == nil || l.ClassID == nil {
		return
	}

	learners, err := s.users.List(ctx, repository.UserFilter{
		Role:    rolePtr(domain.RoleLearner),
		ClassID: l.ClassID,
		Limit:   500,
	})
	if err != nil || len(learners) == 0 {
		return
	}

	ids := make([]int64, 0, len(learners))
	for _, u := range learners {
		ids = append(ids, u.ID)
	}

	link := fmt.Sprintf("/learner/lessons/%d", l.ID)
	title := "New lesson"
	message := l.Title

	_ = s.notifications.NotifyUsers(ctx, ids, domain.NotificationTypeLesson, title, message, &link)
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
	if err := s.canViewLesson(ctx, l, actor); err != nil {
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
	m, err := s.lessons.GetMaterialByID(ctx, materialID)
	if err != nil {
		return err
	}

	if m.LessonID == nil {
		return fmt.Errorf("%w: material has no lesson", domain.ErrInternal)
	}

	l, err := s.lessons.GetByID(ctx, *m.LessonID)
	if err != nil {
		return err
	}

	if err := canEditLesson(l, actor); err != nil {
		return err
	}

	if m.FilePath != nil && *m.FilePath != "" && s.r2 != nil {
		key := s.r2.ExtractKey(*m.FilePath)
		if key != "" {
			s.log.Info("deleting material", "key", key)
			if delErr := s.r2.DeleteObject(ctx, key); delErr != nil {
				return fmt.Errorf("failed to delete file from storage: %w", delErr)
			}
		}
	}

	return s.lessons.DeleteMaterial(ctx, materialID)
}

func (s *LessonService) IsBookmarked(ctx context.Context, userID, lessonID int64) (bool, error) {
	return s.lessons.IsBookmarked(ctx, userID, lessonID)
}

func (s *LessonService) CountMaterials(ctx context.Context, lessonID int64) (int, error) {
	ms, err := s.lessons.ListMaterials(ctx, lessonID)
	if err != nil {
		return 0, err
	}
	return len(ms), nil
}

func (s *LessonService) TeacherName(ctx context.Context, teacherID *int64) *string {
	if teacherID == nil {
		return nil
	}
	u, err := s.users.GetByID(ctx, *teacherID)
	if err != nil || u == nil {
		return nil
	}

	var first, last string
	if u.FirstName != nil {
		first = *u.FirstName
	}
	if u.LastName != nil {
		last = *u.LastName
	}

	name := strings.TrimSpace(first + " " + last)
	if name == "" {
		return nil
	}
	return &name
}

func (s *LessonService) learnerClassID(ctx context.Context, actor ActorContext) (*int64, error) {
	u, err := s.users.GetByID(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}
	return u.ClassID, nil
}

func (s *LessonService) canViewLesson(ctx context.Context, l *domain.Lesson, actor ActorContext) error {
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
		if !l.IsPublished {
			return fmt.Errorf("%w: lesson is not published", domain.ErrForbidden)
		}

		classID, err := s.learnerClassID(ctx, actor)
		if err != nil {
			return err
		}
		if classID != nil && l.ClassID != nil && *classID == *l.ClassID {
			return nil
		}

		isBookmarked, err := s.lessons.IsBookmarked(ctx, actor.UserID, l.ID)
		if err == nil && isBookmarked {
			return nil
		}

		return fmt.Errorf("%w: lesson is not for your class", domain.ErrForbidden)
	}
	return domain.ErrForbidden
}

type ActorContext struct {
	UserID int64
	Role   domain.UserRole
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

func (s *LessonService) CountList(ctx context.Context, actor ActorContext, f repository.LessonFilter) (int, error) {
	switch actor.Role {
	case domain.RoleAdmin:
	case domain.RoleTeacher:
		uid := actor.UserID
		f.TeacherID = &uid
	case domain.RoleLearner, domain.RoleExternal:
		pub := true
		f.PublishedAt = &pub
		classID, err := s.learnerClassID(ctx, actor)
		if err != nil {
			return 0, err
		}
		if classID == nil {
			return 0, nil
		}
		f.ClassID = classID
	default:
		return 0, domain.ErrForbidden
	}
	return s.lessons.Count(ctx, f)
}
