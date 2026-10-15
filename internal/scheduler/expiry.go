package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/mailer"
)

type Notifier interface {
	NotifyUser(
		ctx context.Context,
		userID int64,
		notifType domain.NotificationType,
		title string,
		message string,
		link *string,
	) error
}

type ExpiryScheduler struct {
	db            *sql.DB
	mailer        *mailer.Service
	notifications Notifier
	log           *slog.Logger
	webURL        string
}

func NewExpiryScheduler(
	db *sql.DB,
	mailerSvc *mailer.Service,
	notifications Notifier,
	log *slog.Logger,
	webURL string,
) *ExpiryScheduler {
	return &ExpiryScheduler{
		db:            db,
		mailer:        mailerSvc,
		notifications: notifications,
		log:           log,
		webURL:        webURL,
	}
}

func (s *ExpiryScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	s.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("expiry scheduler stopped")
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *ExpiryScheduler) runOnce(ctx context.Context) {
	s.sendSubscriptionReminders(ctx)
	s.sendTrialReminders(ctx)
}

func (s *ExpiryScheduler) sendSubscriptionReminders(ctx context.Context) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.user_id, s.end_date, u.email, u.first_name
		FROM subscriptions s
		INNER JOIN users u ON u.id = s.user_id
		WHERE s.status = 'active'
		  AND s.expiry_reminder_sent = false
		  AND s.end_date BETWEEN CURRENT_TIMESTAMP + INTERVAL '2 days'
		                     AND CURRENT_TIMESTAMP + INTERVAL '3 days'
	`)
	if err != nil {
		s.log.Error("subscription reminder query", "error", err)
		return
	}
	defer rows.Close()

	type item struct {
		subID     int64
		userID    int64
		endDate   time.Time
		email     string
		firstName sql.NullString
	}

	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.subID, &it.userID, &it.endDate, &it.email, &it.firstName); err != nil {
			s.log.Error("subscription reminder scan", "error", err)
			continue
		}
		items = append(items, it)
	}

	for _, it := range items {
		daysLeft := int(time.Until(it.endDate).Hours() / 24)
		if daysLeft < 0 {
			daysLeft = 0
		}

		name := "there"
		if it.firstName.Valid && it.firstName.String != "" {
			name = it.firstName.String
		}

		vars := map[string]any{
			"FirstName": name,
			"EndDate":   it.endDate.Format("January 2, 2006"),
			"DaysLeft":  daysLeft,
			"RenewLink": s.webURL + "/subscription",
		}

		var firstNamePtr *string
		if it.firstName.Valid {
			firstNamePtr = &it.firstName.String
		}

		if _, err := s.mailer.Enqueue(ctx, "subscription_expiring", it.email, firstNamePtr, vars); err != nil {
			s.log.Warn("subscription reminder enqueue failed", "sub_id", it.subID, "error", err)
			continue
		}

		if s.notifications != nil {
			link := s.webURL + "/subscription"
			title := "Subscription expiring soon"
			msg := fmt.Sprintf("Your subscription expires in %d day(s). Renew to keep full access.", daysLeft)
			_ = s.notifications.NotifyUser(ctx, it.userID, domain.NotificationTypeSubscription, title, msg, &link)
		}

		if _, err := s.db.ExecContext(ctx,
			`UPDATE subscriptions SET expiry_reminder_sent = true WHERE id = $1`, it.subID); err != nil {
			s.log.Warn("mark subscription reminder sent", "sub_id", it.subID, "error", err)
		}
	}

	if len(items) > 0 {
		s.log.Info("subscription reminders queued", "count", len(items))
	}
}

func (s *ExpiryScheduler) sendTrialReminders(ctx context.Context) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.user_id, t.end_date, u.email, u.first_name
		FROM free_trials t
		INNER JOIN users u ON u.id = t.user_id
		WHERE t.notification_sent = false
		  AND t.converted_to_subscriber = false
		  AND t.end_date BETWEEN CURRENT_TIMESTAMP + INTERVAL '2 days'
		                     AND CURRENT_TIMESTAMP + INTERVAL '3 days'
	`)
	if err != nil {
		s.log.Error("trial reminder query", "error", err)
		return
	}
	defer rows.Close()

	type item struct {
		trialID   int64
		userID    int64
		endDate   time.Time
		email     string
		firstName sql.NullString
	}

	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.trialID, &it.userID, &it.endDate, &it.email, &it.firstName); err != nil {
			s.log.Error("trial reminder scan", "error", err)
			continue
		}
		items = append(items, it)
	}

	for _, it := range items {
		daysLeft := int(time.Until(it.endDate).Hours() / 24)
		if daysLeft < 0 {
			daysLeft = 0
		}

		name := "there"
		if it.firstName.Valid && it.firstName.String != "" {
			name = it.firstName.String
		}

		vars := map[string]any{
			"FirstName":     name,
			"EndDate":       it.endDate.Format("January 2, 2006"),
			"DaysLeft":      daysLeft,
			"SubscribeLink": s.webURL + "/subscription",
		}

		var firstNamePtr *string
		if it.firstName.Valid {
			firstNamePtr = &it.firstName.String
		}

		if _, err := s.mailer.Enqueue(ctx, "trial_ending", it.email, firstNamePtr, vars); err != nil {
			s.log.Warn("trial reminder enqueue failed", "trial_id", it.trialID, "error", err)
			continue
		}

		if s.notifications != nil {
			link := s.webURL + "/subscription"
			title := "Trial ending soon"
			msg := fmt.Sprintf("Your free trial ends in %d day(s). Subscribe to keep using the platform.", daysLeft)
			_ = s.notifications.NotifyUser(ctx, it.userID, domain.NotificationTypeTrial, title, msg, &link)
		}

		if _, err := s.db.ExecContext(ctx,
			`UPDATE free_trials SET notification_sent = true WHERE id = $1`, it.trialID); err != nil {
			s.log.Warn("mark trial reminder sent", "trial_id", it.trialID, "error", err)
		}
	}

	if len(items) > 0 {
		s.log.Info("trial reminders queued", "count", len(items))
	}
}
