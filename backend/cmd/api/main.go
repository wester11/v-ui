package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/voidwg/control/internal/application/usecase"
	"github.com/voidwg/control/internal/bootstrap"
	"github.com/voidwg/control/internal/infrastructure/jwtauth"
	"github.com/voidwg/control/internal/infrastructure/logger"
	"github.com/voidwg/control/internal/infrastructure/mtls"
	"github.com/voidwg/control/internal/infrastructure/persistence"
	"github.com/voidwg/control/internal/infrastructure/transport"
	"github.com/voidwg/control/internal/infrastructure/wg"
	netHTTP "github.com/voidwg/control/internal/interfaces/http"
	"github.com/voidwg/control/internal/interfaces/http/handler"
)

func main() {
	cfg := loadConfig()
	log := logger.New(cfg.LogLevel)
	log.Info().Str("addr", cfg.HTTPAddr).Msg("starting void-wg control plane")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := persistence.NewPool(ctx, cfg.DatabaseDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("postgres connect")
	}
	defer pool.Close()

	if err := persistence.EnsureSchemaUpgrades(ctx, pool); err != nil {
		log.Fatal().Err(err).Msg("schema upgrade")
	}

	userRepo := persistence.NewUserRepo(pool)
	peerRepo := persistence.NewPeerRepo(pool)
	srvRepo := persistence.NewServerRepo(pool)
	cfgRepo := persistence.NewConfigRepo(pool)
	invRepo := persistence.NewInviteRepo(pool)
	auditRepo := persistence.NewAuditRepo(pool)
	jobRepo := persistence.NewJobRepo(pool)
	cvRepo := persistence.NewConfigVersionRepo(pool)

	keys := wg.New()
	hasher := jwtauth.NewHasher()
	tokens := jwtauth.New([]byte(cfg.JWTSecret), "void-wg")
	mtlsIssuer, err := mtls.NewIssuer(cfg.MTLSDir)
	if err != nil {
		log.Fatal().Err(err).Msg("mtls issuer init")
	}
	agentTransport := transport.NewAgentClient(cfg.AgentInsecureTLS)

	if err := bootstrap.Admin(ctx, userRepo, hasher, log); err != nil {
		log.Error().Err(err).Msg("bootstrap admin failed")
	}

	authUC := usecase.NewAuth(userRepo, hasher, tokens)
	auditUC := usecase.NewAuditService(auditRepo)
	userUC := usecase.NewUserService(userRepo, hasher)
	peerUC := usecase.NewPeerService(peerRepo, srvRepo, agentTransport)
	cfgUC := usecase.NewConfigService(cfgRepo, srvRepo, peerRepo, agentTransport)
	srvUC := usecase.NewServerService(srvRepo, keys, mtlsIssuer, cfg.PublicBaseURL)
	nodeOpsUC := usecase.NewNodeOpsService(srvRepo, agentTransport)
	jobUC := usecase.NewJobService(jobRepo, srvRepo, cvRepo)
	_ = jobUC // прокидывается в router через AgentJobs handler ниже
	inviteUC := usecase.NewInviteService(invRepo, peerUC, srvRepo)
	statsUC := usecase.NewStatsService(userRepo, peerRepo, srvRepo)

	router := netHTTP.NewRouter(netHTTP.Deps{
		Logger: log,
		Tokens: tokens,
		Auth:   handler.NewAuth(authUC, userUC, auditUC),
		User:   handler.NewUser(userUC, auditUC),
		Peer:   handler.NewPeer(peerUC, auditUC),
		Server: handler.NewServer(srvUC, cfgUC, auditUC),
		Config: handler.NewConfig(cfgUC, auditUC),
		Stats:  handler.NewStats(statsUC),
		Invite: handler.NewInvite(inviteUC, cfg.PublicBaseURL, auditUC),
		Audit:     handler.NewAudit(auditUC),
		NodeOps:   handler.NewNodeOps(nodeOpsUC, srvUC, auditUC),
		AgentJobs: handler.NewAgentJobs(jobUC, srvRepo),
		System:    handler.NewSystem(cfg.InstallDir),
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdown, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		_ = srv.Shutdown(shutdown)
	}()

	// Background: задачи, висящие в running >5 мин, возвращаются в pending.
	go func() {
		t := time.NewTicker(60 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if n, err := jobUC.CleanupStale(ctx); err != nil {
					log.Warn().Err(err).Msg("job-gc failed")
				} else if n > 0 {
					log.Info().Int("count", n).Msg("stale jobs rescheduled")
				}
			}
		}
	}()

	// Background: traffic enforcer — disables peers that exceed their limit.
	trafficEnforcer := usecase.NewTrafficEnforcer(peerRepo, auditUC, &log)
	go trafficEnforcer.Run(ctx)

	// Background monitor: серверы без heartbeat'а >60s помечаются offline.
	// Тикер срабатывает каждые 30s.
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if n, err := srvUC.MarkStaleOffline(ctx, 60*time.Second); err != nil {
					log.Warn().Err(err).Msg("offline-monitor failed")
				} else if n > 0 {
					log.Info().Int("count", n).Msg("nodes marked offline (stale heartbeat)")
				}
			}
		}
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("server error")
	}
	log.Info().Msg("bye")
}

type config struct {
	HTTPAddr         string
	DatabaseDSN      string
	JWTSecret        string
	LogLevel         string
	AgentInsecureTLS bool
	MTLSDir          string
	PublicBaseURL    string
	InstallDir       string
}

func loadConfig() config {
	return config{
		HTTPAddr:         envOr("HTTP_ADDR", ":8080"),
		DatabaseDSN:      envOr("DATABASE_DSN", "postgres://voidwg:voidwg@localhost:5432/voidwg?sslmode=disable"),
		JWTSecret:        envOr("JWT_SECRET", "change-me-please-32-bytes-min!!!"),
		LogLevel:         envOr("LOG_LEVEL", "info"),
		// Phase 4: secure-by-default. Включается только явным флагом.
		AgentInsecureTLS: envOr("AGENT_INSECURE_TLS", "false") == "true",
		MTLSDir:          envOr("MTLS_DIR", "/var/lib/voidwg/agent-ca"),
		PublicBaseURL:    envOr("PUBLIC_BASE_URL", ""),
		InstallDir:       envOr("INSTALL_DIR", "/opt/void-wg"),
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
