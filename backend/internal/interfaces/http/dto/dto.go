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

type ServerResponse struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Endpoint      string     `json:"endpoint"`
	PublicKey     string     `json:"public_key"`
	ListenPort    uint16     `json:"listen_port"`
	Subnet        string     `json:"subnet"`
	ObfsEnabled   bool       `json:"obfs_enabled"`
	Online        bool       `json:"online"`
	LastHeartbeat *time.Time `json:"last_heartbeat,omitempty"`
}

func ServerFromDomain(s *domain.Server) ServerResponse {
	return ServerResponse{
		ID:            s.ID,
		Name:          s.Name,
		Endpoint:      s.Endpoint,
		PublicKey:     s.PublicKey,
		ListenPort:    s.ListenPort,
		Subnet:        s.Subnet.String(),
		ObfsEnabled:   s.ObfsEnabled,
		Online:        s.Online,
		LastHeartbeat: s.LastHeartbeat,
	}
}

type CreateServerRequest struct {
	Name        string   `json:"name"`
	Endpoint    string   `json:"endpoint"`
	ListenPort  uint16   `json:"listen_port"`
	Subnet      string   `json:"subnet"`
	DNS         []string `json:"dns"`
	ObfsEnabled bool     `json:"obfs_enabled"`
}

// ===== Peer =====

type PeerResponse struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	ServerID      uuid.UUID  `json:"server_id"`
	Name          string     `json:"name"`
	PublicKey     string     `json:"public_key"`
	AssignedIP    string     `json:"assigned_ip"`
	Enabled       bool       `json:"enabled"`
	BytesRx       uint64     `json:"bytes_rx"`
	BytesTx       uint64     `json:"bytes_tx"`
	LastHandshake *time.Time `json:"last_handshake,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func PeerFromDomain(p *domain.Peer) PeerResponse {
	return PeerResponse{
		ID:            p.ID,
		UserID:        p.UserID,
		ServerID:      p.ServerID,
		Name:          p.Name,
		PublicKey:     p.PublicKey,
		AssignedIP:    p.AssignedIP.String(),
		Enabled:       p.Enabled,
		BytesRx:       p.BytesRx,
		BytesTx:       p.BytesTx,
		LastHandshake: p.LastHandshake,
		CreatedAt:     p.CreatedAt,
	}
}

type CreatePeerRequest struct {
	ServerID uuid.UUID `json:"server_id"`
	Name     string    `json:"name"`
}

type CreatePeerResponse struct {
	Peer   PeerResponse `json:"peer"`
	Config string       `json:"config"` // wg-quick .conf
}

// ===== Errors =====

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}
