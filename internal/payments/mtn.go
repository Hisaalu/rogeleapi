package payments

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Hisaalu/rogele/internal/config"
)

type MTNProvider struct {
	cfg    config.MomoConfig
	client *http.Client
}

func NewMTNProvider(cfg config.MomoConfig) *MTNProvider {
	return &MTNProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *MTNProvider) Name() string  { return "mtn" }
func (p *MTNProvider) Enabled() bool { return p.cfg.Enabled() }

func (p *MTNProvider) Initiate(ctx context.Context, req InitiateRequest) (*InitiateResult, error) {
	if !p.Enabled() {
		return nil, ErrProviderNotConfigured
	}

	ref := uuid.NewString()

	token, err := p.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	body := map[string]any{
		"amount":       fmt.Sprintf("%.2f", req.Amount),
		"currency":     req.Currency,
		"externalId":   req.ExternalID,
		"payer":        map[string]any{"partyIdType": "MSISDN", "partyId": req.Phone},
		"payerMessage": req.PayerMsg,
		"payeeNote":    req.PayeeMsg,
	}

	buf, _ := json.Marshal(body)

	url := p.cfg.BaseURL + "/collection/v1_0/requesttopay"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("X-Reference-Id", ref)
	httpReq.Header.Set("X-Target-Environment", p.cfg.TargetEnv)
	httpReq.Header.Set("Ocp-Apim-Subscription-Key", p.cfg.SubscriptionKey)
	httpReq.Header.Set("X-Callback-Url", p.cfg.CallbackURL)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: %s %s", ErrProviderRejected, resp.Status, string(raw))
	}

	return &InitiateResult{
		ProviderReference: ref,
		Status:            "pending",
		Raw: map[string]any{
			"provider":  "mtn",
			"reference": ref,
		},
	}, nil
}

func (p *MTNProvider) Status(ctx context.Context, providerReference string) (*StatusResult, error) {
	if !p.Enabled() {
		return nil, ErrProviderNotConfigured
	}

	token, err := p.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	url := fmt.Sprintf("%s/collection/v1_0/requesttopay/%s", p.cfg.BaseURL, providerReference)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("X-Target-Environment", p.cfg.TargetEnv)
	httpReq.Header.Set("Ocp-Apim-Subscription-Key", p.cfg.SubscriptionKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s %s", ErrProviderRejected, resp.Status, string(raw))
	}

	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}

	status, _ := body["status"].(string)
	reason, _ := body["reason"].(string)

	return &StatusResult{
		ProviderReference: providerReference,
		Status:            status,
		Reason:            reason,
		Raw:               body,
	}, nil
}

func (p *MTNProvider) getAccessToken(ctx context.Context) (string, error) {
	auth := base64.StdEncoding.EncodeToString([]byte(p.cfg.APIUser + ":" + p.cfg.APIKey))

	url := p.cfg.BaseURL + "/collection/token/"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Basic "+auth)
	httpReq.Header.Set("Ocp-Apim-Subscription-Key", p.cfg.SubscriptionKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: token %s %s", ErrProviderRejected, resp.Status, string(raw))
	}

	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return "", err
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("empty access token")
	}
	return body.AccessToken, nil
}

var _ Provider = (*MTNProvider)(nil)
