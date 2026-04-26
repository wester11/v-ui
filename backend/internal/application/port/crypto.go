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
//
// Phase 4: используется ТОЛЬКО для серверной части (server.PublicKey),
// peer-keypair генерится на стороне клиента.
type KeyGenerator interface {
	NewKeyPair() (privateKey, publicKey string, err error)
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

// MTLSIssuer — выпуск client-сертификатов для агентов под общий CA.
type MTLSIssuer interface {
	IssueAgentCert(serverID uuid.UUID) (caPEM, certPEM, keyPEM []byte, fingerprint string, err error)
}
