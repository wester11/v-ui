package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/domain"
)

// PasswordHasher — argon2id / bcrypt.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) bool
}

// KeyGenerator — генератор ключевых пар WireGuard (curve25519).
type KeyGenerator interface {
	NewKeyPair() (privateKey, publicKey string, err error)
	NewPresharedKey() (string, error)
}

// SecretBox — симметричное шифрование приватных ключей peer'ов.
type SecretBox interface {
	Seal(plaintext []byte) ([]byte, error)
	Open(ciphertext []byte) ([]byte, error)
}

// TokenIssuer — JWT.
type TokenIssuer interface {
	Issue(userID uuid.UUID, role domain.Role, ttl time.Duration) (string, error)
	IssueRefresh(userID uuid.UUID, ttl time.Duration) (string, error)
	Verify(token string) (Claims, error)
}

type Claims struct {
	UserID uuid.UUID
	Role   domain.Role
	Exp    time.Time
}

// AgentTransport — связь с удалёнными агентами (gRPC/HTTP).
type AgentTransport interface {
	ApplyPeer(ctx context.Context, server *domain.Server, peer *domain.Peer) error
	RevokePeer(ctx context.Context, server *domain.Server, peerID uuid.UUID) error
	Health(ctx context.Context, server *domain.Server) error
}
