package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
	"github.com/voidwg/control/pkg/xray"
)

type ConfigService struct {
	configs port.ConfigRepository
	servers port.ServerRepository
	peers   port.PeerRepository
	agents  port.AgentTransport
}

func NewConfigService(cfgRepo port.ConfigRepository, srvRepo port.ServerRepository, peerRepo port.PeerRepository, agents port.AgentTransport) *ConfigService {
	return &ConfigService{configs: cfgRepo, servers: srvRepo, peers: peerRepo, agents: agents}
}

type CreateConfigInput struct {
	ServerID      uuid.UUID
	Name          string
	Protocol      domain.Protocol
	Template      domain.ConfigTemplate
	SetupMode     domain.ConfigSetupMode
	RoutingMode   domain.ConfigRoutingMode
	Activate      bool
	RawJSON       string
	InboundPort   uint16
	SNI           string
	Dest          string
	Fingerprint   string
	Flow          string
	ShortIDsCount int

	CascadeUpstreamID string
	CascadeStrategy   string
	CascadeRules      []domain.XrayCascadeRule
}

func (s *ConfigService) Create(ctx context.Context, in CreateConfigInput) (*domain.VPNConfig, error) {
	if in.ServerID == uuid.Nil || strings.TrimSpace(in.Name) == "" {
		return nil, domain.ErrValidation
	}
	if !in.Protocol.Valid() || in.Protocol == domain.ProtoNone {
		return nil, domain.ErrValidation
	}

	server, err := s.servers.GetByID(ctx, in.ServerID)
	if err != nil {
		return nil, err
	}

	cfg := &domain.VPNConfig{
		ID:          uuid.New(),
		ServerID:    in.ServerID,
		Name:        strings.TrimSpace(in.Name),
		Protocol:    in.Protocol,
		Template:    in.Template,
		SetupMode:   in.SetupMode,
		RoutingMode: in.RoutingMode,
		IsActive:    false,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	switch in.Protocol {
	case domain.ProtoXray:
		xcfg, err := s.buildXrayConfig(ctx, server, in)
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(xcfg)
		if err != nil {
			return nil, err
		}
		cfg.Settings = raw
	default:
		return nil, domain.ErrInvalidInput
	}

	if err := s.configs.Create(ctx, cfg); err != nil {
		return nil, err
	}
	if in.Activate {
		if err := s.Activate(ctx, cfg.ID); err != nil {
			return nil, err
		}
		active, err := s.configs.GetByID(ctx, cfg.ID)
		if err != nil {
			return nil, err
		}
		return active, nil
	}
	return cfg, nil
}

func (s *ConfigService) ListByServer(ctx context.Context, serverID uuid.UUID) ([]*domain.VPNConfig, error) {
	return s.configs.ListByServer(ctx, serverID)
}

func (s *ConfigService) Activate(ctx context.Context, configID uuid.UUID) error {
	cfg, err := s.configs.GetByID(ctx, configID)
	if err != nil {
		return err
	}
	server, err := s.servers.GetByID(ctx, cfg.ServerID)
	if err != nil {
		return err
	}

	if err := s.configs.SetActive(ctx, cfg.ServerID, cfg.ID); err != nil {
		return err
	}

	server.Protocol = cfg.Protocol
	server.ProtocolConfig = cfg.Settings
	server.UpdatedAt = time.Now().UTC()

	if cfg.Protocol == domain.ProtoXray {
		xc, err := domain.XrayConfigFromJSON(cfg.Settings)
		if err == nil {
			server.TLSPort = xc.InboundPort
		}
	}
	if err := s.servers.Update(ctx, server); err != nil {
		return err
	}

	return s.deployActiveConfig(ctx, server)
}

func (s *ConfigService) deployActiveConfig(ctx context.Context, server *domain.Server) error {
	if server.Protocol != domain.ProtoXray {
		return nil
	}
	peers, err := s.peers.ListByServer(ctx, server.ID)
	if err != nil {
		return err
	}
	full, err := xray.BuildFullConfig(server, peers)
	if err != nil {
		return err
	}
	return s.agents.DeployConfig(ctx, server, full)
}

func (s *ConfigService) buildXrayConfig(ctx context.Context, server *domain.Server, in CreateConfigInput) (domain.XrayConfig, error) {
	if in.SetupMode == "" {
		in.SetupMode = domain.ConfigSetupSimple
	}
	if in.RoutingMode == "" {
		in.RoutingMode = domain.ConfigRoutingSimple
	}

	if in.SetupMode == domain.ConfigSetupAdvanced {
		raw := strings.TrimSpace(in.RawJSON)
		if raw == "" {
			return domain.XrayConfig{}, domain.ErrValidation
		}
		if !json.Valid([]byte(raw)) {
			return domain.XrayConfig{}, domain.ErrValidation
		}
		xc, err := xray.ExtractAdvancedMeta([]byte(raw))
		if err != nil {
			return domain.XrayConfig{}, domain.ErrValidation
		}
		xc.RawConfig = []byte(raw)
		xc.Mode = "standalone"
		if in.RoutingMode == domain.ConfigRoutingCascade {
			c, err := s.buildCascade(ctx, server, in)
			if err != nil {
				return domain.XrayConfig{}, err
			}
			xc.Mode = "cascade"
			xc.Cascade = c
		}
		return xc, nil
	}

	privateKey, publicKey, err := xray.GenerateX25519()
	if err != nil {
		return domain.XrayConfig{}, err
	}
	shortIDsCount := in.ShortIDsCount
	if shortIDsCount <= 0 || shortIDsCount > 16 {
		shortIDsCount = 3
	}
	shortIDs := make([]string, 0, shortIDsCount)
	for i := 0; i < shortIDsCount; i++ {
		sid, err := xray.GenerateShortID()
		if err != nil {
			return domain.XrayConfig{}, err
		}
		shortIDs = append(shortIDs, sid)
	}

	out := domain.XrayConfig{
		InboundPort: in.InboundPort,
		SNI:         ifEmpty(strings.TrimSpace(in.SNI), "www.cloudflare.com"),
		Dest:        ifEmpty(strings.TrimSpace(in.Dest), "www.cloudflare.com:443"),
		PrivateKey:  privateKey,
		PublicKey:   publicKey,
		ShortIDs:    shortIDs,
		Fingerprint: ifEmpty(strings.TrimSpace(in.Fingerprint), "chrome"),
		Flow:        ifEmpty(strings.TrimSpace(in.Flow), "xtls-rprx-vision"),
		Mode:        "standalone",
	}
	if out.InboundPort == 0 {
		out.InboundPort = 443
	}

	if in.RoutingMode == domain.ConfigRoutingCascade {
		c, err := s.buildCascade(ctx, server, in)
		if err != nil {
			return domain.XrayConfig{}, err
		}
		out.Mode = "cascade"
		out.Cascade = c
	}
	return out, nil
}

func (s *ConfigService) buildCascade(ctx context.Context, current *domain.Server, in CreateConfigInput) (*domain.XrayCascadeConfig, error) {
	upID, err := uuid.Parse(strings.TrimSpace(in.CascadeUpstreamID))
	if err != nil || upID == current.ID {
		return nil, domain.ErrValidation
	}
	upstream, err := s.servers.GetByID(ctx, upID)
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
	authUUID := uuid.NewString()
	ensureSystemClient(&upCfg, authUUID, "cascade:"+current.ID.String(), ifEmpty(upCfg.Flow, "xtls-rprx-vision"))
	updatedRaw, err := json.Marshal(upCfg)
	if err == nil {
		upstream.ProtocolConfig = updatedRaw
		upstream.UpdatedAt = time.Now().UTC()
		_ = s.servers.Update(ctx, upstream)
	}

	host := upstream.Endpoint
	port := uint16(443)
	h, p := splitEndpointHostPort(upstream.Endpoint)
	if h != "" {
		host = h
		port = p
	}
	if upCfg.InboundPort > 0 {
		port = upCfg.InboundPort
	}
	return &domain.XrayCascadeConfig{
		UpstreamServerID: upstream.ID.String(),
		UpstreamHost:     host,
		UpstreamPort:     port,
		UpstreamSNI:      ifEmpty(upCfg.SNI, host),
		UpstreamPubKey:   upCfg.PublicKey,
		UpstreamShortID:  upCfg.ShortIDs[0],
		UpstreamAuthUUID: authUUID,
		Strategy:         ifEmpty(in.CascadeStrategy, "leastPing"),
		Rules:            normalizeCascadeRules(in.CascadeRules),
	}, nil
}
