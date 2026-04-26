package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/voidwg/agent/internal/client"
	"github.com/voidwg/agent/internal/obfs"
	"github.com/voidwg/agent/internal/wg"
)

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Str("svc", "void-wg-agent").Logger()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := struct {
		ControlURL  string
		AgentToken  string
		WGInterface string
		ObfsListen  string
		WGAddr      string
		ObfsPSK     string
		HTTPListen  string
	}{
		ControlURL:  envOr("CONTROL_URL", "https://control:8080"),
		AgentToken:  envOr("AGENT_TOKEN", ""),
		WGInterface: envOr("WG_IFACE", "wg0"),
		ObfsListen:  envOr("OBFS_LISTEN", ":51821"),
		WGAddr:      envOr("WG_ADDR", "127.0.0.1:51820"),
		ObfsPSK:     envOr("OBFS_PSK", "voidwg-default-psk-change-me"),
		HTTPListen:  envOr("HTTP_LISTEN", ":7443"),
	}

	mgr := wg.New(cfg.WGInterface)
	ctrl := client.New(cfg.ControlURL, cfg.AgentToken, true)

	// HTTP-сервер для приёма команд от control-plane
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/v1/peers", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Agent-Token") != cfg.AgentToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var p struct {
			ID, PublicKey, PresharedKey, AllowedIP string
		}
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := mgr.AddPeer(r.Context(), p.PublicKey, p.PresharedKey, p.AllowedIP); err != nil {
			log.Error().Err(err).Msg("addpeer")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	// DELETE /v1/peers/{id} - в реальном агенте id mapping → public_key, опускаем для краткости

	go func() {
		srv := &http.Server{Addr: cfg.HTTPListen, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
		log.Info().Str("addr", cfg.HTTPListen).Msg("agent api up")
		_ = srv.ListenAndServe()
	}()

	// Obfuscation proxy
	go func() {
		px, err := obfs.NewProxy(cfg.ObfsListen, cfg.WGAddr, []byte(cfg.ObfsPSK), &log)
		if err != nil {
			log.Error().Err(err).Msg("obfs init")
			return
		}
		log.Info().Str("listen", cfg.ObfsListen).Str("upstream", cfg.WGAddr).Msg("obfs proxy up")
		if err := px.Run(); err != nil {
			log.Error().Err(err).Msg("obfs run")
		}
	}()

	// heartbeat
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("agent stopping")
			return
		case <-t.C:
			if err := ctrl.Heartbeat(ctx); err != nil {
				log.Warn().Err(err).Msg("heartbeat failed")
			}
		}
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
