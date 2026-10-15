package domain

import "time"

type NotificationType string

const (
	NotificationTypeHomework     NotificationType = "homework"
	NotificationTypeQuiz         NotificationType = "quiz"
	NotificationTypeLesson       NotificationType = "lesson"
	NotificationTypePayment      NotificationType = "payment"
	NotificationTypeSubscription NotificationType = "subscription"
	NotificationTypeTrial        NotificationType = "trial"
	NotificationTypeAnnouncement NotificationType = "announcement"
	NotificationTypeSystem       NotificationType = "system"
)

type Notification struct {
	ID        int64            `json:"id"`
	UserID    *int64           `json:"user_id,omitempty"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Message   string           `json:"message"`
	Link      *string          `json:"link,omitempty"`
	IsRead    bool             `json:"is_read"`
	CreatedAt time.Time        `json:"created_at"`
}
