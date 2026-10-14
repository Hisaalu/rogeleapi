package crud

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Hisaalu/rogele/internal/auth"
	"github.com/Hisaalu/rogele/internal/server/middleware"
)

func Mount(r chi.Router, cfg *Config, repo *Repository, tokens *auth.TokenManager, log *slog.Logger) {
	for _, res := range cfg.Resources {
		h := NewHandler(res, repo, log)

		r.Route("/"+res.Name, func(r chi.Router) {
			r.Use(middleware.Authenticate(tokens))

			r.Get("/", h.List)
			r.Get("/{id}", h.Get)

			if len(res.WriteRoles) > 0 {
				r.Post("/", h.Create)
				r.Patch("/{id}", h.Update)
				r.Delete("/{id}", h.Delete)
			}
		})
	}
}

var _ http.Handler
