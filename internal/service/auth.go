package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Hisaalu/rogele/internal/auth"
	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository"
)

type RegisterInput struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
	Phone     string
	Role      domain.UserRole
	ClassID   *int64
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthResult struct {
	User         *domain.User
	AccessToken  string
	RefreshToken string
}

type AuthService struct {
	users  repository.UserRepository
	tokens *auth.TokenManager
}

func NewAuthService(users repository.UserRepository, tokens *auth.TokenManager) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.Email == "" || !strings.Contains(in.Email, "@") {
		return nil, fmt.Errorf("%w: invalid email", domain.ErrInvalidInput)
	}
	if len(in.Password) < 8 {
		return nil, fmt.Errorf("%w: password must be at least 8 characters", domain.ErrInvalidInput)
	}
	if in.Role == "" {
		in.Role = domain.RoleExternal
	}
	if !in.Role.IsValid() {
		return nil, fmt.Errorf("%w: invalid role", domain.ErrInvalidInput)
	}

	if _, err := s.users.GetByEmail(ctx, in.Email); err == nil {
		return nil, fmt.Errorf("%w: email already registered", domain.ErrConflict)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u := &domain.User{
		Email:         in.Email,
		Password:      hash,
		Role:          in.Role,
		ClassID:       in.ClassID,
		IsActive:      true,
		IsSuspended:   false,
		EmailVerified: false,
	}
	if in.FirstName != "" {
		u.FirstName = &in.FirstName
	}
	if in.LastName != "" {
		u.LastName = &in.LastName
	}
	if in.Phone != "" {
		u.Phone = &in.Phone
	}

	created, err := s.users.Create(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.issueTokens(created)
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (*AuthResult, error) {
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))

	u, err := s.users.GetByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("%w: invalid credentials", domain.ErrUnauthorized)
		}
		return nil, err
	}

	if !u.IsActive || u.IsSuspended {
		return nil, fmt.Errorf("%w: account is disabled", domain.ErrForbidden)
	}

	if err := auth.CheckPassword(u.Password, in.Password); err != nil {
		return nil, fmt.Errorf("%w: invalid credentials", domain.ErrUnauthorized)
	}

	_ = s.users.UpdateLastLogin(ctx, u.ID)

	return s.issueTokens(u)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	claims, err := s.tokens.Verify(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid refresh token", domain.ErrUnauthorized)
	}
	if claims.Type != auth.TokenRefresh {
		return nil, fmt.Errorf("%w: refresh token required", domain.ErrUnauthorized)
	}

	u, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if !u.IsActive || u.IsSuspended {
		return nil, fmt.Errorf("%w: account is disabled", domain.ErrForbidden)
	}

	return s.issueTokens(u)
}

func (s *AuthService) issueTokens(u *domain.User) (*AuthResult, error) {
	access, err := s.tokens.GenerateAccess(u)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	refresh, err := s.tokens.GenerateRefresh(u)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	return &AuthResult{
		User:         u,
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}
