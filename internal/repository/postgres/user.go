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

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

const userColumns = `
	id, registration_number, email, password, first_name, last_name,
	phone, bio, qualification, specialization, role, class_id,
	profile_photo, is_active, is_suspended, email_verified,
	last_login, created_at, updated_at, trial_start_date
`

func (r *UserRepo) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	query := `
		INSERT INTO users (
			registration_number, email, password, first_name, last_name,
			phone, bio, qualification, specialization, role, class_id,
			profile_photo, is_active, is_suspended, email_verified
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING ` + userColumns

	row := r.db.QueryRowContext(ctx, query,
		u.RegistrationNumber, u.Email, u.Password, u.FirstName, u.LastName,
		u.Phone, u.Bio, u.Qualification, u.Specialization, u.Role, u.ClassID,
		u.ProfilePhoto, u.IsActive, u.IsSuspended, u.EmailVerified,
	)
	result, err := scanUser(row)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return result, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanUser(row)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE LOWER(email) = LOWER($1)`
	row := r.db.QueryRowContext(ctx, query, email)
	return scanUser(row)
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	query := `
		UPDATE users SET
			registration_number = $1,
			email               = $2,
			first_name          = $3,
			last_name           = $4,
			phone               = $5,
			bio                 = $6,
			qualification       = $7,
			specialization      = $8,
			role                = $9,
			class_id            = $10,
			profile_photo       = $11,
			is_active           = $12,
			is_suspended        = $13,
			email_verified      = $14,
			updated_at          = CURRENT_TIMESTAMP
		WHERE id = $15
	`
	res, err := r.db.ExecContext(ctx, query,
		u.RegistrationNumber, u.Email, u.FirstName, u.LastName,
		u.Phone, u.Bio, u.Qualification, u.Specialization, u.Role,
		u.ClassID, u.ProfilePhoto, u.IsActive, u.IsSuspended,
		u.EmailVerified, u.ID,
	)
	if err != nil {
		return dberr.Translate(fmt.Errorf("update user: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id int64, hash string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET password = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		hash, id,
	)
	if err != nil {
		return dberr.Translate(fmt.Errorf("update password: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepo) Delete(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `UPDATE homework SET teacher_id = NULL WHERE teacher_id = $1`, id)
	if err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return tx.Commit()
}

func (r *UserRepo) List(ctx context.Context, f repository.UserFilter) ([]*domain.User, error) {
	var (
		where []string
		args  []any
		i     = 1
	)

	if f.Role != nil {
		where = append(where, fmt.Sprintf("role = $%d", i))
		args = append(args, *f.Role)
		i++
	}
	if f.ClassID != nil {
		where = append(where, fmt.Sprintf("class_id = $%d", i))
		args = append(args, *f.ClassID)
		i++
	}
	if f.Search != "" {
		where = append(where, fmt.Sprintf(
			"(email ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)",
			i, i, i,
		))
		args = append(args, "%"+f.Search+"%")
		i++
	}

	query := `SELECT ` + userColumns + ` FROM users`
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
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]*domain.User, 0, 16)
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = $1`, id)
	return err
}

type userRowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row *sql.Row) (*domain.User, error) {
	u, err := scanUserRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return u, err
}

func scanUserRow(s userRowScanner) (*domain.User, error) {
	var (
		u              domain.User
		lastLogin      sql.NullTime
		regNum         sql.NullString
		firstName      sql.NullString
		lastName       sql.NullString
		phone          sql.NullString
		bio            sql.NullString
		qualification  sql.NullString
		specialization sql.NullString
		classID        sql.NullInt64
		profilePhoto   sql.NullString
		trialStartDate sql.NullTime
	)

	err := s.Scan(
		&u.ID, &regNum, &u.Email, &u.Password, &firstName, &lastName,
		&phone, &bio, &qualification, &specialization, &u.Role, &classID,
		&profilePhoto, &u.IsActive, &u.IsSuspended, &u.EmailVerified,
		&lastLogin, &u.CreatedAt, &u.UpdatedAt, &trialStartDate,
	)
	if err != nil {
		return nil, err
	}

	if regNum.Valid {
		u.RegistrationNumber = &regNum.String
	}
	if firstName.Valid {
		u.FirstName = &firstName.String
	}
	if lastName.Valid {
		u.LastName = &lastName.String
	}
	if phone.Valid {
		u.Phone = &phone.String
	}
	if bio.Valid {
		u.Bio = &bio.String
	}
	if qualification.Valid {
		u.Qualification = &qualification.String
	}
	if specialization.Valid {
		u.Specialization = &specialization.String
	}
	if classID.Valid {
		u.ClassID = &classID.Int64
	}
	if profilePhoto.Valid {
		u.ProfilePhoto = &profilePhoto.String
	}
	if lastLogin.Valid {
		t := lastLogin.Time
		u.LastLogin = &t
	}
	if trialStartDate.Valid {
		u.TrialStartDate = trialStartDate.Time
	}

	return &u, nil
}

type StudentStats struct {
	UserID       int64
	QuizzesTaken int
	AvgScore     float64
}

func (r *UserRepo) GetStudentStats(ctx context.Context) (map[int64]repository.StudentStats, error) {
	query := `
		SELECT 
			u.id,
			COUNT(a.id) as quizzes_taken,
			COALESCE(AVG(a.score), 0) as avg_score
		FROM users u
		LEFT JOIN quiz_attempts a ON u.id = a.user_id AND a.status = 'completed'
		WHERE u.role IN ('learner', 'external')
		GROUP BY u.id
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get student stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[int64]repository.StudentStats)
	for rows.Next() {
		var s repository.StudentStats
		if err := rows.Scan(&s.UserID, &s.QuizzesTaken, &s.AvgScore); err != nil {
			return nil, err
		}
		stats[s.UserID] = s
	}
	return stats, rows.Err()
}

var _ repository.UserRepository = (*UserRepo)(nil)
