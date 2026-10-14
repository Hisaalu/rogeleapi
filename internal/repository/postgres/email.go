package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/Hisaalu/rogele/internal/dberr"
	"github.com/Hisaalu/rogele/internal/domain"
)

type EmailRepo struct {
	db *sql.DB
}

func NewEmailRepo(db *sql.DB) *EmailRepo {
	return &EmailRepo{db: db}
}

// ─── Templates ───────────────────────────────────────────────────────────────

const templateColumns = `id, name, subject, body, created_by, created_at, updated_at`

func (r *EmailRepo) ListTemplates(ctx context.Context) ([]*domain.EmailTemplate, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+templateColumns+` FROM email_templates ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()
	return scanTemplateRows(rows)
}

func (r *EmailRepo) GetTemplateByName(ctx context.Context, name string) (*domain.EmailTemplate, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+templateColumns+` FROM email_templates WHERE name = $1`, name)
	return scanTemplate(row)
}

func (r *EmailRepo) GetTemplateByID(ctx context.Context, id int64) (*domain.EmailTemplate, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+templateColumns+` FROM email_templates WHERE id = $1`, id)
	return scanTemplate(row)
}

func (r *EmailRepo) UpsertTemplate(ctx context.Context, t *domain.EmailTemplate) (*domain.EmailTemplate, error) {
	query := `
		INSERT INTO email_templates (name, subject, body, created_by)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (name) DO UPDATE SET
			subject = EXCLUDED.subject,
			body = EXCLUDED.body,
			updated_at = CURRENT_TIMESTAMP
		RETURNING ` + templateColumns
	row := r.db.QueryRowContext(ctx, query, t.Name, t.Subject, t.Body, t.CreatedBy)
	result, err := scanTemplate(row)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return result, nil
}

// ─── Queue ───────────────────────────────────────────────────────────────────

func (r *EmailRepo) Enqueue(ctx context.Context, job *domain.EmailJob) (*domain.EmailJob, error) {
	vars, _ := json.Marshal(job.Variables)
	query := `
		INSERT INTO email_queue
			(to_email, to_name, template_name, subject, variables, campaign_id, scheduled_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, to_email, to_name, template_name, subject, variables,
		          campaign_id, attempts, max_attempts, scheduled_at, status, created_at
	`
	row := r.db.QueryRowContext(ctx, query,
		job.ToEmail, job.ToName, job.TemplateName, job.Subject, vars,
		job.CampaignID, job.ScheduledAt,
	)
	result, err := scanJob(row)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return result, nil
}

// ClaimBatch locks up to `limit` queued jobs and returns them.
// Uses FOR UPDATE SKIP LOCKED for safe concurrent workers.
func (r *EmailRepo) ClaimBatch(ctx context.Context, limit int) ([]*domain.EmailJob, error) {
	query := `
		WITH claimed AS (
			SELECT id FROM email_queue
			WHERE status = 'queued' AND scheduled_at <= CURRENT_TIMESTAMP
			ORDER BY scheduled_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE email_queue q
		SET status = 'sending', locked_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		FROM claimed
		WHERE q.id = claimed.id
		RETURNING q.id, q.to_email, q.to_name, q.template_name, q.subject, q.variables,
		          q.campaign_id, q.attempts, q.max_attempts, q.scheduled_at, q.status, q.created_at
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("claim batch: %w", err)
	}
	defer rows.Close()

	var jobs []*domain.EmailJob
	for rows.Next() {
		j, err := scanJobRow(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (r *EmailRepo) MarkJobSent(ctx context.Context, id int64, logID int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE email_queue SET status = 'sent', log_id = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, id, logID)
	return err
}

func (r *EmailRepo) MarkJobFailed(ctx context.Context, id int64, reason string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE email_queue
		SET status = CASE WHEN attempts + 1 >= max_attempts THEN 'failed' ELSE 'queued' END,
		    attempts = attempts + 1,
		    last_error = $2,
		    scheduled_at = CASE WHEN attempts + 1 >= max_attempts
		                        THEN scheduled_at
		                        ELSE CURRENT_TIMESTAMP + INTERVAL '2 minutes' END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, id, reason)
	return err
}

// ─── Logs ────────────────────────────────────────────────────────────────────

func (r *EmailRepo) CreateLog(ctx context.Context, l *domain.EmailLog) (*domain.EmailLog, error) {
	query := `
		INSERT INTO email_logs
			(campaign_id, recipient_email, recipient_user_id, recipient_name,
			 subject, status, error_message, sent_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, campaign_id, recipient_email, recipient_user_id, recipient_name,
		          subject, status, error_message, sent_at, created_at
	`
	row := r.db.QueryRowContext(ctx, query,
		l.CampaignID, l.RecipientEmail, l.RecipientUserID, l.RecipientName,
		l.Subject, l.Status, l.ErrorMessage, l.SentAt,
	)
	result, err := scanLog(row)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return result, nil
}

func (r *EmailRepo) UpdateLog(ctx context.Context, l *domain.EmailLog) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE email_logs SET status = $2, error_message = $3, sent_at = $4
		WHERE id = $1
	`, l.ID, l.Status, l.ErrorMessage, l.SentAt)
	return err
}

type LogFilter struct {
	CampaignID *int64
	Status     *string
	Limit      int
	Offset     int
}

func (r *EmailRepo) ListLogs(ctx context.Context, f LogFilter) ([]*domain.EmailLog, error) {
	var where []string
	var args []any
	i := 1
	if f.CampaignID != nil {
		where = append(where, fmt.Sprintf("campaign_id = $%d", i))
		args = append(args, *f.CampaignID)
		i++
	}
	if f.Status != nil {
		where = append(where, fmt.Sprintf("status = $%d", i))
		args = append(args, *f.Status)
		i++
	}
	q := `SELECT id, campaign_id, recipient_email, recipient_user_id, recipient_name,
	             subject, status, error_message, sent_at, created_at
	      FROM email_logs`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY created_at DESC"
	if f.Limit > 0 {
		q += fmt.Sprintf(" LIMIT $%d", i)
		args = append(args, f.Limit)
		i++
	}
	if f.Offset > 0 {
		q += fmt.Sprintf(" OFFSET $%d", i)
		args = append(args, f.Offset)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list logs: %w", err)
	}
	defer rows.Close()

	var logs []*domain.EmailLog
	for rows.Next() {
		l, err := scanLogRow(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// ─── Campaigns ───────────────────────────────────────────────────────────────

const campaignColumns = `
	id, subject, body, recipient_type, recipient_ids, custom_emails,
	total_recipients, sent_count, failed_count, status, created_by,
	created_at, sent_at
`

func (r *EmailRepo) CreateCampaign(ctx context.Context, c *domain.EmailCampaign) (*domain.EmailCampaign, error) {
	query := `
		INSERT INTO email_campaigns
			(subject, body, recipient_type, recipient_ids, custom_emails, status, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING ` + campaignColumns
	row := r.db.QueryRowContext(ctx, query,
		c.Subject, c.Body, c.RecipientType, c.RecipientIDs, c.CustomEmails,
		c.Status, c.CreatedBy,
	)
	result, err := scanCampaign(row)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return result, nil
}

func (r *EmailRepo) GetCampaign(ctx context.Context, id int64) (*domain.EmailCampaign, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+campaignColumns+` FROM email_campaigns WHERE id = $1`, id)
	return scanCampaign(row)
}

func (r *EmailRepo) UpdateCampaign(ctx context.Context, c *domain.EmailCampaign) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE email_campaigns SET
			recipient_type = $2, recipient_ids = $3, custom_emails = $4,
			total_recipients = $5, sent_count = $6, failed_count = $7,
			status = $8, sent_at = $9
		WHERE id = $1
	`, c.ID, c.RecipientType, c.RecipientIDs, c.CustomEmails,
		c.TotalRecipients, c.SentCount, c.FailedCount, c.Status, c.SentAt)
	return err
}

func (r *EmailRepo) ListCampaigns(ctx context.Context, limit, offset int) ([]*domain.EmailCampaign, error) {
	q := `SELECT ` + campaignColumns + ` FROM email_campaigns ORDER BY created_at DESC`
	args := []any{}
	if limit > 0 {
		q += " LIMIT $1"
		args = append(args, limit)
		if offset > 0 {
			q += " OFFSET $2"
			args = append(args, offset)
		}
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list campaigns: %w", err)
	}
	defer rows.Close()

	var out []*domain.EmailCampaign
	for rows.Next() {
		c, err := scanCampaignRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *EmailRepo) IncrementCampaignStats(ctx context.Context, id int64, sentDelta, failedDelta int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE email_campaigns SET
			sent_count = sent_count + $2,
			failed_count = failed_count + $3
		WHERE id = $1
	`, id, sentDelta, failedDelta)
	return err
}

// ─── Scanners ────────────────────────────────────────────────────────────────

type emailScanner interface {
	Scan(dest ...any) error
}

func scanTemplate(row *sql.Row) (*domain.EmailTemplate, error) {
	t, err := scanTemplateRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return t, err
}

func scanTemplateRow(s emailScanner) (*domain.EmailTemplate, error) {
	var (
		t         domain.EmailTemplate
		createdBy sql.NullInt64
		updatedAt sql.NullTime
	)
	if err := s.Scan(&t.ID, &t.Name, &t.Subject, &t.Body, &createdBy, &t.CreatedAt, &updatedAt); err != nil {
		return nil, err
	}
	if createdBy.Valid {
		t.CreatedBy = &createdBy.Int64
	}
	if updatedAt.Valid {
		t.UpdatedAt = &updatedAt.Time
	}
	return &t, nil
}

func scanTemplateRows(rows *sql.Rows) ([]*domain.EmailTemplate, error) {
	var out []*domain.EmailTemplate
	for rows.Next() {
		t, err := scanTemplateRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func scanJob(row *sql.Row) (*domain.EmailJob, error) {
	j, err := scanJobRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return j, err
}

func scanJobRow(s emailScanner) (*domain.EmailJob, error) {
	var (
		j          domain.EmailJob
		toName     sql.NullString
		variables  []byte
		campaignID sql.NullInt64
	)
	if err := s.Scan(
		&j.ID, &j.ToEmail, &toName, &j.TemplateName, &j.Subject, &variables,
		&campaignID, &j.Attempts, &j.MaxAttempts, &j.ScheduledAt, &j.Status, &j.CreatedAt,
	); err != nil {
		return nil, err
	}
	if toName.Valid {
		j.ToName = &toName.String
	}
	if campaignID.Valid {
		j.CampaignID = &campaignID.Int64
	}
	if len(variables) > 0 {
		_ = json.Unmarshal(variables, &j.Variables)
	}
	return &j, nil
}

func scanLog(row *sql.Row) (*domain.EmailLog, error) {
	l, err := scanLogRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return l, err
}

func scanLogRow(s emailScanner) (*domain.EmailLog, error) {
	var (
		l        domain.EmailLog
		campaign sql.NullInt64
		userID   sql.NullInt64
		name     sql.NullString
		errMsg   sql.NullString
		sentAt   sql.NullTime
	)
	if err := s.Scan(
		&l.ID, &campaign, &l.RecipientEmail, &userID, &name,
		&l.Subject, &l.Status, &errMsg, &sentAt, &l.CreatedAt,
	); err != nil {
		return nil, err
	}
	if campaign.Valid {
		l.CampaignID = &campaign.Int64
	}
	if userID.Valid {
		l.RecipientUserID = &userID.Int64
	}
	if name.Valid {
		l.RecipientName = &name.String
	}
	if errMsg.Valid {
		l.ErrorMessage = &errMsg.String
	}
	if sentAt.Valid {
		l.SentAt = &sentAt.Time
	}
	return &l, nil
}

func scanCampaign(row *sql.Row) (*domain.EmailCampaign, error) {
	c, err := scanCampaignRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return c, err
}

func scanCampaignRow(s emailScanner) (*domain.EmailCampaign, error) {
	var (
		c            domain.EmailCampaign
		recipientIDs sql.NullString
		customEmails sql.NullString
		createdBy    sql.NullInt64
		sentAt       sql.NullTime
	)
	if err := s.Scan(
		&c.ID, &c.Subject, &c.Body, &c.RecipientType, &recipientIDs, &customEmails,
		&c.TotalRecipients, &c.SentCount, &c.FailedCount, &c.Status, &createdBy,
		&c.CreatedAt, &sentAt,
	); err != nil {
		return nil, err
	}
	if recipientIDs.Valid {
		c.RecipientIDs = &recipientIDs.String
	}
	if customEmails.Valid {
		c.CustomEmails = &customEmails.String
	}
	if createdBy.Valid {
		c.CreatedBy = &createdBy.Int64
	}
	if sentAt.Valid {
		c.SentAt = &sentAt.Time
	}
	return &c, nil
}

func (r *EmailRepo) DeleteCampaign(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM email_campaigns WHERE id = $1`, id)
	if err != nil {
		return dberr.Translate(fmt.Errorf("delete campaign: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *EmailRepo) ListRecipientsByRole(ctx context.Context, role string) ([]*domain.EmailRecipient, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, email, first_name, last_name
		FROM users
		WHERE is_active = true AND is_suspended = false AND role = $1
		ORDER BY id
	`, role)
	if err != nil {
		return nil, fmt.Errorf("list recipients by role: %w", err)
	}
	defer rows.Close()
	return scanRecipients(rows)
}

func (r *EmailRepo) ListAllRecipients(ctx context.Context) ([]*domain.EmailRecipient, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, email, first_name, last_name
		FROM users
		WHERE is_active = true AND is_suspended = false
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("list all recipients: %w", err)
	}
	defer rows.Close()
	return scanRecipients(rows)
}

func (r *EmailRepo) ListRecipientsByIDs(ctx context.Context, ids []int64) ([]*domain.EmailRecipient, error) {
	if len(ids) == 0 {
		return []*domain.EmailRecipient{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, email, first_name, last_name
		FROM users
		WHERE id = ANY($1) AND is_active = true AND is_suspended = false
		ORDER BY id
	`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("list recipients by ids: %w", err)
	}
	defer rows.Close()
	return scanRecipients(rows)
}

func scanRecipients(rows *sql.Rows) ([]*domain.EmailRecipient, error) {
	out := make([]*domain.EmailRecipient, 0)
	for rows.Next() {
		var (
			rec   domain.EmailRecipient
			first sql.NullString
			last  sql.NullString
		)
		if err := rows.Scan(&rec.ID, &rec.Email, &first, &last); err != nil {
			return nil, err
		}
		if first.Valid {
			rec.FirstName = &first.String
		}
		if last.Valid {
			rec.LastName = &last.String
		}
		out = append(out, &rec)
	}
	return out, rows.Err()
}

var _ = time.Now
