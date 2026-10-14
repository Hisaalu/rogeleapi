package mailer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository/postgres"
)

type Service struct {
	sender    Sender
	repo      *postgres.EmailRepo
	log       *slog.Logger
	fromName  string
	webURL    string
	pollEvery time.Duration
	batchSize int
}

func NewService(sender Sender, repo *postgres.EmailRepo, log *slog.Logger, fromName, webURL string) *Service {
	return &Service{
		sender:    sender,
		repo:      repo,
		log:       log,
		fromName:  fromName,
		webURL:    webURL,
		pollEvery: 5 * time.Second,
		batchSize: 20,
	}
}

// Enqueue renders the subject and queues the message for delivery.
// This is what services call: they pass a template name and variables.
func (s *Service) Enqueue(
	ctx context.Context,
	templateName string,
	toEmail string,
	toName *string,
	vars map[string]any,
) (*domain.EmailJob, error) {
	tmpl, err := s.repo.GetTemplateByName(ctx, templateName)
	if err != nil {
		return nil, fmt.Errorf("template %q: %w", templateName, err)
	}

	subject, err := RenderText(tmpl.Subject, vars)
	if err != nil {
		return nil, fmt.Errorf("render subject: %w", err)
	}

	job := &domain.EmailJob{
		ToEmail:      toEmail,
		ToName:       toName,
		TemplateName: templateName,
		Subject:      subject,
		Variables:    vars,
		ScheduledAt:  time.Now(),
		Status:       "queued",
	}
	return s.repo.Enqueue(ctx, job)
}

// EnqueueRaw is a convenience wrapper when you already have the subject
// and don't want to look up a template (used by campaigns).
func (s *Service) EnqueueRaw(
	ctx context.Context,
	toEmail string,
	toName *string,
	subject string,
	htmlBody string,
	campaignID *int64,
) (*domain.EmailJob, error) {
	job := &domain.EmailJob{
		ToEmail:      toEmail,
		ToName:       toName,
		TemplateName: "campaign",
		Subject:      subject,
		Variables: map[string]any{
			"_raw_html": htmlBody,
		},
		CampaignID:  campaignID,
		ScheduledAt: time.Now(),
		Status:      "queued",
	}
	return s.repo.Enqueue(ctx, job)
}

// Run is the worker loop. It polls the queue and dispatches emails.
// Cancel the context to stop it.
func (s *Service) Run(ctx context.Context) {
	s.log.Info("mailer worker started", "poll_every", s.pollEvery)
	ticker := time.NewTicker(s.pollEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.Info("mailer worker stopped")
			return
		case <-ticker.C:
			s.drain(ctx)
		}
	}
}

func (s *Service) drain(ctx context.Context) {
	for {
		jobs, err := s.repo.ClaimBatch(ctx, s.batchSize)
		if err != nil {
			s.log.Error("mailer: claim batch", "error", err)
			return
		}
		if len(jobs) == 0 {
			return
		}
		for _, job := range jobs {
			s.deliver(ctx, job)
		}
		if len(jobs) < s.batchSize {
			return
		}
	}
}

func (s *Service) deliver(ctx context.Context, job *domain.EmailJob) {
	// 1. Create a log entry first (status = pending)
	logEntry := &domain.EmailLog{
		CampaignID:     job.CampaignID,
		RecipientEmail: job.ToEmail,
		RecipientName:  job.ToName,
		Subject:        job.Subject,
		Status:         "pending",
	}
	savedLog, err := s.repo.CreateLog(ctx, logEntry)
	if err != nil {
		s.log.Error("mailer: create log", "error", err, "job", job.ID)
		_ = s.repo.MarkJobFailed(ctx, job.ID, "create log: "+err.Error())
		return
	}

	// 2. Render body
	html, err := s.renderBody(ctx, job)
	if err != nil {
		s.log.Error("mailer: render body", "error", err, "job", job.ID)
		s.recordFailure(ctx, job, savedLog, "render: "+err.Error())
		return
	}

	// 3. Send
	msg := Message{
		To:      job.ToEmail,
		Subject: job.Subject,
		HTML:    WrapHTML(html),
		Text:    stripTags(html),
	}
	if job.ToName != nil {
		msg.ToName = *job.ToName
	}

	if err := s.sender.Send(ctx, msg); err != nil {
		s.log.Warn("mailer: send failed",
			"error", err,
			"to", job.ToEmail,
			"template", job.TemplateName,
			"attempts", job.Attempts+1)
		s.recordFailure(ctx, job, savedLog, err.Error())
		return
	}

	// 4. Success
	now := time.Now()
	savedLog.Status = "sent"
	savedLog.SentAt = &now
	_ = s.repo.UpdateLog(ctx, savedLog)
	_ = s.repo.MarkJobSent(ctx, job.ID, savedLog.ID)
	if job.CampaignID != nil {
		_ = s.repo.IncrementCampaignStats(ctx, *job.CampaignID, 1, 0)
	}

	s.log.Info("mailer: sent",
		"to", job.ToEmail,
		"template", job.TemplateName,
		"campaign_id", job.CampaignID)
}

func (s *Service) renderBody(ctx context.Context, job *domain.EmailJob) (string, error) {
	if raw, ok := job.Variables["_raw_html"].(string); ok && raw != "" {
		return raw, nil
	}
	tmpl, err := s.repo.GetTemplateByName(ctx, job.TemplateName)
	if err != nil {
		return "", err
	}
	return RenderText(tmpl.Body, job.Variables)
}

func (s *Service) recordFailure(ctx context.Context, job *domain.EmailJob, logEntry *domain.EmailLog, reason string) {
	logEntry.Status = "failed"
	logEntry.ErrorMessage = &reason
	_ = s.repo.UpdateLog(ctx, logEntry)
	_ = s.repo.MarkJobFailed(ctx, job.ID, reason)
	if job.CampaignID != nil && job.Attempts+1 >= job.MaxAttempts {
		_ = s.repo.IncrementCampaignStats(ctx, *job.CampaignID, 0, 1)
	}
}

// stripTags is a very rough HTML-to-text. Good enough for the plain-text
// alternative part of emails; not meant for display.
func stripTags(s string) string {
	var out []byte
	inTag := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '<':
			inTag = true
		case c == '>':
			inTag = false
		case !inTag:
			out = append(out, c)
		}
	}
	return string(out)
}

// Ensure imports are used
var _ = json.Marshal
