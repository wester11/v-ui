package domain

import (
	"time"

	"github.com/google/uuid"
)

// Invite — одноразовый токен для client-side keygen flow.
//
// Жизненный цикл:
//
//	1) operator/admin создаёт invite -> POST /api/v1/invites
//	2) сервер возвращает URL вида /redeem/<token>
//	3) клиент скачивает с GET  /api/v1/invites/<token>  параметры сервера
//	4) клиент локально генерирует X25519-keypair
//	5) клиент шлёт public_key в  POST /api/v1/invites/<token>/redeem
//	6) сервер создаёт peer, привязывает invite, возвращает финальный wg-quick.
//
// Приватный ключ peer'а никогда не уезжает с устройства.
type Invite struct {
	ID            uuid.UUID
	Token         string
	ServerID      uuid.UUID
	UserID        uuid.UUID
	SuggestedName string
	ExpiresAt     time.Time
	UsedAt        *time.Time
	PeerID        *uuid.UUID
	CreatedAt     time.Time
}

func (i *Invite) Used() bool   { return i.UsedAt != nil }
func (i *Invite) Expired() bool { return time.Now().UTC().After(i.ExpiresAt) }
