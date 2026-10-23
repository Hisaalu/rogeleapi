package handler

import (
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"net/http"

	"github.com/Hisaalu/rogele/internal/auth"
	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/dto"
	"github.com/Hisaalu/rogele/internal/httputil"
	"github.com/Hisaalu/rogele/internal/repository"
	"github.com/Hisaalu/rogele/internal/server/middleware"
	"github.com/Hisaalu/rogele/internal/service"
)

type AuthHandler struct {
	authSvc *service.AuthService
	users   repository.UserRepository
	tokens  *auth.TokenManager
	log     *slog.Logger
}

func NewAuthHandler(
	authSvc *service.AuthService,
	users repository.UserRepository,
	tokens *auth.TokenManager,
	log *slog.Logger,
) *AuthHandler {
	return &AuthHandler{
		authSvc: authSvc,
		users:   users,
		tokens:  tokens,
		log:     log,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	role := domain.UserRole(req.Role)
	if role == "" {
		role = domain.RoleExternal
	}

	result, err := h.authSvc.Register(r.Context(), service.RegisterInput{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Role:      role,
		ClassID:   req.ClassID,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusCreated, dto.AuthResponse{
		User:         dto.FromDomain(result.User),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(h.tokens.AccessTTL().Seconds()),
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	result, err := h.authSvc.Login(r.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusOK, dto.AuthResponse{
		User:         dto.FromDomain(result.User),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(h.tokens.AccessTTL().Seconds()),
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	result, err := h.authSvc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusOK, dto.AuthResponse{
		User:         dto.FromDomain(result.User),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(h.tokens.AccessTTL().Seconds()),
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "no auth claims")
		return
	}

	user, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusOK, dto.FromDomain(user))
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ForgotPasswordRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if err := h.authSvc.ForgotPassword(r.Context(), service.ForgotPasswordInput{
		Email: req.Email,
	}); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{
		"message": "if an account exists for that email, a reset link has been sent",
	})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ResetPasswordRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if err := h.authSvc.ResetPassword(r.Context(), service.ResetPasswordInput{
		Token:    req.Token,
		Password: req.Password,
	}); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{
		"message": "password updated",
	})
}

func (h *AuthHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	// ADDED: Bio, Qualification, Specialization
	var req struct {
		FirstName      *string `json:"first_name,omitempty"`
		LastName       *string `json:"last_name,omitempty"`
		Email          *string `json:"email,omitempty"`
		Phone          *string `json:"phone,omitempty"`
		Bio            *string `json:"bio,omitempty"`
		Qualification  *string `json:"qualification,omitempty"`
		Specialization *string `json:"specialization,omitempty"`
		ProfilePhoto   *string `json:"profile_photo,omitempty"`
		ClassID        *int64  `json:"class_id,omitempty"`
	}

	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	u, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	if req.FirstName != nil {
		u.FirstName = req.FirstName
	}
	if req.LastName != nil {
		u.LastName = req.LastName
	}
	if req.Email != nil {
		u.Email = *req.Email
	}
	if req.Phone != nil {
		u.Phone = req.Phone
	}
	if req.Bio != nil {
		u.Bio = req.Bio
	}
	if req.Qualification != nil {
		u.Qualification = req.Qualification
	}
	if req.Specialization != nil {
		u.Specialization = req.Specialization
	}
	if req.ProfilePhoto != nil {
		u.ProfilePhoto = req.ProfilePhoto
	}
	if req.ClassID != nil {
		u.ClassID = req.ClassID
	}

	if err := h.users.Update(r.Context(), u); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{"message": "profile updated successfully"})
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	u, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.CurrentPassword)); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_PASSWORD", "Current password is incorrect!")
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	if err := h.users.UpdatePassword(r.Context(), u.ID, string(newHash)); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{"message": "password updated successfully"})
}
