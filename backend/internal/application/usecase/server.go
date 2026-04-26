package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/netip"
	"time"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
	"github.com/voidwg/control/internal/infrastructure/obfuscation"
	"github.com/voidwg/control/pkg/xray"
)

type ServerService struct {
	repo port.ServerRepository
	keys port.KeyGenerator
	mtls port.MTLSIssuer
}

func NewServerService(r port.ServerRepository, k port.KeyGenerator, m port.MTLSIssuer) *ServerService {
	return &ServerService{repo: r, keys: k, mtls: m}
}

type RegisterServerInput struct {
	Name        string
	Protocol    domain.Protocol  // "" -> wireguard
	Endpoint    string
	ListenPort  uint16
	TCPPort     uint16
	TLSPort     uint16
	Subnet      string
	DNS         []string
	ObfsEnabled bool

	// Xray-specific (используется только если Protocol=xray)
	XrayInboundPort uint16
	XraySNI         string   // www.cloudflare.com
	XrayDest        string   // www.cloudflare.com:443
	XrayShortIDsN   int      // сколько short_id создать (default 3)
	XrayFingerprint string   // chrome | firefox | safari
	XrayFlow        string   // xtls-rprx-vision
}

type RegisterResult struct {
	Server    *domain.Server
	AgentCA   []byte
	AgentCert []byte
	AgentKey  []byte
}

func (s *ServerService) Register(ctx context.Context, in RegisterServerInput) (*RegisterResult, error) {
	proto := in.Protocol
	if proto == "" {
		proto = domain.ProtoWireGuard
	}
	if !proto.Valid() {
		return nil, domain.ErrValidation
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
		Protocol:             proto,
		Endpoint:             in.Endpoint,
		ListenPort:           in.ListenPort,
		TCPPort:              in.TCPPort,
		TLSPort:              in.TLSPort,
		DNS:                  dns,
		ObfsEnabled:          in.ObfsEnabled,
		AgentToken:           hex.EncodeToString(tokenBytes),
		AgentCertFingerprint: fp,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	switch proto {
	case domain.ProtoWireGuard, domain.ProtoAmneziaWG:
		prefix, err := netip.ParsePrefix(in.Subnet)
		if err != nil {
			return nil, domain.ErrValidation
		}
		_, pub, err := s.keys.NewKeyPair()
		if err != nil {
			return nil, err
		}
		srv.PublicKey = pub
		srv.Subnet = prefix
		if proto == domain.ProtoAmneziaWG {
			srv.AWG = obfuscation.RandomParams()
			srv.ObfsEnabled = true
		}
	case domain.ProtoXray:
		// Reality keys (X25519) — приватник остаётся в БД серверного конфига.
		priv, pub, err := xray.GenerateX25519()
		if err != nil {
			return nil, err
		}
		n := in.XrayShortIDsN
		if n <= 0 || n > 16 {
			n = 3
		}
		shortIDs := make([]string, 0, n)
		for i := 0; i < n; i++ {
			sid, err := xray.GenerateShortID()
			if err != nil {
				return nil, err
			}
			shortIDs = append(shortIDs, sid)
		}
		port := in.XrayInboundPort
		if port == 0 {
			port = 443
		}
		dest := in.XrayDest
		if dest == "" {
			dest = "www.cloudflare.com:443"
		}
		sni := in.XraySNI
		if sni == "" {
			sni = "www.cloudflare.com"
		}
		fp := in.XrayFingerprint
		if fp == "" {
			fp = "chrome"
		}
		flow := in.XrayFlow
		if flow == "" {
			flow = "xtls-rprx-vision"
		}
		xc := domain.XrayConfig{
			InboundPort: port,
			SNI:         sni,
			Dest:        dest,
			PrivateKey:  priv,
			PublicKey:   pub,
			ShortIDs:    shortIDs,
			Fingerprint: fp,
			Flow:        flow,
		}
		raw, err := json.Marshal(xc)
		if err != nil {
			return nil, err
		}
		srv.ProtocolConfig = raw
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

func (s *ServerService) List(ctx context.Context) ([]*domain.Server, error) { return s.repo.List(ctx) }
func (s *ServerService) Get(ctx context.Context, id uuid.UUID) (*domain.Server, error) {
	return s.repo.GetByID(ctx, id)
}
func (s *ServerService) Delete(ctx context.Context, id uuid.UUID) error { return s.repo.Delete(ctx, id) }
