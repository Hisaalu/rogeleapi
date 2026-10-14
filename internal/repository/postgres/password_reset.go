package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Hisaalu/rogele/internal/dberr"
	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository"
)

type PasswordResetRepo struct {
	db *sql.DB
}

func NewPasswordResetRepo(db *sql.DB) *PasswordResetRepo {
	return &PasswordResetRepo{db: db}
}

const passwordResetColumns = `id, user_id, token, expires_at, used, created_at`

func (r *PasswordResetRepo) Create(ctx context.Context, pr *domain.PasswordReset) (*domain.PasswordReset, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO password_resets (user_id, token, expires_at)
		VALUES ($1,$2,$3)
		RETURNING `+passwordResetColumns,
		pr.UserID, pr.Token, pr.ExpiresAt,
	)
	result, err := scanPasswordReset(row)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return result, nil
}

func (r *PasswordResetRepo) GetByToken(ctx context.Context, token string) (*domain.PasswordReset, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+passwordResetColumns+` FROM password_resets WHERE token = $1`, token)
	return scanPasswordReset(row)
}

func (r *PasswordResetRepo) MarkUsed(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE password_resets SET used = true WHERE id = $1`, id)
	if err != nil {
		return dberr.Translate(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PasswordResetRepo) CountRecent(ctx context.Context, userID int64, within time.Duration) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM password_resets
		WHERE user_id = $1 AND created_at > CURRENT_TIMESTAMP - ($2 || ' seconds')::interval
	`, userID, fmt.Sprintf("%d", int(within.Seconds()))).Scan(&n)
	return n, err
}

func scanPasswordReset(row *sql.Row) (*domain.PasswordReset, error) {
	var pr domain.PasswordReset
	err := row.Scan(&pr.ID, &pr.UserID, &pr.Token, &pr.ExpiresAt, &pr.Used, &pr.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

var _ repository.PasswordResetRepository = (*PasswordResetRepo)(nil)
