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

	userRepo := persistence.NewUserRepo(pool)
	peerRepo := persistence.NewPeerRepo(pool)
	srvRepo := persistence.NewServerRepo(pool)

	keys := wg.New()
	hasher := jwtauth.NewHasher()
	tokens := jwtauth.New([]byte(cfg.JWTSecret), "void-wg")
	box, err := jwtauth.NewSecretBox([]byte(cfg.SecretBoxKey))
	if err != nil {
		log.Fatal().Err(err).Msg("secretbox key must be 32 bytes")
	}
	agentTransport := transport.NewAgentClient(cfg.AgentInsecureTLS)

	if err := bootstrap.Admin(ctx, userRepo, hasher, log); err != nil {
		log.Error().Err(err).Msg("bootstrap admin failed")
	}

	authUC := usecase.NewAuth(userRepo, hasher, tokens)
	userUC := usecase.NewUserService(userRepo, hasher)
	peerUC := usecase.NewPeerService(peerRepo, srvRepo, keys, box, agentTransport)
	srvUC := usecase.NewServerService(srvRepo, keys)
	statsUC := usecase.NewStatsService(userRepo, peerRepo, srvRepo)

	router := netHTTP.NewRouter(netHTTP.Deps{
		Logger: log,
		Tokens: tokens,
		Auth:   handler.NewAuth(authUC, userUC),
		User:   handler.NewUser(userUC),
		Peer:   handler.NewPeer(peerUC),
		Server: handler.NewServer(srvUC),
		Stats:  handler.NewStats(statsUC),
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

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("server error")
	}
	log.Info().Msg("bye")
}

type config struct {
	HTTPAddr         string
	DatabaseDSN      string
	JWTSecret        string
	SecretBoxKey     string
	LogLevel         string
	AgentInsecureTLS bool
}

func loadConfig() config {
	return config{
		HTTPAddr:         envOr("HTTP_ADDR", ":8080"),
		DatabaseDSN:      envOr("DATABASE_DSN", "postgres://voidwg:voidwg@localhost:5432/voidwg?sslmode=disable"),
		JWTSecret:        envOr("JWT_SECRET", "change-me-please-32-bytes-min!!!"),
		SecretBoxKey:     envOr("SECRETBOX_KEY", "0123456789abcdef0123456789abcdef"),
		LogLevel:         envOr("LOG_LEVEL", "info"),
		AgentInsecureTLS: envOr("AGENT_INSECURE_TLS", "false") == "true",
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
