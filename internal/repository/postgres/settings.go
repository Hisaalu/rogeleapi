package postgres

import (
	"context"
	"database/sql"
	"fmt"

)

type SettingsRepo struct {
	db *sql.DB
}

func NewSettingsRepo(db *sql.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) ListPublicSettings(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT setting_key, setting_value
		FROM settings
		WHERE setting_value IS NOT NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("list public settings: %w", err)
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var (
			key   string
			value sql.NullString
		)
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		if value.Valid {
			out[key] = value.String
		}
	}
	return out, rows.Err()
}

func (r *SettingsRepo) Upsert(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO settings (setting_key, setting_value)
		VALUES ($1, $2)
		ON CONFLICT (setting_key) DO UPDATE
		SET setting_value = EXCLUDED.setting_value
	`, key, value)
	return err
}

func (r *SettingsRepo) GetByKey(ctx context.Context, key string) (string, error) {
	var val sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT setting_value FROM settings WHERE setting_key = $1`, key).Scan(&val)
	if err != nil {
		return "", err
	}
	return val.String, nil
}