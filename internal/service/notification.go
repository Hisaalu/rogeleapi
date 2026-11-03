package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository"
)

type NotificationService struct {
	repo  repository.NotificationRepository
	users repository.UserRepository
	log   *slog.Logger
}

func NewNotificationService(
	repo repository.NotificationRepository,
	users repository.UserRepository,
	log *slog.Logger,
) *NotificationService {
	return &NotificationService{repo: repo, users: users, log: log}
}

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

type BroadcastAudience struct {
	Kind     string
	Value    string
	ClassID  *int64
	UserIDs  []int64
}

func (s *NotificationService) Broadcast(
	ctx context.Context,
	actor ActorContext,
	title, message string,
	link *string,
	notifType string,
	audience BroadcastAudience,
) (int, string, error) {
	if actor.Role != domain.RoleAdmin {
		return 0, "", fmt.Errorf("%w: admin only", domain.ErrForbidden)
	}
	if title == "" || message == "" {
		return 0, "", fmt.Errorf("%w: title and message are required", domain.ErrInvalidInput)
	}

	t := domain.NotificationType(notifType)
	if t == "" {
		t = domain.NotificationTypeAnnouncement
	}

	userIDs, label, err := s.resolveAudience(ctx, audience)
	if err != nil {
		return 0, "", err
	}

	if userIDs == nil {
		n := &domain.Notification{
			UserID:  nil,
			Type:    t,
			Title:   title,
			Message: message,
			Link:    link,
		}
		if _, err := s.repo.Create(ctx, n); err != nil {
			return 0, "", err
		}
		return 1, label, nil
	}

	if len(userIDs) == 0 {
		return 0, label, nil
	}
	n := &domain.Notification{
		Type:    t,
		Title:   title,
		Message: message,
		Link:    link,
	}
	count, err := s.repo.CreateBulk(ctx, userIDs, n)
	if err != nil {
		return 0, "", err
	}
	return count, label, nil
}

func (s *NotificationService) resolveAudience(ctx context.Context, a BroadcastAudience) ([]int64, string, error) {
	switch a.Kind {
	case "", "all":
		return nil, "all users", nil

	case "role":
		role := domain.UserRole(a.Value)
		if !role.IsValid() {
			return nil, "", fmt.Errorf("%w: invalid audience role", domain.ErrInvalidInput)
		}
		users, err := s.users.List(ctx, repository.UserFilter{
			Role:  &role,
			Limit: 10000,
		})
		if err != nil {
			return nil, "", err
		}
		ids := make([]int64, 0, len(users))
		for _, u := range users {
			if u.IsActive && !u.IsSuspended {
				ids = append(ids, u.ID)
			}
		}
		return ids, fmt.Sprintf("all %ss", a.Value), nil

	case "class":
		if a.ClassID == nil {
			return nil, "", fmt.Errorf("%w: class_id is required for class audience", domain.ErrInvalidInput)
		}
		users, err := s.users.List(ctx, repository.UserFilter{
			ClassID: a.ClassID,
			Limit:   10000,
		})
		if err != nil {
			return nil, "", err
		}
		ids := make([]int64, 0, len(users))
		for _, u := range users {
			if u.IsActive && !u.IsSuspended &&
				(u.Role == domain.RoleLearner || u.Role == domain.RoleExternal) {
				ids = append(ids, u.ID)
			}
		}
		return ids, fmt.Sprintf("class #%d (%d recipients)", *a.ClassID, len(ids)), nil

	case "users":
		seen := make(map[int64]struct{}, len(a.UserIDs))
		ids := make([]int64, 0, len(a.UserIDs))
		for _, id := range a.UserIDs {
			if id <= 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
		return ids, fmt.Sprintf("%d selected users", len(ids)), nil

	default:
		return nil, "", fmt.Errorf("%w: unknown audience kind %q", domain.ErrInvalidInput, a.Kind)
	}
}

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

func (s *NotificationService) ListBroadcasts(ctx context.Context, actor ActorContext, limit, offset int) ([]*domain.Notification, error) {
	if actor.Role != domain.RoleAdmin {
		return nil, fmt.Errorf("%w: admin only", domain.ErrForbidden)
	}
	return s.repo.ListBroadcasts(ctx, limit, offset)
}

func (s *NotificationService) CountBroadcasts(ctx context.Context, actor ActorContext) (int, error) {
	if actor.Role != domain.RoleAdmin {
		return 0, fmt.Errorf("%w: admin only", domain.ErrForbidden)
	}
	return s.repo.CountBroadcasts(ctx)
}