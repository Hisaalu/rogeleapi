package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/mailer"
	"github.com/Hisaalu/rogele/internal/repository/postgres"
)

type CreateCampaignInput struct {
	Subject       string
	Body          string
	RecipientType string
	RecipientIDs  []int64
	CustomEmails  []string
}

type UpdateCampaignInput struct {
	Subject       *string
	Body          *string
	RecipientType *string
	RecipientIDs  []int64
	CustomEmails  []string
}

type PreviewInput struct {
	SampleEmail string
	SampleName  string
}

type CampaignService struct {
	repo   *postgres.EmailRepo
	mailer *mailer.Service
	log    *slog.Logger
}

func NewCampaignService(repo *postgres.EmailRepo, mailerSvc *mailer.Service, log *slog.Logger) *CampaignService {
	return &CampaignService{repo: repo, mailer: mailerSvc, log: log}
}

func (s *CampaignService) List(ctx context.Context, limit, offset int) ([]*domain.EmailCampaign, error) {
	return s.repo.ListCampaigns(ctx, limit, offset)
}

func (s *CampaignService) Get(ctx context.Context, id int64) (*domain.EmailCampaign, error) {
	return s.repo.GetCampaign(ctx, id)
}

func (s *CampaignService) Create(ctx context.Context, actorID int64, in CreateCampaignInput) (*domain.EmailCampaign, error) {
	if strings.TrimSpace(in.Subject) == "" {
		return nil, fmt.Errorf("%w: subject is required", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(in.Body) == "" {
		return nil, fmt.Errorf("%w: body is required", domain.ErrInvalidInput)
	}
	if !isValidRecipientType(in.RecipientType) {
		return nil, fmt.Errorf("%w: invalid recipient_type", domain.ErrInvalidInput)
	}

	var recipientIDs *string
	if len(in.RecipientIDs) > 0 {
		b, _ := json.Marshal(in.RecipientIDs)
		str := string(b)
		recipientIDs = &str
	}

	var customEmails *string
	if len(in.CustomEmails) > 0 {
		b, _ := json.Marshal(in.CustomEmails)
		str := string(b)
		customEmails = &str
	}

	uid := actorID
	c := &domain.EmailCampaign{
		Subject:       in.Subject,
		Body:          in.Body,
		RecipientType: domain.EmailCampaignRecipientType(in.RecipientType),
		RecipientIDs:  recipientIDs,
		CustomEmails:  customEmails,
		Status:        domain.CampaignDraft,
		CreatedBy:     &uid,
	}
	return s.repo.CreateCampaign(ctx, c)
}

func (s *CampaignService) Update(ctx context.Context, id int64, in UpdateCampaignInput) (*domain.EmailCampaign, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Status != domain.CampaignDraft {
		return nil, fmt.Errorf("%w: only draft campaigns can be edited", domain.ErrInvalidInput)
	}

	if in.Subject != nil {
		c.Subject = *in.Subject
	}
	if in.Body != nil {
		c.Body = *in.Body
	}
	if in.RecipientType != nil {
		if !isValidRecipientType(*in.RecipientType) {
			return nil, fmt.Errorf("%w: invalid recipient_type", domain.ErrInvalidInput)
		}
		c.RecipientType = domain.EmailCampaignRecipientType(*in.RecipientType)
	}
	if in.RecipientIDs != nil {
		b, _ := json.Marshal(in.RecipientIDs)
		str := string(b)
		c.RecipientIDs = &str
	}
	if in.CustomEmails != nil {
		b, _ := json.Marshal(in.CustomEmails)
		str := string(b)
		c.CustomEmails = &str
	}

	if err := s.repo.UpdateCampaign(ctx, c); err != nil {
		return nil, err
	}
	return s.repo.GetCampaign(ctx, id)
}

func (s *CampaignService) Delete(ctx context.Context, id int64) error {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return err
	}
	if c.Status != domain.CampaignDraft {
		return fmt.Errorf("%w: only draft campaigns can be deleted", domain.ErrInvalidInput)
	}
	return s.repo.DeleteCampaign(ctx, id)
}

func (s *CampaignService) Preview(ctx context.Context, id int64, in PreviewInput) (string, string, int, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return "", "", 0, err
	}

	recipients, err := s.resolveRecipients(ctx, c)
	if err != nil {
		return "", "", 0, err
	}

	sample := map[string]any{
		"FirstName": in.SampleName,
		"Email":     in.SampleEmail,
	}
	if sample["FirstName"] == "" {
		sample["FirstName"] = "Sample"
	}
	if sample["Email"] == "" {
		sample["Email"] = "sample@example.com"
	}

	subject, err := mailer.RenderText(c.Subject, sample)
	if err != nil {
		return "", "", 0, fmt.Errorf("render subject: %w", err)
	}
	body, err := mailer.RenderText(c.Body, sample)
	if err != nil {
		return "", "", 0, fmt.Errorf("render body: %w", err)
	}

	return subject, body, len(recipients), nil
}

