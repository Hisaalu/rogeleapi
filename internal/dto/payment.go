package dto

import (
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
)

type InitiatePaymentRequest struct {
	PlanID      int64  `json:"plan_id"`
	Method      string `json:"method"`
	PhoneNumber string `json:"phone_number"`
}

type PaymentResponse struct {
	ID              int64          `json:"id"`
	UserID          *int64         `json:"user_id,omitempty"`
	SubscriptionID  *int64         `json:"subscription_id,omitempty"`
	Amount          *float64       `json:"amount,omitempty"`
	PlanType        *string        `json:"plan_type,omitempty"`
	PaymentMethod   *string        `json:"payment_method,omitempty"`
	TransactionID   *string        `json:"transaction_id,omitempty"`
	PhoneNumber     *string        `json:"phone_number,omitempty"`
	Status          string         `json:"status"`
	PaymentDate     *time.Time     `json:"payment_date,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	GatewayResponse map[string]any `json:"gateway_response,omitempty"`

	UserEmail     *string `json:"user_email,omitempty"`
	UserFirstName *string `json:"user_first_name,omitempty"`
	UserLastName  *string `json:"user_last_name,omitempty"`
}

type InitiatePaymentResponse struct {
	Payment           PaymentResponse `json:"payment"`
	ProviderReference string          `json:"provider_reference"`
	Status            string          `json:"status"`
}

type PaymentHistoryResponse struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Amount        float64   `json:"amount"`
	PaymentType   string    `json:"payment_type"`
	Status        string    `json:"status"`
	PaymentMethod *string   `json:"payment_method,omitempty"`
	TransactionID *string   `json:"transaction_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func PaymentFromDomain(p *domain.Payment) PaymentResponse {
	return PaymentResponse{
		ID:              p.ID,
		UserID:          p.UserID,
		SubscriptionID:  p.SubscriptionID,
		Amount:          p.Amount,
		PlanType:        p.PlanType,
		PaymentMethod:   p.PaymentMethod,
		TransactionID:   p.TransactionID,
		PhoneNumber:     p.PhoneNumber,
		Status:          string(p.Status),
		PaymentDate:     p.PaymentDate,
		CreatedAt:       p.CreatedAt,
		GatewayResponse: p.PaymentGatewayResponse,
	}
}

func PaymentsFromDomain(items []*domain.Payment) []PaymentResponse {
	out := make([]PaymentResponse, 0, len(items))
	for _, p := range items {
		out = append(out, PaymentFromDomain(p))
	}
	return out
}

func HistoryFromDomain(items []*domain.PaymentHistory) []PaymentHistoryResponse {
	out := make([]PaymentHistoryResponse, 0, len(items))
	for _, h := range items {
		out = append(out, PaymentHistoryResponse{
			ID:            h.ID,
			UserID:        h.UserID,
			Amount:        h.Amount,
			PaymentType:   h.PaymentType,
			Status:        h.Status,
			PaymentMethod: h.PaymentMethod,
			TransactionID: h.TransactionID,
			CreatedAt:     h.CreatedAt,
		})
	}
	return out
}
