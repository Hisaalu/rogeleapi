package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository"
)

type NotificationService struct {
	repo repository.NotificationRepository
	log  *slog.Logger
}

func NewNotificationService(repo repository.NotificationRepository, log *slog.Logger) *NotificationService {
	return &NotificationService{repo: repo, log: log}
}

// ListForUser returns notifications visible to the caller.
func (s *NotificationService) ListForUser(ctx context.Context, actor ActorContext, limit, offset int) ([]*domain.Notification, error) {
	return s.repo.ListForUser(ctx, actor.UserID, limit, offset)
}

func (s *NotificationService) CountUnread(ctx context.Context, actor ActorContext) (int, error) {
	return s.repo.CountUnreadForUser(ctx, actor.UserID)
}

func (s *NotificationService) MarkRead(ctx context.Context, id int64, actor ActorContext) error {
	return s.repo.MarkRead(ctx, id, actor.UserID)
}

func (s *NotificationService) MarkAllRead(ctx context.Context, actor ActorContext) (int, error) {
	return s.repo.MarkAllRead(ctx, actor.UserID)
}

// Broadcast sends a notification to all active users (admin only).
func (s *NotificationService) Broadcast(ctx context.Context, actor ActorContext, title, message string, link *string, notifType string) (int, error) {
	if actor.Role != domain.RoleAdmin {
		return 0, fmt.Errorf("%w: admin only", domain.ErrForbidden)
	}
	if title == "" || message == "" {
		return 0, fmt.Errorf("%w: title and message are required", domain.ErrInvalidInput)
	}
	t := domain.NotificationType(notifType)
	if t == "" {
		t = domain.NotificationTypeAnnouncement
	}

	// Broadcast row: user_id = NULL. Every user sees it via the OR clause.
	n := &domain.Notification{
		UserID:  nil,
		Type:    t,
		Title:   title,
		Message: message,
		Link:    link,
	}
	if _, err := s.repo.Create(ctx, n); err != nil {
		return 0, err
	}
	return 1, nil
}

// NotifyUsers inserts one notification row per user (used by event emitters).
func (s *NotificationService) NotifyUsers(
	ctx context.Context,
	userIDs []int64,
	notifType domain.NotificationType,
	title string,
	message string,
	link *string,
) error {
	if len(userIDs) == 0 {
		return nil
	}
	n := &domain.Notification{
		Type:    notifType,
		Title:   title,
		Message: message,
		Link:    link,
	}
	_, err := s.repo.CreateBulk(ctx, userIDs, n)
	return err
}

// NotifyUser is a convenience wrapper for a single recipient.
func (s *NotificationService) NotifyUser(
	ctx context.Context,
	userID int64,
	notifType domain.NotificationType,
	title string,
	message string,
	link *string,
) error {
	return s.NotifyUsers(ctx, []int64{userID}, notifType, title, message, link)
}
