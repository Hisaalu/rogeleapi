package server

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/Hisaalu/rogele/internal/auth"
	"github.com/Hisaalu/rogele/internal/handler"
	"github.com/Hisaalu/rogele/internal/repository"
	"github.com/Hisaalu/rogele/internal/server/middleware"
	"github.com/Hisaalu/rogele/internal/service"
)

type RouterDeps struct {
	Log     *slog.Logger
	Version string
	DB      *sql.DB
	Repos   *repository.Repositories
	Tokens  *auth.TokenManager
	AuthSvc *service.AuthService
	UserSvc *service.UserService
}

func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(deps.Log))
	r.Use(middleware.Recover(deps.Log))
	r.Use(middleware.CORS)
	r.Use(chimw.Compress(5))

	health := handler.NewHealthHandler(deps.Version)
	r.Get("/health", health.Check)

	r.Route("/api/v1", func(r chi.Router) {
		authHandler := handler.NewAuthHandler(deps.AuthSvc, deps.UserSvc, deps.Tokens, deps.Log)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)

			r.Group(func(r chi.Router) {
				r.Use(middleware.Authenticate(deps.Tokens))
				r.Get("/me", authHandler.Me)
			})
		})
	})

	return r
}
