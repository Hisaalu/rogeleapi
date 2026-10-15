package dto

import (
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
)

type NotificationResponse struct {
	ID        int64     `json:"id"`
	UserID    *int64    `json:"user_id,omitempty"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Link      *string   `json:"link,omitempty"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

type UnreadCountResponse struct {
	Count int `json:"count"`
}

type BroadcastRequest struct {
	Title   string  `json:"title"`
	Message string  `json:"message"`
	Link    *string `json:"link,omitempty"`
	Type    string  `json:"type,omitempty"`
}

type BroadcastResponse struct {
	Sent int `json:"sent"`
}

func NotificationFromDomain(n *domain.Notification) NotificationResponse {
	return NotificationResponse{
		ID:        n.ID,
		UserID:    n.UserID,
		Type:      string(n.Type),
		Title:     n.Title,
		Message:   n.Message,
		Link:      n.Link,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt,
	}
}

func NotificationsFromDomain(items []*domain.Notification) []NotificationResponse {
	out := make([]NotificationResponse, 0, len(items))
	for _, n := range items {
		out = append(out, NotificationFromDomain(n))
	}
	return out
}
