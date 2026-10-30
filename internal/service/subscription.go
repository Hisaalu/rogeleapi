package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository"
)

type SubscribeInput struct {
	PlanID        int64
	PaymentMethod string
	PhoneNumber   string
}

type SubscriptionService struct {
	subs     repository.SubscriptionRepository
	plans    planLookup
	trials   repository.FreeTrialRepository
	users    repository.UserRepository
	payments repository.PaymentRepository
}

type planLookup interface {
	GetPlan(ctx context.Context, id int64) (*domain.SubscriptionPlan, error)
	GetFreePlan(ctx context.Context) (*domain.SubscriptionPlan, error)
}

func NewSubscriptionService(
	subs repository.SubscriptionRepository,
	plans planLookup,
	trials repository.FreeTrialRepository,
	users repository.UserRepository,
	payments repository.PaymentRepository,
) *SubscriptionService {
	return &SubscriptionService{
		subs:     subs,
		plans:    plans,
		trials:   trials,
		users:    users,
		payments: payments,
	}
}

func (s *SubscriptionService) GetMyActive(ctx context.Context, actor ActorContext) (*domain.Subscription, error) {
	sub, err := s.subs.GetActiveByUser(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *SubscriptionService) ListForUser(ctx context.Context, actor ActorContext) ([]*domain.Subscription, error) {
	uid := actor.UserID
	return s.subs.List(ctx, repository.SubscriptionFilter{UserID: &uid})
}

func (s *SubscriptionService) ListAll(ctx context.Context, actor ActorContext, f repository.SubscriptionFilter) ([]*domain.Subscription, error) {
	if actor.Role != domain.RoleAdmin {
		return nil, fmt.Errorf("%w: admin only", domain.ErrForbidden)
	}
	return s.subs.ListWithUsers(ctx, f)
}

func (s *SubscriptionService) StartTrial(ctx context.Context, actor ActorContext) (*domain.FreeTrial, error) {
	durationDays := 14
	freePlan, err := s.plans.GetFreePlan(ctx)
	if err == nil && freePlan != nil && freePlan.DurationDays > 0 {
		durationDays = freePlan.DurationDays
	}

	u, err := s.users.GetByID(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}

	exists, err := s.trials.ExistsForEmail(ctx, u.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: a trial has already been used for this email", domain.ErrConflict)
	}

	if _, err := s.trials.GetByUser(ctx, actor.UserID); err == nil {
		return nil, fmt.Errorf("%w: you already have a trial", domain.ErrConflict)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	if _, err := s.subs.GetActiveByUser(ctx, actor.UserID); err == nil {
		return nil, fmt.Errorf("%w: you already have an active subscription", domain.ErrConflict)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	now := time.Now().UTC()
	trial := &domain.FreeTrial{
		UserID:    &actor.UserID,
		StartDate: now,
		EndDate:   now.AddDate(0, 0, durationDays),
	}
	return s.trials.Create(ctx, trial)
}

func (s *SubscriptionService) GetMyTrial(ctx context.Context, actor ActorContext) (*domain.FreeTrial, error) {
	trial, err := s.trials.GetByUser(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}

	durationDays := 14
	freePlan, err := s.plans.GetFreePlan(ctx)
	if err == nil && freePlan != nil && freePlan.DurationDays > 0 {
		durationDays = freePlan.DurationDays
	}

	trial.EndDate = trial.StartDate.AddDate(0, 0, durationDays)

	return trial, nil
}

func (s *SubscriptionService) Cancel(ctx context.Context, id int64, actor ActorContext) (*domain.Subscription, error) {
	sub, err := s.subs.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sub.UserID == nil || (*sub.UserID != actor.UserID && actor.Role != domain.RoleAdmin) {
		return nil, domain.ErrForbidden
	}
	if sub.Status != domain.SubscriptionActive && sub.Status != domain.SubscriptionPending {
		return nil, fmt.Errorf("%w: subscription cannot be cancelled in its current state", domain.ErrInvalidInput)
	}
	sub.Status = domain.SubscriptionCancelled
	sub.AutoRenew = false
	if err := s.subs.Update(ctx, sub); err != nil {
		return nil, err
	}
	return s.subs.GetByID(ctx, id)
}

func (s *SubscriptionService) ExpireOverdue(ctx context.Context) (int, error) {
	return s.subs.ExpireOverdue(ctx)
}

func (s *SubscriptionService) ActivateFromPayment(
	ctx context.Context,
	userID int64,
	plan *domain.SubscriptionPlan,
	paymentID int64,
	paymentMethod string,
	transactionID string,
) (*domain.Subscription, error) {
	now := time.Now().UTC()

	planTypeStr := strings.ToLower(plan.Name)
	if planTypeStr == "" {
		planTypeStr = "monthly"
		if plan.DurationDays >= 365 {
			planTypeStr = "yearly"
		} else if plan.DurationDays >= 100 {
			planTypeStr = "termly"
		}
	}

	if existing, err := s.subs.GetActiveByUser(ctx, userID); err == nil {
		existing.PlanType = domain.SubscriptionPlanType(planTypeStr)
		existing.Amount = &plan.Price
		existing.StartDate = now
		existing.EndDate = now.AddDate(0, 0, plan.DurationDays)
		existing.TransactionID = &transactionID
		existing.PaymentMethod = &paymentMethod
		if err := s.subs.Update(ctx, existing); err != nil {
			return nil, err
		}
		existing.Status = domain.SubscriptionActive
		return existing, nil

	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	uid := userID
	sub := &domain.Subscription{
		UserID:        &uid,
		PlanType:      domain.SubscriptionPlanType(planTypeStr),
		Amount:        &plan.Price,
		StartDate:     now,
		EndDate:       now.AddDate(0, 0, plan.DurationDays),
		PaymentMethod: &paymentMethod,
		TransactionID: &transactionID,
		Status:        domain.SubscriptionActive,
	}
	return s.subs.Create(ctx, sub)
}

func (s *SubscriptionService) MarkConverted(ctx context.Context, userID int64) error {
	t, err := s.trials.GetByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}
	if t.ConvertedToSubscriber {
		return nil
	}
	t.ConvertedToSubscriber = true
	return s.trials.Update(ctx, t)
}
