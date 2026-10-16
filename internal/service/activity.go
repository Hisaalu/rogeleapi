package service

import (
	"context"
	"database/sql"
	"log/slog"
)

type ActivityService struct {
	db  *sql.DB
	log *slog.Logger
}

func NewActivityService(db *sql.DB, log *slog.Logger) *ActivityService {
	return &ActivityService{db: db, log: log}
}

func (s *ActivityService) Log(ctx context.Context, userID int64, action, description string) {
	if s == nil || s.db == nil {
		return
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO activity_logs (user_id, action, description)
		VALUES ($1, $2, $3)
	`, userID, action, nullIfEmpty(description))
	if err != nil {
		s.log.Warn("failed to log activity", "action", action, "user_id", userID, "error", err)
	}
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
