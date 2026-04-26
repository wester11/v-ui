// Package awg — AmneziaWG-style обфускация UDP-трафика WireGuard.
//
// Phase 4: ПОЛНОСТЬЮ заменяет старый пакет obfs (XOR/HKDF), который удалён.
//
// Реализует приёмы AmneziaWG (https://github.com/amnezia-vpn/amneziawg-go):
//
//   1) Per-message-type замена первых 4 байт пакета (message_type) на H1..H4:
//        H1 -> handshake_initiation (был 0x01000000)
//        H2 -> handshake_response   (был 0x02000000)
//        H3 -> cookie_reply         (был 0x03000000)
//        H4 -> transport_data       (был 0x04000000)
//      Detect packet type по длине (handshake-кадры WireGuard имеют
//      фиксированный размер: 148/92/64; всё остальное — transport_data).
//
//   2) Перед пакетами initiation / response пристёгивается S1 / S2 случайных байт.
//      Получатель снимает их по первому байту счётчика.
//
//   3) Перед началом сессии клиент (или агент при обратном направлении)
//      шлёт Jc мусорных пакетов длиной [Jmin..Jmax]. Они помечены первым
//      байтом 0xFF и отбрасываются.
//
// Все эти параметры — общие для агента и клиента, выдаются control-plane'ом
// при создании peer'а. Клиент знает их из invite-lookup или из конфига.
//
// Wrap/Unwrap НЕ криптографичны (трафик WG уже зашифрован) — цель:
// разрушить статистические сигнатуры handshake-пакетов для DPI.
package awg

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Params — AmneziaWG-параметры обфускации.
type Params struct {
	Jc   uint8
	Jmin uint16
	Jmax uint16
	S1   uint16
	S2   uint16
	H1   uint32
	H2   uint32
	H3   uint32
	H4   uint32
}

// Маркер мусорного пакета — невозможный H-value (>= 2^31), а потом 0xFFFFFFFF.
const junkMarker = uint32(0xFFFFFFFF)

// WireGuard packet sizes для handshake-фаз.
const (
	wgInitiationLen = 148
	wgResponseLen   = 92
	wgCookieLen     = 64
)

// Wrap — обёртывает исходящий пакет под параметры p.
//
// Возвращает массив кадров (1 нормальный + опционально N junk).
// Для initiation/response пакетов добавляется S1/S2 случайных байт.
func Wrap(p Params, payload []byte) [][]byte {
	out := make([][]byte, 0, 1+int(p.Jc))

	// Mode: первая сессия — пакеты-обманки.
	for i := uint8(0); i < p.Jc; i++ {
		out = append(out, makeJunk(p.Jmin, p.Jmax))
	}

	frame := wrapOne(p, payload)
	out = append(out, frame)
	return out
}

// WrapOne — обёртывает один пакет (без junk preroll).
func WrapOne(p Params, payload []byte) []byte {
	return wrapOne(p, payload)
}

func wrapOne(p Params, payload []byte) []byte {
	if len(payload) < 4 {
		return payload // слишком короткий — не WG, передаём как есть
	}

	mtype := binary.LittleEndian.Uint32(payload[:4])

	// Подмена message_type через H-mapping.
	var newType uint32
	var sPrefix uint16
	switch mtype {
	case 1:
		newType = p.H1
		sPrefix = p.S1
	case 2:
		newType = p.H2
		sPrefix = p.S2
	case 3:
		newType = p.H3
	case 4:
		newType = p.H4
	default:
		newType = mtype // не WG-handshake, не трогаем
	}

	prefixLen := int(sPrefix)
	out := make([]byte, prefixLen+len(payload))
	if prefixLen > 0 {
		_, _ = rand.Read(out[:prefixLen])
	}
	binary.LittleEndian.PutUint32(out[prefixLen:prefixLen+4], newType)
	copy(out[prefixLen+4:], payload[4:])
	return out
}

