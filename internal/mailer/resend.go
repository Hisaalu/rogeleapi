package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ResendSender struct {
	apiKey  string
	from    string
	replyTo string
	client  *http.Client
}

func NewResendSender(apiKey, from, replyTo string) *ResendSender {
	return &ResendSender{
		apiKey:  apiKey,
		from:    from,
		replyTo: replyTo,
		client:  &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *ResendSender) Name() string { return "resend" }

func (s *ResendSender) Send(ctx context.Context, msg Message) error {
	if s.apiKey == "" {
		return fmt.Errorf("resend api key is not configured")
	}

	replyTo := msg.ReplyTo
	if replyTo == "" {
		replyTo = s.replyTo
	}

	toField := msg.To
	if msg.ToName != "" {
		toField = fmt.Sprintf("%s <%s>", msg.ToName, msg.To)
	}

	body := map[string]any{
		"from":    s.from,
		"to":      []string{toField},
		"subject": msg.Subject,
		"html":    msg.HTML,
	}
	if msg.Text != "" {
		body["text"] = msg.Text
	}
	if replyTo != "" {
		body["reply_to"] = replyTo
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.resend.com/emails", bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("resend request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend rejected: %s %s", resp.Status, string(raw))
	}
	return nil
}
