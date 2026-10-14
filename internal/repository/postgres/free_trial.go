package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Hisaalu/rogele/internal/dberr"
	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository"
)

type FreeTrialRepo struct {
	db *sql.DB
}

func NewFreeTrialRepo(db *sql.DB) *FreeTrialRepo {
	return &FreeTrialRepo{db: db}
}

const freeTrialColumns = `
	id, user_id, start_date, end_date,
	notification_sent, converted_to_subscriber, created_at
`

func (r *FreeTrialRepo) Create(ctx context.Context, t *domain.FreeTrial) (*domain.FreeTrial, error) {
	query := `
		INSERT INTO free_trials (
			user_id, start_date, end_date, notification_sent, converted_to_subscriber
		) VALUES ($1,$2,$3,$4,$5)
		RETURNING ` + freeTrialColumns
	row := r.db.QueryRowContext(ctx, query,
		t.UserID, t.StartDate, t.EndDate, t.NotificationSent, t.ConvertedToSubscriber,
	)
	result, err := scanFreeTrial(row)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return result, nil
}

func (r *FreeTrialRepo) GetByUser(ctx context.Context, userID int64) (*domain.FreeTrial, error) {
	query := `SELECT ` + freeTrialColumns + ` FROM free_trials WHERE user_id = $1`
	return scanFreeTrial(r.db.QueryRowContext(ctx, query, userID))
}

func (r *FreeTrialRepo) ExistsForEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM free_trials ft
			INNER JOIN users u ON u.id = ft.user_id
			WHERE LOWER(u.email) = LOWER($1)
		)
	`, email).Scan(&exists)
	return exists, err
}

func (r *FreeTrialRepo) Update(ctx context.Context, t *domain.FreeTrial) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE free_trials SET
			start_date = $1, end_date = $2,
			notification_sent = $3, converted_to_subscriber = $4
		WHERE id = $5
	`, t.StartDate, t.EndDate, t.NotificationSent, t.ConvertedToSubscriber, t.ID)
	if err != nil {
		return dberr.Translate(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanFreeTrial(row *sql.Row) (*domain.FreeTrial, error) {
	var (
		t      domain.FreeTrial
		userID sql.NullInt64
	)
	err := row.Scan(
		&t.ID, &userID, &t.StartDate, &t.EndDate,
		&t.NotificationSent, &t.ConvertedToSubscriber, &t.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if userID.Valid {
		t.UserID = &userID.Int64
	}
	return &t, nil
}

var _ repository.FreeTrialRepository = (*FreeTrialRepo)(nil)
