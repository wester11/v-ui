package transport

import (
	"context"

	"github.com/voidwg/agent/internal/wg"
)

// WireGuardTransport — обёртка над уже существующим wg.Manager (kernel WG).
type WireGuardTransport struct {
	mgr *wg.Manager
}

func NewWireGuard(mgr *wg.Manager) *WireGuardTransport { return &WireGuardTransport{mgr: mgr} }

func (t *WireGuardTransport) Name() string         { return "wireguard" }
func (t *WireGuardTransport) Start() error         { return nil } // wg-link создаётся снаружи (systemd-networkd / docker-init)
func (t *WireGuardTransport) Stop() error          { return nil }
func (t *WireGuardTransport) Reload(_ any) error   { return nil }

func (t *WireGuardTransport) AddPeer(p PeerSpec) error {
	return t.mgr.AddPeer(context.Background(), p.PublicKey, "", p.AllowedIP)
}

func (t *WireGuardTransport) RemovePeer(id string) error {
	// public-key хранит control-plane; здесь по id-стабу ничего не удаляем.
	// При полной интеграции надо будет резолвить id->public_key через локальный кеш.
	return nil
}
