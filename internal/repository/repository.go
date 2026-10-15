package repository

import (
	"context"
	"time"

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
	UpdatePassword(ctx context.Context, id int64, hash string) error
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

type HomeworkFilter struct {
	TeacherID *int64
	ClassID   *int64
	SubjectID *int64
	IsActive  *bool
	Search    string
	Limit     int
	Offset    int
}

type HomeworkRepository interface {
	Create(ctx context.Context, hw *domain.Homework) (*domain.Homework, error)
	GetByID(ctx context.Context, id int64) (*domain.Homework, error)
	Update(ctx context.Context, hw *domain.Homework) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, f HomeworkFilter) ([]*domain.Homework, error)

	ListAttachments(ctx context.Context, homeworkID int64) ([]*domain.HomeworkAttachment, error)
	CreateAttachment(ctx context.Context, a *domain.HomeworkAttachment) (*domain.HomeworkAttachment, error)
	DeleteAttachment(ctx context.Context, id int64) error

	CreateSubmission(ctx context.Context, s *domain.HomeworkSubmission) (*domain.HomeworkSubmission, error)
	GetSubmissionByID(ctx context.Context, id int64) (*domain.HomeworkSubmission, error)
	GetSubmission(ctx context.Context, homeworkID, studentID int64) (*domain.HomeworkSubmission, error)
	ListSubmissions(ctx context.Context, homeworkID int64) ([]*domain.HomeworkSubmission, error)
	ListSubmissionsByStudent(ctx context.Context, studentID int64) ([]*domain.HomeworkSubmission, error)
	UpdateSubmission(ctx context.Context, s *domain.HomeworkSubmission) error
	DeleteSubmission(ctx context.Context, id int64) error

	ListSubmissionFiles(ctx context.Context, submissionID int64) ([]*domain.HomeworkSubmissionFile, error)
	CreateSubmissionFile(ctx context.Context, f *domain.HomeworkSubmissionFile) (*domain.HomeworkSubmissionFile, error)
	DeleteSubmissionFile(ctx context.Context, id int64) error
}

type QuizFilter struct {
	TeacherID *int64
	ClassID   *int64
	SubjectID *int64
	Published *bool
	Limit     int
	Offset    int
}

type QuizRepository interface {
	Create(ctx context.Context, q *domain.Quiz) (*domain.Quiz, error)
	GetByID(ctx context.Context, id int64) (*domain.Quiz, error)
	Update(ctx context.Context, q *domain.Quiz) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, f QuizFilter) ([]*domain.Quiz, error)

	ListQuestions(ctx context.Context, quizID int64) ([]*domain.QuizQuestion, error)
	CreateQuestion(ctx context.Context, q *domain.QuizQuestion) (*domain.QuizQuestion, error)
	DeleteQuestion(ctx context.Context, id int64) error

	CreateAttempt(ctx context.Context, a *domain.QuizAttempt) (*domain.QuizAttempt, error)
	GetAttempt(ctx context.Context, id int64) (*domain.QuizAttempt, error)
	GetInProgressAttempt(ctx context.Context, quizID, userID int64) (*domain.QuizAttempt, error)
	UpdateAttempt(ctx context.Context, a *domain.QuizAttempt) error
	CountAttempts(ctx context.Context, quizID, userID int64) (int, error)
	ListAttemptsByUser(ctx context.Context, userID int64) ([]*domain.QuizAttempt, error)

	SaveAnswer(ctx context.Context, a *domain.QuizAttemptAnswer) error
	ListAnswers(ctx context.Context, attemptID int64) ([]*domain.QuizAttemptAnswer, error)
	ListAttemptsByQuiz(ctx context.Context, quizID int64) ([]*domain.QuizAttempt, error)
	ListAttemptsWithStudents(ctx context.Context, quizID int64) ([]*domain.QuizAttempt, error)

	DeleteAllAttempts(ctx context.Context, quizID int64) error
}

type SubscriptionFilter struct {
	UserID *int64
	Status *domain.SubscriptionStatus
	Limit  int
	Offset int
}

type SubscriptionRepository interface {
	Create(ctx context.Context, s *domain.Subscription) (*domain.Subscription, error)
	GetByID(ctx context.Context, id int64) (*domain.Subscription, error)
	GetActiveByUser(ctx context.Context, userID int64) (*domain.Subscription, error)
	List(ctx context.Context, f SubscriptionFilter) ([]*domain.Subscription, error)
	ListWithUsers(ctx context.Context, f SubscriptionFilter) ([]*domain.Subscription, error)
	Update(ctx context.Context, s *domain.Subscription) error
	ExpireOverdue(ctx context.Context) (int, error)
}

type FreeTrialRepository interface {
	Create(ctx context.Context, t *domain.FreeTrial) (*domain.FreeTrial, error)
	GetByUser(ctx context.Context, userID int64) (*domain.FreeTrial, error)
	ExistsForEmail(ctx context.Context, email string) (bool, error)
	Update(ctx context.Context, t *domain.FreeTrial) error
}

type PaymentFilter struct {
	UserID *int64
	Status *domain.PaymentStatus
	Limit  int
	Offset int
}

type PaymentRepository interface {
	Create(ctx context.Context, p *domain.Payment) (*domain.Payment, error)
	GetByID(ctx context.Context, id int64) (*domain.Payment, error)
	GetByTransactionID(ctx context.Context, txnID string) (*domain.Payment, error)
	List(ctx context.Context, f PaymentFilter) ([]*domain.Payment, error)
	ListWithUsers(ctx context.Context, f PaymentFilter) ([]*domain.Payment, error)
	Update(ctx context.Context, p *domain.Payment) error

	CreateHistory(ctx context.Context, h *domain.PaymentHistory) (*domain.PaymentHistory, error)
	ListHistoryByUser(ctx context.Context, userID int64) ([]*domain.PaymentHistory, error)
	ListHistoryBySubscription(ctx context.Context, subscriptionID int64) ([]*domain.PaymentHistory, error)
}

type PasswordResetRepository interface {
	Create(ctx context.Context, pr *domain.PasswordReset) (*domain.PasswordReset, error)
	GetByToken(ctx context.Context, token string) (*domain.PasswordReset, error)
	MarkUsed(ctx context.Context, id int64) error
	CountRecent(ctx context.Context, userID int64, within time.Duration) (int, error)
}

type NotificationRepository interface {
	Create(ctx context.Context, n *domain.Notification) (*domain.Notification, error)
	CreateBulk(ctx context.Context, userIDs []int64, n *domain.Notification) (int, error)
	ListForUser(ctx context.Context, userID int64, limit, offset int) ([]*domain.Notification, error)
	CountUnreadForUser(ctx context.Context, userID int64) (int, error)
	MarkRead(ctx context.Context, id, userID int64) error
	MarkAllRead(ctx context.Context, userID int64) (int, error)
	AdminList(ctx context.Context, f AdminListFilter) ([]*domain.Notification, error)
	CountBroadcasts(ctx context.Context) (int, error)
	ListBroadcasts(ctx context.Context, limit, offset int) ([]*domain.Notification, error)
}

type AdminListFilter struct {
	UserID *int64
	Type   *string
	Limit  int
	Offset int
}

type Repositories struct {
	User          UserRepository
	Lesson        LessonRepository
	Homework      HomeworkRepository
	Quiz          QuizRepository
	Subscription  SubscriptionRepository
	FreeTrial     FreeTrialRepository
	Payment       PaymentRepository
	PasswordReset PasswordResetRepository
	Notification  NotificationRepository
}
