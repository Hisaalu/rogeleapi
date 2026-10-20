package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Hisaalu/rogele/internal/auth"
	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/dto"
	"github.com/Hisaalu/rogele/internal/httputil"
	"github.com/Hisaalu/rogele/internal/mailer"
	"github.com/Hisaalu/rogele/internal/repository"
	"github.com/Hisaalu/rogele/internal/server/middleware"
)

type AdminUserHandler struct {
	users    repository.UserRepository
	quiz     repository.QuizRepository
	homework repository.HomeworkRepository
	resets   repository.PasswordResetRepository
	mailer   *mailer.Service
	webURL   string
	log      *slog.Logger
}

func NewAdminUserHandler(
	users repository.UserRepository,
	quiz repository.QuizRepository,
	homework repository.HomeworkRepository,
	resets repository.PasswordResetRepository,
	mailerSvc *mailer.Service,
	webURL string,
	log *slog.Logger,
) *AdminUserHandler {
	return &AdminUserHandler{
		users:    users,
		quiz:     quiz,
		homework: homework,
		resets:   resets,
		mailer:   mailerSvc,
		webURL:   webURL,
		log:      log,
	}
}

func (h *AdminUserHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return false
	}
	if claims.Role != domain.RoleAdmin {
		httputil.Error(w, http.StatusForbidden, "FORBIDDEN", "admin only")
		return false
	}
	return true
}

func (h *AdminUserHandler) QuizAttempts(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	attempts, err := h.quiz.ListAttemptsByUser(r.Context(), id)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusOK, dto.AttemptsFromDomain(attempts))
}

func (h *AdminUserHandler) HomeworkSubmissions(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	submissions, err := h.homework.ListSubmissionsByStudent(r.Context(), id)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusOK, dto.SubmissionsFromDomain(submissions, nil))
}

func (h *AdminUserHandler) ForcePasswordReset(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	u, err := h.users.GetByID(r.Context(), id)
	if err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	token, err := generateAdminResetToken()
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "TOKEN_FAILED", "could not generate token")
		return
	}

	pr := &domain.PasswordReset{
		UserID:    u.ID,
		Token:     token,
		ExpiresAt: time.Now().UTC().Add(30 * time.Minute),
	}
	if _, err := h.resets.Create(r.Context(), pr); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	if h.mailer != nil {
		resetLink := strings.TrimRight(h.webURL, "/") + "/reset-password?token=" + token
		firstName := "there"
		if u.FirstName != nil && *u.FirstName != "" {
			firstName = *u.FirstName
		}
		vars := map[string]any{
			"FirstName": firstName,
			"ResetLink": resetLink,
		}
		_, _ = h.mailer.Enqueue(r.Context(), "password_reset", u.Email, u.FirstName, vars)
	}

	httputil.JSON(w, http.StatusOK, map[string]string{
		"message": "a password reset link has been sent to " + u.Email,
	})
}

func generateAdminResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

type SetPasswordRequest struct {
	Password string `json:"password"`
}

func (h *AdminUserHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	var req SetPasswordRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if len(req.Password) < 8 {
		httputil.Error(w, http.StatusBadRequest, "INVALID_INPUT", "password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "HASH_FAILED", "could not hash password")
		return
	}

	if err := h.users.UpdatePassword(r.Context(), id, hash); err != nil {
		httputil.DomainError(w, err, h.log)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{
		"message": "password updated",
	})
}

var _ = context.Background
