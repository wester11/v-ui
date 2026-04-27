// Package xray renders Xray-core config.json and client VLESS+Reality URIs.
package xray

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/crypto/curve25519"

	"github.com/voidwg/control/internal/domain"
)

// GenerateX25519 creates a Reality keypair encoded as base64url (no padding).
func GenerateX25519() (privateB64, publicB64 string, err error) {
	var priv [32]byte
	if _, err := rand.Read(priv[:]); err != nil {
		return "", "", err
	}
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64
	pub, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return "", "", err
	}
	enc := func(b []byte) string { return base64URLNoPad(b) }
	return enc(priv[:]), enc(pub), nil
}

// GenerateShortID returns a 16-char hex shortId for Reality.
func GenerateShortID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	const hex = "0123456789abcdef"
	out := make([]byte, 16)
	for i, x := range b {
		out[i*2] = hex[x>>4]
		out[i*2+1] = hex[x&0x0f]
	}
	return string(out), nil
}

func base64URLNoPad(b []byte) string {
	const alpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	var out strings.Builder
	for i := 0; i < len(b); i += 3 {
		var n uint32
		for j := 0; j < 3; j++ {
			n <<= 8
			if i+j < len(b) {
				n |= uint32(b[i+j])
			}
		}
		out.WriteByte(alpha[(n>>18)&0x3f])
		out.WriteByte(alpha[(n>>12)&0x3f])
		if i+1 < len(b) {
			out.WriteByte(alpha[(n>>6)&0x3f])
		}
		if i+2 < len(b) {
			out.WriteByte(alpha[n&0x3f])
		}
	}
	return out.String()
}

type XrayPeer struct {
	UUID    string
	Email   string
	Flow    string
	ShortID string
}

// BuildFullConfig builds full config.json for one Xray server.
func BuildFullConfig(server *domain.Server, peers []*domain.Peer) ([]byte, error) {
	if server.Protocol != domain.ProtoXray {
		return nil, fmt.Errorf("BuildFullConfig: server is not xray (protocol=%s)", server.Protocol)
	}
	xc, err := domain.XrayConfigFromJSON(server.ProtocolConfig)
	if err != nil {
		return nil, fmt.Errorf("BuildFullConfig: parse protocol_config: %w", err)
	}

	xpeers := make([]XrayPeer, 0, len(peers))
	for _, p := range peers {
		if !p.Enabled || p.Protocol != domain.ProtoXray || p.XrayUUID == "" {
			continue
		}
		flow := p.XrayFlow
		if flow == "" {
			flow = xc.Flow
		}
		email := p.Name
		if email == "" {
			email = p.ID.String()
		}
		xpeers = append(xpeers, XrayPeer{UUID: p.XrayUUID, Email: email, Flow: flow, ShortID: p.XrayShortID})
	}

	return RenderServerConfig(xc, xpeers)
}

