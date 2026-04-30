package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/domain"
)

// UserRepository — порт для пользователей.
type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	Update(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, limit, offset int) ([]*domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int, error)
}

// PeerRepository — порт для peer'ов.
type PeerRepository interface {
	Create(ctx context.Context, p *domain.Peer) error
	Update(ctx context.Context, p *domain.Peer) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Peer, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Peer, error)
	ListByServer(ctx context.Context, serverID uuid.UUID) ([]*domain.Peer, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UsedIPs(ctx context.Context, serverID uuid.UUID) ([]string, error)
	GetByPublicKey(ctx context.Context, pubKey string) (*domain.Peer, error)
	Count(ctx context.Context) (int, error)
	TotalTraffic(ctx context.Context) (rx uint64, tx uint64, err error)
	// ListOverLimit returns enabled peers whose traffic exceeds their limit.
	ListOverLimit(ctx context.Context) ([]*domain.Peer, error)
}

// ServerRepository — порт для серверов.
type ServerRepository interface {
	Create(ctx context.Context, s *domain.Server) error
	Update(ctx context.Context, s *domain.Server) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Server, error)
	GetByToken(ctx context.Context, token string) (*domain.Server, error)
	GetByNodeID(ctx context.Context, nodeID uuid.UUID) (*domain.Server, error)
	List(ctx context.Context) ([]*domain.Server, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CountOnline(ctx context.Context) (total int, online int, err error)
	// MarkStaleOffline помечает сервера как offline, если их last_heartbeat
	// старше threshold. Возвращает количество обновлённых строк.
	MarkStaleOffline(ctx context.Context, threshold time.Duration) (int, error)
}

type ConfigRepository interface {
	Create(ctx context.Context, cfg *domain.VPNConfig) error
	Update(ctx context.Context, cfg *domain.VPNConfig) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.VPNConfig, error)
	ListByServer(ctx context.Context, serverID uuid.UUID) ([]*domain.VPNConfig, error)
	GetActiveByServer(ctx context.Context, serverID uuid.UUID) (*domain.VPNConfig, error)
	SetActive(ctx context.Context, serverID, configID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// InviteRepository — одноразовые инвайты для client-side keygen.
type InviteRepository interface {
	Create(ctx context.Context, inv *domain.Invite) error
	GetByToken(ctx context.Context, token string) (*domain.Invite, error)
	MarkUsed(ctx context.Context, id uuid.UUID, peerID uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Invite, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context) (int, error)
}

// JobRepository — distributed-task queue.
type JobRepository interface {
	Create(ctx context.Context, j *domain.Job) error
	// LeasePending — атомарно достаёт N задач для server'а, помечает running.
	// Используется long-poll'ом с агента.
	LeasePending(ctx context.Context, serverID uuid.UUID, max int) ([]*domain.Job, error)
	Submit(ctx context.Context, jobID uuid.UUID, status domain.JobStatus, result []byte, errMsg string) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error)
	ListByServer(ctx context.Context, serverID uuid.UUID, limit int) ([]*domain.Job, error)
	// RescheduleStale — задачи с status=running >5 минут возвращает в pending
	// (агент упал между lease и submit). Вызывается фоновым тикером.
	RescheduleStale(ctx context.Context, threshold time.Duration) (int, error)
}

// ConfigVersionRepository — снимки конфигов с историей.
type ConfigVersionRepository interface {
	Create(ctx context.Context, v *domain.ConfigVersion) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ConfigVersion, error)
	ListByServer(ctx context.Context, serverID uuid.UUID, limit int) ([]*domain.ConfigVersion, error)
	NextVersion(ctx context.Context, serverID uuid.UUID) (int, error)
	MarkStatus(ctx context.Context, id uuid.UUID, status domain.ConfigVersionStatus) error
}

// AuditRepository — журнал событий.
type AuditRepository interface {
	Append(ctx context.Context, ev *domain.AuditEvent) error
	List(ctx context.Context, limit int, before int64) ([]*domain.AuditEvent, error)
}
