package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/netip"
	"strconv"
	"strings"
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
	Protocol    domain.Protocol // "" -> wireguard
	Endpoint    string
	ListenPort  uint16
	TCPPort     uint16
	TLSPort     uint16
	Subnet      string
	DNS         []string
	ObfsEnabled bool

	// Xray-specific
	XrayInboundPort uint16
	XraySNI         string
	XrayDest        string
	XrayShortIDsN   int
	XrayFingerprint string
	XrayFlow        string

	// Advanced Xray routing mode
	Mode              string
	CascadeUpstreamID string
	CascadeRules      []domain.XrayCascadeRule
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
		fingerprint := in.XrayFingerprint
		if fingerprint == "" {
			fingerprint = "chrome"
		}
		flow := in.XrayFlow
		if flow == "" {
			flow = "xtls-rprx-vision"
		}

		xc := domain.XrayConfig{
			InboundPort:  port,
			SNI:          sni,
			Dest:         dest,
			PrivateKey:   priv,
			PublicKey:    pub,
			ShortIDs:     shortIDs,
			Fingerprint:  fingerprint,
			Flow:         flow,
			Mode:         "standalone",
		}

		mode := strings.ToLower(strings.TrimSpace(in.Mode))
		if mode == "" {
			mode = "standalone"
		}
		if mode != "standalone" && mode != "cascade" {
			return nil, domain.ErrValidation
		}

		if mode == "cascade" {
			upID, err := uuid.Parse(strings.TrimSpace(in.CascadeUpstreamID))
			if err != nil {
				return nil, domain.ErrValidation
			}
			if upID == srvID {
				return nil, domain.ErrValidation
			}
			upstream, err := s.repo.GetByID(ctx, upID)
			if err != nil {
				return nil, err
			}
			if upstream.Protocol != domain.ProtoXray {
				return nil, domain.ErrValidation
			}
			upCfg, err := domain.XrayConfigFromJSON(upstream.ProtocolConfig)
			if err != nil {
				return nil, domain.ErrValidation
			}
			if upCfg.PublicKey == "" || len(upCfg.ShortIDs) == 0 {
				return nil, domain.ErrValidation
			}

			host, upstreamPort := splitEndpointHostPort(upstream.Endpoint)
			if host == "" {
				return nil, domain.ErrValidation
			}
			if upCfg.InboundPort > 0 {
				upstreamPort = upCfg.InboundPort
			}

			authUUID := uuid.NewString()
			xc.Mode = "cascade"
			xc.Cascade = &domain.XrayCascadeConfig{
				UpstreamServerID: upID.String(),
				UpstreamHost:     host,
				UpstreamPort:     upstreamPort,
				UpstreamSNI:      ifEmpty(upCfg.SNI, host),
				UpstreamPubKey:   upCfg.PublicKey,
				UpstreamShortID:  upCfg.ShortIDs[0],
				UpstreamAuthUUID: authUUID,
				Rules:            normalizeCascadeRules(in.CascadeRules),
			}

			ensureSystemClient(&upCfg, authUUID, "cascade:"+srvID.String(), ifEmpty(upCfg.Flow, "xtls-rprx-vision"))
			upRaw, err := json.Marshal(upCfg)
			if err != nil {
				return nil, err
			}
			upstream.ProtocolConfig = upRaw
			upstream.UpdatedAt = now
			if err := s.repo.Update(ctx, upstream); err != nil {
				return nil, err
			}
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

func ensureSystemClient(cfg *domain.XrayConfig, id, email, flow string) {
	for _, c := range cfg.SystemClients {
		if c.ID == id {
			return
		}
	}
	cfg.SystemClients = append(cfg.SystemClients, domain.XraySystemClient{
		ID: id, Email: email, Flow: flow,
	})
}

func normalizeCascadeRules(in []domain.XrayCascadeRule) []domain.XrayCascadeRule {
	if len(in) == 0 {
		return []domain.XrayCascadeRule{
			{Match: "geoip:ru", Outbound: "direct"},
			{Match: "geoip:!ru", Outbound: "proxy"},
		}
	}
	out := make([]domain.XrayCascadeRule, 0, len(in))
	for _, rule := range in {
		m := strings.TrimSpace(rule.Match)
		o := strings.ToLower(strings.TrimSpace(rule.Outbound))
		if m == "" {
			continue
		}
		if o != "proxy" {
			o = "direct"
		}
		out = append(out, domain.XrayCascadeRule{Match: m, Outbound: o})
	}
	if len(out) == 0 {
		return []domain.XrayCascadeRule{{Match: "geoip:ru", Outbound: "direct"}, {Match: "geoip:!ru", Outbound: "proxy"}}
	}
	return out
}

func splitEndpointHostPort(endpoint string) (string, uint16) {
	host, p, err := net.SplitHostPort(endpoint)
	if err == nil {
		port, convErr := strconv.Atoi(p)
		if convErr == nil && port > 0 && port <= 65535 {
			return host, uint16(port)
		}
		return host, 443
	}
	if strings.Contains(err.Error(), "missing port in address") {
		return endpoint, 443
	}
	return "", 0
}

func ifEmpty(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
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
