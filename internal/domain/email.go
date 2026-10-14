package domain

import "time"

type EmailTemplate struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Subject   string     `json:"subject"`
	Body      string     `json:"body"`
	CreatedBy *int64     `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

type EmailJob struct {
	ID           int64          `json:"id"`
	ToEmail      string         `json:"to_email"`
	ToName       *string        `json:"to_name,omitempty"`
	TemplateName string         `json:"template_name"`
	Subject      string         `json:"subject"`
	Variables    map[string]any `json:"variables"`
	CampaignID   *int64         `json:"campaign_id,omitempty"`
	Attempts     int            `json:"attempts"`
	MaxAttempts  int            `json:"max_attempts"`
	ScheduledAt  time.Time      `json:"scheduled_at"`
	Status       string         `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
}

type EmailLog struct {
	ID              int64      `json:"id"`
	CampaignID      *int64     `json:"campaign_id,omitempty"`
	RecipientEmail  string     `json:"recipient_email"`
	RecipientUserID *int64     `json:"recipient_user_id,omitempty"`
	RecipientName   *string    `json:"recipient_name,omitempty"`
	Subject         string     `json:"subject"`
	Status          string     `json:"status"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
	SentAt          *time.Time `json:"sent_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type EmailCampaignRecipientType string

const (
	RecipientAllUsers      EmailCampaignRecipientType = "all_users"
	RecipientTeachers      EmailCampaignRecipientType = "teachers"
	RecipientLearners      EmailCampaignRecipientType = "learners"
	RecipientExternal      EmailCampaignRecipientType = "external"
	RecipientAdmins        EmailCampaignRecipientType = "admins"
	RecipientSpecificUsers EmailCampaignRecipientType = "specific_users"
	RecipientCustomEmails  EmailCampaignRecipientType = "custom_emails"
)

type EmailCampaignStatus string

const (
	CampaignDraft     EmailCampaignStatus = "draft"
	CampaignSending   EmailCampaignStatus = "sending"
	CampaignCompleted EmailCampaignStatus = "completed"
	CampaignFailed    EmailCampaignStatus = "failed"
)

type EmailCampaign struct {
	ID              int64                      `json:"id"`
	Subject         string                     `json:"subject"`
	Body            string                     `json:"body"`
	RecipientType   EmailCampaignRecipientType `json:"recipient_type"`
	RecipientIDs    *string                    `json:"recipient_ids,omitempty"`
	CustomEmails    *string                    `json:"custom_emails,omitempty"`
	TotalRecipients int                        `json:"total_recipients"`
	SentCount       int                        `json:"sent_count"`
	FailedCount     int                        `json:"failed_count"`
	Status          EmailCampaignStatus        `json:"status"`
	CreatedBy       *int64                     `json:"created_by,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
	SentAt          *time.Time                 `json:"sent_at,omitempty"`
}
