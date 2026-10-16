package dto

import (
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
)

type AdminOverviewResponse struct {
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

type ActivityItemResponse struct {
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

type RecentUserResponse struct {
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

type RecentPaymentResponse struct {
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

func AdminOverviewFromDomain(o *domain.AdminOverview) AdminOverviewResponse {
	return AdminOverviewResponse{
		TotalUsers:          o.TotalUsers,
		UsersByRole:         o.UsersByRole,
		NewUsersLast30Days:  o.NewUsersLast30Days,
		ActiveSubscriptions: o.ActiveSubscriptions,
		ActiveTrials:        o.ActiveTrials,
		TotalClasses:        o.TotalClasses,
		TotalSubjects:       o.TotalSubjects,
		TotalLessons:        o.TotalLessons,
		TotalHomework:       o.TotalHomework,
		TotalQuizzes:        o.TotalQuizzes,
		RevenueThisMonth:    o.RevenueThisMonth,
		RevenueLastMonth:    o.RevenueLastMonth,
		TotalPayments:       o.TotalPayments,
		PaymentsByStatus:    o.PaymentsByStatus,
	}
}

func ActivityItemsFromDomain(items []*domain.AdminActivityItem) []ActivityItemResponse {
	out := make([]ActivityItemResponse, 0, len(items))
	for _, i := range items {
		out = append(out, ActivityItemResponse{
			ID:            i.ID,
			Action:        i.Action,
			Description:   i.Description,
			CreatedAt:     i.CreatedAt,
			UserID:        i.UserID,
			UserEmail:     i.UserEmail,
			UserFirstName: i.UserFirstName,
			UserLastName:  i.UserLastName,
			UserRole:      i.UserRole,
		})
	}
	return out
}

func RecentUsersFromDomain(items []*domain.AdminUserSummary) []RecentUserResponse {
	out := make([]RecentUserResponse, 0, len(items))
	for _, i := range items {
		out = append(out, RecentUserResponse{
			ID:          i.ID,
			Email:       i.Email,
			FirstName:   i.FirstName,
			LastName:    i.LastName,
			Role:        i.Role,
			IsActive:    i.IsActive,
			IsSuspended: i.IsSuspended,
			CreatedAt:   i.CreatedAt,
			ClassID:     i.ClassID,
			ClassName:   i.ClassName,
		})
	}
	return out
}

func RecentPaymentsFromDomain(items []*domain.AdminPaymentSummary) []RecentPaymentResponse {
	out := make([]RecentPaymentResponse, 0, len(items))
	for _, i := range items {
		out = append(out, RecentPaymentResponse{
			ID:            i.ID,
			UserID:        i.UserID,
			Amount:        i.Amount,
			PlanType:      i.PlanType,
			PaymentMethod: i.PaymentMethod,
			TransactionID: i.TransactionID,
			Status:        i.Status,
			CreatedAt:     i.CreatedAt,
			PaymentDate:   i.PaymentDate,
			UserEmail:     i.UserEmail,
			UserFirstName: i.UserFirstName,
			UserLastName:  i.UserLastName,
		})
	}
	return out
}

type ClassLearnerCountResponse struct {
	ClassID      int64  `json:"class_id"`
	ClassName    string `json:"class_name"`
	LearnerCount int    `json:"learner_count"`
}

func ClassLearnerCountsFromDomain(items []*domain.ClassLearnerCount) []ClassLearnerCountResponse {
	out := make([]ClassLearnerCountResponse, 0, len(items))
	for _, i := range items {
		out = append(out, ClassLearnerCountResponse{
			ClassID:      i.ClassID,
			ClassName:    i.ClassName,
			LearnerCount: i.LearnerCount,
		})
	}
	return out
}