// Unwrap — снимает обфускацию.
//
// Возвращает (payload, isJunk, err).
// Чтобы понять, какой это был тип пакета, мы используем известную длину:
//   - len == p.S1 + 148 -> initiation, strip S1, restore type=1
//   - len == p.S2 + 92  -> response,   strip S2, restore type=2
//   - len == 64          -> cookie_reply, restore type=3
//   - junk marker first 4 bytes — это junk-пакет.
//   - всё остальное — transport_data, restore type=4
func Unwrap(p Params, frame []byte) ([]byte, bool, error) {
	if len(frame) < 4 {
		return nil, false, errors.New("frame too short")
	}
	first4 := binary.LittleEndian.Uint32(frame[:4])
	if first4 == junkMarker {
		return nil, true, nil
	}

	// Try strip S1+initiation.
	if int(p.S1)+wgInitiationLen == len(frame) {
		body := frame[p.S1:]
		out := make([]byte, len(body))
		binary.LittleEndian.PutUint32(out[:4], 1)
		copy(out[4:], body[4:])
		return out, false, nil
	}
	// Try strip S2+response.
	if int(p.S2)+wgResponseLen == len(frame) {
		body := frame[p.S2:]
		out := make([]byte, len(body))
		binary.LittleEndian.PutUint32(out[:4], 2)
		copy(out[4:], body[4:])
		return out, false, nil
	}
	// cookie_reply — без префикса.
	if len(frame) == wgCookieLen {
		out := make([]byte, len(frame))
		binary.LittleEndian.PutUint32(out[:4], 3)
		copy(out[4:], frame[4:])
		return out, false, nil
	}
	// transport_data — длина переменная, type=4.
	out := make([]byte, len(frame))
	binary.LittleEndian.PutUint32(out[:4], 4)
	copy(out[4:], frame[4:])
	return out, false, nil
}

func makeJunk(jmin, jmax uint16) []byte {
	if jmax < jmin {
		jmin, jmax = jmax, jmin
	}
	if jmax == 0 {
		jmax = 100
	}
	span := int(jmax) - int(jmin)
	if span <= 0 {
		span = 1
	}
	var b [4]byte
	_, _ = rand.Read(b[:])
	size := int(jmin) + int(binary.BigEndian.Uint32(b[:])%uint32(span))
	if size < 4 {
		size = 4
	}
	out := make([]byte, size)
	binary.LittleEndian.PutUint32(out[:4], junkMarker)
	_, _ = rand.Read(out[4:])
	return out
}

// Proxy — UDP-relay между клиентом и локальным WG-портом, с AWG-обфускацией.
type Proxy struct {
	p          Params
	listenAddr *net.UDPAddr
	wgAddr     *net.UDPAddr
	log        *zerolog.Logger

	mu       sync.Mutex
	sessions map[string]*session
}

type session struct {
	wgConn     *net.UDPConn
	clientAddr *net.UDPAddr
	lastSeen   time.Time
}

func NewProxy(listenAddr, wgAddr string, p Params, log *zerolog.Logger) (*Proxy, error) {
	la, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return nil, err
	}
	wa, err := net.ResolveUDPAddr("udp", wgAddr)
	if err != nil {
		return nil, err
	}
	return &Proxy{p: p, listenAddr: la, wgAddr: wa, log: log, sessions: map[string]*session{}}, nil
}

// Run — блокирующий обработчик UDP.
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
		payload, junk, err := Unwrap(p.p, buf[:n])
		if err != nil {
			p.log.Debug().Err(err).Msg("unwrap failed")
			continue
		}
		if junk {
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

	// reverse direction: wg → client (с обфускацией). На первой пачке шлём Jc junk-кадров.
	go func() {
		buf := make([]byte, 65535)
		first := true
		for {
			n, _, err := wgConn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			if first {
				for i := uint8(0); i < p.p.Jc; i++ {
					_, _ = listenConn.WriteToUDP(makeJunk(p.p.Jmin, p.p.Jmax), addr)
				}
				first = false
			}
			frame := WrapOne(p.p, buf[:n])
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
