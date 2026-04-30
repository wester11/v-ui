package transport

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/voidwg/agent/internal/awg"
	"github.com/voidwg/agent/internal/wg"
)

// AmneziaWGTransport — WireGuard + AmneziaWG-style UDP-обфускация.
//
// Композиция:
//   * wg.Manager  — управляет ядерным wg-интерфейсом (peers).
//   * awg.Proxy   — UDP-relay client <-AWG-> agent <-plain-> wg0
type AmneziaWGTransport struct {
	mgr   *wg.Manager
	proxy *awg.Proxy
	stop  chan struct{}
	log   *zerolog.Logger
}

func NewAmneziaWG(mgr *wg.Manager, listenAddr, wgAddr string, p awg.Params, log *zerolog.Logger) (*AmneziaWGTransport, error) {
	px, err := awg.NewProxy(listenAddr, wgAddr, p, log)
	if err != nil {
		return nil, err
	}
	return &AmneziaWGTransport{mgr: mgr, proxy: px, stop: make(chan struct{}), log: log}, nil
}

func (t *AmneziaWGTransport) Name() string { return "amneziawg" }

func (t *AmneziaWGTransport) Start() error {
	go func() {
		if err := t.proxy.Run(); err != nil {
			t.log.Error().Err(err).Msg("awg proxy stopped")
		}
	}()
	return nil
}

func (t *AmneziaWGTransport) Stop() error {
	close(t.stop)
	return nil
}

func (t *AmneziaWGTransport) Reload(_ any) error { return nil }

func (t *AmneziaWGTransport) AddPeer(p PeerSpec) error {
	return t.mgr.AddPeer(context.Background(), p.PublicKey, "", p.AllowedIP)
}

func (t *AmneziaWGTransport) RemovePeer(id string) error { return nil }
