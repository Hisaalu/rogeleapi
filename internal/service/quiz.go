package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository"
)

type CreateQuizInput struct {
	Title        string
	Description  *string
	SubjectID    *int64
	ClassID      *int64
	TimeLimit    *int
	PassingScore int
	MaxAttempts  int
	StartDate    *time.Time
	EndDate      *time.Time
}

type UpdateQuizInput struct {
	Title        *string
	Description  *string
	SubjectID    *int64
	ClassID      *int64
	TimeLimit    *int
	PassingScore *int
	MaxAttempts  *int
	IsPublished  *bool
	StartDate    *time.Time
	EndDate      *time.Time
}

type CreateQuestionInput struct {
	Question      string
	OptionA       *string
	OptionB       *string
	OptionC       *string
	OptionD       *string
	CorrectAnswer *string
	Points        int
	Explanation   *string
}

type AnswerInput struct {
	QuestionID     int64
	SelectedAnswer *int
}

type QuizService struct {
	quizzes repository.QuizRepository
	users   repository.UserRepository
}

func NewQuizService(quizzes repository.QuizRepository, users repository.UserRepository) *QuizService {
	return &QuizService{quizzes: quizzes, users: users}
}

func (s *QuizService) Create(ctx context.Context, teacherID int64, in CreateQuizInput) (*domain.Quiz, error) {
	if in.Title == "" {
		return nil, fmt.Errorf("%w: title is required", domain.ErrInvalidInput)
	}
	if in.PassingScore < 0 || in.PassingScore > 100 {
		in.PassingScore = 50
	}
	if in.MaxAttempts < 1 {
		in.MaxAttempts = 3
	}
	q := &domain.Quiz{
		Title:        in.Title,
		Description:  in.Description,
		SubjectID:    in.SubjectID,
		ClassID:      in.ClassID,
		TeacherID:    &teacherID,
		TimeLimit:    in.TimeLimit,
		PassingScore: in.PassingScore,
		MaxAttempts:  in.MaxAttempts,
		StartDate:    in.StartDate,
		EndDate:      in.EndDate,
		Status:       "draft",
	}
	return s.quizzes.Create(ctx, q)
}

func (s *QuizService) GetByID(ctx context.Context, id int64, actor ActorContext) (*domain.Quiz, error) {
	q, err := s.quizzes.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.canViewQuiz(ctx, q, actor); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *QuizService) List(ctx context.Context, actor ActorContext, f repository.QuizFilter) ([]*domain.Quiz, error) {
	switch actor.Role {
	case domain.RoleAdmin:
	case domain.RoleTeacher:
		tid := actor.UserID
		f.TeacherID = &tid
	case domain.RoleLearner, domain.RoleExternal:
		pub := true
		f.Published = &pub
		classID, err := s.learnerClassID(ctx, actor)
		if err != nil {
			return nil, err
		}
		if classID == nil {
			return []*domain.Quiz{}, nil
		}
		f.ClassID = classID
	default:
		return nil, domain.ErrForbidden
	}
	return s.quizzes.List(ctx, f)
}

func (s *QuizService) Update(ctx context.Context, id int64, actor ActorContext, in UpdateQuizInput) (*domain.Quiz, error) {
	q, err := s.quizzes.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := canEditQuiz(q, actor); err != nil {
		return nil, err
	}
	if in.Title != nil {
		q.Title = *in.Title
	}
	if in.Description != nil {
		q.Description = in.Description
	}
	if in.SubjectID != nil {
		q.SubjectID = in.SubjectID
	}
	if in.ClassID != nil {
		q.ClassID = in.ClassID
	}
	if in.TimeLimit != nil {
		q.TimeLimit = in.TimeLimit
	}
	if in.PassingScore != nil {
		q.PassingScore = *in.PassingScore
	}
	if in.MaxAttempts != nil {
		q.MaxAttempts = *in.MaxAttempts
	}
	if in.StartDate != nil {
		q.StartDate = in.StartDate
	}
	if in.EndDate != nil {
		q.EndDate = in.EndDate
	}
	if in.IsPublished != nil {
		q.IsPublished = *in.IsPublished
		if *in.IsPublished {
			q.Status = "published"
		} else {
			q.Status = "draft"
		}
	}
	if err := s.quizzes.Update(ctx, q); err != nil {
		return nil, err
	}
	return s.quizzes.GetByID(ctx, id)
}

func (s *QuizService) Delete(ctx context.Context, id int64, actor ActorContext) error {
	q, err := s.quizzes.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := canEditQuiz(q, actor); err != nil {
		return err
	}
	return s.quizzes.Delete(ctx, id)
}

