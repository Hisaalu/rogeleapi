package mailer

import (
	"context"
	"log/slog"
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository/postgres"
)

type Service struct {
	sender    Sender
	repo      *postgres.EmailRepo
	branding  *BrandingProvider
	log       *slog.Logger
	fromName  string
	webURL    string
	pollEvery time.Duration
	batchSize int
}

func NewService(
	sender Sender,
	repo *postgres.EmailRepo,
	branding *BrandingProvider,
	log *slog.Logger,
	fromName, webURL string,
) *Service {
	return &Service{
		sender:    sender,
		repo:      repo,
		branding:  branding,
		log:       log,
		fromName:  fromName,
		webURL:    webURL,
		pollEvery: 5 * time.Second,
		batchSize: 20,
	}
}

func (s *Service) Enqueue(
	ctx context.Context,
	templateName string,
	toEmail string,
	toName *string,
	vars map[string]any,
) (*domain.EmailJob, error) {
	tmpl, err := s.repo.GetTemplateByName(ctx, templateName)
	if err != nil {
		return nil, err
	}

	subject, err := RenderText(tmpl.Subject, vars)
	if err != nil {
		return nil, err
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

func (s *Service) Run(ctx context.Context) {
	s.log.Info("mailer worker started", "poll_every", s.pollEvery)
	drainTicker := time.NewTicker(s.pollEvery)
	defer drainTicker.Stop()

	reconcileTicker := time.NewTicker(10 * time.Second)
	defer reconcileTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.Info("mailer worker stopped")
			return
		case <-drainTicker.C:
			s.drain(ctx)
		case <-reconcileTicker.C:
			s.reconcileCampaigns(ctx)
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
	html, err := s.renderBody(ctx, job)
	if err != nil {
		s.log.Error("mailer: render body", "error", err, "job", job.ID)
		s.persistFailureIfFinal(ctx, job, "render: "+err.Error())
		return
	}

	var branding Branding
	if s.branding != nil {
		branding = s.branding.Get(ctx)
	} else {
		branding = defaultBranding
	}

	wrapped := WrapHTML(html, branding)

	msg := Message{
		To:      job.ToEmail,
		Subject: job.Subject,
		HTML:    wrapped,
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
		s.persistFailureIfFinal(ctx, job, err.Error())
		_ = s.repo.MarkJobFailed(ctx, job.ID, err.Error())
		return
	}

	now := time.Now()
	logEntry := &domain.EmailLog{
		CampaignID:     job.CampaignID,
		RecipientEmail: job.ToEmail,
		RecipientName:  job.ToName,
		Subject:        job.Subject,
		Status:         "sent",
		SentAt:         &now,
	}
	savedLog, err := s.repo.CreateLog(ctx, logEntry)
	if err != nil {
		s.log.Error("mailer: create log", "error", err, "job", job.ID)
	} else {
		_ = s.repo.MarkJobSent(ctx, job.ID, savedLog.ID)
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

	var branding Branding
	if s.branding != nil {
		branding = s.branding.Get(ctx)
	} else {
		branding = defaultBranding
	}

	merged := make(map[string]any, len(job.Variables)+6)
	for k, v := range job.Variables {
		merged[k] = v
	}
	merged["_brand_primary"] = branding.ColorPrimary
	merged["_brand_accent"] = branding.ColorAccent
	merged["_school_name"] = branding.SchoolName
	merged["_website_url"] = branding.WebsiteURL
	merged["_logo_url"] = branding.LogoURL
	merged["_support_email"] = branding.SupportEmail

	return RenderText(tmpl.Body, merged)
}

func (s *Service) persistFailureIfFinal(ctx context.Context, job *domain.EmailJob, reason string) {
	if job.Attempts+1 < job.MaxAttempts {
		return
	}
	logEntry := &domain.EmailLog{
		CampaignID:     job.CampaignID,
		RecipientEmail: job.ToEmail,
		RecipientName:  job.ToName,
		Subject:        job.Subject,
		Status:         "failed",
		ErrorMessage:   &reason,
	}
	if _, err := s.repo.CreateLog(ctx, logEntry); err != nil {
		s.log.Error("mailer: create failure log", "error", err, "job", job.ID)
	}
}

func (s *Service) reconcileCampaigns(ctx context.Context) {
	ids, err := s.repo.ListCampaignsInProgress(ctx)
	if err != nil {
		s.log.Error("reconciler: list campaigns", "error", err)
		return
	}
	for _, id := range ids {
		pending, err := s.repo.CountPendingJobs(ctx, id)
		if err != nil {
			s.log.Error("reconciler: count pending", "campaign_id", id, "error", err)
			continue
		}
		if pending > 0 {
			continue
		}
		sent, failed, err := s.repo.RecalcCampaignStats(ctx, id)
		if err != nil {
			s.log.Error("reconciler: recalc stats", "campaign_id", id, "error", err)
			continue
		}
		_ = s.repo.SetCampaignCounts(ctx, id, sent, failed)
		if err := s.repo.MarkCampaignCompleted(ctx, id); err == nil {
			s.log.Info("campaign completed",
				"campaign_id", id,
				"sent", sent,
				"failed", failed)
		}
	}
}

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
