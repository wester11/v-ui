// Package dto — DTO между REST-слоем и use-case'ами.
package dto

import (
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

type ServerResponse struct {
	ID            uuid.UUID    `json:"id"`
	Name          string       `json:"name"`
	Protocol      string       `json:"protocol"`
	Endpoint      string       `json:"endpoint"`
	PublicKey     string       `json:"public_key,omitempty"`
	ListenPort    uint16       `json:"listen_port,omitempty"`
	TCPPort       uint16       `json:"tcp_port,omitempty"`
	TLSPort       uint16       `json:"tls_port,omitempty"`
	Subnet        string       `json:"subnet,omitempty"`
	ObfsEnabled   bool         `json:"obfs_enabled"`
	AWG           AWGParamsDTO `json:"awg,omitempty"`
	XrayInbound   uint16       `json:"xray_inbound_port,omitempty"`
	XraySNI       string       `json:"xray_sni,omitempty"`
	XrayPubKey    string       `json:"xray_public_key,omitempty"`
	Online        bool         `json:"online"`
	LastHeartbeat *time.Time   `json:"last_heartbeat,omitempty"`
}

func ServerFromDomain(s *domain.Server) ServerResponse {
	resp := ServerResponse{
		ID:            s.ID,
		Name:          s.Name,
		Protocol:      string(s.Protocol),
		Endpoint:      s.Endpoint,
		PublicKey:     s.PublicKey,
		ListenPort:    s.ListenPort,
		TCPPort:       s.TCPPort,
		TLSPort:       s.TLSPort,
		ObfsEnabled:   s.ObfsEnabled,
		Online:        s.Online,
		LastHeartbeat: s.LastHeartbeat,
	}
	if s.Subnet.IsValid() {
		resp.Subnet = s.Subnet.String()
	}
	if s.Protocol == domain.ProtoAmneziaWG {
		resp.AWG = AWGParamsDTO{
			Jc: s.AWG.Jc, Jmin: s.AWG.Jmin, Jmax: s.AWG.Jmax,
			S1: s.AWG.S1, S2: s.AWG.S2,
			H1: s.AWG.H1, H2: s.AWG.H2, H3: s.AWG.H3, H4: s.AWG.H4,
		}
	}
	if s.Protocol == domain.ProtoXray {
		if xc, err := domain.XrayConfigFromJSON(s.ProtocolConfig); err == nil {
			resp.XrayInbound = xc.InboundPort
			resp.XraySNI = xc.SNI
			resp.XrayPubKey = xc.PublicKey // public — можно показать в UI
		}
	}
	return resp
}

type CreateServerRequest struct {
	Name        string   `json:"name"`
	Protocol    string   `json:"protocol"` // wireguard | amneziawg | xray (default: wireguard)
	Endpoint    string   `json:"endpoint"`
	ListenPort  uint16   `json:"listen_port"`
	TCPPort     uint16   `json:"tcp_port"`
	TLSPort     uint16   `json:"tls_port"`
	Subnet      string   `json:"subnet"`
	DNS         []string `json:"dns"`
	ObfsEnabled bool     `json:"obfs_enabled"`

	// Xray-only (если protocol=xray):
	XrayInboundPort uint16 `json:"xray_inbound_port,omitempty"`
	XraySNI         string `json:"xray_sni,omitempty"`
	XrayDest        string `json:"xray_dest,omitempty"`
	XrayShortIDsN   int    `json:"xray_short_ids,omitempty"`
	XrayFingerprint string `json:"xray_fingerprint,omitempty"`
	XrayFlow        string `json:"xray_flow,omitempty"`
}

// CreateServerResponse — отдаётся ОДИН РАЗ при регистрации:
// agent_token + mTLS-материал. После этого CA/cert/key больше не получить.
type CreateServerResponse struct {
	ServerResponse
	AgentToken string `json:"agent_token"`
	AgentCA    string `json:"agent_ca"`   // PEM, ставится в trusted store агента
	AgentCert  string `json:"agent_cert"` // PEM, mTLS client cert
	AgentKey   string `json:"agent_key"`  // PEM, приватник client cert
}

// ===== Peer =====

type PeerResponse struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	ServerID      uuid.UUID  `json:"server_id"`
	Protocol      string     `json:"protocol"`
	Name          string     `json:"name"`
	PublicKey     string     `json:"public_key,omitempty"` // WG/AWG only
	XrayUUID      string     `json:"xray_uuid,omitempty"`  // Xray only
	XrayShortID   string     `json:"xray_short_id,omitempty"`
	AssignedIP    string     `json:"assigned_ip,omitempty"`
	Enabled       bool       `json:"enabled"`
	BytesRx       uint64     `json:"bytes_rx"`
	BytesTx       uint64     `json:"bytes_tx"`
	LastHandshake *time.Time `json:"last_handshake,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func PeerFromDomain(p *domain.Peer) PeerResponse {
	r := PeerResponse{
		ID:            p.ID,
		UserID:        p.UserID,
		ServerID:      p.ServerID,
		Protocol:      string(p.Protocol),
		Name:          p.Name,
		PublicKey:     p.PublicKey,
		XrayUUID:      p.XrayUUID,
		XrayShortID:   p.XrayShortID,
		Enabled:       p.Enabled,
		BytesRx:       p.BytesRx,
		BytesTx:       p.BytesTx,
		LastHandshake: p.LastHandshake,
		CreatedAt:     p.CreatedAt,
	}
	if p.AssignedIP.IsValid() {
		r.AssignedIP = p.AssignedIP.String()
	}
	return r
}

// CreatePeerRequest — protocol-aware.
//
//	WG/AWG:  обязателен public_key (X25519, base64).
//	Xray:    public_key игнорируется; сервис генерит UUID и shortID.
type CreatePeerRequest struct {
	ServerID  uuid.UUID `json:"server_id"`
	Name      string    `json:"name"`
	PublicKey string    `json:"public_key,omitempty"`
}

type CreatePeerResponse struct {
	Peer       PeerResponse `json:"peer"`
	ConfigStub string       `json:"config"` // wg-quick stub (без PrivateKey — клиент подставит сам)
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
	URL           string     `json:"url"` // /redeem/<token>
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

// InviteLookupResponse — публичный (без auth) ответ на GET /api/v1/invites/<token>.
// Содержит метаданные сервера, нужные клиенту для генерации конфига.
// public_key самого peer'а ещё не известен — клиент сгенерит и пришлёт через Redeem.
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
