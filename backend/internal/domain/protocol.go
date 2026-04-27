package domain

import "encoding/json"

// Protocol defines VPN transport type for a server node.
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

// XrayConfig stores server-side VLESS+Reality settings in Server.ProtocolConfig.
type XrayConfig struct {
	InboundPort  uint16   `json:"inbound_port"`
	SNI          string   `json:"sni"`
	Dest         string   `json:"dest"`
	PrivateKey   string   `json:"private_key"`
	PublicKey    string   `json:"public_key"`
	ShortIDs     []string `json:"short_ids"`
	Fingerprint  string   `json:"fingerprint"`
	Flow         string   `json:"flow"`
	FallbackDest string   `json:"fallback_dest,omitempty"`

	// Mode: standalone | cascade
	Mode string `json:"mode,omitempty"`
	// Cascade keeps upstream routing chain configuration for multi-hop.
	Cascade *XrayCascadeConfig `json:"cascade,omitempty"`
	// SystemClients are non-user internal identities (e.g. cascade interconnect).
	SystemClients []XraySystemClient `json:"system_clients,omitempty"`
}

type XrayCascadeRule struct {
	// Match examples: geoip:ru, geoip:!ru, geosite:netflix
	Match string `json:"match"`
	// Outbound: direct | proxy
	Outbound string `json:"outbound"`
}

type XrayCascadeConfig struct {
	UpstreamServerID string            `json:"upstream_server_id"`
	UpstreamHost     string            `json:"upstream_host"`
	UpstreamPort     uint16            `json:"upstream_port"`
	UpstreamSNI      string            `json:"upstream_sni"`
	UpstreamPubKey   string            `json:"upstream_public_key"`
	UpstreamShortID  string            `json:"upstream_short_id"`
	UpstreamAuthUUID string            `json:"upstream_auth_uuid"`
	Rules            []XrayCascadeRule `json:"rules,omitempty"`
}

type XraySystemClient struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Flow  string `json:"flow,omitempty"`
}

// XrayConfigFromJSON safely decodes protocol_config.
func XrayConfigFromJSON(b []byte) (XrayConfig, error) {
	var c XrayConfig
	if len(b) == 0 || string(b) == "null" {
		return c, nil
	}
	err := json.Unmarshal(b, &c)
	return c, err
}

// XrayPublic is a client-safe view without private key material.
type XrayPublic struct {
	InboundPort uint16
	SNI         string
	PublicKey   string
	Fingerprint string
	Flow        string
	ShortID     string
}

// PublicView creates a public config fragment for one client.
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
