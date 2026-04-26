package port

import (
	"context"

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
	Count(ctx context.Context) (int, error)
	TotalTraffic(ctx context.Context) (rx uint64, tx uint64, err error)
}

// ServerRepository — порт для серверов.
type ServerRepository interface {
	Create(ctx context.Context, s *domain.Server) error
	Update(ctx context.Context, s *domain.Server) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Server, error)
	GetByToken(ctx context.Context, token string) (*domain.Server, error)
	List(ctx context.Context) ([]*domain.Server, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CountOnline(ctx context.Context) (total int, online int, err error)
}
