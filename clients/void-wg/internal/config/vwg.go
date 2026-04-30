// Package config handles .vwg config files — a superset of wg-quick format.
//
// Standard wg-quick fields work unchanged.  Additional [Interface] fields:
//
//   Obfuscation = on|off   enable AWG-style obfuscation (default off)
//   Jc    = 4              junk packet count   (AmneziaWG Jc)
//   Jmin  = 40             junk min size bytes (AmneziaWG Jmin)
//   Jmax  = 70             junk max size bytes (AmneziaWG Jmax)
//   S1    = 0              init handshake padding  bytes
//   S2    = 0              resp handshake padding  bytes
//   H1    = 0              handshake init   magic  (uint32)
//   H2    = 0              handshake resp   magic  (uint32)
//   H3    = 0              handshake cookie magic  (uint32)
//   H4    = 0              transport data   magic  (uint32)
package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ObfsParams are the AmneziaWG obfuscation parameters.
type ObfsParams struct {
	Enabled bool
	Jc      uint8
	Jmin    uint16
	Jmax    uint16
	S1      uint16
	S2      uint16
	H1      uint32
	H2      uint32
	H3      uint32
	H4      uint32
}

// DefaultObfsParams returns safe default obfuscation parameters.
func DefaultObfsParams() ObfsParams {
	return ObfsParams{
		Enabled: true,
		Jc:      4,
		Jmin:    40,
		Jmax:    70,
		S1:      0,
		S2:      0,
		H1:      1,
		H2:      2,
		H3:      3,
		H4:      4,
	}
}

// Config holds a parsed .vwg configuration.
type Config struct {
	Name      string
	Interface Interface
	Obfs      ObfsParams
	Peers     []Peer
}

type Interface struct {
	PrivateKey string
	Address    []string
	DNS        []string
	MTU        int
	ListenPort int
	Table      string
	PreUp      []string
	PostUp     []string
	PreDown    []string
	PostDown   []string
}

type Peer struct {
	PublicKey           string
	PresharedKey        string
	Endpoint            string
	AllowedIPs          []string
	PersistentKeepalive int
}

// ParseFile reads a .vwg or standard .conf file.
func ParseFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	name := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		name = path[idx+1:]
	}
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".vwg"), ".conf")

	cfg := &Config{Name: name}
	var section string
	var curPeer *Peer

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			if curPeer != nil {
				cfg.Peers = append(cfg.Peers, *curPeer)
				curPeer = nil
			}
			section = strings.ToLower(strings.Trim(line, "[]"))
			if section == "peer" {
				curPeer = &Peer{}
			}
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if idx := strings.Index(val, " #"); idx >= 0 {
			val = strings.TrimSpace(val[:idx])
		}
		switch section {
		case "interface":
			parseInterface(&cfg.Interface, &cfg.Obfs, key, val)
		case "peer":
			if curPeer != nil {
				parsePeer(curPeer, key, val)
			}
		}
	}
	if curPeer != nil {
		cfg.Peers = append(cfg.Peers, *curPeer)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if cfg.Interface.PrivateKey == "" {
		return nil, fmt.Errorf("missing PrivateKey in [Interface]")
	}
	return cfg, nil
}

func parseInterface(iface *Interface, obfs *ObfsParams, key, val string) {
	switch strings.ToLower(key) {
	case "privatekey":
		iface.PrivateKey = val
	case "address":
		iface.Address = splitTrim(val)
	case "dns":
		iface.DNS = splitTrim(val)
	case "mtu":
		iface.MTU, _ = strconv.Atoi(val)
	case "listenport":
		iface.ListenPort, _ = strconv.Atoi(val)
	case "table":
		iface.Table = val
	case "preup":
		iface.PreUp = append(iface.PreUp, val)
	case "postup":
		iface.PostUp = append(iface.PostUp, val)
	case "predown":
		iface.PreDown = append(iface.PreDown, val)
	case "postdown":
		iface.PostDown = append(iface.PostDown, val)
	// Obfuscation params
	case "obfuscation":
		obfs.Enabled = strings.ToLower(val) == "on" || val == "1" || strings.ToLower(val) == "true"
	case "jc":
		v, _ := strconv.ParseUint(val, 10, 8)
		obfs.Jc = uint8(v)
	case "jmin":
		v, _ := strconv.ParseUint(val, 10, 16)
		obfs.Jmin = uint16(v)
	case "jmax":
		v, _ := strconv.ParseUint(val, 10, 16)
		obfs.Jmax = uint16(v)
	case "s1":
		v, _ := strconv.ParseUint(val, 10, 16)
		obfs.S1 = uint16(v)
	case "s2":
		v, _ := strconv.ParseUint(val, 10, 16)
		obfs.S2 = uint16(v)
	case "h1":
		v, _ := strconv.ParseUint(val, 10, 32)
		obfs.H1 = uint32(v)
	case "h2":
		v, _ := strconv.ParseUint(val, 10, 32)
		obfs.H2 = uint32(v)
	case "h3":
		v, _ := strconv.ParseUint(val, 10, 32)
		obfs.H3 = uint32(v)
	case "h4":
		v, _ := strconv.ParseUint(val, 10, 32)
		obfs.H4 = uint32(v)
	}
}

