// Package obfuscation предоставляет ГЕНЕРАТОР AmneziaWG-параметров.
//
// Phase 4: вся бывшая XOR/HKDF-схема снесена. Реальная обфускация теперь
// выполняется на уровне самого WireGuard-протокола (форк amneziawg-go) —
// см. agent/internal/awg. Backend здесь только рандомизирует параметры
// под каждый сервер, чтобы у разных нод были разные fingerprint'ы.
package obfuscation

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/voidwg/control/internal/domain"
)

// Минимально-разумные значения, проверенные на устойчивость к DPI:
// см. amnezia-vpn/amneziawg-windows-client/issues и обсуждения параметров.
const (
	defaultJcMin   = 3
	defaultJcMax   = 10
	defaultPadMin  = 50
	defaultPadMax  = 1000
	defaultSPadMin = 15
	defaultSPadMax = 150
)

// RandomParams — генерирует уникальный набор AWG-параметров.
//
// Гарантии:
//   - Jc ∈ [3..10]
//   - Jmin < Jmax, оба ∈ [50..1000]
//   - S1, S2 ∈ [15..150]
//   - H1..H4 — попарно-различные uint32 в диапазоне [5..2^31), чтобы не пересечься
//     с реальными значениями WireGuard message_type (1..4 + 0).
func RandomParams() domain.AWGParams {
	jc := uint8(defaultJcMin + randInt(defaultJcMax-defaultJcMin))

	jmin := uint16(defaultPadMin + randInt(200))
	jmax := uint16(int(jmin) + 200 + randInt(defaultPadMax-int(jmin)-200))

	s1 := uint16(defaultSPadMin + randInt(defaultSPadMax-defaultSPadMin))
	s2 := uint16(defaultSPadMin + randInt(defaultSPadMax-defaultSPadMin))

	hs := uniqueU32x4()

	return domain.AWGParams{
		Jc: jc, Jmin: jmin, Jmax: jmax,
		S1: s1, S2: s2,
		H1: hs[0], H2: hs[1], H3: hs[2], H4: hs[3],
	}
}

func randInt(n int) int {
	if n <= 0 {
		return 0
	}
	var b [4]byte
	_, _ = rand.Read(b[:])
	return int(binary.BigEndian.Uint32(b[:]) % uint32(n))
}

func uniqueU32x4() [4]uint32 {
	var out [4]uint32
	seen := map[uint32]bool{0: true, 1: true, 2: true, 3: true, 4: true}
	for i := 0; i < 4; i++ {
		for {
			var b [4]byte
			_, _ = rand.Read(b[:])
			v := binary.BigEndian.Uint32(b[:]) | (1 << 31) // ensure высокий бит = 0 не у всех
			v &^= 1 << 31                                  // clear, чтоб остаться в [0, 2^31)
			if v < 5 {
				continue
			}
			if !seen[v] {
				seen[v] = true
				out[i] = v
				break
			}
		}
	}
	return out
}
