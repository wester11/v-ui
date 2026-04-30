// Package tunnel manages a void-wg tunnel:
// 1. Writes the wg-quick-compatible portion of the .vwg config
// 2. Brings up the WireGuard interface via wg-quick
// 3. If obfuscation is enabled, starts the UDP obfs proxy
// 4. Optionally enables the kill switch
package tunnel

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/voidwg/void-wg/internal/config"
	"github.com/voidwg/void-wg/internal/obfs"
)

// Tunnel manages a single void-wg interface.
type Tunnel struct {
	mu         sync.Mutex
	cfg        *config.Config
	confDir    string
	proxy      *obfs.Proxy
	running    bool
	KillSwitch bool
	BypassNets []string
}

// New creates a Tunnel.
func New(cfg *config.Config, confDir string) *Tunnel {
	return &Tunnel{cfg: cfg, confDir: confDir}
}

// Up brings the tunnel up.
func (t *Tunnel) Up() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return fmt.Errorf("tunnel %s already running", t.cfg.Name)
	}

	// Write wg-quick compatible config (no obfs fields)
	wgConf := t.cfg.ToWGConf()
	confPath := filepath.Join(t.confDir, t.cfg.Name+".conf")
	tmp, err := os.CreateTemp(t.confDir, ".vwg-*.conf")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	if _, err := tmp.WriteString(wgConf); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	_ = tmp.Close()
	_ = os.Chmod(tmp.Name(), 0600)
	if err := os.Rename(tmp.Name(), confPath); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("install conf: %w", err)
	}

	// wg-quick up
	if out, err := exec.Command("wg-quick", "up", t.cfg.Name).CombinedOutput(); err != nil {
		return fmt.Errorf("wg-quick up: %w\n%s", err, out)
	}

	// Start obfs proxy if enabled
	if t.cfg.Obfs.Enabled && len(t.cfg.Peers) > 0 {
		// The proxy sits between local WG (:51820) and the obfuscated server
		peer := t.cfg.Peers[0]
		if peer.Endpoint != "" {
			proxy := &obfs.Proxy{
				LocalAddr:  "127.0.0.1:51821",
				RemoteAddr: peer.Endpoint,
				Params: obfs.Params{
					Jc:   t.cfg.Obfs.Jc,
					Jmin: t.cfg.Obfs.Jmin,
					Jmax: t.cfg.Obfs.Jmax,
					S1:   t.cfg.Obfs.S1,
					S2:   t.cfg.Obfs.S2,
					H1:   t.cfg.Obfs.H1,
					H2:   t.cfg.Obfs.H2,
					H3:   t.cfg.Obfs.H3,
					H4:   t.cfg.Obfs.H4,
				},
			}
			if err := proxy.Start(); err != nil {
				// non-fatal: tunnel works without proxy (plain WG)
				fmt.Fprintf(os.Stderr, "warn: obfs proxy: %v\n", err)
			} else {
				t.proxy = proxy
			}
		}
	}

	// Kill switch
	if t.KillSwitch {
		if err := enableKillSwitch(t.cfg.Name, t.BypassNets); err != nil {
			fmt.Fprintf(os.Stderr, "warn: kill switch: %v\n", err)
		}
	}

	t.running = true
	return nil
}

// Down brings the tunnel down.
func (t *Tunnel) Down() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return fmt.Errorf("tunnel %s is not running", t.cfg.Name)
	}

	// Kill switch off first
	if t.KillSwitch {
		_ = disableKillSwitch()
	}

	// Stop obfs proxy
	if t.proxy != nil {
		t.proxy.Stop()
		t.proxy = nil
	}

	if out, err := exec.Command("wg-quick", "down", t.cfg.Name).CombinedOutput(); err != nil {
		return fmt.Errorf("wg-quick down: %w\n%s", err, out)
	}

	t.running = false
	return nil
}

// Status returns a human-readable status string.
func (t *Tunnel) Status() (string, error) {
	out, err := exec.Command("wg", "show", t.cfg.Name).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("wg show: %w\n%s", err, out)
	}
	return string(out), nil
}

// enableKillSwitch installs iptables rules.
func enableKillSwitch(iface string, allowedNets []string) error {
	rules := [][]string{
		{"-N", "VOID_KS"},
		{"-F", "VOID_KS"},
		{"-A", "VOID_KS", "-i", "lo", "-j", "ACCEPT"},
		{"-A", "VOID_KS", "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT"},
		{"-A", "VOID_KS", "-o", iface, "-j", "ACCEPT"},
		{"-A", "VOID_KS", "-i", iface, "-j", "ACCEPT"},
	}
	for _, net := range allowedNets {
		if net = strings.TrimSpace(net); net == "" {
			continue
		}
		rules = append(rules,
			[]string{"-A", "VOID_KS", "-d", net, "-j", "ACCEPT"},
			[]string{"-A", "VOID_KS", "-s", net, "-j", "ACCEPT"},
		)
	}
	rules = append(rules, []string{"-A", "VOID_KS", "-j", "DROP"})
	for _, r := range rules {
		_ = exec.Command("iptables", r...).Run()
	}
	for _, chain := range []string{"OUTPUT", "FORWARD"} {
		_ = exec.Command("iptables", "-I", chain, "1", "-j", "VOID_KS").Run()
	}
	return nil
}

func disableKillSwitch() error {
	for _, chain := range []string{"OUTPUT", "FORWARD"} {
		_ = exec.Command("iptables", "-D", chain, "-j", "VOID_KS").Run()
	}
	_ = exec.Command("iptables", "-F", "VOID_KS").Run()
	_ = exec.Command("iptables", "-X", "VOID_KS").Run()
	return nil
}
