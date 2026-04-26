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

	// Какой transport включить на этом агенте.
	// wireguard | amneziawg | xray   (default: amneziawg).
	Protocol string

	// WG / AWG
	WGInterface string
	WGAddr      string
	ObfsListen  string
	TCPListen   string
	TLSListen   string
	TLSCert     string
	TLSKey      string
	HTTPListen  string
	AWG         awg.Params

	// Xray
	XrayBin    string
	XrayConfig string

	// mTLS to control-plane
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

	wgMgr := wg.New(cfg.WGInterface)

	ctrl, err := client.New(cfg.ControlURL, cfg.AgentToken, client.TLSConfig{
		CAFile:   cfg.ControlCA,
		CertFile: cfg.ControlCert,
		KeyFile:  cfg.ControlKey,
		Insecure: cfg.InsecureTLS,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("client init")
	}

	// === protocol selection ===
	var trans transport.Transport
	switch cfg.Protocol {
	case "wireguard":
		trans = transport.NewWireGuard(wgMgr)
	case "amneziawg", "":
		t, err := transport.NewAmneziaWG(wgMgr, cfg.ObfsListen, cfg.WGAddr, cfg.AWG, &log)
		if err != nil {
			log.Fatal().Err(err).Msg("amneziawg init")
		}
		trans = t
	case "xray":
		trans = transport.NewXray(cfg.XrayBin, cfg.XrayConfig, &log)
	default:
		log.Fatal().Str("protocol", cfg.Protocol).Msg("unknown protocol")
	}
	if err := trans.Start(); err != nil {
		log.Fatal().Err(err).Msg("transport start")
	}
	log.Info().Str("transport", trans.Name()).Msg("transport up")

	// === HTTP API: control-plane вызывает /v1/peers и /v1/xray/reload ===
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
		switch r.Method {
		case http.MethodPost:
			var p transport.PeerSpec
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if err := trans.AddPeer(p); err != nil {
				log.Error().Err(err).Msg("AddPeer")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			if err := trans.RemovePeer(id); err != nil {
				log.Error().Err(err).Msg("RemovePeer")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/v1/xray/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Agent-Token") != cfg.AgentToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var in transport.XrayInbound
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := trans.Reload(in); err != nil {
			log.Error().Err(err).Msg("xray reload")
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

	// fallback transports — только для WG/AWG; для xray поверх TLS уже сидит сам Reality.
	if (cfg.Protocol == "wireguard" || cfg.Protocol == "amneziawg" || cfg.Protocol == "") && cfg.TCPListen != "" {
		go func() {
			t := transport.NewTCP(cfg.TCPListen, cfg.WGAddr, cfg.AWG, &log)
			if err := t.Run(); err != nil {
				log.Error().Err(err).Msg("tcp tunnel")
			}
		}()
	}

	// heartbeat
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = trans.Stop()
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
func envU8(k string) uint8 {
	v, _ := strconv.ParseUint(envOr(k, "0"), 10, 8)
	return uint8(v)
}
func envU16(k string) uint16 {
	v, _ := strconv.ParseUint(envOr(k, "0"), 10, 16)
	return uint16(v)
}
func envU32(k string) uint32 {
	v, _ := strconv.ParseUint(envOr(k, "0"), 10, 32)
	return uint32(v)
}

func loadConfig() config {
	return config{
		ControlURL: envOr("CONTROL_URL", "https://control:8080"),
		AgentToken: envOr("AGENT_TOKEN", ""),
		Protocol:   envOr("AGENT_PROTOCOL", "amneziawg"),

		WGInterface: envOr("WG_IFACE", "wg0"),
		WGAddr:      envOr("WG_ADDR", "127.0.0.1:51820"),
		ObfsListen:  envOr("OBFS_LISTEN", ":51821"),
		TCPListen:   envOr("TCP_LISTEN", ""),
		TLSListen:   envOr("TLS_LISTEN", ""),
		TLSCert:     envOr("TLS_CERT", ""),
		TLSKey:      envOr("TLS_KEY", ""),
		HTTPListen:  envOr("HTTP_LISTEN", ":7443"),
		AWG: awg.Params{
			Jc: envU8("AWG_JC"), Jmin: envU16("AWG_JMIN"), Jmax: envU16("AWG_JMAX"),
			S1: envU16("AWG_S1"), S2: envU16("AWG_S2"),
			H1: envU32("AWG_H1"), H2: envU32("AWG_H2"), H3: envU32("AWG_H3"), H4: envU32("AWG_H4"),
		},

		XrayBin:    envOr("XRAY_BIN", "xray"),
		XrayConfig: envOr("XRAY_CONFIG", "/etc/xray/config.json"),

		ControlCA:   envOr("CONTROL_CA", ""),
		ControlCert: envOr("CONTROL_CERT", ""),
		ControlKey:  envOr("CONTROL_KEY", ""),
		InsecureTLS: envOr("AGENT_INSECURE_TLS", "false") == "true",
	}
}