// RenderServerConfig renders server-side config.json.
func RenderServerConfig(cfg domain.XrayConfig, peers []XrayPeer) ([]byte, error) {
	clients := make([]map[string]any, 0, len(peers)+len(cfg.SystemClients))
	for _, p := range peers {
		flow := p.Flow
		if flow == "" {
			flow = cfg.Flow
		}
		clients = append(clients, map[string]any{
			"id":    p.UUID,
			"flow":  flow,
			"email": p.Email,
		})
	}
	for _, sc := range cfg.SystemClients {
		if strings.TrimSpace(sc.ID) == "" {
			continue
		}
		clients = append(clients, map[string]any{
			"id":    sc.ID,
			"flow":  ifEmpty(sc.Flow, ifEmpty(cfg.Flow, "xtls-rprx-vision")),
			"email": ifEmpty(sc.Email, "system"),
		})
	}

	port := cfg.InboundPort
	if port == 0 {
		port = 443
	}
	shortIDs := cfg.ShortIDs
	if len(shortIDs) == 0 {
		shortIDs = []string{""}
	}
	dest := cfg.Dest
	if dest == "" {
		dest = "www.cloudflare.com:443"
	}
	sni := cfg.SNI
	if sni == "" {
		sni = strings.SplitN(dest, ":", 2)[0]
	}

	outbounds := []any{
		map[string]any{"protocol": "freedom", "tag": "direct"},
		map[string]any{"protocol": "blackhole", "tag": "block"},
	}
	rules := []any{}

	if cfg.Mode == "cascade" && cfg.Cascade != nil {
		outbounds = append(outbounds, map[string]any{
			"protocol": "vless",
			"tag":      "cascade-upstream",
			"settings": map[string]any{
				"vnext": []any{map[string]any{
					"address": cfg.Cascade.UpstreamHost,
					"port":    cfg.Cascade.UpstreamPort,
					"users": []any{map[string]any{
						"id":         cfg.Cascade.UpstreamAuthUUID,
						"encryption": "none",
						"flow":       ifEmpty(cfg.Flow, "xtls-rprx-vision"),
					}},
				}},
			},
			"streamSettings": map[string]any{
				"network":  "tcp",
				"security": "reality",
				"realitySettings": map[string]any{
					"serverName":  ifEmpty(cfg.Cascade.UpstreamSNI, cfg.Cascade.UpstreamHost),
					"publicKey":   cfg.Cascade.UpstreamPubKey,
					"shortId":     cfg.Cascade.UpstreamShortID,
					"fingerprint": ifEmpty(cfg.Fingerprint, "chrome"),
				},
			},
		})

		rules = append(rules, map[string]any{"type": "field", "ip": []string{"geoip:private"}, "outboundTag": "direct"})
		for _, rule := range cfg.Cascade.Rules {
			if rr := cascadeRoutingRule(rule); rr != nil {
				rules = append(rules, rr)
			}
		}
		// catch-all fallback for non-matched traffic in cascade mode
		rules = append(rules, map[string]any{"type": "field", "outboundTag": "cascade-upstream"})
	} else {
		rules = append(rules, map[string]any{"type": "field", "ip": []string{"geoip:private"}, "outboundTag": "block"})
	}

	out := map[string]any{
		"log": map[string]any{"loglevel": "warning"},
		"inbounds": []any{map[string]any{
			"listen":   "0.0.0.0",
			"port":     port,
			"protocol": "vless",
			"tag":      "vless-reality",
			"settings": map[string]any{
				"clients":    clients,
				"decryption": "none",
			},
			"streamSettings": map[string]any{
				"network":  "tcp",
				"security": "reality",
				"realitySettings": map[string]any{
					"show":        false,
					"dest":        dest,
					"xver":        0,
					"serverNames": []string{sni},
					"privateKey":  cfg.PrivateKey,
					"shortIds":    shortIDs,
					"fingerprint": ifEmpty(cfg.Fingerprint, "chrome"),
				},
			},
			"sniffing": map[string]any{
				"enabled":      true,
				"destOverride": []string{"http", "tls", "quic"},
			},
		}},
		"outbounds": outbounds,
		"routing":   map[string]any{"rules": rules},
	}
	return json.MarshalIndent(out, "", "  ")
}

func cascadeRoutingRule(rule domain.XrayCascadeRule) map[string]any {
	match := strings.TrimSpace(rule.Match)
	if match == "" {
		return nil
	}
	tag := "direct"
	if strings.EqualFold(strings.TrimSpace(rule.Outbound), "proxy") {
		tag = "cascade-upstream"
	}
	out := map[string]any{"type": "field", "outboundTag": tag}
	if strings.HasPrefix(match, "geoip:") {
		out["ip"] = []string{match}
		return out
	}
	if strings.HasPrefix(match, "geosite:") || strings.HasPrefix(match, "domain:") {
		out["domain"] = []string{match}
		return out
	}
	if strings.Contains(match, "/") || strings.Count(match, ".") >= 1 {
		out["ip"] = []string{match}
		return out
	}
	out["domain"] = []string{match}
	return out
}

// RenderVLESSURI renders a client import URI.
func RenderVLESSURI(server *domain.Server, peer *domain.Peer, view domain.XrayPublic) string {
	host, port := splitHostPort(server.Endpoint)
	if view.InboundPort > 0 {
		port = fmt.Sprintf("%d", view.InboundPort)
	}
	q := url.Values{}
	q.Set("type", "tcp")
	q.Set("security", "reality")
	q.Set("pbk", view.PublicKey)
	q.Set("fp", ifEmpty(view.Fingerprint, "chrome"))
	q.Set("sni", view.SNI)
	q.Set("sid", view.ShortID)
	q.Set("flow", ifEmpty(view.Flow, "xtls-rprx-vision"))
	q.Set("encryption", "none")

	u := url.URL{
		Scheme:   "vless",
		User:     url.User(peer.XrayUUID),
		Host:     host + ":" + port,
		RawQuery: q.Encode(),
		Fragment: peer.Name,
	}
	return u.String()
}

func ifEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func splitHostPort(endpoint string) (host, port string) {
	if i := strings.LastIndex(endpoint, ":"); i > 0 {
		return endpoint[:i], endpoint[i+1:]
	}
	return endpoint, "443"
}
