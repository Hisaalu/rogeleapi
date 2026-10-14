package dto

import (
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
)

type StartTrialResponse struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Days      int       `json:"days"`
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
		StartDate:     s.StartDate,
		EndDate:       s.EndDate,
		PaymentMethod: s.PaymentMethod,
		TransactionID: s.TransactionID,
		Status:        string(s.Status),
		AutoRenew:     s.AutoRenew,
		IsUpgrade:     s.IsUpgrade,
		CreatedAt:     s.CreatedAt,
	}
}

func SubscriptionsFromDomain(items []*domain.Subscription) []SubscriptionResponse {
	out := make([]SubscriptionResponse, 0, len(items))
	for _, s := range items {
		out = append(out, SubscriptionFromDomain(s))
	}
	return out
}

func TrialFromDomain(t *domain.FreeTrial) StartTrialResponse {
	days := int(t.EndDate.Sub(t.StartDate).Hours() / 24)
	return StartTrialResponse{
		StartDate: t.StartDate,
		EndDate:   t.EndDate,
		Days:      days,
	}
}
