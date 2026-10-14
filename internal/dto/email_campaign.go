package dto

import (
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
)

type CreateCampaignRequest struct {
	Subject       string   `json:"subject"`
	Body          string   `json:"body"`
	RecipientType string   `json:"recipient_type"`
	RecipientIDs  []int64  `json:"recipient_ids,omitempty"`
	CustomEmails  []string `json:"custom_emails,omitempty"`
}

type UpdateCampaignRequest struct {
	Subject       *string  `json:"subject,omitempty"`
	Body          *string  `json:"body,omitempty"`
	RecipientType *string  `json:"recipient_type,omitempty"`
	RecipientIDs  []int64  `json:"recipient_ids,omitempty"`
	CustomEmails  []string `json:"custom_emails,omitempty"`
}

type CampaignPreviewRequest struct {
	SampleEmail string `json:"sample_email,omitempty"`
	SampleName  string `json:"sample_name,omitempty"`
}

type CampaignPreviewResponse struct {
	Subject        string `json:"subject"`
	HTML           string `json:"html"`
	RecipientCount int    `json:"recipient_count"`
}

type CampaignResponse struct {
	ID              int64      `json:"id"`
	Subject         string     `json:"subject"`
	Body            string     `json:"body"`
	RecipientType   string     `json:"recipient_type"`
	RecipientIDs    []int64    `json:"recipient_ids,omitempty"`
	CustomEmails    []string   `json:"custom_emails,omitempty"`
	TotalRecipients int        `json:"total_recipients"`
	SentCount       int        `json:"sent_count"`
	FailedCount     int        `json:"failed_count"`
	Status          string     `json:"status"`
	CreatedBy       *int64     `json:"created_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	SentAt          *time.Time `json:"sent_at,omitempty"`
}

type CampaignSendResponse struct {
	CampaignID  int64  `json:"campaign_id"`
	QueuedCount int    `json:"queued_count"`
	Status      string `json:"status"`
}

type EmailLogResponse struct {
	ID             int64      `json:"id"`
	CampaignID     *int64     `json:"campaign_id,omitempty"`
	RecipientEmail string     `json:"recipient_email"`
	RecipientName  *string    `json:"recipient_name,omitempty"`
	Subject        string     `json:"subject"`
	Status         string     `json:"status"`
	ErrorMessage   *string    `json:"error_message,omitempty"`
	SentAt         *time.Time `json:"sent_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

func CampaignFromDomain(c *domain.EmailCampaign) CampaignResponse {
	return CampaignResponse{
		ID:              c.ID,
		Subject:         c.Subject,
		Body:            c.Body,
		RecipientType:   string(c.RecipientType),
		TotalRecipients: c.TotalRecipients,
		SentCount:       c.SentCount,
		FailedCount:     c.FailedCount,
		Status:          string(c.Status),
		CreatedBy:       c.CreatedBy,
		CreatedAt:       c.CreatedAt,
		SentAt:          c.SentAt,
	}
}

func CampaignsFromDomain(items []*domain.EmailCampaign) []CampaignResponse {
	out := make([]CampaignResponse, 0, len(items))
	for _, c := range items {
		out = append(out, CampaignFromDomain(c))
	}
	return out
}

func LogsFromDomain(items []*domain.EmailLog) []EmailLogResponse {
	out := make([]EmailLogResponse, 0, len(items))
	for _, l := range items {
		out = append(out, EmailLogResponse{
			ID:             l.ID,
			CampaignID:     l.CampaignID,
			RecipientEmail: l.RecipientEmail,
			RecipientName:  l.RecipientName,
			Subject:        l.Subject,
			Status:         l.Status,
			ErrorMessage:   l.ErrorMessage,
			SentAt:         l.SentAt,
			CreatedAt:      l.CreatedAt,
		})
	}
	return out
}
