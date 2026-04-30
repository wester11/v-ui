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

// AgentTransport — связь с удалёнными агентами.
//
// Phase 5.1 (refactor под Remnawave-style):
//   * для WG/AWG — runtime peer-mutation (ApplyPeer / RevokePeer)
//   * для Xray — НЕ peer-mutation. Control-plane целиком собирает config.json
//     и пушит на агента через DeployConfig. Агент только пишет файл и
//     перезапускает контейнер xray. Никакого in-memory состояния peer'ов.
type AgentTransport interface {
	ApplyPeer(ctx context.Context, server *domain.Server, peer *domain.Peer) error
	RevokePeer(ctx context.Context, server *domain.Server, peerID uuid.UUID) error
	DeployConfig(ctx context.Context, server *domain.Server, configJSON []byte) error
	Health(ctx context.Context, server *domain.Server) error

	// Phase 7: remote node control.
	RestartService(ctx context.Context, server *domain.Server) error
	Metrics(ctx context.Context, server *domain.Server) ([]byte, error)
}

// MTLSIssuer — выпуск client-сертификатов для агентов под общий CA.
type MTLSIssuer interface {
	IssueAgentCert(serverID uuid.UUID) (caPEM, certPEM, keyPEM []byte, fingerprint string, err error)
}
