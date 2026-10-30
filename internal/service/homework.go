package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository"
	"github.com/Hisaalu/rogele/internal/storage"
)

type CreateHomeworkInput struct {
	Title       string
	Description *string
	ClassID     int64
	SubjectID   int64
	DueDate     time.Time
}

type UpdateHomeworkInput struct {
	Title       *string
	Description *string
	ClassID     *int64
	SubjectID   *int64
	DueDate     *time.Time
	IsActive    *bool
}

type HomeworkService struct {
	repo          repository.HomeworkRepository
	users         repository.UserRepository
	notifications *NotificationService
	activity      *ActivityService
	r2            *storage.R2Client
	log           *slog.Logger
}

func NewHomeworkService(
	repo repository.HomeworkRepository,
	users repository.UserRepository,
	notifications *NotificationService,
	activity *ActivityService,
	r2 *storage.R2Client,
	log *slog.Logger,
) *HomeworkService {
	return &HomeworkService{
		repo:          repo,
		users:         users,
		notifications: notifications,
		activity:      activity,
		r2:            r2,
		log:           log,
	}
}

func (s *HomeworkService) Create(ctx context.Context, teacherID int64, in CreateHomeworkInput) (*domain.Homework, error) {
	if in.Title == "" {
		return nil, fmt.Errorf("%w: title is required", domain.ErrInvalidInput)
	}
	if in.ClassID == 0 || in.SubjectID == 0 {
		return nil, fmt.Errorf("%w: class and subject are required", domain.ErrInvalidInput)
	}

	hw := &domain.Homework{
		Title:       in.Title,
		Description: in.Description,
		ClassID:     in.ClassID,
		SubjectID:   in.SubjectID,
		TeacherID:   teacherID,
		DueDate:     in.DueDate.UTC(),
		IsActive:    true,
	}
	created, err := s.repo.Create(ctx, hw)
	if err != nil {
		return nil, err
	}

	s.notifyNewHomework(ctx, created)
	return created, nil
}

func (s *HomeworkService) notifyNewHomework(ctx context.Context, hw *domain.Homework) {
	if s.notifications == nil {
		return
	}

	learners, err := s.users.List(ctx, repository.UserFilter{
		Role:    rolePtr(domain.RoleLearner),
		ClassID: &hw.ClassID,
		Limit:   500,
	})
	if err != nil || len(learners) == 0 {
		return
	}

	ids := make([]int64, 0, len(learners))
	for _, u := range learners {
		ids = append(ids, u.ID)
	}

	link := fmt.Sprintf("/learner/homework/%d", hw.ID)
	title := "New homework"
	message := hw.Title

	_ = s.notifications.NotifyUsers(ctx, ids, domain.NotificationTypeHomework, title, message, &link)
}

func (s *HomeworkService) GetByID(ctx context.Context, id int64, actor ActorContext) (*domain.Homework, error) {
	hw, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.canView(ctx, hw, actor); err != nil {
		return nil, err
	}
	return hw, nil
}

func (s *HomeworkService) List(ctx context.Context, actor ActorContext, f repository.HomeworkFilter) ([]*domain.Homework, error) {
	switch actor.Role {
	case domain.RoleAdmin:
	case domain.RoleTeacher:
		f.TeacherID = &actor.UserID
	case domain.RoleLearner, domain.RoleExternal:
		classID, err := s.learnerClassID(ctx, actor)
		if err != nil {
			return nil, err
		}
		if classID == nil {
			return []*domain.Homework{}, nil
		}
		f.ClassID = classID
		active := true
		f.IsActive = &active
	default:
		return nil, domain.ErrForbidden
	}
	return s.repo.List(ctx, f)
}

func (s *HomeworkService) ListMySubmissions(ctx context.Context, actor ActorContext) ([]*domain.HomeworkSubmission, error) {
	if actor.Role != domain.RoleLearner && actor.Role != domain.RoleExternal {
		return []*domain.HomeworkSubmission{}, nil
	}
	return s.repo.ListSubmissionsByStudent(ctx, actor.UserID)
}

