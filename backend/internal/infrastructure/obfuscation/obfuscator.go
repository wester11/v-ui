// Package obfuscation реализует базовую обфускацию UDP-трафика.
//
// Стратегии:
//  - Random padding: каждый пакет получает 0..MaxPad байт случайного «мусора»,
//    в начало кадра пишется 2-байтовая длина оригинального payload.
//  - XOR-маска: payload XOR-ится с keystream HKDF(psk, nonce). Не криптографически
//    стойко (трафик WG уже шифрован), цель — устранить статистические сигнатуры
//    handshake-пакетов от DPI.
//  - Decoy noise: периодически инжектится «мусорный» пакет случайного размера,
//    распознаётся флагом 0x00 в первом байте длины — приёмник его отбрасывает.
//
// Эта же логика реализована и в пакете agent/internal/obfs.
package obfuscation

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	mrand "math/rand"

	"golang.org/x/crypto/hkdf"
)

const (
	HeaderLen = 2
	MaxPad    = 96
	DecoyTag  = byte(0x00) // признак decoy-пакета (длина = 0)
)

type Obfuscator struct {
	key []byte // 32-байтовый секрет, общий для агента и клиента
}

func New(psk []byte) (*Obfuscator, error) {
	if len(psk) == 0 {
		return nil, errors.New("obfuscation: empty psk")
	}
	r := hkdf.New(sha256.New, psk, nil, []byte("voidwg-obfs/v1"))
	key := make([]byte, 32)
	if _, err := r.Read(key); err != nil {
		return nil, err
	}
	return &Obfuscator{key: key}, nil
}

// Wrap — маскирует исходящий пакет.
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

// Unwrap — снимает маскировку. Возвращает (payload, isDecoy, err).
func (o *Obfuscator) Unwrap(frame []byte) ([]byte, bool, error) {
	if len(frame) < HeaderLen {
		return nil, false, errors.New("frame too short")
	}
	nonce := append([]byte(nil), frame[:HeaderLen]...)
	body := append([]byte(nil), frame[HeaderLen:]...)
	o.xor(body, nonce)

	plen := int(binary.BigEndian.Uint16(nonce))
	if plen == 0 {
		return nil, true, nil // decoy
	}
	if plen > len(body) {
		return nil, false, errors.New("decoded length mismatch")
	}
	return body[:plen], false, nil
}

// MakeDecoy — пакет-обманка (длина=0, тело — случайный шум).
func (o *Obfuscator) MakeDecoy() []byte {
	size := 32 + mrand.Intn(MaxPad)
	out := make([]byte, HeaderLen+size)
	out[0] = DecoyTag
	out[1] = DecoyTag
	_, _ = rand.Read(out[HeaderLen:])
	o.xor(out[HeaderLen:], out[:HeaderLen])
	return out
}

// xor — keystream через HMAC-SHA256(key, nonce || counter).
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
