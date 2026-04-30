// Package stats parses `wg show <iface> dump` output for peer statistics.
package stats

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// PeerStats holds real-time statistics for a single WireGuard peer.
type PeerStats struct {
	PublicKey       string
	PresharedKey    string
	Endpoint        string
	AllowedIPs      []string
	LastHandshake   time.Time
	RxBytes         int64
	TxBytes         int64
	PersistentKA    int
}

// InterfaceStats holds the interface key + all peer stats.
type InterfaceStats struct {
	Interface  string
	PublicKey  string
	PrivateKey string
	ListenPort int
	Peers      []PeerStats
}

// Dump runs `wg show <iface> dump` and parses the output.
// Requires the `wg` binary and appropriate permissions.
func Dump(iface string) (*InterfaceStats, error) {
	out, err := exec.Command("wg", "show", iface, "dump").Output()
	if err != nil {
		return nil, fmt.Errorf("wg show %s dump: %w", iface, err)
	}

	stats := &InterfaceStats{Interface: iface}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	for i, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if i == 0 {
			// first line: private-key public-key listen-port fwmark
			if len(fields) >= 3 {
				stats.PrivateKey = fields[0]
				stats.PublicKey  = fields[1]
				stats.ListenPort, _ = strconv.Atoi(fields[2])
			}
			continue
		}
		// peer lines: public-key preshared-key endpoint allowed-ips
		//             latest-handshake transfer-rx transfer-tx persistent-keepalive
		if len(fields) < 8 {
			continue
		}
		ps := PeerStats{
			PublicKey:    fields[0],
			PresharedKey: fields[1],
			Endpoint:     fields[2],
			AllowedIPs:   splitTrim(fields[3]),
		}
		if ts, err := strconv.ParseInt(fields[4], 10, 64); err == nil && ts > 0 {
			ps.LastHandshake = time.Unix(ts, 0)
		}
		ps.RxBytes, _ = strconv.ParseInt(fields[5], 10, 64)
		ps.TxBytes, _ = strconv.ParseInt(fields[6], 10, 64)
		if fields[7] != "off" {
			ps.PersistentKA, _ = strconv.Atoi(fields[7])
		}
		stats.Peers = append(stats.Peers, ps)
	}
	return stats, nil
}

// FormatBytes returns a human-readable byte count (B / KiB / MiB / GiB).
func FormatBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.2f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.2f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.2f KiB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func splitTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
