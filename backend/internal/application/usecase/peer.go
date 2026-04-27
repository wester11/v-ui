package usecase

import (
	"context"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
	"github.com/voidwg/control/pkg/wireguard"
	"github.com/voidwg/control/pkg/xray"
)

// PeerService is protocol-aware: WG/AWG and Xray.
type PeerService struct {
	peers   port.PeerRepository
	servers port.ServerRepository
	agents  port.AgentTransport
}

func NewPeerService(p port.PeerRepository, s port.ServerRepository, a port.AgentTransport) *PeerService {
	return &PeerService{peers: p, servers: s, agents: a}
}

type CreatePeerInput struct {
	UserID    uuid.UUID
	ServerID  uuid.UUID
	Name      string
	PublicKey string // WG/AWG only
}

type FleetRedeployResult struct {
	ServerID uuid.UUID `json:"server_id"`
	Name     string    `json:"name"`
	Status   string    `json:"status"`
	Retries  int       `json:"retries"`
	Error    string    `json:"error,omitempty"`
}

type FleetHealthResult struct {
	ServerID      uuid.UUID  `json:"server_id"`
	Name          string     `json:"name"`
	Protocol      string     `json:"protocol"`
	Status        string     `json:"status"`
	LastHeartbeat *time.Time `json:"last_heartbeat,omitempty"`
	Error         string     `json:"error,omitempty"`
}

func (s *PeerService) Create(ctx context.Context, in CreatePeerInput) (*domain.Peer, string, error) {
	srv, err := s.servers.GetByID(ctx, in.ServerID)
	if err != nil {
		return nil, "", err
	}
	switch srv.Protocol {
	case "", domain.ProtoWireGuard, domain.ProtoAmneziaWG:
		return s.createWG(ctx, srv, in)
	case domain.ProtoXray:
		return s.createXray(ctx, srv, in)
	case domain.ProtoNone:
		return nil, "", domain.ErrValidation
	default:
		return nil, "", domain.ErrInvalidInput
	}
}

func (s *PeerService) createWG(ctx context.Context, srv *domain.Server, in CreatePeerInput) (*domain.Peer, string, error) {
	pub := strings.TrimSpace(in.PublicKey)
	if !validBase64Key(pub) {
		return nil, "", domain.ErrInvalidInput
	}
	if existing, _ := s.peers.GetByPublicKey(ctx, pub); existing != nil {
		return nil, "", domain.ErrAlreadyExists
	}
	ip, err := s.allocateIP(ctx, srv)
	if err != nil {
		return nil, "", err
	}
	peer := domain.NewWGPeer(in.UserID, in.ServerID, in.Name, pub)
	peer.Protocol = srv.Protocol
	if peer.Protocol == "" {
		peer.Protocol = domain.ProtoWireGuard
	}
	peer.AssignedIP = ip
	peer.AllowedIPs = []netip.Prefix{netip.PrefixFrom(ip, 32)}

	if err := s.peers.Create(ctx, peer); err != nil {
		return nil, "", err
	}
	if err := s.agents.ApplyPeer(ctx, srv, peer); err != nil {
		return peer, wireguard.RenderClientConfigStub(peer, srv), err
	}
	return peer, wireguard.RenderClientConfigStub(peer, srv), nil
}

func (s *PeerService) createXray(ctx context.Context, srv *domain.Server, in CreatePeerInput) (*domain.Peer, string, error) {
	xc, err := domain.XrayConfigFromJSON(srv.ProtocolConfig)
	if err != nil || len(xc.ShortIDs) == 0 {
		return nil, "", domain.ErrValidation
	}

	vlessUUID := uuid.NewString()
	shortID := pickShortID(xc.ShortIDs, vlessUUID)
	flow := xc.Flow
	if flow == "" {
		flow = "xtls-rprx-vision"
	}

	peer := domain.NewXrayPeer(in.UserID, in.ServerID, in.Name, vlessUUID, shortID, flow)
	if err := s.peers.Create(ctx, peer); err != nil {
		return nil, "", err
	}
	cfgURI := xray.RenderVLESSURI(srv, peer, xc.PublicView(shortID))
	if err := s.deployXray(ctx, srv); err != nil {
		return peer, cfgURI, err
	}
	return peer, cfgURI, nil
}

// deployXray rebuilds full config.json and pushes it to the node agent.
func (s *PeerService) deployXray(ctx context.Context, srv *domain.Server) error {
	allPeers, err := s.peers.ListByServer(ctx, srv.ID)
	if err != nil {
		return err
	}
	cfg, err := xray.BuildFullConfig(srv, allPeers)
	if err != nil {
		return err
	}
	return s.agents.DeployConfig(ctx, srv, cfg)
}

func (s *PeerService) Redeploy(ctx context.Context, serverID uuid.UUID) error {
	srv, err := s.servers.GetByID(ctx, serverID)
	if err != nil {
		return err
	}
	if srv.Protocol != domain.ProtoXray {
		return nil
	}
	return s.deployWithRetry(ctx, srv, 3)
}