func (s *HomeworkService) Update(ctx context.Context, id int64, actor ActorContext, in UpdateHomeworkInput) (*domain.Homework, error) {
	hw, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.canEdit(hw, actor); err != nil {
		return nil, err
	}

	if in.Title != nil {
		hw.Title = *in.Title
	}
	if in.Description != nil {
		hw.Description = in.Description
	}
	if in.ClassID != nil {
		hw.ClassID = *in.ClassID
	}
	if in.SubjectID != nil {
		hw.SubjectID = *in.SubjectID
	}
	if in.DueDate != nil {
		hw.DueDate = in.DueDate.UTC()
	}
	if in.IsActive != nil {
		hw.IsActive = *in.IsActive
	}

	if err := s.repo.Update(ctx, hw); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *HomeworkService) Delete(ctx context.Context, id int64, actor ActorContext) error {
	hw, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.canEdit(hw, actor); err != nil {
		return err
	}

	attachments, _ := s.repo.ListAttachments(ctx, id)
	for _, a := range attachments {
		if a.FilePath != "" && s.r2 != nil {
			key := s.r2.ExtractKey(a.FilePath)
			if key != "" {
				s.log.Info("deleting homework attachment on homework delete", "key", key)
				_ = s.r2.DeleteObject(ctx, key)
			}
		}
	}

	submissions, _ := s.repo.ListSubmissions(ctx, id)
	for _, sub := range submissions {
		files, _ := s.repo.ListSubmissionFiles(ctx, sub.ID)
		for _, f := range files {
			if f.FilePath != "" && s.r2 != nil {
				key := s.r2.ExtractKey(f.FilePath)
				if key != "" {
					s.log.Info("deleting submission file on homework delete", "key", key)
					_ = s.r2.DeleteObject(ctx, key)
				}
			}
		}
	}

	return s.repo.Delete(ctx, id)
}

func (s *HomeworkService) ListAttachments(ctx context.Context, homeworkID int64, actor ActorContext) ([]*domain.HomeworkAttachment, error) {
	hw, err := s.repo.GetByID(ctx, homeworkID)
	if err != nil {
		return nil, err
	}
	if err := s.canView(ctx, hw, actor); err != nil {
		return nil, err
	}
	return s.repo.ListAttachments(ctx, homeworkID)
}

func (s *HomeworkService) AddAttachment(ctx context.Context, homeworkID int64, actor ActorContext, a *domain.HomeworkAttachment) (*domain.HomeworkAttachment, error) {
	hw, err := s.repo.GetByID(ctx, homeworkID)
	if err != nil {
		return nil, err
	}
	if err := s.canEdit(hw, actor); err != nil {
		return nil, err
	}
	a.HomeworkID = homeworkID
	return s.repo.CreateAttachment(ctx, a)
}

func (s *HomeworkService) DeleteAttachment(ctx context.Context, homeworkID, attachmentID int64, actor ActorContext) error {
	hw, err := s.repo.GetByID(ctx, homeworkID)
	if err != nil {
		return err
	}
	if err := s.canEdit(hw, actor); err != nil {
		return err
	}

	a, err := s.repo.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return err
	}

	if a.FilePath != "" && s.r2 != nil {
		key := s.r2.ExtractKey(a.FilePath)
		if key != "" {
			s.log.Info("deleting homework attachment", "key", key)
			if delErr := s.r2.DeleteObject(ctx, key); delErr != nil {
				return fmt.Errorf("failed to delete file from storage: %w", delErr)
			}
		}
	}

	return s.repo.DeleteAttachment(ctx, attachmentID)
}

