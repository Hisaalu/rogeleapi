package mailer

import (
	"context"
	"log/slog"
)

type NoopSender struct {
	log *slog.Logger
}

func NewNoopSender(log *slog.Logger) *NoopSender {
	return &NoopSender{log: log}
}

func (s *NoopSender) Name() string { return "noop" }

func (s *NoopSender) Send(ctx context.Context, msg Message) error {
	s.log.Info("email suppressed (noop sender)",
		"to", msg.To,
		"subject", msg.Subject,
		"preview", preview(msg.HTML, 120),
	)
	return nil
}

func preview(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
