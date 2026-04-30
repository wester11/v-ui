// Package dto defines HTTP DTOs between handlers and use-cases.
package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/domain"
)

// ===== Auth =====

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ===== Stats =====

type StatsResponse struct {
	Users         int    `json:"users"`
	Peers         int    `json:"peers"`
	Servers       int    `json:"servers"`
	ServersOnline int    `json:"servers_online"`
	BytesRxTotal  uint64 `json:"bytes_rx_total"`
	BytesTxTotal  uint64 `json:"bytes_tx_total"`
}

// ===== User =====

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Disabled  bool      `json:"disabled"`
	CreatedAt time.Time `json:"created_at"`
}

func UserFromDomain(u *domain.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Role:      string(u.Role),
		Disabled:  u.Disabled,
		CreatedAt: u.CreatedAt,
	}
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// ===== Server =====

type AWGParamsDTO struct {
	Jc   uint8  `json:"Jc"`
	Jmin uint16 `json:"Jmin"`
	Jmax uint16 `json:"Jmax"`
	S1   uint16 `json:"S1"`
	S2   uint16 `json:"S2"`
	H1   uint32 `json:"H1"`
	H2   uint32 `json:"H2"`
	H3   uint32 `json:"H3"`
	H4   uint32 `json:"H4"`
}

type CascadeRuleDTO struct {
	Match    string `json:"match"`
	Outbound string `json:"outbound"`
}

type ServerResponse struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	NodeID        uuid.UUID  `json:"node_id"`
	Endpoint      string     `json:"endpoint"`
	Hostname      string     `json:"hostname,omitempty"`
	IP            string     `json:"ip,omitempty"`
	Status        string     `json:"status"`
	AgentVersion  string     `json:"agent_version,omitempty"`
	Online        bool       `json:"online"`
	LastHeartbeat *time.Time `json:"last_heartbeat,omitempty"`
	Protocol      string     `json:"protocol,omitempty"`
	Mode          string     `json:"mode,omitempty"`
}

func ServerFromDomain(s *domain.Server) ServerResponse {
	resp := ServerResponse{
		ID:            s.ID,
		Name:          s.Name,
		NodeID:        s.NodeID,
		Endpoint:      s.Endpoint,
		Hostname:      s.Hostname,
		IP:            s.IP,
		Status:        s.Status,
		AgentVersion:  s.AgentVersion,
		Online:        s.Online,
		LastHeartbeat: s.LastHeartbeat,
		Protocol:      string(s.Protocol),
	}
	if s.Protocol == domain.ProtoXray {
		if xc, err := domain.XrayConfigFromJSON(s.ProtocolConfig); err == nil {
			resp.Mode = ifEmpty(xc.Mode, "standalone")
		}
	}
	return resp
}

type CreateServerRequest struct {
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
}

// CreateServerResponse is returned once at registration time with agent secrets.
type CreateServerResponse struct {
	ServerResponse
	NodeID        string `json:"node_id"`
	Secret        string `json:"secret"`
	InstallCommand string `json:"install_command"`
	ComposeSnippet string `json:"compose_snippet"`
}