func (s *HomeworkService) ListSubmissions(ctx context.Context, homeworkID int64, actor ActorContext) ([]*domain.HomeworkSubmission, error) {
	hw, err := s.repo.GetByID(ctx, homeworkID)
	if err != nil {
		return nil, err
	}

	if actor.Role == domain.RoleLearner || actor.Role == domain.RoleExternal {
		classID, err := s.learnerClassID(ctx, actor)
		if err != nil {
			return nil, err
		}
		if classID == nil || *classID != hw.ClassID {
			return nil, fmt.Errorf("%w: homework is not for your class", domain.ErrForbidden)
		}
		own, err := s.repo.GetSubmission(ctx, homeworkID, actor.UserID)
		if err != nil {
			if err == domain.ErrNotFound {
				return []*domain.HomeworkSubmission{}, nil
			}
			return nil, err
		}
		return []*domain.HomeworkSubmission{own}, nil
	} else if actor.Role == domain.RoleTeacher {
		if hw.TeacherID != actor.UserID {
			return nil, fmt.Errorf("%w: teachers can only view their own homework", domain.ErrForbidden)
		}
	} else if actor.Role != domain.RoleAdmin {
		return nil, domain.ErrForbidden
	}

	return s.repo.ListSubmissions(ctx, homeworkID)
}

func (s *HomeworkService) Submit(ctx context.Context, homeworkID int64, actor ActorContext, textAnswer *string) (*domain.HomeworkSubmission, error) {
	hw, err := s.repo.GetByID(ctx, homeworkID)
	if err != nil {
		return nil, err
	}
	if !hw.IsActive {
		return nil, fmt.Errorf("%w: homework is not active", domain.ErrForbidden)
	}

	now := time.Now().UTC()
	dueDate := hw.DueDate.UTC()

	existing, err := s.repo.GetSubmission(ctx, homeworkID, actor.UserID)
	if err == nil && existing != nil {
		if string(existing.Status) == "graded" {
			return nil, fmt.Errorf("%w: cannot resubmit graded homework", domain.ErrForbidden)
		}

		existing.TextAnswer = textAnswer
		if now.After(dueDate) {
			existing.Status = domain.HomeworkStatus("late")
		} else {
			existing.Status = domain.HomeworkStatus("submitted")
		}
		existing.SubmittedAt = &now

		if err := s.repo.UpdateSubmission(ctx, existing); err != nil {
			return nil, err
		}
		return s.repo.GetSubmissionByID(ctx, existing.ID)
	}

	if err != nil && err != domain.ErrNotFound {
		return nil, err
	}

	status := domain.HomeworkStatus("submitted")
	if now.After(dueDate) {
		status = domain.HomeworkStatus("late")
	}

	sub := &domain.HomeworkSubmission{
		HomeworkID:  homeworkID,
		StudentID:   actor.UserID,
		TextAnswer:  textAnswer,
		Status:      status,
		SubmittedAt: &now,
	}

	return s.repo.CreateSubmission(ctx, sub)
}

func (s *HomeworkService) GradeSubmission(ctx context.Context, submissionID int64, actor ActorContext, grade float64, feedback string) (*domain.HomeworkSubmission, error) {
	sub, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	hw, err := s.repo.GetByID(ctx, sub.HomeworkID)
	if err != nil {
		return nil, err
	}
	if err := s.canEdit(hw, actor); err != nil {
		return nil, err
	}

	if grade < 0 {
		grade = 0
	}
	if grade > 100 {
		grade = 100
	}

	now := time.Now().UTC()

	sub.Grade = &grade
	sub.Feedback = &feedback
	sub.Status = domain.HomeworkStatus("graded")
	sub.GradedAt = &now

	if err := s.repo.UpdateSubmission(ctx, sub); err != nil {
		return nil, err
	}
	return s.repo.GetSubmissionByID(ctx, submissionID)
}

func (s *HomeworkService) DeleteSubmission(ctx context.Context, submissionID int64, actor ActorContext) error {
	sub, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return err
	}

	if actor.Role == domain.RoleLearner {
		if sub.StudentID != actor.UserID {
			return fmt.Errorf("%w: you can only delete your own submission", domain.ErrForbidden)
		}
	} else if actor.Role == domain.RoleTeacher {
		hw, err := s.repo.GetByID(ctx, sub.HomeworkID)
		if err != nil {
			return err
		}
		if hw.TeacherID != actor.UserID {
			return fmt.Errorf("%w: teachers can only delete submissions for their own homework", domain.ErrForbidden)
		}
	} else if actor.Role != domain.RoleAdmin {
		return fmt.Errorf("%w: insufficient permissions", domain.ErrForbidden)
	}

	files, _ := s.repo.ListSubmissionFiles(ctx, submissionID)
	for _, f := range files {
		if f.FilePath != "" && s.r2 != nil {
			key := s.r2.ExtractKey(f.FilePath)
			if key != "" {
				s.log.Info("deleting submission file on submission delete", "key", key)
				_ = s.r2.DeleteObject(ctx, key)
			}
		}
	}

	return s.repo.DeleteSubmission(ctx, submissionID)
}

