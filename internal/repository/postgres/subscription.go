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

type SubscriptionRepo struct {
	db *sql.DB
}

func NewSubscriptionRepo(db *sql.DB) *SubscriptionRepo {
	return &SubscriptionRepo{db: db}
}

const subscriptionColumns = `
	id, user_id, plan_type, amount, start_date, end_date,
	payment_method, transaction_id, status, auto_renew,
	is_upgrade, upgraded_from, upgraded_at, upgraded_to,
	original_subscription_id, created_at
`

func (r *SubscriptionRepo) Create(ctx context.Context, s *domain.Subscription) (*domain.Subscription, error) {
	query := `
		INSERT INTO subscriptions (
			user_id, plan_type, amount, start_date, end_date,
			payment_method, transaction_id, status, auto_renew,
			is_upgrade, upgraded_from, upgraded_at, upgraded_to,
			original_subscription_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING ` + subscriptionColumns

	row := r.db.QueryRowContext(ctx, query,
		s.UserID, s.PlanType, s.Amount, s.StartDate, s.EndDate,
		s.PaymentMethod, s.TransactionID, s.Status, s.AutoRenew,
		s.IsUpgrade, s.UpgradedFrom, s.UpgradedAt, s.UpgradedTo,
		s.OriginalSubscriptionID,
	)
	result, err := scanSubscription(row)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return result, nil
}

func (r *SubscriptionRepo) GetByID(ctx context.Context, id int64) (*domain.Subscription, error) {
	query := `SELECT ` + subscriptionColumns + ` FROM subscriptions WHERE id = $1`
	return scanSubscription(r.db.QueryRowContext(ctx, query, id))
}

func (r *SubscriptionRepo) GetActiveByUser(ctx context.Context, userID int64) (*domain.Subscription, error) {
	query := `SELECT ` + subscriptionColumns + ` FROM subscriptions
	          WHERE user_id = $1 AND status = 'active'
	          ORDER BY end_date DESC LIMIT 1`
	return scanSubscription(r.db.QueryRowContext(ctx, query, userID))
}

