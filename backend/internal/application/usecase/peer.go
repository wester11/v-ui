package usecase

import (
	"context"
	"net/netip"
	"strings"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
	"github.com/voidwg/control/pkg/wireguard"
)

// PeerService — управление peer'ами БЕЗ хранения приватных ключей.
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
	PublicKey string // WireGuard X25519 public key (base64), генерится на клиенте
}

// Create — добавляет peer'а с public_key, полученным от клиента.
// Сервер выделяет IP, пушит конфиг агенту, рендерит [Interface]-stub
// БЕЗ PrivateKey (клиент сам подставит свой приватник в конфиг).
func (s *PeerService) Create(ctx context.Context, in CreatePeerInput) (*domain.Peer, string, error) {
	pub := strings.TrimSpace(in.PublicKey)
	if !validBase64Key(pub) {
		return nil, "", domain.ErrInvalidInput
	}
	srv, err := s.servers.GetByID(ctx, in.ServerID)
	if err != nil {
		return nil, "", err
	}
	if existing, _ := s.peers.GetByPublicKey(ctx, pub); existing != nil {
		return nil, "", domain.ErrAlreadyExists
	}

	ip, err := s.allocateIP(ctx, srv)
	if err != nil {
		return nil, "", err
	}

	peer := domain.NewPeer(in.UserID, in.ServerID, in.Name, pub)
	peer.AssignedIP = ip
	peer.AllowedIPs = []netip.Prefix{netip.PrefixFrom(ip, 32)}

	if err := s.peers.Create(ctx, peer); err != nil {
		return nil, "", err
	}
	if err := s.agents.ApplyPeer(ctx, srv, peer); err != nil {
		// мягкая ошибка: peer создан, но не применён — оператор увидит в UI
		return peer, wireguard.RenderClientConfigStub(peer, srv), err
	}
	return peer, wireguard.RenderClientConfigStub(peer, srv), nil
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

// Config — повторно отрендерить конфиг-stub (без PrivateKey).
// Клиент должен сам подставить свой приватник, который есть только у него.
func (s *PeerService) Config(ctx context.Context, peerID uuid.UUID) (string, error) {
	p, err := s.peers.GetByID(ctx, peerID)
	if err != nil {
		return "", err
	}
	srv, err := s.servers.GetByID(ctx, p.ServerID)
	if err != nil {
		return "", err
	}
	return wireguard.RenderClientConfigStub(p, srv), nil
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

// validBase64Key — поверхностная проверка X25519 public key в base64.
func validBase64Key(k string) bool {
	if len(k) != 44 {
		return false
	}
	if !strings.HasSuffix(k, "=") {
		return false
	}
	for _, c := range k {
		switch {
		case c >= 'A' && c <= 'Z':
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '+' || c == '/' || c == '=':
		default:
			return false
		}
	}
	return true
}
