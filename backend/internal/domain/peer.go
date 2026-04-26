package domain

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// Peer — клиентское подключение WireGuard.
type Peer struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	ServerID      uuid.UUID
	Name          string
	PublicKey     string
	PrivateKeyEnc []byte // AES-GCM зашифрованный приватник
	PresharedKey  string
	AssignedIP    netip.Addr
	AllowedIPs    []netip.Prefix
	Enabled       bool
	LastHandshake *time.Time
	BytesRx       uint64
	BytesTx       uint64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewPeer(userID, serverID uuid.UUID, name string) *Peer {
	now := time.Now().UTC()
	return &Peer{
		ID:        uuid.New(),
		UserID:    userID,
		ServerID:  serverID,
		Name:      name,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
