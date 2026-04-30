// Package config parses WireGuard wg-quick compatible .conf files.
// Supports all standard fields: Interface (PrivateKey, Address, DNS, MTU,
// PreUp/PostUp/PreDown/PostDown, Table, ListenPort) and Peer (PublicKey,
// PresharedKey, Endpoint, AllowedIPs, PersistentKeepalive).
package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds a parsed wg-quick configuration.
type Config struct {
	Name      string // derived from filename (without .conf)
	Interface Interface
	Peers     []Peer
}

type Interface struct {
	PrivateKey string
	Address    []string // CIDR list e.g. ["10.0.0.2/32","fd00::2/128"]
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
	Comment             string // optional inline comment
}

// ParseFile reads and parses a .conf file.
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
	name = strings.TrimSuffix(name, ".conf")

	cfg := &Config{Name: name}
	var section string
	var curPeer *Peer

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// skip blanks and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			// flush peer
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
		// strip inline comments
		if idx := strings.Index(val, " #"); idx >= 0 {
			val = strings.TrimSpace(val[:idx])
		}

		switch section {
		case "interface":
			parseInterface(&cfg.Interface, key, val)
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

func parseInterface(iface *Interface, key, val string) {
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

// Marshal serialises a Config back into wg-quick .conf format.
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
	if iface.Table != "" {
		b.WriteString("Table = " + iface.Table + "\n")
	}
	for _, s := range iface.PreUp {
		b.WriteString("PreUp = " + s + "\n")
	}
	for _, s := range iface.PostUp {
		b.WriteString("PostUp = " + s + "\n")
	}
	for _, s := range iface.PreDown {
		b.WriteString("PreDown = " + s + "\n")
	}
	for _, s := range iface.PostDown {
		b.WriteString("PostDown = " + s + "\n")
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
