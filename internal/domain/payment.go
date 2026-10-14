package domain

import "time"

type PaymentStatus string
type PaymentMethod string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentCompleted PaymentStatus = "completed"
	PaymentFailed    PaymentStatus = "failed"
	PaymentRefunded  PaymentStatus = "refunded"

	MethodMTN    PaymentMethod = "mtn"
	MethodAirtel PaymentMethod = "airtel"
)

type Payment struct {
	ID                     int64          `json:"id"`
	UserID                 *int64         `json:"user_id,omitempty"`
	SubscriptionID         *int64         `json:"subscription_id,omitempty"`
	Amount                 *float64       `json:"amount,omitempty"`
	PlanType               *string        `json:"plan_type,omitempty"`
	PaymentMethod          *string        `json:"payment_method,omitempty"`
	TransactionID          *string        `json:"transaction_id,omitempty"`
	PaymentGatewayResponse map[string]any `json:"payment_gateway_response,omitempty"`
	PhoneNumber            *string        `json:"phone_number,omitempty"`
	Status                 PaymentStatus  `json:"status"`
	PaymentDate            *time.Time     `json:"payment_date,omitempty"`
	CreatedAt              time.Time      `json:"created_at"`
}

type PaymentHistory struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	SubscriptionID *int64    `json:"subscription_id,omitempty"`
	Amount         float64   `json:"amount"`
	PaymentType    string    `json:"payment_type"`
	FromPlan       *string   `json:"from_plan,omitempty"`
	ToPlan         *string   `json:"to_plan,omitempty"`
	Status         string    `json:"status"`
	PaymentMethod  *string   `json:"payment_method,omitempty"`
	TransactionID  *string   `json:"transaction_id,omitempty"`
	PaymentData    *string   `json:"payment_data,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}
