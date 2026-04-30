package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
)

const DefaultInviteTTL = 24 * time.Hour

type InviteService struct {
	invites port.InviteRepository
	peers   *PeerService
	servers port.ServerRepository
}

func NewInviteService(i port.InviteRepository, p *PeerService, s port.ServerRepository) *InviteService {
	return &InviteService{invites: i, peers: p, servers: s}
}

type CreateInviteInput struct {
	UserID        uuid.UUID
	ServerID      uuid.UUID
	SuggestedName string
	TTL           time.Duration
}

// Create — генерирует одноразовый invite, привязанный к user'у и серверу.
func (s *InviteService) Create(ctx context.Context, in CreateInviteInput) (*domain.Invite, error) {
	if _, err := s.servers.GetByID(ctx, in.ServerID); err != nil {
		return nil, err
	}
	ttl := in.TTL
	if ttl <= 0 || ttl > 30*24*time.Hour {
		ttl = DefaultInviteTTL
	}
	tok, err := randomToken(24)
	if err != nil {
		return nil, err
	}
	inv := &domain.Invite{
		ID:            uuid.New(),
		Token:         tok,
		ServerID:      in.ServerID,
		UserID:        in.UserID,
		SuggestedName: strings.TrimSpace(in.SuggestedName),
		ExpiresAt:     time.Now().UTC().Add(ttl),
		CreatedAt:     time.Now().UTC(),
	}
	if err := s.invites.Create(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

// Lookup — публичный (без auth) доступ к invite'у по токену:
// возвращает только метаданные сервера для генерации ключей на клиенте.
type InviteView struct {
	Server        *domain.Server
	SuggestedName string
	ExpiresAt     time.Time
}

func (s *InviteService) Lookup(ctx context.Context, token string) (*InviteView, error) {
	inv, err := s.invites.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if inv.Used() {
		return nil, domain.ErrAlreadyExists
	}
	if inv.Expired() {
		return nil, domain.ErrForbidden
	}
	srv, err := s.servers.GetByID(ctx, inv.ServerID)
	if err != nil {
		return nil, err
	}
	return &InviteView{Server: srv, SuggestedName: inv.SuggestedName, ExpiresAt: inv.ExpiresAt}, nil
}

type RedeemInput struct {
	Token     string
	PublicKey string
	Name      string
}

// Redeem — выдаёт peer-конфиг под client-generated ключ.
func (s *InviteService) Redeem(ctx context.Context, in RedeemInput) (*domain.Peer, string, error) {
	inv, err := s.invites.GetByToken(ctx, in.Token)
	if err != nil {
		return nil, "", err
	}
	if inv.Used() {
		return nil, "", domain.ErrAlreadyExists
	}
	if inv.Expired() {
		return nil, "", domain.ErrForbidden
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = inv.SuggestedName
	}
	if name == "" {
		name = "client-" + inv.Token[:6]
	}
	peer, cfg, err := s.peers.Create(ctx, CreatePeerInput{
		UserID:    inv.UserID,
		ServerID:  inv.ServerID,
		Name:      name,
		PublicKey: in.PublicKey,
	})
	if err != nil {
		return nil, "", err
	}
	if err := s.invites.MarkUsed(ctx, inv.ID, peer.ID); err != nil {
		// peer создан — мягко логируем, но конфиг отдаём
		return peer, cfg, nil
	}
	return peer, cfg, nil
}

func (s *InviteService) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Invite, error) {
	return s.invites.ListByUser(ctx, userID)
}

func (s *InviteService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.invites.Delete(ctx, id)
}

func randomToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
