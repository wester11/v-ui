package usecase

import (
	"context"
	"encoding/json"
	"net/netip"
	"strings"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
	"github.com/voidwg/control/pkg/wireguard"
	"github.com/voidwg/control/pkg/xray"
)

// PeerService — protocol-aware: WG/AWG vs Xray.
type PeerService struct {
	peers   port.PeerRepository
	servers port.ServerRepository
	agents  port.AgentTransport
}

func NewPeerService(p port.PeerRepository, s port.ServerRepository, a port.AgentTransport) *PeerService {
	return &PeerService{peers: p, servers: s, agents: a}
}

// CreatePeerInput — для wireguard/amneziawg обязателен PublicKey;
// для xray PublicKey игнорируется, сервис генерит UUID и берёт shortID из пула.
type CreatePeerInput struct {
	UserID    uuid.UUID
	ServerID  uuid.UUID
	Name      string
	PublicKey string // только WG/AWG
}

// Create — создаёт peer'а, branching по Server.Protocol.
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
	if err != nil {
		return nil, "", domain.ErrValidation
	}
	if len(xc.ShortIDs) == 0 {
		return nil, "", domain.ErrValidation
	}

	vlessUUID := uuid.NewString()
	// Round-robin: peer'ы распределяются по пулу shortId по hash UUID.
	shortID := pickShortID(xc.ShortIDs, vlessUUID)
	flow := xc.Flow
	if flow == "" {
		flow = "xtls-rprx-vision"
	}

	peer := domain.NewXrayPeer(in.UserID, in.ServerID, in.Name, vlessUUID, shortID, flow)
	if err := s.peers.Create(ctx, peer); err != nil {
		return nil, "", err
	}
	// Уведомить агента о новом peer (тот перерендерит config.json и reload).
	if err := s.agents.ApplyPeer(ctx, srv, peer); err != nil {
		return peer, xray.RenderVLESSURI(srv, peer, xc.PublicView(shortID)), err
	}
	return peer, xray.RenderVLESSURI(srv, peer, xc.PublicView(shortID)), nil
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
	if err := s.agents.RevokePeer(ctx, srv, p.ID); err != nil {
		return err
	}
	return s.peers.Delete(ctx, peerID)
}

func (s *PeerService) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Peer, error) {
	return s.peers.ListByUser(ctx, userID)
}

// Config — рендерит клиентский конфиг под протокол peer'а.
//
//	WG/AWG:  wg-quick stub (без PrivateKey — клиент подставит свой)
//	Xray:    vless:// URI
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

// pickShortID — детерминированный выбор по хэшу UUID, чтобы при перерендеринге
// config.json id не "прыгал" между shortId.
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

// ensure json import remains used even if no caller marshals here directly
var _ = json.RawMessage{}
