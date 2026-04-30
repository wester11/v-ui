package main

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/voidwg/agent/internal/awg"
	"github.com/voidwg/agent/internal/client"
	"github.com/voidwg/agent/internal/sysstat"
	"github.com/voidwg/agent/internal/transport"
	"github.com/voidwg/agent/internal/wg"
)

type config struct {
	ControlURL  string
	AgentToken  string
	NodeID      string
	Secret      string

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

	// Xray (stateless model)
	XrayConfig    string // путь к config.json (общий volume с контейнером xray)
	XrayContainer string // docker-имя sidecar-контейнера xray
	XrayHealth    string // host:port для TCP health-probe

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

	ctrl, err := client.New(cfg.ControlURL, cfg.AgentToken, cfg.NodeID, cfg.Secret, client.TLSConfig{
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
		trans = transport.NewXray(cfg.XrayConfig, cfg.XrayContainer, cfg.XrayHealth, &log)
	default:
		log.Fatal().Str("protocol", cfg.Protocol).Msg("unknown protocol")
	}
	if err := trans.Start(); err != nil {
		log.Fatal().Err(err).Msg("transport start")
	}
	log.Info().Str("transport", trans.Name()).Msg("transport up")

	hostname, _ := os.Hostname()
	ip := primaryIP()
	if err := ctrl.Register(ctx, hostname, ip, envOr("AGENT_VERSION", "dev")); err != nil {
		log.Warn().Err(err).Msg("register failed")
	}

	// === HTTP API: control-plane вызывает /v1/peers и /v1/xray/reload ===
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	// /v1/peers — runtime peer-mutation для WG/AWG.
	// Для xray этот endpoint отдаёт 410 Gone — все peer-операции для xray
	// идут через /v1/xray/deploy (полный config push).
	mux.HandleFunc("/v1/peers", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Agent-Token") != cfg.AgentToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if cfg.Protocol == "xray" {
			http.Error(w, "use /v1/xray/deploy for xray peer changes", http.StatusGone)
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

	// /v1/xray/deploy — получить полный config.json от control-plane,
	// записать атомарно, перезапустить xray-контейнер.
	mux.HandleFunc("/v1/xray/deploy", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Agent-Token") != cfg.AgentToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		xt, ok := trans.(*transport.XrayTransport)
		if !ok {
			http.Error(w, "this agent is not in xray mode", http.StatusConflict)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB cap
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := xt.ApplyConfig(body); err != nil {
			log.Error().Err(err).Msg("ApplyConfig")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := xt.Restart(); err != nil {
			log.Error().Err(err).Msg("xray Restart")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// /v1/restart — control-plane просит ребутнуть рантайм. Для xray-режима
	// делегируется в XrayTransport.Restart() (docker restart sidecar). Для WG/AWG
	// перезапуск интерфейса делается через wg-quick down/up — пока no-op.
	mux.HandleFunc("/v1/restart", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Agent-Token") != cfg.AgentToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if xt, ok := trans.(*transport.XrayTransport); ok {
			if err := xt.Restart(); err != nil {
				log.Error().Err(err).Msg("xray restart")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// /v1/metrics — реальная статистика: система + WG peers + транспорт.
	mux.HandleFunc("/v1/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Agent-Token") != cfg.AgentToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		host, _ := os.Hostname()
		out := map[string]any{
			"transport":     trans.Name(),
			"hostname":      host,
			"agent_version": envOr("AGENT_VERSION", "dev"),
			"protocol":      cfg.Protocol,
			"timestamp":     time.Now().UTC().Format(time.RFC3339),
		}
		// System stats (CPU load, memory, uptime)
		if sys, err := sysstat.CollectSystem(); err == nil {
			out["uptime_seconds"] = sys.UptimeSeconds
			out["load_avg_1"]     = sys.LoadAvg1
			out["mem_total_kb"]   = sys.MemTotalKB
			out["mem_avail_kb"]   = sys.MemAvailKB
			out["mem_used_pct"]   = sys.MemUsedPct
		}
		// WireGuard peer stats (only for WG/AWG transport)
		if cfg.Protocol == "wireguard" || cfg.Protocol == "amneziawg" || cfg.Protocol == "" {
			if wgst, err := sysstat.CollectWG(cfg.WGInterface); err == nil {
				peers := make([]map[string]any, 0, len(wgst.Peers))
				onlineCount := 0
				for _, p := range wgst.Peers {
					pm := map[string]any{
						"public_key":  p.PublicKey,
						"endpoint":    p.Endpoint,
						"allowed_ips": p.AllowedIPs,
						"rx_bytes":    p.RxBytes,
						"tx_bytes":    p.TxBytes,
						"online":      p.Online,
					}
					if !p.LastHandshake.IsZero() {
						pm["last_handshake"] = p.LastHandshake.UTC().Format(time.RFC3339)
					}
					peers = append(peers, pm)
					if p.Online {
						onlineCount++
					}
				}
				out["wg_peers"]        = peers
				out["wg_peers_total"]  = len(peers)
				out["wg_peers_online"] = onlineCount
				out["wg_public_key"]   = wgst.PublicKey
				out["wg_listen_port"]  = wgst.ListenPort
			}
		}
		if xt, ok := trans.(*transport.XrayTransport); ok {
			out["xray_health"] = xt.HealthCheck() == nil
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})

	// /v1/xray/health — простой TCP-probe inbound-порта.
	mux.HandleFunc("/v1/xray/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Agent-Token") != cfg.AgentToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		xt, ok := trans.(*transport.XrayTransport)
		if !ok {
			w.WriteHeader(http.StatusConflict)
			return
		}
		if err := xt.HealthCheck(); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
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
		NodeID:     envOr("NODE_ID", ""),
		Secret:     envOr("SECRET", ""),
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

		XrayConfig:    envOr("XRAY_CONFIG", "/etc/xray/config.json"),
		XrayContainer: envOr("XRAY_CONTAINER", "xray"),
		XrayHealth:    envOr("XRAY_HEALTH", "127.0.0.1:443"),

		ControlCA:   envOr("CONTROL_CA", ""),
		ControlCert: envOr("CONTROL_CERT", ""),
		ControlKey:  envOr("CONTROL_KEY", ""),
		InsecureTLS: envOr("AGENT_INSECURE_TLS", "false") == "true",
	}
}

func primaryIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP == nil || ipNet.IP.IsLoopback() {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}
			s := ip.String()
			if strings.HasPrefix(s, "127.") {
				continue
			}
			return s
		}
	}
	return ""
}