func (s *QuizService) Approve(ctx context.Context, id int64, actor ActorContext, approved bool) (*domain.Quiz, error) {
	if actor.Role != domain.RoleAdmin {
		return nil, fmt.Errorf("%w: only admins can approve", domain.ErrForbidden)
	}
	q, err := s.quizzes.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	q.IsApproved = approved
	if approved {
		q.ApprovedAt = &now
		q.ApprovedBy = &actor.UserID
	} else {
		q.ApprovedAt = nil
		q.ApprovedBy = nil
	}
	if err := s.quizzes.Update(ctx, q); err != nil {
		return nil, err
	}
	return s.quizzes.GetByID(ctx, id)
}

func (s *QuizService) ListQuestions(ctx context.Context, quizID int64, actor ActorContext) ([]*domain.QuizQuestion, error) {
	q, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if err := s.canViewQuiz(ctx, q, actor); err != nil {
		return nil, err
	}
	questions, err := s.quizzes.ListQuestions(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if actor.Role == domain.RoleLearner {
		for _, question := range questions {
			question.CorrectAnswer = nil
			question.Explanation = nil
		}
	}
	return questions, nil
}

func (s *QuizService) AddQuestion(ctx context.Context, quizID int64, actor ActorContext, in CreateQuestionInput) (*domain.QuizQuestion, error) {
	q, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if err := canEditQuiz(q, actor); err != nil {
		return nil, err
	}
	if in.Question == "" {
		return nil, fmt.Errorf("%w: question text is required", domain.ErrInvalidInput)
	}
	if in.Points < 1 {
		in.Points = 1
	}
	qq := &domain.QuizQuestion{
		QuizID:        &quizID,
		Question:      in.Question,
		OptionA:       in.OptionA,
		OptionB:       in.OptionB,
		OptionC:       in.OptionC,
		OptionD:       in.OptionD,
		CorrectAnswer: in.CorrectAnswer,
		Points:        in.Points,
		Explanation:   in.Explanation,
	}
	return s.quizzes.CreateQuestion(ctx, qq)
}

func (s *QuizService) DeleteQuestion(ctx context.Context, quizID, questionID int64, actor ActorContext) error {
	q, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return err
	}
	if err := canEditQuiz(q, actor); err != nil {
		return err
	}
	return s.quizzes.DeleteQuestion(ctx, questionID)
}

func (s *QuizService) StartAttempt(ctx context.Context, quizID int64, actor ActorContext) (*domain.QuizAttempt, error) {
	if actor.Role != domain.RoleLearner {
		return nil, fmt.Errorf("%w: only learners can attempt quizzes", domain.ErrForbidden)
	}
	q, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if !q.IsPublished {
		return nil, fmt.Errorf("%w: quiz is not published", domain.ErrForbidden)
	}
	classID, err := s.learnerClassID(ctx, actor)
	if err != nil {
		return nil, err
	}
	if classID == nil || q.ClassID == nil || *classID != *q.ClassID {
		return nil, fmt.Errorf("%w: quiz is not for your class", domain.ErrForbidden)
	}
	now := time.Now()
	if q.StartDate != nil && now.Before(*q.StartDate) {
		return nil, fmt.Errorf("%w: quiz has not started yet", domain.ErrForbidden)
	}
	if q.EndDate != nil && now.After(*q.EndDate) {
		return nil, fmt.Errorf("%w: quiz has ended", domain.ErrForbidden)
	}

	if existing, err := s.quizzes.GetInProgressAttempt(ctx, quizID, actor.UserID); err == nil {
		if q.TimeLimit != nil && *q.TimeLimit > 0 {
			deadline := existing.StartedAt.Add(time.Duration(*q.TimeLimit) * time.Minute)
			if time.Now().After(deadline) {
				existing.Status = domain.QuizAttemptExpired
				completed := time.Now()
				existing.CompletedAt = &completed
				_ = s.quizzes.UpdateAttempt(ctx, existing)
			} else {
				return existing, nil
			}
		} else {
			return existing, nil
		}
	} else if err != domain.ErrNotFound {
		return nil, err
	}

	if q.MaxAttempts > 0 {
		completed, err := s.quizzes.CountAttempts(ctx, quizID, actor.UserID)
		if err != nil {
			return nil, err
		}
		if completed >= q.MaxAttempts {
			return nil, fmt.Errorf("%w: max attempts reached", domain.ErrForbidden)
		}
	}

	attempt := &domain.QuizAttempt{
		QuizID:    &quizID,
		UserID:    &actor.UserID,
		Status:    domain.QuizAttemptInProgress,
		StartedAt: time.Now(),
	}
	return s.quizzes.CreateAttempt(ctx, attempt)
}

