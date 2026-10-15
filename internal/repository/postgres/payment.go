package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Hisaalu/rogele/internal/dberr"
	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository"
)

type PaymentRepo struct {
	db *sql.DB
}

func NewPaymentRepo(db *sql.DB) *PaymentRepo {
	return &PaymentRepo{db: db}
}

const paymentColumns = `
	id, user_id, subscription_id, amount, plan_type, payment_method,
	transaction_id, payment_gateway_response, phone_number, status,
	payment_date, created_at
`

func (r *PaymentRepo) Create(ctx context.Context, p *domain.Payment) (*domain.Payment, error) {
	var gatewayJSON []byte
	if p.PaymentGatewayResponse != nil {
		gatewayJSON, _ = json.Marshal(p.PaymentGatewayResponse)
	}

	query := `
		INSERT INTO payments (
			user_id, subscription_id, amount, plan_type, payment_method,
			transaction_id, payment_gateway_response, phone_number, status,
			payment_date
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING ` + paymentColumns

	row := r.db.QueryRowContext(ctx, query,
		p.UserID, p.SubscriptionID, p.Amount, p.PlanType, p.PaymentMethod,
		p.TransactionID, gatewayJSON, p.PhoneNumber, p.Status,
		p.PaymentDate,
	)
	result, err := scanPayment(row)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return result, nil
}

func (r *PaymentRepo) GetByID(ctx context.Context, id int64) (*domain.Payment, error) {
	query := `SELECT ` + paymentColumns + ` FROM payments WHERE id = $1`
	return scanPayment(r.db.QueryRowContext(ctx, query, id))
}

func (r *PaymentRepo) GetByTransactionID(ctx context.Context, txnID string) (*domain.Payment, error) {
	query := `SELECT ` + paymentColumns + ` FROM payments WHERE transaction_id = $1`
	return scanPayment(r.db.QueryRowContext(ctx, query, txnID))
}

