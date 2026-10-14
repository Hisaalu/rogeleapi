package domain

import "time"

type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleTeacher  UserRole = "teacher"
	RoleLearner  UserRole = "learner"
	RoleExternal UserRole = "external"
)

func (r UserRole) IsValid() bool {
	switch r {
	case RoleAdmin, RoleTeacher, RoleLearner, RoleExternal:
		return true
	}
	return false
}

type User struct {
	ID                 int64      `json:"id"`
	RegistrationNumber *string    `json:"registration_number,omitempty"`
	Email              string     `json:"email"`
	Password           string     `json:"-"`
	FirstName          *string    `json:"first_name,omitempty"`
	LastName           *string    `json:"last_name,omitempty"`
	Phone              *string    `json:"phone,omitempty"`
	Bio                *string    `json:"bio,omitempty"`
	Qualification      *string    `json:"qualification,omitempty"`
	Specialization     *string    `json:"specialization,omitempty"`
	Role               UserRole   `json:"role"`
	ClassID            *int64     `json:"class_id,omitempty"`
	ProfilePhoto       *string    `json:"profile_photo,omitempty"`
	IsActive           bool       `json:"is_active"`
	IsSuspended        bool       `json:"is_suspended"`
	EmailVerified      bool       `json:"email_verified"`
	LastLogin          *time.Time `json:"last_login,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	TrialStartDate     time.Time  `json:"trial_start_date"`
}

func (u *User) FullName() string {
	fn, ln := "", ""
	if u.FirstName != nil {
		fn = *u.FirstName
	}
	if u.LastName != nil {
		ln = *u.LastName
	}
	if fn == "" && ln == "" {
		return ""
	}
	if fn == "" {
		return ln
	}
	if ln == "" {
		return fn
	}
	return fn + " " + ln
}
