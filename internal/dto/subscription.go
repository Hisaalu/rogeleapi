package dto

import (
	"math"
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
)

type StartTrialResponse struct {
	IsInTrial     bool      `json:"is_in_trial"`
	DaysRemaining int       `json:"days_remaining"`
	TotalDays     int       `json:"total_days"`
	EndDate       time.Time `json:"end_date"`
	StartDate     time.Time `json:"start_date"`
}

type TrialResponse struct {
	IsInTrial     bool      `json:"is_in_trial"`
	DaysRemaining int       `json:"days_remaining"`
	TotalDays     int       `json:"total_days"`
	EndDate       time.Time `json:"end_date"`
	StartDate     time.Time `json:"start_date"`
}

type SubscriptionResponse struct {
	ID            int64     `json:"id"`
	UserID        *int64    `json:"user_id,omitempty"`
	PlanType      string    `json:"plan_type"`
	Amount        *float64  `json:"amount,omitempty"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	PaymentMethod *string   `json:"payment_method,omitempty"`
	TransactionID *string   `json:"transaction_id,omitempty"`
	Status        string    `json:"status"`
	AutoRenew     bool      `json:"auto_renew"`
	IsUpgrade     bool      `json:"is_upgrade"`
	CreatedAt     time.Time `json:"created_at"`
}

func SubscriptionFromDomain(s *domain.Subscription) SubscriptionResponse {
	return SubscriptionResponse{
		ID:            s.ID,
		UserID:        s.UserID,
		PlanType:      string(s.PlanType),
		Amount:        s.Amount,
		StartDate:     s.StartDate.UTC(),
		EndDate:       s.EndDate.UTC(),
		PaymentMethod: s.PaymentMethod,
		TransactionID: s.TransactionID,
		Status:        string(s.Status),
		AutoRenew:     s.AutoRenew,
		IsUpgrade:     s.IsUpgrade,
		CreatedAt:     s.CreatedAt.UTC(),
	}
}

func SubscriptionsFromDomain(items []*domain.Subscription) []SubscriptionResponse {
	out := make([]SubscriptionResponse, 0, len(items))
	for _, s := range items {
		out = append(out, SubscriptionFromDomain(s))
	}
	return out
}

func calendarDaysBetween(from, to time.Time) int {
	from = from.UTC()
	to = to.UTC()
	fromDay := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toDay := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	diff := int(math.Ceil(toDay.Sub(fromDay).Hours() / 24))
	if diff < 0 {
		return 0
	}
	return diff
}

func remainingDays(end, now time.Time) int {
	end = end.UTC()
	now = now.UTC()
	if !end.After(now) {
		return 0
	}
	return int(math.Ceil(end.Sub(now).Hours() / 24))
}

func TrialFromDomain(t *domain.FreeTrial) StartTrialResponse {
	now := time.Now().UTC()
	endDate := t.EndDate.UTC()
	startDate := t.StartDate.UTC()

	isInTrial := endDate.After(now)
	totalDays := calendarDaysBetween(startDate, endDate)
	daysRemaining := 0
	if isInTrial {
		daysRemaining = remainingDays(endDate, now)
	}

	return StartTrialResponse{
		IsInTrial:     isInTrial,
		DaysRemaining: daysRemaining,
		TotalDays:     totalDays,
		EndDate:       endDate,
		StartDate:     startDate,
	}
}

func TrialResponseFromDomain(t *domain.FreeTrial) TrialResponse {
	resp := TrialFromDomain(t)
	return TrialResponse{
		IsInTrial:     resp.IsInTrial,
		DaysRemaining: resp.DaysRemaining,
		TotalDays:     resp.TotalDays,
		EndDate:       resp.EndDate,
		StartDate:     resp.StartDate,
	}
}
