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
	"github.com/voidwg/control/internal/infrastructure/obfuscation"
)

type ServerService struct {
	repo  port.ServerRepository
	keys  port.KeyGenerator
	mtls  port.MTLSIssuer
}

func NewServerService(r port.ServerRepository, k port.KeyGenerator, m port.MTLSIssuer) *ServerService {
	return &ServerService{repo: r, keys: k, mtls: m}
}

type RegisterServerInput struct {
	Name        string
	Endpoint    string
	ListenPort  uint16
	TCPPort     uint16
	TLSPort     uint16
	Subnet      string
	DNS         []string
	ObfsEnabled bool
}

// RegisterResult — sensitive material, выдаётся ОДИН РАЗ при создании сервера.
type RegisterResult struct {
	Server    *domain.Server
	AgentCA   []byte
	AgentCert []byte
	AgentKey  []byte
}

func (s *ServerService) Register(ctx context.Context, in RegisterServerInput) (*RegisterResult, error) {
	prefix, err := netip.ParsePrefix(in.Subnet)
	if err != nil {
		return nil, domain.ErrValidation
	}
	_, pub, err := s.keys.NewKeyPair()
	if err != nil {
		return nil, err
	}

	dns := make([]netip.Addr, 0, len(in.DNS))
	for _, d := range in.DNS {
		if a, err := netip.ParseAddr(d); err == nil {
			dns = append(dns, a)
		}
	}

	tokenBytes := make([]byte, 32)
	_, _ = rand.Read(tokenBytes)
	now := time.Now().UTC()

	srvID := uuid.New()

	caPEM, certPEM, keyPEM, fp, err := s.mtls.IssueAgentCert(srvID)
	if err != nil {
		return nil, err
	}

	srv := &domain.Server{
		ID:                   srvID,
		Name:                 in.Name,
		Endpoint:             in.Endpoint,
		PublicKey:            pub,
		ListenPort:           in.ListenPort,
		TCPPort:              in.TCPPort,
		TLSPort:              in.TLSPort,
		Subnet:               prefix,
		DNS:                  dns,
		ObfsEnabled:          in.ObfsEnabled,
		AWG:                  obfuscation.RandomParams(),
		AgentToken:           hex.EncodeToString(tokenBytes),
		AgentCertFingerprint: fp,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := s.repo.Create(ctx, srv); err != nil {
		return nil, err
	}
	return &RegisterResult{
		Server:    srv,
		AgentCA:   caPEM,
		AgentCert: certPEM,
		AgentKey:  keyPEM,
	}, nil
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
