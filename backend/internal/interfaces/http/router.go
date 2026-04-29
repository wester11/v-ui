package http

import (
	"net/http"
	"time"

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
	Config *handler.ConfigHandler
	Stats  *handler.StatsHandler
	Invite *handler.InviteHandler
	Audit     *handler.AuditHandler
	NodeOps   *handler.NodeOpsHandler
	AgentJobs *handler.AgentJobsHandler
	System   *handler.SystemHandler
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
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-Id", "X-Agent-Token", "X-Node-ID", "X-Node-Secret"},
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

	r.Route("/api/v1/auth", func(r chi.Router) {
		r.With(mw.RateLimit(5, 5*time.Minute)).Post("/login", d.Auth.Login)
		r.Post("/refresh", d.Auth.Refresh)
	})

	r.Route("/api/v1/invites", func(r chi.Router) {
		r.With(mw.RateLimit(30, 5*time.Minute)).Get("/{token}", d.Invite.Lookup)
		r.With(mw.RateLimit(10, 5*time.Minute)).Post("/{token}/redeem", d.Invite.Redeem)
	})

	r.Post("/api/v1/agent/register", d.Server.RegisterAgent)
	r.Post("/api/v1/agent/heartbeat", d.Server.Heartbeat)

	// Phase 8: pull-mode job queue для агентов (auth: X-Node-ID + X-Node-Secret).
	r.Get("/api/v1/agent/jobs",       d.AgentJobs.Pull)
	r.Post("/api/v1/agent/jobs/{id}", d.AgentJobs.Submit)
	r.Get("/install-node.sh", d.Server.InstallNodeScript)

	r.Group(func(r chi.Router) {
		r.Use(mw.Auth(d.Tokens))
		r.Get("/api/v1/me", d.Auth.Me)
		r.Patch("/api/v1/me/password", d.Auth.ChangePassword)

		r.Route("/api/v1/peers", func(r chi.Router) {
			r.Get("/", d.Peer.List)
			r.Post("/", d.Peer.Create)
			r.Get("/{id}/config", d.Peer.Config)
			r.Delete("/{id}", d.Peer.Delete)
			r.Patch("/{id}", d.Peer.Patch)
		})

		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole(domain.RoleAdmin, domain.RoleOperator))
			r.Get("/api/v1/stats", d.Stats.Get)

			r.Route("/api/v1/users", func(r chi.Router) {
				r.Post("/", d.User.Create)
				r.Get("/", d.User.List)
				r.Delete("/{id}", d.User.Delete)
				r.Patch("/{id}", d.User.Patch)
			})

			r.Route("/api/v1/servers", func(r chi.Router) {
				r.Post("/", d.Server.Create)
				r.Get("/", d.Server.List)
				r.Delete("/{id}", d.Server.Delete)
				r.Get("/{id}/check", d.Server.Check)
			})

			r.Route("/api/v1/configs", func(r chi.Router) {
				r.Post("/", d.Config.Create)
				r.Post("/{id}/activate", d.Config.Activate)
			})
			r.Route("/api/v1/servers/{serverID}/configs", func(r chi.Router) {
				r.Get("/", d.Config.ListByServer)
			})

			r.Post("/api/v1/admin/servers/{id}/redeploy", d.Peer.Redeploy)
			r.Post("/api/v1/admin/servers/redeploy-all", d.Peer.RedeployAll)
			r.Get("/api/v1/admin/servers/health", d.Peer.Health)

			// Phase 7: config deploy + node remote-control.
			r.Post("/api/v1/admin/servers/{id}/deploy",        d.Config.Deploy)
			r.Post("/api/v1/admin/servers/{id}/restart",       d.NodeOps.Restart)
			r.Post("/api/v1/admin/servers/{id}/rotate-secret", d.NodeOps.RotateSecret)
			r.Get("/api/v1/admin/servers/{id}/metrics",        d.NodeOps.Metrics)

			r.Post("/api/v1/admin/invites", d.Invite.Create)
			r.Get("/api/v1/admin/invites", d.Invite.List)
			r.Delete("/api/v1/admin/invites/{id}", d.Invite.Delete)
			r.Get("/api/v1/admin/system/version", d.System.Version)
			r.Post("/api/v1/admin/system/update",  d.System.Update)
		})

		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole(domain.RoleAdmin))
			r.Get("/api/v1/audit", d.Audit.List)
		})
	})

	return r
}
