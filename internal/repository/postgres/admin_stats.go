package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
)

type AdminStatsRepo struct {
	db *sql.DB
}

func NewAdminStatsRepo(db *sql.DB) *AdminStatsRepo {
	return &AdminStatsRepo{db: db}
}

func (r *AdminStatsRepo) GetOverview(ctx context.Context) (*domain.AdminOverview, error) {
	out := &domain.AdminOverview{
		UsersByRole:      make(map[string]int),
		PaymentsByStatus: make(map[string]int),
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT role, COUNT(*) FROM users GROUP BY role
	`)
	if err != nil {
		return nil, fmt.Errorf("users by role: %w", err)
	}
	for rows.Next() {
		var (
			role  string
			count int
		)
		if err := rows.Scan(&role, &count); err != nil {
			rows.Close()
			return nil, err
		}
		out.UsersByRole[role] = count
		out.TotalUsers += count
	}
	rows.Close()

	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM users
		WHERE created_at >= CURRENT_TIMESTAMP - INTERVAL '30 days'
	`).Scan(&out.NewUsersLast30Days); err != nil {
		return nil, fmt.Errorf("new users 30d: %w", err)
	}

	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM subscriptions WHERE status = 'active'
	`).Scan(&out.ActiveSubscriptions); err != nil {
		return nil, fmt.Errorf("active subscriptions: %w", err)
	}

	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM free_trials
		WHERE end_date >= CURRENT_TIMESTAMP AND converted_to_subscriber = false
	`).Scan(&out.ActiveTrials); err != nil {
		return nil, fmt.Errorf("active trials: %w", err)
	}

	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM classes
	`).Scan(&out.TotalClasses); err != nil {
		return nil, fmt.Errorf("total classes: %w", err)
	}

	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM subjects
	`).Scan(&out.TotalSubjects); err != nil {
		return nil, fmt.Errorf("total subjects: %w", err)
	}

	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM lessons
	`).Scan(&out.TotalLessons); err != nil {
		return nil, fmt.Errorf("total lessons: %w", err)
	}

	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM homework
	`).Scan(&out.TotalHomework); err != nil {
		return nil, fmt.Errorf("total homework: %w", err)
	}

	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM quizzes
	`).Scan(&out.TotalQuizzes); err != nil {
		return nil, fmt.Errorf("total quizzes: %w", err)
	}

	if err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM payments
		WHERE status = 'completed'
		  AND payment_date >= date_trunc('month', CURRENT_TIMESTAMP)
	`).Scan(&out.RevenueThisMonth); err != nil {
		return nil, fmt.Errorf("revenue this month: %w", err)
	}

	if err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM payments
		WHERE status = 'completed'
		  AND payment_date >= date_trunc('month', CURRENT_TIMESTAMP) - INTERVAL '1 month'
		  AND payment_date < date_trunc('month', CURRENT_TIMESTAMP)
	`).Scan(&out.RevenueLastMonth); err != nil {
		return nil, fmt.Errorf("revenue last month: %w", err)
	}

	payRows, err := r.db.QueryContext(ctx, `
		SELECT status, COUNT(*) FROM payments GROUP BY status
	`)
	if err != nil {
		return nil, fmt.Errorf("payments by status: %w", err)
	}
	for payRows.Next() {
		var (
			status string
			count  int
		)
		if err := payRows.Scan(&status, &count); err != nil {
			payRows.Close()
			return nil, err
		}
		out.PaymentsByStatus[status] = count
		out.TotalPayments += count
	}
	payRows.Close()

	return out, nil
}

func (r *AdminStatsRepo) ListRecentActivity(ctx context.Context, limit int) ([]*domain.AdminActivityItem, error) {
	if limit <= 0 || limit > 500 {
		limit = 20
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			l.id, l.action, l.description, l.created_at,
			u.id, u.email, u.first_name, u.last_name, u.role
		FROM activity_logs l
		LEFT JOIN users u ON u.id = l.user_id
		ORDER BY l.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent activity: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.AdminActivityItem, 0, limit)
	for rows.Next() {
		var (
			item      domain.AdminActivityItem
			desc      sql.NullString
			userID    sql.NullInt64
			email     sql.NullString
			firstName sql.NullString
			lastName  sql.NullString
			role      sql.NullString
		)
		if err := rows.Scan(
			&item.ID, &item.Action, &desc, &item.CreatedAt,
			&userID, &email, &firstName, &lastName, &role,
		); err != nil {
			return nil, err
		}
		if desc.Valid {
			item.Description = &desc.String
		}
		if userID.Valid {
			item.UserID = &userID.Int64
		}
		if email.Valid {
			item.UserEmail = &email.String
		}
		if firstName.Valid {
			item.UserFirstName = &firstName.String
		}
		if lastName.Valid {
			item.UserLastName = &lastName.String
		}
		if role.Valid {
			item.UserRole = &role.String
		}
		out = append(out, &item)
	}
	return out, rows.Err()
}

func (r *AdminStatsRepo) ListRecentSignups(ctx context.Context, limit int) ([]*domain.AdminUserSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			u.id, u.email, u.first_name, u.last_name, u.role,
			u.is_active, u.is_suspended, u.created_at,
			u.class_id, c.name
		FROM users u
		LEFT JOIN classes c ON c.id = u.class_id
		ORDER BY u.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent signups: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.AdminUserSummary, 0, limit)
	for rows.Next() {
		var (
			u         domain.AdminUserSummary
			firstName sql.NullString
			lastName  sql.NullString
			classID   sql.NullInt64
			className sql.NullString
		)
		if err := rows.Scan(
			&u.ID, &u.Email, &firstName, &lastName, &u.Role,
			&u.IsActive, &u.IsSuspended, &u.CreatedAt,
			&classID, &className,
		); err != nil {
			return nil, err
		}
		if firstName.Valid {
			u.FirstName = &firstName.String
		}
		if lastName.Valid {
			u.LastName = &lastName.String
		}
		if classID.Valid {
			u.ClassID = &classID.Int64
		}
		if className.Valid {
			u.ClassName = &className.String
		}
		out = append(out, &u)
	}
	return out, rows.Err()
}

func (r *AdminStatsRepo) ListRecentPayments(ctx context.Context, limit int) ([]*domain.AdminPaymentSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			p.id, p.user_id, p.amount, p.plan_type, p.payment_method,
			p.transaction_id, p.status, p.created_at, p.payment_date,
			u.email, u.first_name, u.last_name
		FROM payments p
		LEFT JOIN users u ON u.id = p.user_id
		ORDER BY p.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent payments: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.AdminPaymentSummary, 0, limit)
	for rows.Next() {
		var (
			p         domain.AdminPaymentSummary
			userID    sql.NullInt64
			amount    sql.NullFloat64
			planType  sql.NullString
			method    sql.NullString
			txnID     sql.NullString
			payDate   sql.NullTime
			email     sql.NullString
			firstName sql.NullString
			lastName  sql.NullString
		)
		if err := rows.Scan(
			&p.ID, &userID, &amount, &planType, &method,
			&txnID, &p.Status, &p.CreatedAt, &payDate,
			&email, &firstName, &lastName,
		); err != nil {
			return nil, err
		}
		if userID.Valid {
			p.UserID = &userID.Int64
		}
		if amount.Valid {
			p.Amount = &amount.Float64
		}
		if planType.Valid {
			p.PlanType = &planType.String
		}
		if method.Valid {
			p.PaymentMethod = &method.String
		}
		if txnID.Valid {
			p.TransactionID = &txnID.String
		}
		if payDate.Valid {
			p.PaymentDate = &payDate.Time
		}
		if email.Valid {
			p.UserEmail = &email.String
		}
		if firstName.Valid {
			p.UserFirstName = &firstName.String
		}
		if lastName.Valid {
			p.UserLastName = &lastName.String
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (r *AdminStatsRepo) LearnersByClass(ctx context.Context) ([]*domain.ClassLearnerCount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			c.id,
			c.name,
			COUNT(u.id) FILTER (
				WHERE u.role IN ('learner','external')
				  AND u.is_active = true
				  AND u.is_suspended = false
			)::int AS learner_count
		FROM classes c
		LEFT JOIN users u ON u.class_id = c.id
		GROUP BY c.id, c.name
		ORDER BY c.name
	`)
	if err != nil {
		return nil, fmt.Errorf("learners by class: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.ClassLearnerCount, 0)
	for rows.Next() {
		var item domain.ClassLearnerCount
		if err := rows.Scan(&item.ClassID, &item.ClassName, &item.LearnerCount); err != nil {
			return nil, err
		}
		out = append(out, &item)
	}
	return out, rows.Err()
}

var _ = time.Now