func (s *QuizService) GetAttempt(ctx context.Context, attemptID int64, actor ActorContext) (*domain.QuizAttempt, error) {
	a, err := s.quizzes.GetAttempt(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if a.UserID == nil || *a.UserID != actor.UserID {
		if actor.Role != domain.RoleAdmin {
			return nil, domain.ErrForbidden
		}
	}
	return a, nil
}

func (s *QuizService) SubmitAttempt(ctx context.Context, attemptID int64, actor ActorContext, answers []AnswerInput) (*domain.QuizAttempt, error) {
	a, err := s.quizzes.GetAttempt(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if a.UserID == nil || *a.UserID != actor.UserID {
		return nil, domain.ErrForbidden
	}
	if a.Status != domain.QuizAttemptInProgress {
		return nil, fmt.Errorf("%w: attempt is not in progress", domain.ErrInvalidInput)
	}
	if a.QuizID == nil {
		return nil, fmt.Errorf("%w: attempt has no quiz", domain.ErrInternal)
	}
	if _, err := s.quizzes.GetByID(ctx, *a.QuizID); err != nil {
		return nil, err
	}

	questions, err := s.quizzes.ListQuestions(ctx, *a.QuizID)
	if err != nil {
		return nil, err
	}
	questionMap := make(map[int64]*domain.QuizQuestion, len(questions))
	for _, question := range questions {
		questionMap[question.ID] = question
	}

	correct := 0
	for _, ans := range answers {
		question, ok := questionMap[ans.QuestionID]
		if !ok {
			continue
		}
		isCorrect := false
		if ans.SelectedAnswer != nil && question.CorrectAnswer != nil {
			letter := answerLetter(*ans.SelectedAnswer)
			isCorrect = letter != "" && letter == *question.CorrectAnswer
		}
		if err := s.quizzes.SaveAnswer(ctx, &domain.QuizAttemptAnswer{
			AttemptID:      attemptID,
			QuestionID:     ans.QuestionID,
			SelectedAnswer: ans.SelectedAnswer,
			IsCorrect:      isCorrect,
		}); err != nil {
			return nil, err
		}
		if isCorrect {
			correct++
		}
	}

	totalQuestions := len(questions)
	score := 0
	if totalQuestions > 0 {
		score = correct * 100 / totalQuestions
	}
	timeTaken := int(time.Since(a.StartedAt.UTC()).Seconds())
	if timeTaken < 0 {
		timeTaken = 0
	}
	completed := time.Now()

	a.Score = &score
	a.TotalQuestions = &totalQuestions
	a.CorrectAnswers = &correct
	a.TimeTaken = &timeTaken
	a.Status = domain.QuizAttemptCompleted
	a.CompletedAt = &completed

	if err := s.quizzes.UpdateAttempt(ctx, a); err != nil {
		return nil, err
	}
	return s.quizzes.GetAttempt(ctx, attemptID)
}

func (s *QuizService) ListMyAttempts(ctx context.Context, actor ActorContext) ([]*domain.QuizAttempt, error) {
	if actor.Role != domain.RoleLearner {
		return []*domain.QuizAttempt{}, nil
	}
	return s.quizzes.ListAttemptsByUser(ctx, actor.UserID)
}

func (s *QuizService) learnerClassID(ctx context.Context, actor ActorContext) (*int64, error) {
	u, err := s.users.GetByID(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}
	return u.ClassID, nil
}

func (s *QuizService) canViewQuiz(ctx context.Context, q *domain.Quiz, actor ActorContext) error {
	switch actor.Role {
	case domain.RoleAdmin:
		return nil
	case domain.RoleTeacher:
		if q.TeacherID != nil && *q.TeacherID == actor.UserID {
			return nil
		}
		if q.IsPublished {
			return nil
		}
		return fmt.Errorf("%w: teachers can only view their own unpublished quizzes", domain.ErrForbidden)
	case domain.RoleLearner, domain.RoleExternal:
		if !q.IsPublished {
			return fmt.Errorf("%w: quiz is not published", domain.ErrForbidden)
		}
		classID, err := s.learnerClassID(ctx, actor)
		if err != nil {
			return err
		}
		if classID == nil || q.ClassID == nil || *classID != *q.ClassID {
			return fmt.Errorf("%w: quiz is not for your class", domain.ErrForbidden)
		}
		return nil
	}
	return domain.ErrForbidden
}

func canEditQuiz(q *domain.Quiz, actor ActorContext) error {
	switch actor.Role {
	case domain.RoleAdmin:
		return nil
	case domain.RoleTeacher:
		if q.TeacherID != nil && *q.TeacherID == actor.UserID {
			return nil
		}
		return fmt.Errorf("%w: teachers can only modify their own quizzes", domain.ErrForbidden)
	}
	return fmt.Errorf("%w: insufficient role", domain.ErrForbidden)
}

func answerLetter(n int) string {
	switch n {
	case 0:
		return "A"
	case 1:
		return "B"
	case 2:
		return "C"
	case 3:
		return "D"
	}
	return ""
}
