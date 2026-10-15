package domain

import "time"

type AdminOverview struct {
	TotalUsers          int            `json:"total_users"`
	UsersByRole         map[string]int `json:"users_by_role"`
	NewUsersLast30Days  int            `json:"new_users_last_30_days"`
	ActiveSubscriptions int            `json:"active_subscriptions"`
	ActiveTrials        int            `json:"active_trials"`
	TotalClasses        int            `json:"total_classes"`
	TotalSubjects       int            `json:"total_subjects"`
	TotalLessons        int            `json:"total_lessons"`
	TotalHomework       int            `json:"total_homework"`
	TotalQuizzes        int            `json:"total_quizzes"`
	RevenueThisMonth    float64        `json:"revenue_this_month"`
	RevenueLastMonth    float64        `json:"revenue_last_month"`
	TotalPayments       int            `json:"total_payments"`
	PaymentsByStatus    map[string]int `json:"payments_by_status"`
}

type AdminActivityItem struct {
	ID            int64     `json:"id"`
	Action        string    `json:"action"`
	Description   *string   `json:"description,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UserID        *int64    `json:"user_id,omitempty"`
	UserEmail     *string   `json:"user_email,omitempty"`
	UserFirstName *string   `json:"user_first_name,omitempty"`
	UserLastName  *string   `json:"user_last_name,omitempty"`
	UserRole      *string   `json:"user_role,omitempty"`
}

type AdminUserSummary struct {
	ID          int64     `json:"id"`
	Email       string    `json:"email"`
	FirstName   *string   `json:"first_name,omitempty"`
	LastName    *string   `json:"last_name,omitempty"`
	Role        string    `json:"role"`
	IsActive    bool      `json:"is_active"`
	IsSuspended bool      `json:"is_suspended"`
	CreatedAt   time.Time `json:"created_at"`
	ClassID     *int64    `json:"class_id,omitempty"`
	ClassName   *string   `json:"class_name,omitempty"`
}

type AdminPaymentSummary struct {
	ID            int64      `json:"id"`
	UserID        *int64     `json:"user_id,omitempty"`
	Amount        *float64   `json:"amount,omitempty"`
	PlanType      *string    `json:"plan_type,omitempty"`
	PaymentMethod *string    `json:"payment_method,omitempty"`
	TransactionID *string    `json:"transaction_id,omitempty"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	PaymentDate   *time.Time `json:"payment_date,omitempty"`
	UserEmail     *string    `json:"user_email,omitempty"`
	UserFirstName *string    `json:"user_first_name,omitempty"`
	UserLastName  *string    `json:"user_last_name,omitempty"`
}
