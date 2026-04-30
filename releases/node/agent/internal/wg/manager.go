// Package wg — обёртка над утилитой `wg` (через exec).
// В проде имеет смысл заменить на golang.zx2c4.com/wireguard/wgctrl,
// но прямой вызов wg-команд проще и не требует root-кода в Go-процессе.
package wg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type Manager struct {
	iface string
	mu    sync.Mutex
}

func New(iface string) *Manager { return &Manager{iface: iface} }

// AddPeer — `wg set <iface> peer <pub> preshared-key <psk-file> allowed-ips <ip>`
func (m *Manager) AddPeer(ctx context.Context, publicKey, presharedKey, allowedIP string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := []string{"set", m.iface, "peer", publicKey, "allowed-ips", allowedIP}
	if presharedKey != "" {
		// preshared передаём через stdin
		cmd := exec.CommandContext(ctx, "wg", "set", m.iface, "peer", publicKey,
			"preshared-key", "/dev/stdin", "allowed-ips", allowedIP)
		cmd.Stdin = strings.NewReader(presharedKey + "\n")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("wg set: %w: %s", err, out)
		}
		return nil
	}
	out, err := exec.CommandContext(ctx, "wg", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg set: %w: %s", err, out)
	}
	return nil
}

func (m *Manager) RemovePeer(ctx context.Context, publicKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	out, err := exec.CommandContext(ctx, "wg", "set", m.iface, "peer", publicKey, "remove").CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg remove: %w: %s", err, out)
	}
	return nil
}

func (m *Manager) Status(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "wg", "show", m.iface).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
