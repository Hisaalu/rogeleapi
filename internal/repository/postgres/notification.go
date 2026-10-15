package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Hisaalu/rogele/internal/dberr"
	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository"
)

type NotificationRepo struct {
	db *sql.DB
}

func NewNotificationRepo(db *sql.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

const notificationColumns = `
	id, user_id, type, title, message, link, is_read, created_at
`

// Create inserts a single notification. If n.UserID is nil, it's a broadcast.
func (r *NotificationRepo) Create(ctx context.Context, n *domain.Notification) (*domain.Notification, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO notifications (user_id, type, title, message, link)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING `+notificationColumns,
		n.UserID, n.Type, n.Title, n.Message, n.Link,
	)
	return scanNotification(row)
}

// CreateBulk inserts one row per userID. Used by event emitters.
func (r *NotificationRepo) CreateBulk(ctx context.Context, userIDs []int64, n *domain.Notification) (int, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}

	var (
		placeholders []string
		args         []any
		i            = 1
	)

	// Column list is fixed; each row is ($1,$2,$3,$4,$5,NULL)
	// We don't reuse the same args — we insert type/title/message/link once
	// per row. Postgres doesn't support "reuse" in VALUES lists.
	for _, uid := range userIDs {
		placeholders = append(placeholders, fmt.Sprintf(
			"($%d,$%d,$%d,$%d,$%d)",
			i, i+1, i+2, i+3, i+4,
		))
		args = append(args, uid, n.Type, n.Title, n.Message, n.Link)
		i += 5
	}

	query := `INSERT INTO notifications (user_id, type, title, message, link) VALUES ` +
		strings.Join(placeholders, ", ")

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, dberr.Translate(fmt.Errorf("create notifications bulk: %w", err))
	}
	count, _ := res.RowsAffected()
	return int(count), nil
}

// ListForUser returns notifications visible to a user: their own + broadcasts.
func (r *NotificationRepo) ListForUser(ctx context.Context, userID int64, limit, offset int) ([]*domain.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT `+notificationColumns+`
		FROM notifications
		WHERE user_id = $1 OR user_id IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	return scanNotificationRows(rows)
}

// CountUnreadForUser counts unread notifications the user can see.
func (r *NotificationRepo) CountUnreadForUser(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notifications
		WHERE (user_id = $1 OR user_id IS NULL) AND is_read = false
	`, userID).Scan(&n)
	return n, err
}

// MarkRead marks a single notification as read if the user can see it.
func (r *NotificationRepo) MarkRead(ctx context.Context, id, userID int64) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE notifications SET is_read = true
		WHERE id = $1 AND (user_id = $2 OR user_id IS NULL)
	`, id, userID)
	if err != nil {
		return dberr.Translate(fmt.Errorf("mark notification read: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// MarkAllRead marks all of the user's own + broadcast notifications as read.
func (r *NotificationRepo) MarkAllRead(ctx context.Context, userID int64) (int, error) {
	// Broadcast rows are shared. Marking a broadcast as read affects every user.
	// To avoid that, we only mark the user's own notifications.
	// A cleaner design would use a join table, but for now, only per-user rows.
	res, err := r.db.ExecContext(ctx, `
		UPDATE notifications SET is_read = true
		WHERE user_id = $1 AND is_read = false
	`, userID)
	if err != nil {
		return 0, dberr.Translate(fmt.Errorf("mark all read: %w", err))
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (r *NotificationRepo) AdminList(ctx context.Context, f repository.AdminListFilter) ([]*domain.Notification, error) {
	var where []string
	var args []any
	i := 1

	if f.UserID != nil {
		where = append(where, fmt.Sprintf("user_id = $%d", i))
		args = append(args, *f.UserID)
		i++
	}
	if f.Type != nil {
		where = append(where, fmt.Sprintf("type = $%d", i))
		args = append(args, *f.Type)
		i++
	}

	query := `SELECT ` + notificationColumns + ` FROM notifications`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC"

	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT $%d", i)
	args = append(args, limit)
	i++

	if f.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", i)
		args = append(args, f.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("admin list notifications: %w", err)
	}
	defer rows.Close()
	return scanNotificationRows(rows)
}

// CountSentBroadcast returns total broadcasts sent (user_id IS NULL).
func (r *NotificationRepo) CountSentBroadcast(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id IS NULL`).Scan(&n)
	return n, err
}

// Scanner helpers

type notificationScanner interface {
	Scan(dest ...any) error
}

func scanNotification(row *sql.Row) (*domain.Notification, error) {
	n, err := scanNotificationRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return n, err
}

func scanNotificationRow(s notificationScanner) (*domain.Notification, error) {
	var (
		n      domain.Notification
		userID sql.NullInt64
		link   sql.NullString
	)
	if err := s.Scan(
		&n.ID, &userID, &n.Type, &n.Title, &n.Message, &link, &n.IsRead, &n.CreatedAt,
	); err != nil {
		return nil, err
	}
	if userID.Valid {
		n.UserID = &userID.Int64
	}
	if link.Valid {
		n.Link = &link.String
	}
	return &n, nil
}

func scanNotificationRows(rows *sql.Rows) ([]*domain.Notification, error) {
	out := make([]*domain.Notification, 0, 16)
	for rows.Next() {
		n, err := scanNotificationRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

var _ = time.Now
