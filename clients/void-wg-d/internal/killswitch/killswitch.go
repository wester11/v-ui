// Package killswitch manages iptables/ip6tables kill-switch rules.
// When active, all traffic that does NOT go through the WireGuard interface
// is dropped, preventing leaks if the VPN connection drops.
package killswitch

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	chainName = "VOID_KILLSWITCH"
)

// Enable installs the kill-switch rules for the given WG interface.
// allowedNets are subnets that bypass the kill switch (e.g. LAN ranges).
func Enable(iface string, allowedNets []string) error {
	// Create chain if it doesn't exist
	_ = ipt("-N", chainName)

	// Flush chain
	if err := ipt("-F", chainName); err != nil {
		return fmt.Errorf("flush chain: %w", err)
	}

	// Allow loopback
	if err := ipt("-A", chainName, "-i", "lo", "-j", "ACCEPT"); err != nil {
		return err
	}
	// Allow established/related
	if err := ipt("-A", chainName, "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT"); err != nil {
		return err
	}
	// Allow traffic on WG interface
	if err := ipt("-A", chainName, "-o", iface, "-j", "ACCEPT"); err != nil {
		return err
	}
	if err := ipt("-A", chainName, "-i", iface, "-j", "ACCEPT"); err != nil {
		return err
	}
	// Allow bypass subnets (LAN, etc.)
	for _, net := range allowedNets {
		if net = strings.TrimSpace(net); net == "" {
			continue
		}
		if err := ipt("-A", chainName, "-d", net, "-j", "ACCEPT"); err != nil {
			return err
		}
		if err := ipt("-A", chainName, "-s", net, "-j", "ACCEPT"); err != nil {
			return err
		}
	}
	// Drop everything else
	if err := ipt("-A", chainName, "-j", "DROP"); err != nil {
		return err
	}

	// Hook into OUTPUT and FORWARD
	for _, chain := range []string{"OUTPUT", "FORWARD"} {
		// check if already hooked
		out, _ := exec.Command("iptables", "-C", chain, "-j", chainName).CombinedOutput()
		if len(out) == 0 {
			continue // already present
		}
		if err := ipt("-I", chain, "1", "-j", chainName); err != nil {
			return fmt.Errorf("hook %s: %w", chain, err)
		}
	}
	return nil
}

// Disable removes the kill-switch rules.
func Disable() error {
	for _, chain := range []string{"OUTPUT", "FORWARD"} {
		_ = ipt("-D", chain, "-j", chainName)
	}
	_ = ipt("-F", chainName)
	_ = ipt("-X", chainName)
	return nil
}

// IsEnabled returns true if the kill-switch chain exists.
func IsEnabled() bool {
	err := exec.Command("iptables", "-L", chainName).Run()
	return err == nil
}

func ipt(args ...string) error {
	cmd := exec.Command("iptables", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("iptables %s: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
