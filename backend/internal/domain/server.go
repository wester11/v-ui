package domain

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// AWGParams — параметры AmneziaWG-style обфускации, передаются клиенту.
//
// Эти значения встраиваются в peer-конфиг и реализуют:
//   - Jc (JunkPacketCount): N мусорных пакетов перед handshake'ом
//   - Jmin/Jmax: длина каждого мусорного пакета (random в [Jmin, Jmax])
//   - S1/S2: случайные байты, препендятся к InitiationPacket / ResponsePacket
//   - H1/H2/H3/H4: подменяемые значения message_type для handshake-кадров
//
// Совпадает с reference-имплементацией amnezia-vpn/amneziawg-go.
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

// Server — VPN-нода (агент).
type Server struct {
	ID                     uuid.UUID
	Name                   string
	Endpoint               string // host:port (UDP)
	PublicKey              string
	ListenPort             uint16
	TCPPort                uint16 // fallback: UDP-over-TCP (0 = выключено)
	TLSPort                uint16 // fallback: UDP-over-TLS (0 = выключено)
	Subnet                 netip.Prefix
	DNS                    []netip.Addr
	ObfsEnabled            bool
	AWG                    AWGParams
	AgentToken             string
	AgentCertFingerprint   string // SHA-256 client-cert fingerprint для mTLS
	LastHeartbeat          *time.Time
	Online                 bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
}