func (s *CampaignService) Send(ctx context.Context, id int64) (int, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return 0, err
	}
	if c.Status != domain.CampaignDraft {
		return 0, fmt.Errorf("%w: campaign has already been sent", domain.ErrInvalidInput)
	}

	recipients, err := s.resolveRecipients(ctx, c)
	if err != nil {
		return 0, err
	}
	if len(recipients) == 0 {
		return 0, fmt.Errorf("%w: no recipients matched", domain.ErrInvalidInput)
	}

	queued := 0
	for _, r := range recipients {
		vars := map[string]any{
			"FirstName": derefOr(r.FirstName, "there"),
			"FullName":  r.DisplayName(),
			"Email":     r.Email,
		}
		subject, err := mailer.RenderText(c.Subject, vars)
		if err != nil {
			s.log.Warn("campaign render subject failed", "email", r.Email, "error", err)
			continue
		}
		body, err := mailer.RenderText(c.Body, vars)
		if err != nil {
			s.log.Warn("campaign render body failed", "email", r.Email, "error", err)
			continue
		}

		if _, err := s.mailer.EnqueueRaw(ctx, r.Email, r.FirstName, subject, body, &c.ID); err != nil {
			s.log.Warn("campaign enqueue failed", "email", r.Email, "error", err)
			continue
		}
		queued++
	}

	now := time.Now()
	c.Status = domain.CampaignSending
	c.TotalRecipients = queued
	c.SentAt = &now
	if err := s.repo.UpdateCampaign(ctx, c); err != nil {
		return queued, err
	}

	return queued, nil
}

func (s *CampaignService) Logs(ctx context.Context, campaignID int64, limit, offset int) ([]*domain.EmailLog, error) {
	cid := campaignID
	return s.repo.ListLogs(ctx, postgres.LogFilter{
		CampaignID: &cid,
		Limit:      limit,
		Offset:     offset,
	})
}

func (s *CampaignService) resolveRecipients(ctx context.Context, c *domain.EmailCampaign) ([]*domain.EmailRecipient, error) {
	switch c.RecipientType {
	case domain.RecipientAllUsers:
		return s.repo.ListAllRecipients(ctx)
	case domain.RecipientTeachers:
		return s.repo.ListRecipientsByRole(ctx, string(domain.RoleTeacher))
	case domain.RecipientLearners:
		return s.repo.ListRecipientsByRole(ctx, string(domain.RoleLearner))
	case domain.RecipientExternal:
		return s.repo.ListRecipientsByRole(ctx, string(domain.RoleExternal))
	case domain.RecipientAdmins:
		return s.repo.ListRecipientsByRole(ctx, string(domain.RoleAdmin))
	case domain.RecipientSpecificUsers:
		ids, err := parseIDList(c.RecipientIDs)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid recipient_ids", domain.ErrInvalidInput)
		}
		return s.repo.ListRecipientsByIDs(ctx, ids)
	case domain.RecipientCustomEmails:
		emails, err := parseEmailList(c.CustomEmails)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid custom_emails", domain.ErrInvalidInput)
		}
		out := make([]*domain.EmailRecipient, 0, len(emails))
		for i, e := range emails {
			out = append(out, &domain.EmailRecipient{
				ID:    int64(-(i + 1)),
				Email: e,
			})
		}
		return out, nil
	}
	return nil, fmt.Errorf("%w: unknown recipient_type", domain.ErrInvalidInput)
}

func isValidRecipientType(t string) bool {
	switch domain.EmailCampaignRecipientType(t) {
	case domain.RecipientAllUsers,
		domain.RecipientTeachers,
		domain.RecipientLearners,
		domain.RecipientExternal,
		domain.RecipientAdmins,
		domain.RecipientSpecificUsers,
		domain.RecipientCustomEmails:
		return true
	}
	return false
}

func parseIDList(s *string) ([]int64, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	var ids []int64
	if err := json.Unmarshal([]byte(*s), &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func parseEmailList(s *string) ([]string, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	var emails []string
	if err := json.Unmarshal([]byte(*s), &emails); err != nil {
		return nil, err
	}
	clean := make([]string, 0, len(emails))
	for _, e := range emails {
		e = strings.TrimSpace(strings.ToLower(e))
		if e != "" && strings.Contains(e, "@") {
			clean = append(clean, e)
		}
	}
	return clean, nil
}
