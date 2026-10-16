package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Hisaalu/rogele/internal/auth"
	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/mailer"
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

type ForgotPasswordInput struct {
	Email string
}

type ResetPasswordInput struct {
	Token    string
	Password string
}

type AuthResult struct {
	User         *domain.User
	AccessToken  string
	RefreshToken string
}

const (
	resetTokenTTL        = 30 * time.Minute
	resetRateLimit       = 3
	resetRateLimitWindow = 15 * time.Minute
)

type AuthService struct {
	users      repository.UserRepository
	resetRepo  repository.PasswordResetRepository
	mailer     *mailer.Service
	activity   *ActivityService
	webBaseURL string
	tokens     *auth.TokenManager
}

func NewAuthService(
	users repository.UserRepository,
	resetRepo repository.PasswordResetRepository,
	mailerSvc *mailer.Service,
	activity *ActivityService,
	webBaseURL string,
	tokens *auth.TokenManager,
) *AuthService {
	return &AuthService{
		users:      users,
		resetRepo:  resetRepo,
		mailer:     mailerSvc,
		activity:   activity,
		webBaseURL: webBaseURL,
		tokens:     tokens,
	}
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
	s.activity.Log(ctx, u.ID, "login", "signed in")

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

func (s *AuthService) ForgotPassword(ctx context.Context, in ForgotPasswordInput) error {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	if email == "" {
		return fmt.Errorf("%w: email is required", domain.ErrInvalidInput)
	}

	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}

	if s.resetRepo == nil {
		return fmt.Errorf("%w: password reset is not available", domain.ErrInternal)
	}

	recent, err := s.resetRepo.CountRecent(ctx, u.ID, resetRateLimitWindow)
	if err != nil {
		return err
	}
	if recent >= resetRateLimit {
		return nil
	}

	token, err := generateResetToken()
	if err != nil {
		return err
	}

	pr := &domain.PasswordReset{
		UserID:    u.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(resetTokenTTL),
	}
	if _, err := s.resetRepo.Create(ctx, pr); err != nil {
		return err
	}

	if s.mailer != nil {
		resetLink := strings.TrimRight(s.webBaseURL, "/") + "/reset-password?token=" + token
		vars := map[string]any{
			"FirstName": derefOr(u.FirstName, "there"),
			"ResetLink": resetLink,
		}
		if _, err := s.mailer.Enqueue(ctx, "password_reset", u.Email, u.FirstName, vars); err != nil {
			return err
		}
	}

	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, in ResetPasswordInput) error {
	if in.Token == "" {
		return fmt.Errorf("%w: token is required", domain.ErrInvalidInput)
	}
	if len(in.Password) < 8 {
		return fmt.Errorf("%w: password must be at least 8 characters", domain.ErrInvalidInput)
	}

	pr, err := s.resetRepo.GetByToken(ctx, in.Token)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("%w: invalid or expired token", domain.ErrInvalidInput)
		}
		return err
	}
	if pr.Used {
		return fmt.Errorf("%w: token has already been used", domain.ErrInvalidInput)
	}
	if time.Now().After(pr.ExpiresAt) {
		return fmt.Errorf("%w: token has expired", domain.ErrInvalidInput)
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.users.UpdatePassword(ctx, pr.UserID, hash); err != nil {
		return err
	}

	return s.resetRepo.MarkUsed(ctx, pr.ID)
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

func generateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func derefOr(s *string, fallback string) string {
	if s == nil || *s == "" {
		return fallback
	}
	return *s
}
