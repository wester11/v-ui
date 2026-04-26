package domain

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// Server — VPN-нода (агент).
type Server struct {
	ID            uuid.UUID
	Name          string
	Endpoint      string // host:port (UDP)
	PublicKey     string
	ListenPort    uint16
	Subnet        netip.Prefix // CIDR для peer'ов
	DNS           []netip.Addr
	ObfsEnabled   bool
	AgentToken    string // токен для аутентификации агента
	LastHeartbeat *time.Time
	Online        bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
