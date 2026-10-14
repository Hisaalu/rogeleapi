package payments

import (
	"context"
	"errors"
)

var (
	ErrProviderNotConfigured = errors.New("payment provider is not configured")
	ErrProviderRejected      = errors.New("payment provider rejected the request")
)

type InitiateRequest struct {
	Amount     float64
	Currency   string
	Phone      string
	ExternalID string
	Reference  string
	PayerMsg   string
	PayeeMsg   string
}

type InitiateResult struct {
	ProviderReference string
	Status            string
	Raw               map[string]any
}

type StatusResult struct {
	ProviderReference string
	Status            string
	Reason            string
	Raw               map[string]any
}

type Provider interface {
	Name() string
	Enabled() bool
	Initiate(ctx context.Context, req InitiateRequest) (*InitiateResult, error)
	Status(ctx context.Context, providerReference string) (*StatusResult, error)
}
