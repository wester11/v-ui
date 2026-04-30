// Package sysstat collects system and WireGuard peer statistics.
package sysstat

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SystemStats holds CPU, memory and uptime info.
type SystemStats struct {
	UptimeSeconds int64
	LoadAvg1      float64
	MemTotalKB    int64
	MemAvailKB    int64
	MemUsedPct    float64
}

// PeerStats holds per-peer WireGuard statistics.
type PeerStats struct {
	PublicKey     string
	Endpoint      string
	AllowedIPs    []string
	LastHandshake time.Time
	RxBytes       int64
	TxBytes       int64
	Online        bool // handshake within last 3 minutes
}

// WGStats holds interface + peer stats from `wg show dump`.
type WGStats struct {
	Interface  string
	PublicKey  string
	ListenPort int
	Peers      []PeerStats
}

// CollectSystem reads /proc/uptime, /proc/loadavg, /proc/meminfo.
func CollectSystem() (*SystemStats, error) {
	s := &SystemStats{}

	// /proc/uptime
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			if f, err := strconv.ParseFloat(fields[0], 64); err == nil {
				s.UptimeSeconds = int64(f)
			}
		}
	}

	// /proc/loadavg
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			s.LoadAvg1, _ = strconv.ParseFloat(fields[0], 64)
		}
	}

	// /proc/meminfo
	if f, err := os.Open("/proc/meminfo"); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			key, val, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			val = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(val), "kB"))
			n, _ := strconv.ParseInt(val, 10, 64)
			switch key {
			case "MemTotal":
				s.MemTotalKB = n
			case "MemAvailable":
				s.MemAvailKB = n
			}
		}
	}
	if s.MemTotalKB > 0 {
		s.MemUsedPct = float64(s.MemTotalKB-s.MemAvailKB) / float64(s.MemTotalKB) * 100
	}
	return s, nil
}

// CollectWG runs `wg show <iface> dump` and parses peer statistics.
func CollectWG(iface string) (*WGStats, error) {
	out, err := exec.Command("wg", "show", iface, "dump").Output()
	if err != nil {
		return nil, fmt.Errorf("wg show %s dump: %w", iface, err)
	}

	stats := &WGStats{Interface: iface}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	now := time.Now()

	for i, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if i == 0 {
			if len(fields) >= 3 {
				stats.PublicKey = fields[1]
				stats.ListenPort, _ = strconv.Atoi(fields[2])
			}
			continue
		}
		if len(fields) < 8 {
			continue
		}
		ps := PeerStats{
			PublicKey:  fields[0],
			Endpoint:   fields[2],
			AllowedIPs: splitTrim(fields[3]),
		}
		if ts, err := strconv.ParseInt(fields[4], 10, 64); err == nil && ts > 0 {
			ps.LastHandshake = time.Unix(ts, 0)
			ps.Online = now.Sub(ps.LastHandshake) < 3*time.Minute
		}
		ps.RxBytes, _ = strconv.ParseInt(fields[5], 10, 64)
		ps.TxBytes, _ = strconv.ParseInt(fields[6], 10, 64)
		stats.Peers = append(stats.Peers, ps)
	}
	return stats, nil
}

// FormatBytes returns a human-readable byte count.
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
