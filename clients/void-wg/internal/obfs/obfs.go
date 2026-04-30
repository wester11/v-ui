// Package obfs implements the AmneziaWG-compatible obfuscation layer.
//
// How it works:
//   1. Before a real WireGuard handshake, Jc junk UDP packets of random
//      size [Jmin, Jmax] bytes are sent to confuse DPI systems.
//   2. The 4-byte magic fields in the WireGuard handshake headers
//      (message type) are XOR'd with H1/H2/H3/H4 to make them
//      unrecognisable to signature-based DPI.
//   3. S1/S2 bytes of random padding are prepended to the init/response
//      handshake messages.
//
// This is a UDP proxy that sits between the wireguard-go kernel module
// and the remote endpoint. The Proxy listens on a local port and forwards
// to the real server with obfuscation applied.
package obfs

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

// Params are the AmneziaWG obfuscation parameters.
type Params struct {
	Jc   uint8  // junk packet count
	Jmin uint16 // junk min size
	Jmax uint16 // junk max size
	S1   uint16 // init padding bytes
	S2   uint16 // response padding bytes
	H1   uint32 // init magic xor
	H2   uint32 // response magic xor
	H3   uint32 // cookie magic xor
	H4   uint32 // transport magic xor
}

// WireGuard message types (first 4 bytes of each packet).
const (
	msgInitiation = 1
	msgResponse   = 2
	msgCookieReply = 3
	msgTransport  = 4
)

// Proxy is a UDP obfuscation proxy.
// Listen on LocalAddr, forward to RemoteAddr with AWG obfuscation.
type Proxy struct {
	LocalAddr  string
	RemoteAddr string
	Params     Params

	mu      sync.Mutex
	conn    *net.UDPConn
	remote  *net.UDPAddr
	peers   map[string]*net.UDPAddr // local client addr -> remote mapping
	running bool
}

// Start begins proxying UDP packets.
func (p *Proxy) Start() error {
	laddr, err := net.ResolveUDPAddr("udp", p.LocalAddr)
	if err != nil {
		return fmt.Errorf("resolve local: %w", err)
	}
	raddr, err := net.ResolveUDPAddr("udp", p.RemoteAddr)
	if err != nil {
		return fmt.Errorf("resolve remote: %w", err)
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return fmt.Errorf("listen udp %s: %w", p.LocalAddr, err)
	}

	p.mu.Lock()
	p.conn = conn
	p.remote = raddr
	p.peers = make(map[string]*net.UDPAddr)
	p.running = true
	p.mu.Unlock()

	go p.loop()
	return nil
}

// Stop shuts down the proxy.
func (p *Proxy) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn != nil {
		_ = p.conn.Close()
		p.running = false
	}
}

func (p *Proxy) loop() {
	buf := make([]byte, 65535)
	for {
		n, addr, err := p.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		pkt := make([]byte, n)
		copy(pkt, buf[:n])
		go p.handlePacket(pkt, addr)
	}
}

func (p *Proxy) handlePacket(pkt []byte, from *net.UDPAddr) {
	if len(pkt) < 4 {
		return
	}
	msgType := binary.LittleEndian.Uint32(pkt[:4])

	// Outbound (client → server): obfuscate
	outbound := p.isLocalClient(from)
	if outbound {
		p.sendObfuscated(pkt, msgType)
	} else {
		// Inbound (server → client): deobfuscate
		p.sendDeobfuscated(pkt, msgType, from)
	}
}

// sendObfuscated applies obfuscation and sends to remote.
func (p *Proxy) sendObfuscated(pkt []byte, msgType uint32) {
	out := make([]byte, len(pkt))
	copy(out, pkt)

	// XOR message type with Hx magic
	xorMagic := p.magicFor(msgType)
	binary.LittleEndian.PutUint32(out[:4], msgType^xorMagic)

	// Add padding (S1 for init, S2 for response)
	var padding uint16
	switch msgType {
	case msgInitiation:
		padding = p.Params.S1
	case msgResponse:
		padding = p.Params.S2
	}
	if padding > 0 {
		pad := make([]byte, padding)
		_, _ = rand.Read(pad)
		out = append(pad, out...)
	}

	// Send junk packets first for handshake messages
	if msgType == msgInitiation {
		p.sendJunk()
	}

	conn := p.conn
	if conn != nil {
		_, _ = conn.WriteToUDP(out, p.remote)
	}
}

// sendDeobfuscated strips obfuscation and delivers to local WG.
func (p *Proxy) sendDeobfuscated(pkt []byte, _ uint32, _ *net.UDPAddr) {
	if len(pkt) < 4 {
		return
	}
	out := make([]byte, len(pkt))
	copy(out, pkt)

	// Read XOR'd type
	xorType := binary.LittleEndian.Uint32(out[:4])
	// Try each magic to find original type
	for _, orig := range []uint32{msgInitiation, msgResponse, msgCookieReply, msgTransport} {
		xor := p.magicFor(orig)
		if xorType == orig^xor {
			binary.LittleEndian.PutUint32(out[:4], orig)
			// Strip padding
			var padding uint16
			switch orig {
			case msgInitiation:
				padding = p.Params.S1
			case msgResponse:
				padding = p.Params.S2
			}
			if int(padding) < len(out) {
				out = out[padding:]
			}
			break
		}
	}

	// Deliver to the local WG endpoint (loopback)
	// In a real deployment the wg iface listens on WGAddr
	// For now, we write back to the local wg listener
	wgAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:51820")
	if p.conn != nil {
		_, _ = p.conn.WriteToUDP(out, wgAddr)
	}
}

func (p *Proxy) sendJunk() {
	if p.Params.Jc == 0 {
		return
	}
	jmin := int(p.Params.Jmin)
	jmax := int(p.Params.Jmax)
	if jmin <= 0 {
		jmin = 40
	}
	if jmax < jmin {
		jmax = jmin + 30
	}

	for i := 0; i < int(p.Params.Jc); i++ {
		size := jmin + randInt(jmax-jmin+1)
		junk := make([]byte, size)
		_, _ = rand.Read(junk)
		if p.conn != nil {
			_, _ = p.conn.WriteToUDP(junk, p.remote)
		}
		// tiny delay so junk packets don't arrive in a burst
		time.Sleep(time.Millisecond)
	}
}

func (p *Proxy) magicFor(msgType uint32) uint32 {
	switch msgType {
	case msgInitiation:
		return p.Params.H1
	case msgResponse:
		return p.Params.H2
	case msgCookieReply:
		return p.Params.H3
	case msgTransport:
		return p.Params.H4
	default:
		return 0
	}
}

func (p *Proxy) isLocalClient(addr *net.UDPAddr) bool {
	return addr.IP.IsLoopback()
}

// randInt returns a random int in [0, n).
func randInt(n int) int {
	if n <= 1 {
		return 0
	}
	var b [4]byte
	_, _ = rand.Read(b[:])
	return int(binary.LittleEndian.Uint32(b[:])) % n
}