type ConfigResponse struct {
	ID          uuid.UUID `json:"id"`
	ServerID    uuid.UUID `json:"server_id"`
	Name        string    `json:"name"`
	Protocol    string    `json:"protocol"`
	Template    string    `json:"template"`
	SetupMode   string    `json:"setup_mode"`
	RoutingMode string    `json:"routing_mode"`
	IsActive    bool      `json:"is_active"`
	Settings    any       `json:"settings,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func ConfigFromDomain(cfg *domain.VPNConfig, includeSettings bool) ConfigResponse {
	out := ConfigResponse{
		ID:          cfg.ID,
		ServerID:    cfg.ServerID,
		Name:        cfg.Name,
		Protocol:    string(cfg.Protocol),
		Template:    string(cfg.Template),
		SetupMode:   string(cfg.SetupMode),
		RoutingMode: string(cfg.RoutingMode),
		IsActive:    cfg.IsActive,
		CreatedAt:   cfg.CreatedAt,
		UpdatedAt:   cfg.UpdatedAt,
	}
	if includeSettings && len(cfg.Settings) > 0 {
		var parsed any
		if err := json.Unmarshal(cfg.Settings, &parsed); err == nil {
			out.Settings = parsed
		}
	}
	return out
}

type CreateConfigRequest struct {
	ServerID      uuid.UUID        `json:"server_id"`
	Name          string           `json:"name"`
	Protocol      string           `json:"protocol"`
	Template      string           `json:"template"`
	SetupMode     string           `json:"setup_mode"`
	RoutingMode   string           `json:"routing_mode"`
	Activate      bool             `json:"activate"`
	RawJSON       string           `json:"raw_json,omitempty"`
	InboundPort   uint16           `json:"inbound_port,omitempty"`
	SNI           string           `json:"sni,omitempty"`
	Dest          string           `json:"dest,omitempty"`
	Fingerprint   string           `json:"fingerprint,omitempty"`
	Flow          string           `json:"flow,omitempty"`
	ShortIDsCount int              `json:"short_ids_count,omitempty"`
	CascadeUpstreamID string       `json:"cascade_upstream_id,omitempty"`
	CascadeStrategy string         `json:"cascade_strategy,omitempty"`
	CascadeRules   []CascadeRuleDTO `json:"cascade_rules,omitempty"`
}

// ===== Peer =====

type PeerResponse struct {
	ID                uuid.UUID  `json:"id"`
	UserID            uuid.UUID  `json:"user_id"`
	ServerID          uuid.UUID  `json:"server_id"`
	Protocol          string     `json:"protocol"`
	Name              string     `json:"name"`
	PublicKey         string     `json:"public_key,omitempty"`
	XrayUUID          string     `json:"xray_uuid,omitempty"`
	XrayShortID       string     `json:"xray_short_id,omitempty"`
	AssignedIP        string     `json:"assigned_ip,omitempty"`
	Enabled           bool       `json:"enabled"`
	BytesRx           uint64     `json:"bytes_rx"`
	BytesTx           uint64     `json:"bytes_tx"`
	TrafficLimitBytes uint64     `json:"traffic_limit_bytes"`
	TrafficLimitedAt  *time.Time `json:"traffic_limited_at,omitempty"`
	LastHandshake     *time.Time `json:"last_handshake,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

func PeerFromDomain(p *domain.Peer) PeerResponse {
	r := PeerResponse{
		ID:                p.ID,
		UserID:            p.UserID,
		ServerID:          p.ServerID,
		Protocol:          string(p.Protocol),
		Name:              p.Name,
		PublicKey:         p.PublicKey,
		XrayUUID:          p.XrayUUID,
		XrayShortID:       p.XrayShortID,
		Enabled:           p.Enabled,
		BytesRx:           p.BytesRx,
		BytesTx:           p.BytesTx,
		TrafficLimitBytes: p.TrafficLimitBytes,
		TrafficLimitedAt:  p.TrafficLimitedAt,
		LastHandshake:     p.LastHandshake,
		CreatedAt:         p.CreatedAt,
	}
	if p.AssignedIP.IsValid() {
		r.AssignedIP = p.AssignedIP.String()
	}
	return r
}

type CreatePeerRequest struct {
	ServerID          uuid.UUID `json:"server_id"`
	Name              string    `json:"name"`
	PublicKey         string    `json:"public_key,omitempty"`
	TrafficLimitBytes uint64    `json:"traffic_limit_bytes,omitempty"`
}

type CreatePeerResponse struct {
	Peer       PeerResponse `json:"peer"`
	ConfigStub string       `json:"config"`
}

// ===== Invite =====

type CreateInviteRequest struct {
	ServerID      uuid.UUID `json:"server_id"`
	SuggestedName string    `json:"suggested_name"`
	TTLSeconds    int64     `json:"ttl_seconds"`
}

type InviteResponse struct {
	ID            uuid.UUID  `json:"id"`
	Token         string     `json:"token"`
	URL           string     `json:"url"`
	ServerID      uuid.UUID  `json:"server_id"`
	SuggestedName string     `json:"suggested_name"`
	ExpiresAt     time.Time  `json:"expires_at"`
	UsedAt        *time.Time `json:"used_at,omitempty"`
	PeerID        *uuid.UUID `json:"peer_id,omitempty"`
}

func InviteFromDomain(i *domain.Invite, baseURL string) InviteResponse {
	return InviteResponse{
		ID:            i.ID,
		Token:         i.Token,
		URL:           baseURL + "/redeem/" + i.Token,
		ServerID:      i.ServerID,
		SuggestedName: i.SuggestedName,
		ExpiresAt:     i.ExpiresAt,
		UsedAt:        i.UsedAt,
		PeerID:        i.PeerID,
	}
}

type InviteLookupResponse struct {
	Endpoint    string       `json:"endpoint"`
	PublicKey   string       `json:"public_key"`
	ListenPort  uint16       `json:"listen_port"`
	TCPPort     uint16       `json:"tcp_port,omitempty"`
	TLSPort     uint16       `json:"tls_port,omitempty"`
	DNS         []string     `json:"dns"`
	ObfsEnabled bool         `json:"obfs_enabled"`
	AWG         AWGParamsDTO `json:"awg,omitempty"`
	ExpiresAt   time.Time    `json:"expires_at"`
	Suggested   string       `json:"suggested_name"`
}

type RedeemInviteRequest struct {
	PublicKey string `json:"public_key"`
	Name      string `json:"name"`
}

// ===== Audit =====

type AuditEntry struct {
	ID         int64          `json:"id"`
	TS         time.Time      `json:"ts"`
	ActorID    *uuid.UUID     `json:"actor_id,omitempty"`
	ActorEmail string         `json:"actor_email,omitempty"`
	Action     string         `json:"action"`
	TargetType string         `json:"target_type,omitempty"`
	TargetID   string         `json:"target_id,omitempty"`
	IP         string         `json:"ip,omitempty"`
	UserAgent  string         `json:"user_agent,omitempty"`
	Result     string         `json:"result"`
	Meta       map[string]any `json:"meta,omitempty"`
}

func AuditFromDomain(e *domain.AuditEvent) AuditEntry {
	return AuditEntry{
		ID:         e.ID,
		TS:         e.TS,
		ActorID:    e.ActorID,
		ActorEmail: e.ActorEmail,
		Action:     e.Action,
		TargetType: e.TargetType,
		TargetID:   e.TargetID,
		IP:         e.IP,
		UserAgent:  e.UserAgent,
		Result:     e.Result,
		Meta:       e.Meta,
	}
}

// ===== Errors =====

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func ifEmpty(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
