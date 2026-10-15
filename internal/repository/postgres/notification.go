package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
	n.id, n.user_id, n.type, n.title, n.message, n.link, n.created_at
`

func (r *NotificationRepo) Create(ctx context.Context, n *domain.Notification) (*domain.Notification, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO notifications (user_id, type, title, message, link)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, user_id, type, title, message, link, is_read, created_at
	`, n.UserID, n.Type, n.Title, n.Message, n.Link)

	out, err := scanNotificationSingle(row)
	if err != nil {
		return nil, dberr.Translate(fmt.Errorf("create notification: %w", err))
	}
	return out, nil
}

func (r *NotificationRepo) CreateBulk(ctx context.Context, userIDs []int64, n *domain.Notification) (int, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}

	var (
		placeholders []string
		args         []any
		i            = 1
	)

	for _, uid := range userIDs {
		placeholders = append(placeholders, fmt.Sprintf(
			"($%d,$%d,$%d,$%d,$%d)", i, i+1, i+2, i+3, i+4,
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

// ListForUser returns notifications visible to a user with a computed `is_read` flag.
// is_read is true if:
//   - the notification's own `is_read` column is true (legacy per-user rows), OR
//   - the user has an entry in notification_reads
func (r *NotificationRepo) ListForUser(ctx context.Context, userID int64, limit, offset int) ([]*domain.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			n.id, n.user_id, n.type, n.title, n.message, n.link,
			CASE
				WHEN n.user_id = $1 THEN n.is_read
				WHEN nr.user_id IS NOT NULL THEN true
				ELSE false
			END AS is_read,
			n.created_at
		FROM notifications n
		LEFT JOIN notification_reads nr
			ON nr.notification_id = n.id AND nr.user_id = $1
		WHERE n.user_id = $1 OR n.user_id IS NULL
		ORDER BY n.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	return scanNotificationRowsCompat(rows)
}

func (r *NotificationRepo) CountUnreadForUser(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM notifications n
		LEFT JOIN notification_reads nr
			ON nr.notification_id = n.id AND nr.user_id = $1
		WHERE (n.user_id = $1 OR n.user_id IS NULL)
		  AND (
			(n.user_id = $1 AND n.is_read = false)
			OR
			(n.user_id IS NULL AND nr.user_id IS NULL)
		  )
	`, userID).Scan(&n)
	return n, err
}

// MarkRead marks one notification as read for a specific user.
func (r *NotificationRepo) MarkRead(ctx context.Context, id, userID int64) error {
	// Own notification: flip its own is_read column.
	// Broadcast: insert into notification_reads.
	var ownerID sql.NullInt64
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM notifications WHERE id = $1`, id).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}

	if !ownerID.Valid {
		_, err = r.db.ExecContext(ctx, `
			INSERT INTO notification_reads (notification_id, user_id)
			VALUES ($1, $2)
			ON CONFLICT (notification_id, user_id) DO NOTHING
		`, id, userID)
		if err != nil {
			return dberr.Translate(fmt.Errorf("mark broadcast read: %w", err))
		}
		return nil
	}

	if ownerID.Int64 != userID {
		return domain.ErrNotFound
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE notifications SET is_read = true WHERE id = $1
	`, id)
	if err != nil {
		return dberr.Translate(fmt.Errorf("mark notification read: %w", err))
	}
	return nil
}

// MarkAllRead marks every notification the user can see as read.
func (r *NotificationRepo) MarkAllRead(ctx context.Context, userID int64) (int, error) {
	// 1. Flip is_read on the user's own unread notifications.
	ownRes, err := r.db.ExecContext(ctx, `
		UPDATE notifications SET is_read = true
		WHERE user_id = $1 AND is_read = false
	`, userID)
	if err != nil {
		return 0, dberr.Translate(fmt.Errorf("mark own read: %w", err))
	}
	ownCount, _ := ownRes.RowsAffected()

	// 2. Insert notification_reads for every visible broadcast they haven't read.
	readRes, err := r.db.ExecContext(ctx, `
		INSERT INTO notification_reads (notification_id, user_id)
		SELECT n.id, $1
		FROM notifications n
		LEFT JOIN notification_reads nr
			ON nr.notification_id = n.id AND nr.user_id = $1
		WHERE n.user_id IS NULL AND nr.user_id IS NULL
		ON CONFLICT (notification_id, user_id) DO NOTHING
	`, userID)
	if err != nil {
		return 0, dberr.Translate(fmt.Errorf("mark broadcasts read: %w", err))
	}
	broadcastCount, _ := readRes.RowsAffected()

	return int(ownCount + broadcastCount), nil
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

	query := `SELECT id, user_id, type, title, message, link, is_read, created_at FROM notifications`
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
	return scanNotificationRowsCompat(rows)
}

func (r *NotificationRepo) ListBroadcasts(ctx context.Context, limit, offset int) ([]*domain.Notification, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, type, title, message, link, is_read, created_at
		FROM notifications
		WHERE user_id IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list broadcasts: %w", err)
	}
	defer rows.Close()
	return scanNotificationRowsCompat(rows)
}

func (r *NotificationRepo) CountBroadcasts(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id IS NULL`).Scan(&n)
	return n, err
}

// Scanners

type notificationScanner interface {
	Scan(dest ...any) error
}

func scanNotificationSingle(s notificationScanner) (*domain.Notification, error) {
	var (
		n      domain.Notification
		userID sql.NullInt64
		link   sql.NullString
		isRead bool
	)
	if err := s.Scan(
		&n.ID, &userID, &n.Type, &n.Title, &n.Message, &link, &isRead, &n.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if userID.Valid {
		n.UserID = &userID.Int64
	}
	if link.Valid {
		n.Link = &link.String
	}
	n.IsRead = isRead
	return &n, nil
}

func scanNotificationRowsCompat(rows *sql.Rows) ([]*domain.Notification, error) {
	out := make([]*domain.Notification, 0, 16)
	for rows.Next() {
		var (
			n      domain.Notification
			userID sql.NullInt64
			link   sql.NullString
			isRead bool
		)
		if err := rows.Scan(
			&n.ID, &userID, &n.Type, &n.Title, &n.Message, &link, &isRead, &n.CreatedAt,
		); err != nil {
			return nil, err
		}
		if userID.Valid {
			n.UserID = &userID.Int64
		}
		if link.Valid {
			n.Link = &link.String
		}
		n.IsRead = isRead
		out = append(out, &n)
	}
	return out, rows.Err()
}
