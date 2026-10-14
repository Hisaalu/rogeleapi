package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Hisaalu/rogele/internal/auth"
	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/httputil"
)

type ctxKeyAuth string

const AuthClaimsKey ctxKeyAuth = "auth_claims"

func Authenticate(tm *auth.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				httputil.Error(w, http.StatusUnauthorized, "MISSING_TOKEN", "authorization header required")
				return
			}
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				httputil.Error(w, http.StatusUnauthorized, "INVALID_AUTH_HEADER", "expected 'Bearer <token>'")
				return
			}

			claims, err := tm.Verify(parts[1])
			if err != nil {
				httputil.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired token")
				return
			}
			if claims.Type != auth.TokenAccess {
				httputil.Error(w, http.StatusUnauthorized, "WRONG_TOKEN_TYPE", "access token required")
				return
			}

			ctx := context.WithValue(r.Context(), AuthClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns a middleware that only allows the given roles through.
// Use after Authenticate.
func RequireRole(roles ...domain.UserRole) func(http.Handler) http.Handler {
	allowed := make(map[domain.UserRole]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				httputil.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "no auth claims in context")
				return
			}
			if _, ok := allowed[claims.Role]; !ok {
				httputil.Error(w, http.StatusForbidden, "FORBIDDEN", "insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClaimsFromContext extracts the verified JWT claims.
func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	c, ok := ctx.Value(AuthClaimsKey).(*auth.Claims)
	return c, ok
}
