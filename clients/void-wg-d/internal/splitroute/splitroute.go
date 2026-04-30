// Package splitroute implements split tunneling for void-wg-d.
//
// Split tunneling works by injecting explicit routes for bypass
// destinations via the original (pre-VPN) default gateway, so that
// traffic to those destinations bypasses the WireGuard tunnel even when
// AllowedIPs = 0.0.0.0/0.
//
// Steps:
//  1. Capture the default gateway + outbound interface before wg-quick up.
//  2. After wg-quick up, add "ip route add <cidr> via <gw> dev <dev>".
//  3. On tunnel down, remove those routes.
//
// Both IP/CIDR and domain names are accepted; domains are resolved to
// their A records at connect time.
package splitroute

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// Gateway holds the original default route before the VPN comes up.
type Gateway struct {
	IP  string // e.g. "192.168.1.1"
	Dev string // e.g. "eth0"
}

// GetDefaultGateway returns the system's current default IPv4 gateway.
// It parses "ip route show default" output.
func GetDefaultGateway() (*Gateway, error) {
	out, err := exec.Command("ip", "route", "show", "default").Output()
	if err != nil {
		return nil, fmt.Errorf("ip route show default: %w", err)
	}
	// Example line: "default via 192.168.1.1 dev eth0 proto dhcp src ..."
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		gw := &Gateway{}
		for i, f := range fields {
			switch f {
			case "via":
				if i+1 < len(fields) {
					gw.IP = fields[i+1]
				}
			case "dev":
				if i+1 < len(fields) {
					gw.Dev = fields[i+1]
				}
			}
		}
		if gw.IP != "" {
			return gw, nil
		}
	}
	return nil, fmt.Errorf("no default gateway found")
}

// ParseBypass converts a mixed list of IPs, CIDRs, and domain names into
// a deduplicated list of CIDR strings ready for routing.
// Domains are resolved to A records; each address becomes a /32.
func ParseBypass(items []string) ([]string, error) {
	seen := make(map[string]struct{})
	var out []string

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		// Already a CIDR?
		if _, ipnet, err := net.ParseCIDR(item); err == nil {
			cidr := ipnet.String()
			if _, dup := seen[cidr]; !dup {
				seen[cidr] = struct{}{}
				out = append(out, cidr)
			}
			continue
		}

		// Plain IP?
		if ip := net.ParseIP(item); ip != nil {
			var cidr string
			if ip.To4() != nil {
				cidr = ip.String() + "/32"
			} else {
				cidr = ip.String() + "/128"
			}
			if _, dup := seen[cidr]; !dup {
				seen[cidr] = struct{}{}
				out = append(out, cidr)
			}
			continue
		}

		// Treat as domain — resolve A records
		addrs, err := net.LookupHost(item)
		if err != nil {
			return nil, fmt.Errorf("resolve %q: %w", item, err)
		}
		if len(addrs) == 0 {
			return nil, fmt.Errorf("no addresses for domain %q", item)
		}
		for _, addr := range addrs {
			ip := net.ParseIP(addr)
			if ip == nil {
				continue
			}
			var cidr string
			if ip.To4() != nil {
				cidr = ip.String() + "/32"
			} else {
				cidr = ip.String() + "/128"
			}
			if _, dup := seen[cidr]; !dup {
				seen[cidr] = struct{}{}
				out = append(out, cidr)
			}
		}
	}
	return out, nil
}

// AddBypassRoutes installs routes for each CIDR via the pre-VPN gateway.
// It returns the list of CIDRs that were successfully added (for cleanup).
func AddBypassRoutes(cidrs []string, gw *Gateway) ([]string, error) {
	var added []string
	for _, cidr := range cidrs {
		args := []string{"route", "add", cidr, "via", gw.IP, "dev", gw.Dev}
		out, err := exec.Command("ip", args...).CombinedOutput()
		if err != nil {
			// "File exists" means the route is already there — not an error
			msg := string(out)
			if strings.Contains(msg, "File exists") || strings.Contains(msg, "RTNETLINK answers: File exists") {
				added = append(added, cidr)
				continue
			}
			return added, fmt.Errorf("ip route add %s via %s dev %s: %w\n%s",
				cidr, gw.IP, gw.Dev, err, msg)
		}
		added = append(added, cidr)
	}
	return added, nil
}

// RemoveBypassRoutes removes the routes added by AddBypassRoutes.
// Errors are collected and returned as a single string but do not abort.
func RemoveBypassRoutes(cidrs []string) error {
	var errs []string
	for _, cidr := range cidrs {
		out, err := exec.Command("ip", "route", "del", cidr).CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			// "No such process" / "No such file" — route already gone, ignore
			if strings.Contains(msg, "No such") {
				continue
			}
			errs = append(errs, fmt.Sprintf("del %s: %v (%s)", cidr, err, msg))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("remove bypass routes: %s", strings.Join(errs, "; "))
	}
	return nil
}

// Print returns a human-readable summary for status output.
func Print(cidrs []string, gw *Gateway) string {
	if len(cidrs) == 0 {
		return "none"
	}
	return fmt.Sprintf("%s (via %s dev %s)", strings.Join(cidrs, ", "), gw.IP, gw.Dev)
}
