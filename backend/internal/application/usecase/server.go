package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/netip"
	"time"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
)

type ServerService struct {
	repo port.ServerRepository
	keys port.KeyGenerator
}

func NewServerService(r port.ServerRepository, k port.KeyGenerator) *ServerService {
	return &ServerService{repo: r, keys: k}
}

type RegisterServerInput struct {
	Name        string
	Endpoint    string
	ListenPort  uint16
	Subnet      string
	DNS         []string
	ObfsEnabled bool
}

func (s *ServerService) Register(ctx context.Context, in RegisterServerInput) (*domain.Server, error) {
	prefix, err := netip.ParsePrefix(in.Subnet)
	if err != nil {
		return nil, domain.ErrValidation
	}
	priv, pub, err := s.keys.NewKeyPair()
	if err != nil {
		return nil, err
	}
	_ = priv // приватник сервера будет передан в агент при первой инициализации (через secure pipe)

	dns := make([]netip.Addr, 0, len(in.DNS))
	for _, d := range in.DNS {
		if a, err := netip.ParseAddr(d); err == nil {
			dns = append(dns, a)
		}
	}

	tokenBytes := make([]byte, 32)
	_, _ = rand.Read(tokenBytes)
	now := time.Now().UTC()

	srv := &domain.Server{
		ID:          uuid.New(),
		Name:        in.Name,
		Endpoint:    in.Endpoint,
		PublicKey:   pub,
		ListenPort:  in.ListenPort,
		Subnet:      prefix,
		DNS:         dns,
		ObfsEnabled: in.ObfsEnabled,
		AgentToken:  hex.EncodeToString(tokenBytes),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, srv); err != nil {
		return nil, err
	}
	return srv, nil
}

func (s *ServerService) Heartbeat(ctx context.Context, token string) (*domain.Server, error) {
	srv, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	srv.LastHeartbeat = &now
	srv.Online = true
	if err := s.repo.Update(ctx, srv); err != nil {
		return nil, err
	}
	return srv, nil
}

func (s *ServerService) List(ctx context.Context) ([]*domain.Server, error) {
	return s.repo.List(ctx)
}

func (s *ServerService) Get(ctx context.Context, id uuid.UUID) (*domain.Server, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ServerService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
