package domain

import (
	"encoding/json"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// AWGParams — параметры AmneziaWG-обфускации (см. Phase 4).
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

// Server — VPN-нода. Поле Protocol определяет, какой транспорт запускает агент.
type Server struct {
	ID                   uuid.UUID
	Name                 string
	Protocol             Protocol         // wireguard | amneziawg | xray
	ProtocolConfig       json.RawMessage  // protocol-specific (XrayConfig для xray)
	Endpoint             string           // host:port (для WG/AWG — UDP, для xray — TCP)
	PublicKey            string           // WG public key (для xray не используется)
	ListenPort           uint16
	TCPPort              uint16
	TLSPort              uint16
	Subnet               netip.Prefix
	DNS                  []netip.Addr
	ObfsEnabled          bool
	AWG                  AWGParams
	AgentToken           string
	AgentCertFingerprint string
	LastHeartbeat        *time.Time
	Online               bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
