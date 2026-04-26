// Package obfs — UDP proxy с обфускацией для агента.
// Слушает obfs-порт, разворачивает кадры → форвардит в локальный wg.
// Идентичная логика реализована на стороне backend, чтобы клиентская
// библиотека могла её переиспользовать.
package obfs

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"net"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/hkdf"
)

const (
	HeaderLen = 2
	MaxPad    = 96
)

type Obfuscator struct{ key []byte }

func NewObfuscator(psk []byte) (*Obfuscator, error) {
	if len(psk) == 0 {
		return nil, errors.New("empty psk")
	}
	r := hkdf.New(sha256.New, psk, nil, []byte("voidwg-obfs/v1"))
	key := make([]byte, 32)
	if _, err := r.Read(key); err != nil {
		return nil, err
	}
	return &Obfuscator{key: key}, nil
}

func (o *Obfuscator) Wrap(payload []byte) []byte {
	pad := mrand.Intn(MaxPad)
	out := make([]byte, HeaderLen+len(payload)+pad)
	binary.BigEndian.PutUint16(out[:HeaderLen], uint16(len(payload)))
	copy(out[HeaderLen:], payload)
	if pad > 0 {
		_, _ = rand.Read(out[HeaderLen+len(payload):])
	}
	o.xor(out[HeaderLen:], out[:HeaderLen])
	return out
}

func (o *Obfuscator) Unwrap(frame []byte) ([]byte, bool, error) {
	if len(frame) < HeaderLen {
		return nil, false, errors.New("frame too short")
	}
	nonce := append([]byte(nil), frame[:HeaderLen]...)
	body := append([]byte(nil), frame[HeaderLen:]...)
	o.xor(body, nonce)
	plen := int(binary.BigEndian.Uint16(nonce))
	if plen == 0 {
		return nil, true, nil
	}
	if plen > len(body) {
		return nil, false, errors.New("len mismatch")
	}
	return body[:plen], false, nil
}

func (o *Obfuscator) xor(buf, nonce []byte) {
	h := hmac.New(sha256.New, o.key)
	var ctr uint32
	var block []byte
	for i := range buf {
		if i%32 == 0 {
			h.Reset()
			h.Write(nonce)
			var c [4]byte
			binary.BigEndian.PutUint32(c[:], ctr)
			h.Write(c[:])
			block = h.Sum(nil)
			ctr++
		}
		buf[i] ^= block[i%32]
	}
}

// Proxy — двусторонний UDP-relay: client <obfs> agent <plain> wg0.
type Proxy struct {
	o          *Obfuscator
	listenAddr *net.UDPAddr
	wgAddr     *net.UDPAddr
	log        *zerolog.Logger

	mu       sync.Mutex
	sessions map[string]*session
}

type session struct {
	wgConn   *net.UDPConn
	clientAddr *net.UDPAddr
	lastSeen time.Time
}

func NewProxy(listenAddr, wgAddr string, psk []byte, log *zerolog.Logger) (*Proxy, error) {
	la, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return nil, err
	}
	wa, err := net.ResolveUDPAddr("udp", wgAddr)
	if err != nil {
		return nil, err
	}
	o, err := NewObfuscator(psk)
	if err != nil {
		return nil, err
	}
	return &Proxy{o: o, listenAddr: la, wgAddr: wa, log: log, sessions: map[string]*session{}}, nil
}

// Run — запускает обработчик. Блокирующий.
func (p *Proxy) Run() error {
	conn, err := net.ListenUDP("udp", p.listenAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	go p.gcLoop()

	buf := make([]byte, 65535)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		payload, decoy, err := p.o.Unwrap(buf[:n])
		if err != nil {
			p.log.Debug().Err(err).Msg("unwrap failed")
			continue
		}
		if decoy {
			continue
		}
		s := p.sessionFor(addr, conn)
		_, _ = s.wgConn.Write(payload)
		s.lastSeen = time.Now()
	}
}

func (p *Proxy) sessionFor(addr *net.UDPAddr, listenConn *net.UDPConn) *session {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := addr.String()
	if s, ok := p.sessions[key]; ok {
		return s
	}
	wgConn, err := net.DialUDP("udp", nil, p.wgAddr)
	if err != nil {
		p.log.Error().Err(err).Msg("dial wg")
		return &session{}
	}
	s := &session{wgConn: wgConn, clientAddr: addr, lastSeen: time.Now()}
	p.sessions[key] = s

	// reverse direction: wg → client (с обфускацией)
	go func() {
		buf := make([]byte, 65535)
		for {
			n, _, err := wgConn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			frame := p.o.Wrap(buf[:n])
			_, _ = listenConn.WriteToUDP(frame, addr)
		}
	}()
	return s
}

func (p *Proxy) gcLoop() {
	t := time.NewTicker(2 * time.Minute)
	for range t.C {
		p.mu.Lock()
		for k, s := range p.sessions {
			if time.Since(s.lastSeen) > 5*time.Minute {
				_ = s.wgConn.Close()
				delete(p.sessions, k)
			}
		}
		p.mu.Unlock()
	}
}
