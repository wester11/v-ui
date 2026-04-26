package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/voidwg/agent/internal/awg"
	"github.com/voidwg/agent/internal/client"
	"github.com/voidwg/agent/internal/transport"
	"github.com/voidwg/agent/internal/wg"
)

type config struct {
	ControlURL  string
	AgentToken  string
	WGInterface string
	WGAddr      string
	ObfsListen  string // UDP с AWG-обфускацией
	TCPListen   string // optional, fallback UDP-over-TCP
	TLSListen   string // optional, fallback UDP-over-TLS
	TLSCert     string // путь к cert (для TLS-fallback)
	TLSKey      string
	HTTPListen  string

	// AWG params (получаются из control-plane при первом heartbeat;
	// пока для standalone-теста читаются из env).
	AWG awg.Params

	// mTLS — Phase 4 secure-by-default.
	ControlCA   string
	ControlCert string
	ControlKey  string
	InsecureTLS bool
}

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Str("svc", "void-wg-agent").Logger()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := loadConfig()

	mgr := wg.New(cfg.WGInterface)

	ctrl, err := client.New(cfg.ControlURL, cfg.AgentToken, client.TLSConfig{
		CAFile:   cfg.ControlCA,
		CertFile: cfg.ControlCert,
		KeyFile:  cfg.ControlKey,
		Insecure: cfg.InsecureTLS,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("client init")
	}

	// HTTP-сервер для приёма команд от control-plane.
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
			ID, PublicKey, AllowedIP string
		}
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// PSK больше не передаётся — сервер не хранит.
		if err := mgr.AddPeer(r.Context(), p.PublicKey, "", p.AllowedIP); err != nil {
			log.Error().Err(err).Msg("addpeer")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	go func() {
		srv := &http.Server{Addr: cfg.HTTPListen, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
		log.Info().Str("addr", cfg.HTTPListen).Msg("agent api up")
		_ = srv.ListenAndServe()
	}()

	// AWG UDP proxy (основной канал)
	go func() {
		px, err := awg.NewProxy(cfg.ObfsListen, cfg.WGAddr, cfg.AWG, &log)
		if err != nil {
			log.Error().Err(err).Msg("awg init")
			return
		}
		log.Info().
			Str("listen", cfg.ObfsListen).
			Str("upstream", cfg.WGAddr).
			Uint8("Jc", cfg.AWG.Jc).
			Uint16("S1", cfg.AWG.S1).Uint16("S2", cfg.AWG.S2).
			Msg("AmneziaWG proxy up")
		if err := px.Run(); err != nil {
			log.Error().Err(err).Msg("awg run")
		}
	}()

	// Fallback: UDP-over-TCP (если ISP режет UDP)
	if cfg.TCPListen != "" {
		go func() {
			t := transport.NewTCP(cfg.TCPListen, cfg.WGAddr, cfg.AWG, &log)
			if err := t.Run(); err != nil {
				log.Error().Err(err).Msg("tcp tunnel")
			}
		}()
	}

	// Fallback: UDP-over-TLS (если режется TCP-trafic, но TLS пропускают)
	if cfg.TLSListen != "" && cfg.TLSCert != "" && cfg.TLSKey != "" {
		go func() {
			t, err := transport.NewTLS(cfg.TLSListen, cfg.WGAddr, cfg.TLSCert, cfg.TLSKey, cfg.AWG, &log)
			if err != nil {
				log.Error().Err(err).Msg("tls tunnel init")
				return
			}
			if err := t.Run(); err != nil {
				log.Error().Err(err).Msg("tls tunnel")
			}
		}()
	}

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

func envU16(k string) uint16 {
	if v, err := strconv.ParseUint(envOr(k, "0"), 10, 16); err == nil {
		return uint16(v)
	}
	return 0
}

func envU32(k string) uint32 {
	if v, err := strconv.ParseUint(envOr(k, "0"), 10, 32); err == nil {
		return uint32(v)
	}
	return 0
}

func envU8(k string) uint8 {
	if v, err := strconv.ParseUint(envOr(k, "0"), 10, 8); err == nil {
		return uint8(v)
	}
	return 0
}

func loadConfig() config {
	return config{
		ControlURL:  envOr("CONTROL_URL", "https://control:8080"),
		AgentToken:  envOr("AGENT_TOKEN", ""),
		WGInterface: envOr("WG_IFACE", "wg0"),
		WGAddr:      envOr("WG_ADDR", "127.0.0.1:51820"),
		ObfsListen:  envOr("OBFS_LISTEN", ":51821"),
		TCPListen:   envOr("TCP_LISTEN", ""),
		TLSListen:   envOr("TLS_LISTEN", ""),
		TLSCert:     envOr("TLS_CERT", ""),
		TLSKey:      envOr("TLS_KEY", ""),
		HTTPListen:  envOr("HTTP_LISTEN", ":7443"),

		AWG: awg.Params{
			Jc:   envU8("AWG_JC"),
			Jmin: envU16("AWG_JMIN"),
			Jmax: envU16("AWG_JMAX"),
			S1:   envU16("AWG_S1"),
			S2:   envU16("AWG_S2"),
			H1:   envU32("AWG_H1"),
			H2:   envU32("AWG_H2"),
			H3:   envU32("AWG_H3"),
			H4:   envU32("AWG_H4"),
		},

		ControlCA:   envOr("CONTROL_CA", ""),
		ControlCert: envOr("CONTROL_CERT", ""),
		ControlKey:  envOr("CONTROL_KEY", ""),
		// secure-by-default; включить для отладки можно AGENT_INSECURE_TLS=true.
		InsecureTLS: envOr("AGENT_INSECURE_TLS", "false") == "true",
	}
}
