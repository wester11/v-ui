package usecase

import (
	"context"
	"net/netip"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
	"github.com/voidwg/control/pkg/wireguard"
)

type PeerService struct {
	peers   port.PeerRepository
	servers port.ServerRepository
	keys    port.KeyGenerator
	box     port.SecretBox
	agents  port.AgentTransport
}

func NewPeerService(
	p port.PeerRepository,
	s port.ServerRepository,
	k port.KeyGenerator,
	b port.SecretBox,
	a port.AgentTransport,
) *PeerService {
	return &PeerService{peers: p, servers: s, keys: k, box: b, agents: a}
}

type CreatePeerInput struct {
	UserID   uuid.UUID
	ServerID uuid.UUID
	Name     string
}

// Provision — создаёт нового peer'а: генерирует ключи, выделяет IP, толкает на агент.
func (s *PeerService) Provision(ctx context.Context, in CreatePeerInput) (*domain.Peer, string, error) {
	srv, err := s.servers.GetByID(ctx, in.ServerID)
	if err != nil {
		return nil, "", err
	}

	priv, pub, err := s.keys.NewKeyPair()
	if err != nil {
		return nil, "", err
	}
	psk, err := s.keys.NewPresharedKey()
	if err != nil {
		return nil, "", err
	}
	encPriv, err := s.box.Seal([]byte(priv))
	if err != nil {
		return nil, "", err
	}

	ip, err := s.allocateIP(ctx, srv)
	if err != nil {
		return nil, "", err
	}

	peer := domain.NewPeer(in.UserID, in.ServerID, in.Name)
	peer.PublicKey = pub
	peer.PrivateKeyEnc = encPriv
	peer.PresharedKey = psk
	peer.AssignedIP = ip
	peer.AllowedIPs = []netip.Prefix{netip.PrefixFrom(ip, 32)}

	if err := s.peers.Create(ctx, peer); err != nil {
		return nil, "", err
	}

	if err := s.agents.ApplyPeer(ctx, srv, peer); err != nil {
		// мягкая ошибка: peer создан, но не применён — оператор увидит в UI
		return peer, wireguard.RenderClientConfig(peer, srv, priv), err
	}
	return peer, wireguard.RenderClientConfig(peer, srv, priv), nil
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

// Config — повторно отрендерить .conf (без приватного ключа в незашифрованном виде в БД мы
// должны расшифровать только по запросу владельца).
func (s *PeerService) Config(ctx context.Context, peerID uuid.UUID) (string, error) {
	p, err := s.peers.GetByID(ctx, peerID)
	if err != nil {
		return "", err
	}
	srv, err := s.servers.GetByID(ctx, p.ServerID)
	if err != nil {
		return "", err
	}
	priv, err := s.box.Open(p.PrivateKeyEnc)
	if err != nil {
		return "", err
	}
	return wireguard.RenderClientConfig(p, srv, string(priv)), nil
}

// allocateIP — линейный алгоритм выбора первого свободного IP в подсети.
func (s *PeerService) allocateIP(ctx context.Context, srv *domain.Server) (netip.Addr, error) {
	used, err := s.peers.UsedIPs(ctx, srv.ID)
	if err != nil {
		return netip.Addr{}, err
	}
	usedSet := make(map[string]struct{}, len(used)+1)
	for _, ip := range used {
		usedSet[ip] = struct{}{}
	}
	// .1 зарезервирован за самим сервером
	usedSet[srv.Subnet.Addr().Next().String()] = struct{}{}

	addr := srv.Subnet.Addr().Next().Next() // начинаем с .2
	for srv.Subnet.Contains(addr) {
		if _, taken := usedSet[addr.String()]; !taken {
			return addr, nil
		}
		addr = addr.Next()
	}
	return netip.Addr{}, domain.ErrPoolExhausted
}
