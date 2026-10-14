package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository"
)

type CreateHomeworkInput struct {
	ClassID     int64
	SubjectID   int64
	Title       string
	Description *string
	DueDate     time.Time
}

type UpdateHomeworkInput struct {
	ClassID     *int64
	SubjectID   *int64
	Title       *string
	Description *string
	DueDate     *time.Time
	IsActive    *bool
}

type GradeInput struct {
	Grade    float64
	Feedback *string
}

type HomeworkService struct {
	hw    repository.HomeworkRepository
	users repository.UserRepository
}

func NewHomeworkService(hw repository.HomeworkRepository, users repository.UserRepository) *HomeworkService {
	return &HomeworkService{hw: hw, users: users}
}

func (s *HomeworkService) Create(ctx context.Context, teacherID int64, in CreateHomeworkInput) (*domain.Homework, error) {
	if in.Title == "" {
		return nil, fmt.Errorf("%w: title is required", domain.ErrInvalidInput)
	}
	hw := &domain.Homework{
		TeacherID:   teacherID,
		ClassID:     in.ClassID,
		SubjectID:   in.SubjectID,
		Title:       in.Title,
		Description: in.Description,
		DueDate:     in.DueDate,
		IsActive:    true,
	}
	return s.hw.Create(ctx, hw)
}

func (s *HomeworkService) GetByID(ctx context.Context, id int64, actor ActorContext) (*domain.Homework, error) {
	hw, err := s.hw.GetByID(ctx, id)
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
		tid := actor.UserID
		f.TeacherID = &tid
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
	return s.hw.List(ctx, f)
}

func (s *HomeworkService) Update(ctx context.Context, id int64, actor ActorContext, in UpdateHomeworkInput) (*domain.Homework, error) {
	hw, err := s.hw.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := canEditHomework(hw, actor); err != nil {
		return nil, err
	}
	if in.ClassID != nil {
		hw.ClassID = *in.ClassID
	}
	if in.SubjectID != nil {
		hw.SubjectID = *in.SubjectID
	}
	if in.Title != nil {
		hw.Title = *in.Title
	}
	if in.Description != nil {
		hw.Description = in.Description
	}
	if in.DueDate != nil {
		hw.DueDate = *in.DueDate
	}
	if in.IsActive != nil {
		hw.IsActive = *in.IsActive
	}
	if err := s.hw.Update(ctx, hw); err != nil {
		return nil, err
	}
	return s.hw.GetByID(ctx, id)
}

func (s *HomeworkService) Delete(ctx context.Context, id int64, actor ActorContext) error {
	hw, err := s.hw.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := canEditHomework(hw, actor); err != nil {
		return err
	}
	return s.hw.Delete(ctx, id)
}

func (s *HomeworkService) Submit(ctx context.Context, homeworkID int64, actor ActorContext, textAnswer *string) (*domain.HomeworkSubmission, error) {
	if actor.Role != domain.RoleLearner {
		return nil, fmt.Errorf("%w: only learners can submit homework", domain.ErrForbidden)
	}
	hw, err := s.hw.GetByID(ctx, homeworkID)
	if err != nil {
		return nil, err
	}
	classID, err := s.learnerClassID(ctx, actor)
	if err != nil {
		return nil, err
	}
	if classID == nil || *classID != hw.ClassID {
		return nil, fmt.Errorf("%w: homework is not for your class", domain.ErrForbidden)
	}
	if !hw.IsActive {
		return nil, fmt.Errorf("%w: homework is not active", domain.ErrForbidden)
	}

	if _, err := s.hw.GetSubmission(ctx, homeworkID, actor.UserID); err == nil {
		return nil, fmt.Errorf("%w: you already submitted this homework", domain.ErrConflict)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	now := time.Now()
	status := domain.HomeworkStatusSubmitted
	if now.After(hw.DueDate) {
		status = domain.HomeworkStatusLate
	}

	sub := &domain.HomeworkSubmission{
		HomeworkID:  homeworkID,
		StudentID:   actor.UserID,
		TextAnswer:  textAnswer,
		Status:      status,
		SubmittedAt: &now,
	}
	return s.hw.CreateSubmission(ctx, sub)
}

func (s *HomeworkService) ListSubmissions(ctx context.Context, homeworkID int64, actor ActorContext) ([]*domain.HomeworkSubmission, error) {
	hw, err := s.hw.GetByID(ctx, homeworkID)
	if err != nil {
		return nil, err
	}
	if err := canEditHomework(hw, actor); err != nil {
		return nil, err
	}
	return s.hw.ListSubmissions(ctx, homeworkID)
}

func (s *HomeworkService) MySubmissions(ctx context.Context, actor ActorContext) ([]*domain.HomeworkSubmission, error) {
	if actor.Role != domain.RoleLearner {
		return []*domain.HomeworkSubmission{}, nil
	}
	return s.hw.ListSubmissionsByStudent(ctx, actor.UserID)
}

func (s *HomeworkService) Grade(ctx context.Context, submissionID int64, actor ActorContext, in GradeInput) (*domain.HomeworkSubmission, error) {
	sub, err := s.hw.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	hw, err := s.hw.GetByID(ctx, sub.HomeworkID)
	if err != nil {
		return nil, err
	}
	if err := canEditHomework(hw, actor); err != nil {
		return nil, err
	}
	now := time.Now()
	sub.Grade = &in.Grade
	sub.Feedback = in.Feedback
	sub.Status = domain.HomeworkStatusGraded
	sub.GradedAt = &now
	if err := s.hw.UpdateSubmission(ctx, sub); err != nil {
		return nil, err
	}
	return s.hw.GetSubmissionByID(ctx, submissionID)
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
		if !hw.IsActive {
			return fmt.Errorf("%w: homework is not active", domain.ErrForbidden)
		}
		return nil
	}
	return domain.ErrForbidden
}

func canEditHomework(hw *domain.Homework, actor ActorContext) error {
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