func (s *PeerService) RedeployAll(ctx context.Context) ([]FleetRedeployResult, error) {
	servers, err := s.servers.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]FleetRedeployResult, 0, len(servers))
	for _, srv := range servers {
		if srv.Protocol != domain.ProtoXray {
			continue
		}
		result := FleetRedeployResult{ServerID: srv.ID, Name: srv.Name, Status: "ok", Retries: 1}
		if err := s.deployWithRetry(ctx, srv, 3); err != nil {
			result.Status = "error"
			result.Error = err.Error()
			result.Retries = 3
		}
		out = append(out, result)
	}
	return out, nil
}

func (s *PeerService) HealthReport(ctx context.Context) ([]FleetHealthResult, error) {
	servers, err := s.servers.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]FleetHealthResult, 0, len(servers))
	for _, srv := range servers {
		status := "offline"
		item := FleetHealthResult{
			ServerID:      srv.ID,
			Name:          srv.Name,
			Protocol:      string(srv.Protocol),
			Status:        status,
			LastHeartbeat: srv.LastHeartbeat,
		}
		probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := s.agents.Health(probeCtx, srv)
		cancel()
		if err == nil {
			if srv.Online {
				item.Status = "online"
			} else {
				item.Status = "degraded"
			}
		} else {
			item.Error = err.Error()
			if srv.Online {
				item.Status = "degraded"
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *PeerService) deployWithRetry(ctx context.Context, srv *domain.Server, attempts int) error {
	if attempts <= 1 {
		return s.deployXray(ctx, srv)
	}
	var last error
	for i := 0; i < attempts; i++ {
		if err := s.deployXray(ctx, srv); err == nil {
			return nil
		} else {
			last = err
		}
		if i < attempts-1 {
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
		}
	}
	return last
}

func (s *PeerService) Revoke(ctx context.Context, peerID uuid.UUID) error {
	p, err := s.peers.GetByID(ctx, peerID)
	if err != nil {
		return err
	}
	srv, err := s.servers.GetByID(ctx, p.ServerID)
	if err != nil {
		return err
	}
	if srv.Protocol == domain.ProtoXray {
		if err := s.peers.Delete(ctx, peerID); err != nil {
			return err
		}
		return s.deployXray(ctx, srv)
	}
	if err := s.agents.RevokePeer(ctx, srv, p.ID); err != nil {
		return err
	}
	return s.peers.Delete(ctx, peerID)
}

func (s *PeerService) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Peer, error) {
	return s.peers.ListByUser(ctx, userID)
}

func (s *PeerService) Config(ctx context.Context, peerID uuid.UUID) (string, error) {
	p, err := s.peers.GetByID(ctx, peerID)
	if err != nil {
		return "", err
	}
	srv, err := s.servers.GetByID(ctx, p.ServerID)
	if err != nil {
		return "", err
	}
	switch p.Protocol {
	case "", domain.ProtoWireGuard, domain.ProtoAmneziaWG:
		return wireguard.RenderClientConfigStub(p, srv), nil
	case domain.ProtoXray:
		xc, err := domain.XrayConfigFromJSON(srv.ProtocolConfig)
		if err != nil {
			return "", err
		}
		return xray.RenderVLESSURI(srv, p, xc.PublicView(p.XrayShortID)), nil
	}
	return "", domain.ErrInvalidInput
}

func (s *PeerService) allocateIP(ctx context.Context, srv *domain.Server) (netip.Addr, error) {
	used, err := s.peers.UsedIPs(ctx, srv.ID)
	if err != nil {
		return netip.Addr{}, err
	}
	usedSet := make(map[string]struct{}, len(used)+1)
	for _, ip := range used {
		usedSet[ip] = struct{}{}
	}
	if !srv.Subnet.IsValid() {
		return netip.Addr{}, domain.ErrPoolExhausted
	}
	usedSet[srv.Subnet.Addr().Next().String()] = struct{}{}
	addr := srv.Subnet.Addr().Next().Next()
	for srv.Subnet.Contains(addr) {
		if _, taken := usedSet[addr.String()]; !taken {
			return addr, nil
		}
		addr = addr.Next()
	}
	return netip.Addr{}, domain.ErrPoolExhausted
}

func validBase64Key(k string) bool {
	if len(k) != 44 || !strings.HasSuffix(k, "=") {
		return false
	}
	for _, c := range k {
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9',
			c == '+', c == '/', c == '=':
		default:
			return false
		}
	}
	return true
}

func pickShortID(pool []string, key string) string {
	if len(pool) == 0 {
		return ""
	}
	h := uint32(0)
	for i := 0; i < len(key); i++ {
		h = h*31 + uint32(key[i])
	}
	return pool[int(h)%len(pool)]
}
