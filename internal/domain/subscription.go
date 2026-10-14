package domain

import "time"

type SubscriptionPlanType string
type SubscriptionStatus string

const (
	PlanMonthly SubscriptionPlanType = "monthly"
	PlanTermly  SubscriptionPlanType = "termly"
	PlanYearly  SubscriptionPlanType = "yearly"

	SubscriptionActive    SubscriptionStatus = "active"
	SubscriptionExpired   SubscriptionStatus = "expired"
	SubscriptionCancelled SubscriptionStatus = "cancelled"
	SubscriptionPending   SubscriptionStatus = "pending"
)

type SubscriptionPlan struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Description  *string    `json:"description,omitempty"`
	Price        float64    `json:"price"`
	DurationDays int        `json:"duration_days"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}

type Subscription struct {
	ID                     int64                `json:"id"`
	UserID                 *int64               `json:"user_id,omitempty"`
	PlanType               SubscriptionPlanType `json:"plan_type"`
	Amount                 *float64             `json:"amount,omitempty"`
	StartDate              time.Time            `json:"start_date"`
	EndDate                time.Time            `json:"end_date"`
	PaymentMethod          *string              `json:"payment_method,omitempty"`
	TransactionID          *string              `json:"transaction_id,omitempty"`
	Status                 SubscriptionStatus   `json:"status"`
	AutoRenew              bool                 `json:"auto_renew"`
	IsUpgrade              bool                 `json:"is_upgrade"`
	UpgradedFrom           *string              `json:"upgraded_from,omitempty"`
	UpgradedAt             *time.Time           `json:"upgraded_at,omitempty"`
	UpgradedTo             *string              `json:"upgraded_to,omitempty"`
	OriginalSubscriptionID *int64               `json:"original_subscription_id,omitempty"`
	CreatedAt              time.Time            `json:"created_at"`
}

type FreeTrial struct {
	ID                    int64     `json:"id"`
	UserID                *int64    `json:"user_id,omitempty"`
	StartDate             time.Time `json:"start_date"`
	EndDate               time.Time `json:"end_date"`
	NotificationSent      bool      `json:"notification_sent"`
	ConvertedToSubscriber bool      `json:"converted_to_subscriber"`
	CreatedAt             time.Time `json:"created_at"`
}
