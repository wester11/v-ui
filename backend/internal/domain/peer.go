package domain

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// Peer — клиентское подключение. Phase 4: приватники не хранятся.
// Phase 5: поле Protocol определяет формат конфига; для xray вместо
// PublicKey используются XrayUUID + XrayShortID.
type Peer struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	ServerID      uuid.UUID
	Protocol      Protocol
	Name          string
	PublicKey     string     // WG/AWG: X25519 public key, генерится клиентом
	XrayUUID      string     // VLESS UUID
	XrayFlow      string     // xtls-rprx-vision
	XrayShortID   string     // выбран из пула server.XrayConfig.ShortIDs
	AssignedIP    netip.Addr // только для WG/AWG
	AllowedIPs    []netip.Prefix
	Enabled       bool
	LastHandshake *time.Time
	BytesRx       uint64
	BytesTx       uint64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewWGPeer(userID, serverID uuid.UUID, name, publicKey string) *Peer {
	now := time.Now().UTC()
	return &Peer{
		ID:        uuid.New(),
		UserID:    userID,
		ServerID:  serverID,
		Protocol:  ProtoWireGuard,
		Name:      name,
		PublicKey: publicKey,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func NewXrayPeer(userID, serverID uuid.UUID, name, vlessUUID, shortID, flow string) *Peer {
	now := time.Now().UTC()
	return &Peer{
		ID:          uuid.New(),
		UserID:      userID,
		ServerID:    serverID,
		Protocol:    ProtoXray,
		Name:        name,
		XrayUUID:    vlessUUID,
		XrayShortID: shortID,
		XrayFlow:    flow,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// для backward-compat со старыми вызовами:
func NewPeer(userID, serverID uuid.UUID, name, publicKey string) *Peer {
	return NewWGPeer(userID, serverID, name, publicKey)
}
