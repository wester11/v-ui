// Package transport — единый интерфейс над WireGuard / AmneziaWG / Xray
// движками внутри агента. Phase 5: позволяет одному агенту обслуживать
// разные протоколы без рестартов всего процесса.
package transport

// Transport — общий контракт для всех движков.
//
// Контракт:
//   * Start() — поднять процесс/inteface (idempotent).
//   * Stop()  — корректно остановить.
//   * Reload(cfg) — применить новую конфигурацию без рестарта (для xray
//     это перерендерить config.json и SIGHUP; для WG — wg syncconf).
//   * AddPeer / RemovePeer — point-in-time изменения без полного reload.
type Transport interface {
	Name() string
	Start() error
	Stop() error
	Reload(cfg any) error
	AddPeer(p PeerSpec) error
	RemovePeer(id string) error
}

// PeerSpec — protocol-agnostic описание peer'а, приходящее с control-plane.
type PeerSpec struct {
	ID        string
	Protocol  string

	// WG/AWG
	PublicKey string
	AllowedIP string

	// Xray
	XrayUUID    string
	XrayFlow    string
	XrayShortID string
	XrayEmail   string // для логирования; обычно peer.Name
}
