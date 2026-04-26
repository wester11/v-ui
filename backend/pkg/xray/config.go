// Package xray — генератор Xray-core config.json (серверная часть)
// и клиентских VLESS+Reality URI.
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

// GenerateX25519 — выпускает пару ключей для Reality.
// Reality использует X25519, как и WireGuard, но Xray ждёт base64(URL, no padding).
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
	enc := func(b []byte) string {
		return base64URLNoPad(b)
	}
	return enc(priv[:]), enc(pub), nil
}

// GenerateShortID — 8-байтовый shortId в hex (как в Reality).
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

// XrayPeer — peer для рендеринга в config.json.
type XrayPeer struct {
	UUID    string
	Email   string
	Flow    string
	ShortID string
}

// RenderServerConfig — генерирует config.json для Xray-core (серверная сторона).
//
// Inbound: VLESS на :443 с Reality, dest на легитимный TLS-сервер.
// Outbound: freedom (direct).
func RenderServerConfig(cfg domain.XrayConfig, peers []XrayPeer) ([]byte, error) {
	clients := make([]map[string]any, 0, len(peers))
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

	out := map[string]any{
		"log": map[string]any{
			"loglevel": "warning",
		},
		"inbounds": []any{
			map[string]any{
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
						"show":         false,
						"dest":         dest,
						"xver":         0,
						"serverNames":  []string{sni},
						"privateKey":   cfg.PrivateKey,
						"shortIds":     shortIDs,
						"fingerprint":  ifEmpty(cfg.Fingerprint, "chrome"),
					},
				},
				"sniffing": map[string]any{
					"enabled":      true,
					"destOverride": []string{"http", "tls", "quic"},
				},
			},
		},
		"outbounds": []any{
			map[string]any{"protocol": "freedom", "tag": "direct"},
			map[string]any{"protocol": "blackhole", "tag": "block"},
		},
		"routing": map[string]any{
			"rules": []any{
				map[string]any{"type": "field", "ip": []string{"geoip:private"}, "outboundTag": "block"},
			},
		},
	}
	return json.MarshalIndent(out, "", "  ")
}

// RenderVLESSURI — клиентский импорт-link.
//
// Формат:
//
//	vless://<uuid>@<host>:<port>?type=tcp&security=reality
//	  &pbk=<server pubkey>&fp=<fingerprint>&sni=<sni>&sid=<shortid>
//	  &flow=xtls-rprx-vision#<peer-name>
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