func (r *SubscriptionRepo) List(ctx context.Context, f repository.SubscriptionFilter) ([]*domain.Subscription, error) {
	var where []string
	var args []any
	i := 1

	if f.UserID != nil {
		where = append(where, fmt.Sprintf("user_id = $%d", i))
		args = append(args, *f.UserID)
		i++
	}
	if f.Status != nil {
		where = append(where, fmt.Sprintf("status = $%d", i))
		args = append(args, *f.Status)
		i++
	}

	query := `SELECT ` + subscriptionColumns + ` FROM subscriptions`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC"

	if f.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", i)
		args = append(args, f.Limit)
		i++
	}
	if f.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", i)
		args = append(args, f.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.Subscription, 0, 16)
	for rows.Next() {
		s, err := scanSubscriptionRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

func (r *SubscriptionRepo) Update(ctx context.Context, s *domain.Subscription) error {
	query := `
		UPDATE subscriptions SET
			plan_type = $1, amount = $2, start_date = $3, end_date = $4,
			payment_method = $5, transaction_id = $6, status = $7,
			auto_renew = $8, is_upgrade = $9, upgraded_from = $10,
			upgraded_at = $11, upgraded_to = $12, original_subscription_id = $13
		WHERE id = $14
	`
	res, err := r.db.ExecContext(ctx, query,
		s.PlanType, s.Amount, s.StartDate, s.EndDate,
		s.PaymentMethod, s.TransactionID, s.Status,
		s.AutoRenew, s.IsUpgrade, s.UpgradedFrom,
		s.UpgradedAt, s.UpgradedTo, s.OriginalSubscriptionID,
		s.ID,
	)
	if err != nil {
		return dberr.Translate(fmt.Errorf("update subscription: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SubscriptionRepo) ExpireOverdue(ctx context.Context) (int, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE subscriptions SET status = 'expired'
		 WHERE status = 'active' AND end_date < CURRENT_TIMESTAMP`)
	if err != nil {
		return 0, fmt.Errorf("expire overdue: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

type subscriptionScanner interface {
	Scan(dest ...any) error
}

func scanSubscription(row *sql.Row) (*domain.Subscription, error) {
	s, err := scanSubscriptionRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return s, err
}

func scanSubscriptionRow(s subscriptionScanner) (*domain.Subscription, error) {
	var (
		sub        domain.Subscription
		userID     sql.NullInt64
		amount     sql.NullFloat64
		payMethod  sql.NullString
		txnID      sql.NullString
		upFrom     sql.NullString
		upgradedAt sql.NullTime
		upTo       sql.NullString
		origSubID  sql.NullInt64
	)
	err := s.Scan(
		&sub.ID, &userID, &sub.PlanType, &amount, &sub.StartDate, &sub.EndDate,
		&payMethod, &txnID, &sub.Status, &sub.AutoRenew,
		&sub.IsUpgrade, &upFrom, &upgradedAt, &upTo,
		&origSubID, &sub.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if userID.Valid {
		sub.UserID = &userID.Int64
	}
	if amount.Valid {
		sub.Amount = &amount.Float64
	}
	if payMethod.Valid {
		sub.PaymentMethod = &payMethod.String
	}
	if txnID.Valid {
		sub.TransactionID = &txnID.String
	}
	if upFrom.Valid {
		sub.UpgradedFrom = &upFrom.String
	}
	if upgradedAt.Valid {
		sub.UpgradedAt = &upgradedAt.Time
	}
	if upTo.Valid {
		sub.UpgradedTo = &upTo.String
	}
	if origSubID.Valid {
		sub.OriginalSubscriptionID = &origSubID.Int64
	}
	return &sub, nil
}

func (r *SubscriptionRepo) ListWithUsers(ctx context.Context, f repository.SubscriptionFilter) ([]*domain.Subscription, error) {
	var where []string
	var args []any
	i := 1

	if f.UserID != nil {
		where = append(where, fmt.Sprintf("s.user_id = $%d", i))
		args = append(args, *f.UserID)
		i++
	}
	if f.Status != nil {
		where = append(where, fmt.Sprintf("s.status = $%d", i))
		args = append(args, *f.Status)
		i++
	}

	query := `
		SELECT
			s.id, s.user_id, s.plan_type, s.amount, s.start_date, s.end_date,
			s.payment_method, s.transaction_id, s.status, s.auto_renew,
			s.is_upgrade, s.upgraded_from, s.upgraded_at, s.upgraded_to,
			s.original_subscription_id, s.created_at,
			u.email, u.first_name, u.last_name
		FROM subscriptions s
		LEFT JOIN users u ON u.id = s.user_id
	`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY s.created_at DESC"

	if f.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", i)
		args = append(args, f.Limit)
		i++
	}
	if f.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", i)
		args = append(args, f.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions with users: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.Subscription, 0, 16)
	for rows.Next() {
		var (
			s          domain.Subscription
			userID     sql.NullInt64
			amount     sql.NullFloat64
			payMethod  sql.NullString
			txnID      sql.NullString
			upFrom     sql.NullString
			upgradedAt sql.NullTime
			upTo       sql.NullString
			origSubID  sql.NullInt64
			email      sql.NullString
			firstName  sql.NullString
			lastName   sql.NullString
		)
		if err := rows.Scan(
			&s.ID, &userID, &s.PlanType, &amount, &s.StartDate, &s.EndDate,
			&payMethod, &txnID, &s.Status, &s.AutoRenew,
			&s.IsUpgrade, &upFrom, &upgradedAt, &upTo,
			&origSubID, &s.CreatedAt,
			&email, &firstName, &lastName,
		); err != nil {
			return nil, err
		}
		if userID.Valid {
			s.UserID = &userID.Int64
		}
		if amount.Valid {
			s.Amount = &amount.Float64
		}
		if payMethod.Valid {
			s.PaymentMethod = &payMethod.String
		}
		if txnID.Valid {
			s.TransactionID = &txnID.String
		}
		if upFrom.Valid {
			s.UpgradedFrom = &upFrom.String
		}
		if upgradedAt.Valid {
			s.UpgradedAt = &upgradedAt.Time
		}
		if upTo.Valid {
			s.UpgradedTo = &upTo.String
		}
		if origSubID.Valid {
			s.OriginalSubscriptionID = &origSubID.Int64
		}
		if email.Valid {
			s.UserEmail = &email.String
		}
		if firstName.Valid {
			s.UserFirstName = &firstName.String
		}
		if lastName.Valid {
			s.UserLastName = &lastName.String
		}
		results = append(results, &s)
	}
	return results, rows.Err()
}

var _ repository.SubscriptionRepository = (*SubscriptionRepo)(nil)
