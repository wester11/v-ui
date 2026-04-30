// Package tunnel manages a WireGuard tunnel via wg-quick.
// It writes the config atomically and calls wg-quick up/down.
package tunnel

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/voidwg/void-wg-d/internal/config"
	"github.com/voidwg/void-wg-d/internal/killswitch"
	"github.com/voidwg/void-wg-d/internal/splitroute"
	"github.com/voidwg/void-wg-d/internal/stats"
)

const defaultConfDir = "/etc/wireguard"

// Tunnel manages a single WireGuard interface.
type Tunnel struct {
	mu        sync.Mutex
	cfg       *config.Config
	confPath  string
	running   bool
	KillSwitch bool
	BypassNets  []string // subnets that bypass kill switch (e.g. LAN)
	SplitBypass []string // IPs/CIDRs/domains routed via original gateway (split tunneling)

	// internal state for cleanup
	splitGateway *splitroute.Gateway
	splitRoutes  []string // resolved CIDRs that were added to the routing table
}

// New creates a Tunnel from a parsed config.
func New(cfg *config.Config) *Tunnel {
	return &Tunnel{
		cfg:      cfg,
		confPath: filepath.Join(defaultConfDir, cfg.Name+".conf"),
	}
}

// Up brings up the WireGuard tunnel using wg-quick.
func (t *Tunnel) Up() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return fmt.Errorf("tunnel %s already running", t.cfg.Name)
	}

	// Write config atomically
	tmp, err := os.CreateTemp(defaultConfDir, ".vwgd-*.conf")
	if err != nil {
		return fmt.Errorf("create temp conf: %w", err)
	}
	if _, err := tmp.WriteString(t.cfg.Marshal()); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("write conf: %w", err)
	}
	_ = tmp.Close()
	if err := os.Chmod(tmp.Name(), 0600); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), t.confPath); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("install conf: %w", err)
	}

	// Capture default gateway BEFORE wg-quick changes the routing table.
	var gw *splitroute.Gateway
	if len(t.SplitBypass) > 0 {
		var err error
		gw, err = splitroute.GetDefaultGateway()
		if err != nil {
			// non-fatal: warn and skip split routing
			fmt.Fprintf(os.Stderr, "warn: split-bypass: get default gateway: %v\n", err)
		}
	}

	// wg-quick up
	if out, err := exec.Command("wg-quick", "up", t.cfg.Name).CombinedOutput(); err != nil {
		return fmt.Errorf("wg-quick up: %w\n%s", err, out)
	}

	// Inject split-tunnel routes via the original gateway.
	if len(t.SplitBypass) > 0 && gw != nil {
		cidrs, err := splitroute.ParseBypass(t.SplitBypass)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: split-bypass: resolve: %v\n", err)
		} else {
			added, err := splitroute.AddBypassRoutes(cidrs, gw)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warn: split-bypass: add routes: %v\n", err)
			}
			t.splitGateway = gw
			t.splitRoutes = added
		}
	}

	// Enable kill switch if requested
	if t.KillSwitch {
		if err := killswitch.Enable(t.cfg.Name, t.BypassNets); err != nil {
			// non-fatal: tunnel is up, warn about kill switch failure
			fmt.Fprintf(os.Stderr, "warn: kill switch: %v\n", err)
		}
	}

	t.running = true
	return nil
}

// Down brings down the WireGuard tunnel.
func (t *Tunnel) Down() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return fmt.Errorf("tunnel %s is not running", t.cfg.Name)
	}

	// Remove kill switch first
	if t.KillSwitch {
		_ = killswitch.Disable()
	}

	// Remove split-tunnel routes before wg-quick clears the interface.
	if len(t.splitRoutes) > 0 {
		if err := splitroute.RemoveBypassRoutes(t.splitRoutes); err != nil {
			fmt.Fprintf(os.Stderr, "warn: split-bypass: remove routes: %v\n", err)
		}
		t.splitRoutes = nil
		t.splitGateway = nil
	}

	if out, err := exec.Command("wg-quick", "down", t.cfg.Name).CombinedOutput(); err != nil {
		return fmt.Errorf("wg-quick down: %w\n%s", err, out)
	}

	t.running = false
	return nil
}

// IsRunning returns whether the tunnel interface is currently up.
func (t *Tunnel) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

// Stats returns real-time peer statistics.
func (t *Tunnel) Stats() (*stats.InterfaceStats, error) {
	return stats.Dump(t.cfg.Name)
}

// Status returns a human-readable status string.
func (t *Tunnel) Status() (string, error) {
	st, err := t.Stats()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Interface : %s\n", st.Interface)
	fmt.Fprintf(&b, "Public key: %s\n", st.PublicKey)
	if st.ListenPort > 0 {
		fmt.Fprintf(&b, "Listen    : %d\n", st.ListenPort)
	}
	// Show split-tunnel routes if active
	t.mu.Lock()
	splitSummary := splitroute.Print(t.splitRoutes, t.splitGateway)
	t.mu.Unlock()
	fmt.Fprintf(&b, "Split bypass: %s\n", splitSummary)
	for _, p := range st.Peers {
		fmt.Fprintf(&b, "\nPeer      : %s\n", p.PublicKey[:12]+"...")
		if p.Endpoint != "" {
			fmt.Fprintf(&b, "  Endpoint: %s\n", p.Endpoint)
		}
		fmt.Fprintf(&b, "  AllowedIPs: %s\n", strings.Join(p.AllowedIPs, ", "))
		if !p.LastHandshake.IsZero() {
			fmt.Fprintf(&b, "  Last handshake: %s ago\n", formatDuration(p.LastHandshake))
		}
		fmt.Fprintf(&b, "  RX: %s  TX: %s\n", stats.FormatBytes(p.RxBytes), stats.FormatBytes(p.TxBytes))
	}
	return b.String(), nil
}

func formatDuration(t interface{ Unix() int64 }) string {
	return fmt.Sprintf("(handshake at unix ts %d)", t.(interface{ Unix() int64 }).Unix())
}