func parsePeer(p *Peer, key, val string) {
	switch strings.ToLower(key) {
	case "publickey":
		p.PublicKey = val
	case "presharedkey":
		p.PresharedKey = val
	case "endpoint":
		p.Endpoint = val
	case "allowedips":
		p.AllowedIPs = splitTrim(val)
	case "persistentkeepalive":
		p.PersistentKeepalive, _ = strconv.Atoi(val)
	}
}

// Marshal serialises to .vwg format (wg-quick compatible + obfs fields).
func (c *Config) Marshal() string {
	var b strings.Builder
	iface := c.Interface
	b.WriteString("[Interface]\n")
	b.WriteString("PrivateKey = " + iface.PrivateKey + "\n")
	if len(iface.Address) > 0 {
		b.WriteString("Address = " + strings.Join(iface.Address, ", ") + "\n")
	}
	if len(iface.DNS) > 0 {
		b.WriteString("DNS = " + strings.Join(iface.DNS, ", ") + "\n")
	}
	if iface.MTU > 0 {
		b.WriteString("MTU = " + strconv.Itoa(iface.MTU) + "\n")
	}
	if iface.ListenPort > 0 {
		b.WriteString("ListenPort = " + strconv.Itoa(iface.ListenPort) + "\n")
	}
	// Obfuscation block
	obfs := c.Obfs
	if obfs.Enabled {
		b.WriteString("\n# VoidWG obfuscation parameters\n")
		b.WriteString("Obfuscation = on\n")
		b.WriteString(fmt.Sprintf("Jc   = %d\n", obfs.Jc))
		b.WriteString(fmt.Sprintf("Jmin = %d\n", obfs.Jmin))
		b.WriteString(fmt.Sprintf("Jmax = %d\n", obfs.Jmax))
		b.WriteString(fmt.Sprintf("S1   = %d\n", obfs.S1))
		b.WriteString(fmt.Sprintf("S2   = %d\n", obfs.S2))
		b.WriteString(fmt.Sprintf("H1   = %d\n", obfs.H1))
		b.WriteString(fmt.Sprintf("H2   = %d\n", obfs.H2))
		b.WriteString(fmt.Sprintf("H3   = %d\n", obfs.H3))
		b.WriteString(fmt.Sprintf("H4   = %d\n", obfs.H4))
	}
	for _, p := range c.Peers {
		b.WriteString("\n[Peer]\n")
		b.WriteString("PublicKey = " + p.PublicKey + "\n")
		if p.PresharedKey != "" {
			b.WriteString("PresharedKey = " + p.PresharedKey + "\n")
		}
		if p.Endpoint != "" {
			b.WriteString("Endpoint = " + p.Endpoint + "\n")
		}
		if len(p.AllowedIPs) > 0 {
			b.WriteString("AllowedIPs = " + strings.Join(p.AllowedIPs, ", ") + "\n")
		}
		if p.PersistentKeepalive > 0 {
			b.WriteString("PersistentKeepalive = " + strconv.Itoa(p.PersistentKeepalive) + "\n")
		}
	}
	return b.String()
}

// UpgradeFromWG imports a standard WG config and adds default obfuscation params.
func UpgradeFromWG(path string) (*Config, error) {
	cfg, err := ParseFile(path)
	if err != nil {
		return nil, err
	}
	cfg.Obfs = DefaultObfsParams()
	return cfg, nil
}

// ToWGConf returns a standard WireGuard .conf (strips obfs fields).
// Use this when passing config to wg-quick.
func (c *Config) ToWGConf() string {
	// Temporarily zero obfs so Marshal skips the block
	saved := c.Obfs
	c.Obfs = ObfsParams{}
	out := c.Marshal()
	c.Obfs = saved
	return out
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
