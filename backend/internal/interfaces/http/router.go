package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
	"github.com/voidwg/control/internal/interfaces/http/handler"
	mw "github.com/voidwg/control/internal/interfaces/http/middleware"
)

type Deps struct {
	Logger *zerolog.Logger
	Tokens port.TokenIssuer
	Auth   *handler.AuthHandler
	User   *handler.UserHandler
	Peer   *handler.PeerHandler
	Server *handler.ServerHandler
	Stats  *handler.StatsHandler
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RealIP)
	r.Use(mw.RequestID)
	r.Use(mw.Recover(d.Logger))
	r.Use(mw.Logger(d.Logger))
	r.Use(mw.Metrics)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-Id", "X-Agent-Token"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Handle("/metrics", promhttp.Handler())

	// public
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/login", d.Auth.Login)
		r.Post("/refresh", d.Auth.Refresh)
	})
	// agents (token, не JWT)
	r.Post("/api/v1/agent/heartbeat", d.Server.Heartbeat)

	// authenticated
	r.Group(func(r chi.Router) {
		r.Use(mw.Auth(d.Tokens))
		r.Get("/api/v1/me", d.Auth.Me)
		r.Patch("/api/v1/me/password", d.Auth.ChangePassword)

		// peers — пользовательские
		r.Route("/api/v1/peers", func(r chi.Router) {
			r.Get("/", d.Peer.List)
			r.Post("/", d.Peer.Create)
			r.Get("/{id}/config", d.Peer.Config)
			r.Delete("/{id}", d.Peer.Delete)
		})

		// admin / operator
		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole(domain.RoleAdmin, domain.RoleOperator))
			r.Get("/api/v1/stats", d.Stats.Get)
			r.Route("/api/v1/users", func(r chi.Router) {
				r.Post("/", d.User.Create)
				r.Get("/", d.User.List)
				r.Delete("/{id}", d.User.Delete)
			})
			r.Route("/api/v1/servers", func(r chi.Router) {
				r.Post("/", d.Server.Create)
				r.Get("/", d.Server.List)
				r.Delete("/{id}", d.Server.Delete)
			})
		})
	})

	return r
}
