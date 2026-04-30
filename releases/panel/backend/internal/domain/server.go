package domain

import (
	"encoding/json"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type AWGParams struct {
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

// Server is infrastructure metadata for a node.
// VPN logic is configured separately via VPNConfig.
type Server struct {
	ID           uuid.UUID
	Name         string
	NodeID       uuid.UUID
	NodeSecret   string
	Hostname     string
	IP           string
	Status       string // pending | online | offline | error
	AgentVersion string

	Protocol       Protocol        // active config protocol snapshot
	ProtocolConfig json.RawMessage // active config snapshot
	Endpoint       string          // ip/hostname:port of agent API

	PublicKey      string // legacy WG field
	ListenPort     uint16
	TCPPort        uint16
	TLSPort        uint16
	Subnet         netip.Prefix
	DNS            []netip.Addr
	ObfsEnabled    bool
	AWG            AWGParams
	AgentToken     string
	AgentCertFingerprint string
	LastHeartbeat  *time.Time
	Online         bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