func (r *PaymentRepo) List(ctx context.Context, f repository.PaymentFilter) ([]*domain.Payment, error) {
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

	query := `SELECT ` + paymentColumns + ` FROM payments`
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
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.Payment, 0, 16)
	for rows.Next() {
		p, err := scanPaymentRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

func (r *PaymentRepo) Update(ctx context.Context, p *domain.Payment) error {
	var gatewayJSON []byte
	if p.PaymentGatewayResponse != nil {
		gatewayJSON, _ = json.Marshal(p.PaymentGatewayResponse)
	}

	query := `
		UPDATE payments SET
			subscription_id = $1, amount = $2, plan_type = $3,
			payment_method = $4, transaction_id = $5,
			payment_gateway_response = $6, phone_number = $7,
			status = $8, payment_date = $9
		WHERE id = $10
	`
	res, err := r.db.ExecContext(ctx, query,
		p.SubscriptionID, p.Amount, p.PlanType,
		p.PaymentMethod, p.TransactionID,
		gatewayJSON, p.PhoneNumber,
		p.Status, p.PaymentDate,
		p.ID,
	)
	if err != nil {
		return dberr.Translate(fmt.Errorf("update payment: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const historyColumns = `
	id, user_id, subscription_id, amount, payment_type, from_plan, to_plan,
	status, payment_method, transaction_id, payment_data, created_at
`

func (r *PaymentRepo) CreateHistory(ctx context.Context, h *domain.PaymentHistory) (*domain.PaymentHistory, error) {
	query := `
		INSERT INTO payment_history (
			user_id, subscription_id, amount, payment_type, from_plan, to_plan,
			status, payment_method, transaction_id, payment_data
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING ` + historyColumns

	var (
		out       domain.PaymentHistory
		subID     sql.NullInt64
		fromPlan  sql.NullString
		toPlan    sql.NullString
		payMethod sql.NullString
		txnID     sql.NullString
		payData   sql.NullString
	)
	err := r.db.QueryRowContext(ctx, query,
		h.UserID, h.SubscriptionID, h.Amount, h.PaymentType,
		h.FromPlan, h.ToPlan, h.Status, h.PaymentMethod,
		h.TransactionID, h.PaymentData,
	).Scan(&out.ID, &out.UserID, &subID, &out.Amount, &out.PaymentType,
		&fromPlan, &toPlan, &out.Status, &payMethod, &txnID, &payData, &out.CreatedAt)
	if err != nil {
		return nil, dberr.Translate(fmt.Errorf("create history: %w", err))
	}
	if subID.Valid {
		out.SubscriptionID = &subID.Int64
	}
	if fromPlan.Valid {
		out.FromPlan = &fromPlan.String
	}
	if toPlan.Valid {
		out.ToPlan = &toPlan.String
	}
	if payMethod.Valid {
		out.PaymentMethod = &payMethod.String
	}
	if txnID.Valid {
		out.TransactionID = &txnID.String
	}
	if payData.Valid {
		out.PaymentData = &payData.String
	}
	return &out, nil
}

func (r *PaymentRepo) ListHistoryByUser(ctx context.Context, userID int64) ([]*domain.PaymentHistory, error) {
	query := `SELECT ` + historyColumns + ` FROM payment_history WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	defer rows.Close()
	return scanHistoryRows(rows)
}

func (r *PaymentRepo) ListHistoryBySubscription(ctx context.Context, subscriptionID int64) ([]*domain.PaymentHistory, error) {
	query := `SELECT ` + historyColumns + ` FROM payment_history WHERE subscription_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	defer rows.Close()
	return scanHistoryRows(rows)
}

func scanHistoryRows(rows *sql.Rows) ([]*domain.PaymentHistory, error) {
	results := make([]*domain.PaymentHistory, 0, 16)
	for rows.Next() {
		var (
			h         domain.PaymentHistory
			subID     sql.NullInt64
			fromPlan  sql.NullString
			toPlan    sql.NullString
			payMethod sql.NullString
			txnID     sql.NullString
			payData   sql.NullString
		)
		if err := rows.Scan(&h.ID, &h.UserID, &subID, &h.Amount, &h.PaymentType,
			&fromPlan, &toPlan, &h.Status, &payMethod, &txnID, &payData, &h.CreatedAt); err != nil {
			return nil, err
		}
		if subID.Valid {
			h.SubscriptionID = &subID.Int64
		}
		if fromPlan.Valid {
			h.FromPlan = &fromPlan.String
		}
		if toPlan.Valid {
			h.ToPlan = &toPlan.String
		}
		if payMethod.Valid {
			h.PaymentMethod = &payMethod.String
		}
		if txnID.Valid {
			h.TransactionID = &txnID.String
		}
		if payData.Valid {
			h.PaymentData = &payData.String
		}
		results = append(results, &h)
	}
	return results, rows.Err()
}

type paymentScanner interface {
	Scan(dest ...any) error
}

func scanPayment(row *sql.Row) (*domain.Payment, error) {
	p, err := scanPaymentRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return p, err
}

func scanPaymentRow(s paymentScanner) (*domain.Payment, error) {
	var (
		p          domain.Payment
		userID     sql.NullInt64
		subID      sql.NullInt64
		amount     sql.NullFloat64
		planType   sql.NullString
		method     sql.NullString
		txnID      sql.NullString
		gatewayRaw []byte
		phone      sql.NullString
		payDate    sql.NullTime
	)
	err := s.Scan(
		&p.ID, &userID, &subID, &amount, &planType, &method,
		&txnID, &gatewayRaw, &phone, &p.Status,
		&payDate, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if userID.Valid {
		p.UserID = &userID.Int64
	}
	if subID.Valid {
		p.SubscriptionID = &subID.Int64
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
	if len(gatewayRaw) > 0 {
		var gateway map[string]any
		if err := json.Unmarshal(gatewayRaw, &gateway); err == nil {
			p.PaymentGatewayResponse = gateway
		}
	}
	if phone.Valid {
		p.PhoneNumber = &phone.String
	}
	if payDate.Valid {
		p.PaymentDate = &payDate.Time
	}
	return &p, nil
}

func (r *PaymentRepo) ListWithUsers(ctx context.Context, f repository.PaymentFilter) ([]*domain.Payment, error) {
	var where []string
	var args []any
	i := 1

	if f.UserID != nil {
		where = append(where, fmt.Sprintf("p.user_id = $%d", i))
		args = append(args, *f.UserID)
		i++
	}
	if f.Status != nil {
		where = append(where, fmt.Sprintf("p.status = $%d", i))
		args = append(args, *f.Status)
		i++
	}

	query := `
		SELECT
			p.id, p.user_id, p.subscription_id, p.amount, p.plan_type,
			p.payment_method, p.transaction_id, p.payment_gateway_response,
			p.phone_number, p.status, p.payment_date, p.created_at,
			u.email, u.first_name, u.last_name
		FROM payments p
		LEFT JOIN users u ON u.id = p.user_id
	`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY p.created_at DESC"

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
		return nil, fmt.Errorf("list payments with users: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.Payment, 0, 16)
	for rows.Next() {
		var (
			p          domain.Payment
			userID     sql.NullInt64
			subID      sql.NullInt64
			amount     sql.NullFloat64
			planType   sql.NullString
			method     sql.NullString
			txnID      sql.NullString
			gatewayRaw []byte
			phone      sql.NullString
			payDate    sql.NullTime
			email      sql.NullString
			firstName  sql.NullString
			lastName   sql.NullString
		)
		if err := rows.Scan(
			&p.ID, &userID, &subID, &amount, &planType, &method,
			&txnID, &gatewayRaw, &phone, &p.Status, &payDate, &p.CreatedAt,
			&email, &firstName, &lastName,
		); err != nil {
			return nil, err
		}
		if userID.Valid {
			p.UserID = &userID.Int64
		}
		if subID.Valid {
			p.SubscriptionID = &subID.Int64
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
		if len(gatewayRaw) > 0 {
			var gateway map[string]any
			if err := json.Unmarshal(gatewayRaw, &gateway); err == nil {
				p.PaymentGatewayResponse = gateway
			}
		}
		if phone.Valid {
			p.PhoneNumber = &phone.String
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
		results = append(results, &p)
	}
	return results, rows.Err()
}

var _ repository.PaymentRepository = (*PaymentRepo)(nil)
