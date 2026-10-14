package repository

import (
	"context"

	"github.com/Hisaalu/rogele/internal/domain"
)

type UserFilter struct {
	Role    *domain.UserRole
	ClassID *int64
	Search  string
	Limit   int
	Offset  int
}

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) (*domain.User, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, u *domain.User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, f UserFilter) ([]*domain.User, error)
	UpdateLastLogin(ctx context.Context, id int64) error
}

type Repositories struct {
	User UserRepository
}
