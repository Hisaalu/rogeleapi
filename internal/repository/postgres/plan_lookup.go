package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Hisaalu/rogele/internal/domain"
)

type PlanRepo struct {
	db *sql.DB
}

func NewPlanRepo(db *sql.DB) *PlanRepo {
	return &PlanRepo{db: db}
}

func (r *PlanRepo) GetPlan(ctx context.Context, id int64) (*domain.SubscriptionPlan, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, price, duration_days, is_active, created_at, updated_at
		FROM subscription_plans WHERE id = $1
	`, id)

	var (
		p       domain.SubscriptionPlan
		desc    sql.NullString
		updated sql.NullTime
	)
	err := row.Scan(&p.ID, &p.Name, &desc, &p.Price, &p.DurationDays, &p.IsActive, &p.CreatedAt, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		p.Description = &desc.String
	}
	if updated.Valid {
		p.UpdatedAt = &updated.Time
	}
	return &p, nil
}
