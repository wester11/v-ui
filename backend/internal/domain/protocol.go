package domain

import "encoding/json"

// Protocol — тип транспорта VPN-ноды.
type Protocol string

const (
	ProtoWireGuard Protocol = "wireguard"
	ProtoAmneziaWG Protocol = "amneziawg"
	ProtoXray      Protocol = "xray"
)

func (p Protocol) Valid() bool {
	switch p {
	case ProtoWireGuard, ProtoAmneziaWG, ProtoXray:
		return true
	}
	return false
}

// XrayConfig — серверная сторона VLESS+Reality.
//
// Хранится в Server.ProtocolConfig (jsonb). Public части (PublicKey, ShortIDs,
// SNI, Fingerprint, Flow, InboundPort) выдаются клиенту; PrivateKey никогда
// не покидает control-plane (как и WG-приватник сервера в чистом WG-режиме).
type XrayConfig struct {
	InboundPort uint16   `json:"inbound_port"`            // 443 — выглядит как HTTPS
	SNI         string   `json:"sni"`                      // www.cloudflare.com
	Dest        string   `json:"dest"`                     // www.cloudflare.com:443
	PrivateKey  string   `json:"private_key"`              // X25519 (server-side)
	PublicKey   string   `json:"public_key"`               // X25519 (для клиентов)
	ShortIDs    []string `json:"short_ids"`                // пул shortId, peer'ам выдаётся один из этих
	Fingerprint string   `json:"fingerprint"`              // chrome | firefox | safari | random
	Flow        string   `json:"flow"`                     // xtls-rprx-vision (по умолчанию)
	FallbackDest string   `json:"fallback_dest,omitempty"` // куда уйдёт трафик не-VLESS клиентов (опц.)
}

// XrayConfigFromJSON — безопасный декод RawMessage.
func XrayConfigFromJSON(b []byte) (XrayConfig, error) {
	var c XrayConfig
	if len(b) == 0 || string(b) == "null" {
		return c, nil
	}
	err := json.Unmarshal(b, &c)
	return c, err
}

// XrayPublic — то, что отдаётся клиентам (без приватника).
type XrayPublic struct {
	InboundPort uint16
	SNI         string
	PublicKey   string
	Fingerprint string
	Flow        string
	ShortID     string
}

// PublicView — выдаёт клиентский view конфига для одного peer'а.
// shortID берётся из пула серверного конфига по индексу (peer.uuid → один из ShortIDs).
func (c XrayConfig) PublicView(shortID string) XrayPublic {
	return XrayPublic{
		InboundPort: c.InboundPort,
		SNI:         c.SNI,
		PublicKey:   c.PublicKey,
		Fingerprint: c.Fingerprint,
		Flow:        c.Flow,
		ShortID:     shortID,
	}
}
