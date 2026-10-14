package payments

import (
	"context"
	"fmt"

	"github.com/Hisaalu/rogele/internal/config"
)

type AirtelProvider struct {
	cfg config.AirtelConfig
}

func NewAirtelProvider(cfg config.AirtelConfig) *AirtelProvider {
	return &AirtelProvider{cfg: cfg}
}

func (p *AirtelProvider) Name() string  { return "airtel" }
func (p *AirtelProvider) Enabled() bool { return p.cfg.Enabled() }

func (p *AirtelProvider) Initiate(ctx context.Context, req InitiateRequest) (*InitiateResult, error) {
	if !p.Enabled() {
		return nil, ErrProviderNotConfigured
	}
	return nil, fmt.Errorf("airtel provider not yet implemented")
}

func (p *AirtelProvider) Status(ctx context.Context, providerReference string) (*StatusResult, error) {
	if !p.Enabled() {
		return nil, ErrProviderNotConfigured
	}
	return nil, fmt.Errorf("airtel provider not yet implemented")
}

var _ Provider = (*AirtelProvider)(nil)