func (s *HomeworkService) ListSubmissionFiles(ctx context.Context, submissionID int64, actor ActorContext) ([]*domain.HomeworkSubmissionFile, error) {
	sub, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	hw, err := s.repo.GetByID(ctx, sub.HomeworkID)
	if err != nil {
		return nil, err
	}
	if err := s.canView(ctx, hw, actor); err != nil {
		return nil, err
	}
	if actor.Role == domain.RoleLearner && sub.StudentID != actor.UserID {
		return nil, fmt.Errorf("%w: you can only view your own submission files", domain.ErrForbidden)
	}
	return s.repo.ListSubmissionFiles(ctx, submissionID)
}

func (s *HomeworkService) AddSubmissionFile(ctx context.Context, submissionID int64, actor ActorContext, f *domain.HomeworkSubmissionFile) (*domain.HomeworkSubmissionFile, error) {
	sub, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if sub.StudentID != actor.UserID {
		return nil, fmt.Errorf("%w: you can only add files to your own submission", domain.ErrForbidden)
	}
	f.SubmissionID = submissionID
	return s.repo.CreateSubmissionFile(ctx, f)
}

func (s *HomeworkService) DeleteSubmissionFile(ctx context.Context, fileID int64, actor ActorContext) error {
	f, err := s.repo.GetSubmissionFileByID(ctx, fileID)
	if err != nil {
		return err
	}

	sub, err := s.repo.GetSubmissionByID(ctx, f.SubmissionID)
	if err == nil {
		if actor.Role == domain.RoleLearner && sub.StudentID != actor.UserID {
			return fmt.Errorf("%w: you can only delete your own files", domain.ErrForbidden)
		}
	}

	if f.FilePath != "" && s.r2 != nil {
		key := s.r2.ExtractKey(f.FilePath)
		if key != "" {
			s.log.Info("deleting submission file", "key", key)
			if delErr := s.r2.DeleteObject(ctx, key); delErr != nil {
				return fmt.Errorf("failed to delete file from storage: %w", delErr)
			}
		}
	}

	return s.repo.DeleteSubmissionFile(ctx, fileID)
}

func (s *HomeworkService) learnerClassID(ctx context.Context, actor ActorContext) (*int64, error) {
	u, err := s.users.GetByID(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}
	return u.ClassID, nil
}

func (s *HomeworkService) canView(ctx context.Context, hw *domain.Homework, actor ActorContext) error {
	switch actor.Role {
	case domain.RoleAdmin:
		return nil
	case domain.RoleTeacher:
		if hw.TeacherID == actor.UserID {
			return nil
		}
		return fmt.Errorf("%w: teachers can only view their own homework", domain.ErrForbidden)
	case domain.RoleLearner, domain.RoleExternal:
		classID, err := s.learnerClassID(ctx, actor)
		if err != nil {
			return err
		}
		if classID == nil || *classID != hw.ClassID {
			return fmt.Errorf("%w: homework is not for your class", domain.ErrForbidden)
		}
		return nil
	}
	return domain.ErrForbidden
}

func (s *HomeworkService) canEdit(hw *domain.Homework, actor ActorContext) error {
	switch actor.Role {
	case domain.RoleAdmin:
		return nil
	case domain.RoleTeacher:
		if hw.TeacherID == actor.UserID {
			return nil
		}
		return fmt.Errorf("%w: teachers can only modify their own homework", domain.ErrForbidden)
	}
	return fmt.Errorf("%w: insufficient role", domain.ErrForbidden)
}

func rolePtr(r domain.UserRole) *domain.UserRole {
	return &r
}
