// Package xray renders Xray-core config.json and client VLESS+Reality URIs.
package xray

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
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

func PublicKeyFromPrivate(privateB64 string) (string, error) {
	privateB64 = strings.TrimSpace(privateB64)
	if privateB64 == "" {
		return "", errors.New("private key is empty")
	}
	raw, err := base64.RawURLEncoding.DecodeString(privateB64)
	if err != nil {
		return "", err
	}
	if len(raw) != 32 {
		return "", errors.New("private key must be 32 bytes")
	}
	pub, err := curve25519.X25519(raw, curve25519.Basepoint)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(pub), nil
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

	if len(cfg.RawConfig) > 0 {
		return renderRawConfig(cfg, clients)
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

func renderRawConfig(cfg domain.XrayConfig, clients []map[string]any) ([]byte, error) {
	var raw map[string]any
	if err := json.Unmarshal(cfg.RawConfig, &raw); err != nil {
		return nil, fmt.Errorf("invalid raw xray config: %w", err)
	}

	inbounds, _ := raw["inbounds"].([]any)
	for i := range inbounds {
		inb, ok := inbounds[i].(map[string]any)
		if !ok {
			continue
		}
		proto, _ := inb["protocol"].(string)
		if proto != "vless" {
			continue
		}
		settings, _ := inb["settings"].(map[string]any)
		if settings == nil {
			settings = map[string]any{}
		}
		settings["clients"] = clients
		inb["settings"] = settings
		inbounds[i] = inb
	}
	raw["inbounds"] = inbounds

	if cfg.Mode == "cascade" && cfg.Cascade != nil {
		mergeCascadeIntoRaw(raw, cfg)
	}
	return json.MarshalIndent(raw, "", "  ")
}

func mergeCascadeIntoRaw(raw map[string]any, cfg domain.XrayConfig) {
	outbounds, _ := raw["outbounds"].([]any)
	hasCascadeOutbound := false
	for _, ob := range outbounds {
		obb, ok := ob.(map[string]any)
		if !ok {
			continue
		}
		if tag, _ := obb["tag"].(string); tag == "cascade-upstream" {
			hasCascadeOutbound = true
			break
		}
	}
	if !hasCascadeOutbound {
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
	}
	raw["outbounds"] = outbounds

	routing, _ := raw["routing"].(map[string]any)
	if routing == nil {
		routing = map[string]any{}
	}
	rules, _ := routing["rules"].([]any)
	if len(rules) == 0 {
		rules = []any{map[string]any{"type": "field", "ip": []string{"geoip:private"}, "outboundTag": "direct"}}
	}
	for _, rr := range cfg.Cascade.Rules {
		if rule := cascadeRoutingRule(rr); rule != nil {
			rules = append(rules, rule)
		}
	}
	rules = append(rules, map[string]any{"type": "field", "outboundTag": "cascade-upstream"})
	routing["rules"] = rules
	raw["routing"] = routing
}

func ExtractAdvancedMeta(raw []byte) (domain.XrayConfig, error) {
	var cfg domain.XrayConfig
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return cfg, err
	}

	inbounds, _ := root["inbounds"].([]any)
	for _, in := range inbounds {
		inb, ok := in.(map[string]any)
		if !ok {
			continue
		}
		proto, _ := inb["protocol"].(string)
		if proto != "vless" {
			continue
		}
		if p, ok := inb["port"].(float64); ok && p > 0 {
			cfg.InboundPort = uint16(p)
		}
		stream, _ := inb["streamSettings"].(map[string]any)
		reality, _ := stream["realitySettings"].(map[string]any)
		if sn, ok := reality["serverNames"].([]any); ok && len(sn) > 0 {
			if s, _ := sn[0].(string); s != "" {
				cfg.SNI = s
			}
		}
		if sid, ok := reality["shortIds"].([]any); ok {
			for _, s := range sid {
				if ss, _ := s.(string); ss != "" {
					cfg.ShortIDs = append(cfg.ShortIDs, ss)
				}
			}
		}
		if fp, ok := reality["fingerprint"].(string); ok {
			cfg.Fingerprint = fp
		}
		if pk, ok := reality["privateKey"].(string); ok {
			cfg.PrivateKey = pk
			pub, _ := PublicKeyFromPrivate(pk)
			cfg.PublicKey = pub
		}
		break
	}
	if cfg.InboundPort == 0 {
		cfg.InboundPort = 443
	}
	if cfg.SNI == "" {
		cfg.SNI = "www.cloudflare.com"
	}
	if cfg.Fingerprint == "" {
		cfg.Fingerprint = "chrome"
	}
	if cfg.Flow == "" {
		cfg.Flow = "xtls-rprx-vision"
	}
	if len(cfg.ShortIDs) == 0 {
		cfg.ShortIDs = []string{""}
	}
	return cfg, nil
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
