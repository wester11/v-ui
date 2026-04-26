package domain

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// Peer — клиентское подключение WireGuard.
//
// Phase 4: приватный ключ клиента НИКОГДА не покидает устройство пользователя.
// Сервер хранит только публичный ключ.
type Peer struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	ServerID      uuid.UUID
	Name          string
	PublicKey     string
	AssignedIP    netip.Addr
	AllowedIPs    []netip.Prefix
	Enabled       bool
	LastHandshake *time.Time
	BytesRx       uint64
	BytesTx       uint64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewPeer(userID, serverID uuid.UUID, name, publicKey string) *Peer {
	now := time.Now().UTC()
	return &Peer{
		ID:        uuid.New(),
		UserID:    userID,
		ServerID:  serverID,
		Name:      name,
		PublicKey: publicKey,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
