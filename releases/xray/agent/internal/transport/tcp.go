// Package transport — UDP-over-TCP / UDP-over-TLS туннель для WireGuard.
//
// Предназначение: fallback-транспорт на случай, если ISP режет/блокирует UDP.
// Wire format кадров:
//
//	+---------------+----------------------------+
//	|  uint16 BE    | payload (длина = uint16)   |
//	|  payload_len  |                            |
//	+---------------+----------------------------+
//
// Каждое TCP/TLS-соединение = одна WG-сессия, форвардится в локальный
// UDP-сокет на 127.0.0.1:wg_port.
package transport

import (
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"time"

	"github.com/rs/zerolog"

	"github.com/voidwg/agent/internal/awg"
)

// TCPTunnel — слушает TCP, при коннекте устанавливает UDP-сессию к WG.
type TCPTunnel struct {
	listenAddr string
	wgAddr     string
	awgParams  awg.Params
	log        *zerolog.Logger
	tlsCfg     *tls.Config // nil = чистый TCP, не-nil = TLS
}

func NewTCP(listenAddr, wgAddr string, p awg.Params, log *zerolog.Logger) *TCPTunnel {
	return &TCPTunnel{listenAddr: listenAddr, wgAddr: wgAddr, awgParams: p, log: log}
}

// NewTLS — то же самое, но с TLS-терминацией.
// certFile/keyFile — серверный cert (тот же, что у панели), может быть LE.
func NewTLS(listenAddr, wgAddr, certFile, keyFile string, p awg.Params, log *zerolog.Logger) (*TCPTunnel, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &TCPTunnel{
		listenAddr: listenAddr,
		wgAddr:     wgAddr,
		awgParams:  p,
		log:        log,
		tlsCfg: &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
			NextProtos:   []string{"h2", "http/1.1"}, // mimicry: выглядит как обычный HTTPS-сервер
		},
	}, nil
}

func (t *TCPTunnel) Run() error {
	var ln net.Listener
	var err error
	if t.tlsCfg != nil {
		ln, err = tls.Listen("tcp", t.listenAddr, t.tlsCfg)
	} else {
		ln, err = net.Listen("tcp", t.listenAddr)
	}
	if err != nil {
		return err
	}
	defer ln.Close()
	t.log.Info().Str("addr", t.listenAddr).Bool("tls", t.tlsCfg != nil).Msg("transport listening")

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go t.handle(conn)
	}
}

func (t *TCPTunnel) handle(conn net.Conn) {
	defer conn.Close()
	wg, err := net.Dial("udp", t.wgAddr)
	if err != nil {
		t.log.Error().Err(err).Msg("dial wg")
		return
	}
	defer wg.Close()

	// TCP -> UDP (incoming)
	go func() {
		defer conn.Close()
		defer wg.Close()
		for {
			frame, err := readFrame(conn)
			if err != nil {
				return
			}
			payload, junk, err := awg.Unwrap(t.awgParams, frame)
			if err != nil || junk {
				continue
			}
			_ = conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
			_, _ = wg.Write(payload)
		}
	}()

	// UDP -> TCP (outgoing)
	buf := make([]byte, 65535)
	first := true
	for {
		n, err := wg.Read(buf)
		if err != nil {
			return
		}
		if first {
			// preroll Jc junk-frames
			for i := uint8(0); i < t.awgParams.Jc; i++ {
				_ = writeFrame(conn, awg.WrapOne(t.awgParams, []byte{0xFF, 0xFF, 0xFF, 0xFF}))
			}
			first = false
		}
		frame := awg.WrapOne(t.awgParams, buf[:n])
		if err := writeFrame(conn, frame); err != nil {
			return
		}
	}
}

func readFrame(r io.Reader) ([]byte, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := int(binary.BigEndian.Uint16(hdr[:]))
	if n == 0 || n > 65535 {
		return nil, errors.New("invalid frame length")
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func writeFrame(w io.Writer, payload []byte) error {
	var hdr [2]byte
	binary.BigEndian.PutUint16(hdr[:], uint16(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}
