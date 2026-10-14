package dto

import (
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
)

type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Role      string `json:"role,omitempty"`
	ClassID   *int64 `json:"class_id,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID                 int64      `json:"id"`
	RegistrationNumber *string    `json:"registration_number,omitempty"`
	Email              string     `json:"email"`
	FirstName          *string    `json:"first_name,omitempty"`
	LastName           *string    `json:"last_name,omitempty"`
	Phone              *string    `json:"phone,omitempty"`
	Bio                *string    `json:"bio,omitempty"`
	Qualification      *string    `json:"qualification,omitempty"`
	Specialization     *string    `json:"specialization,omitempty"`
	Role               string     `json:"role"`
	ClassID            *int64     `json:"class_id,omitempty"`
	ProfilePhoto       *string    `json:"profile_photo,omitempty"`
	IsActive           bool       `json:"is_active"`
	IsSuspended        bool       `json:"is_suspended"`
	EmailVerified      bool       `json:"email_verified"`
	LastLogin          *time.Time `json:"last_login,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"`
}

func FromDomain(u *domain.User) UserResponse {
	return UserResponse{
		ID:                 u.ID,
		RegistrationNumber: u.RegistrationNumber,
		Email:              u.Email,
		FirstName:          u.FirstName,
		LastName:           u.LastName,
		Phone:              u.Phone,
		Bio:                u.Bio,
		Qualification:      u.Qualification,
		Specialization:     u.Specialization,
		Role:               string(u.Role),
		ClassID:            u.ClassID,
		ProfilePhoto:       u.ProfilePhoto,
		IsActive:           u.IsActive,
		IsSuspended:        u.IsSuspended,
		EmailVerified:      u.EmailVerified,
		LastLogin:          u.LastLogin,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}
}
