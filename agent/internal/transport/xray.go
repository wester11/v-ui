package transport

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

// XrayTransport — управляет процессом xray-core на агенте.
//
// Архитектура:
//   * config.json генерится управлением (control-plane передаёт уже готовый JSON
//     или Reality-параметры + список peer'ов).
//   * Процесс запускается как subprocess: `xray run -c <ConfigPath>`.
//   * Reload: переписать config.json -> SIGHUP (xray hot-reload), либо restart.
//   * AddPeer / RemovePeer — мерджит локальный реестр и вызывает Reload.
//
// Альтернатива subprocess'у — Xray API (gRPC). Здесь идём более простым путём
// (config-file + SIGHUP), что проще для контейнеризации.
type XrayTransport struct {
	binary     string // путь к xray binary (default: "xray")
	configPath string // /etc/xray/config.json
	log        *zerolog.Logger

	mu         sync.Mutex
	cmd        *exec.Cmd
	currentCfg map[string]any
	peers      map[string]map[string]any // id -> client object
}

type XrayInbound struct {
	Port            uint16   `json:"port"`
	SNI             string   `json:"sni"`
	Dest            string   `json:"dest"`
	PrivateKey      string   `json:"private_key"`
	ShortIDs        []string `json:"short_ids"`
	Fingerprint     string   `json:"fingerprint"`
	Flow            string   `json:"flow"`
}

func NewXray(binary, configPath string, log *zerolog.Logger) *XrayTransport {
	if binary == "" {
		binary = "xray"
	}
	if configPath == "" {
		configPath = "/etc/xray/config.json"
	}
	return &XrayTransport{
		binary:     binary,
		configPath: configPath,
		log:        log,
		peers:      map[string]map[string]any{},
	}
}

func (t *XrayTransport) Name() string { return "xray" }

// Start — запускает xray subprocess. Если config ещё не был сгенерён —
// пишем минимальный заглушечный конфиг (inbound с Reality, без клиентов).
func (t *XrayTransport) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cmd != nil && t.cmd.Process != nil {
		return nil // уже запущен
	}
	if _, err := os.Stat(t.configPath); os.IsNotExist(err) {
		if err := t.writeMinimalStub(); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(t.configPath), 0o755); err != nil {
		return err
	}

	cmd := exec.Command(t.binary, "run", "-c", t.configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("xray start: %w", err)
	}
	t.cmd = cmd
	t.log.Info().Str("bin", t.binary).Str("cfg", t.configPath).Msg("xray started")

	go func() {
		_ = cmd.Wait()
		t.log.Warn().Msg("xray process exited")
	}()
	return nil
}

func (t *XrayTransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cmd == nil || t.cmd.Process == nil {
		return nil
	}
	_ = t.cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan struct{})
	go func() { _, _ = t.cmd.Process.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = t.cmd.Process.Kill()
	}
	t.cmd = nil
	return nil
}

// Reload — принимает XrayInbound (или совместимую map) и пересобирает config.json.
func (t *XrayTransport) Reload(cfg any) error {
	t.mu.Lock()
	in, err := coerceInbound(cfg)
	if err != nil {
		t.mu.Unlock()
		return err
	}
	t.currentCfg = inboundToMap(in)
	if err := t.renderLocked(); err != nil {
		t.mu.Unlock()
		return err
	}
	t.mu.Unlock()
	return t.sighup()
}

// AddPeer — добавляет VLESS-client в config.json и делает SIGHUP.
func (t *XrayTransport) AddPeer(p PeerSpec) error {
	t.mu.Lock()
	if p.Protocol != "xray" {
		t.mu.Unlock()
		return fmt.Errorf("xray transport: wrong protocol %q", p.Protocol)
	}
	t.peers[p.ID] = map[string]any{
		"id":    p.XrayUUID,
		"flow":  p.XrayFlow,
		"email": p.XrayEmail,
	}
	if err := t.renderLocked(); err != nil {
		t.mu.Unlock()
		return err
	}
	t.mu.Unlock()
	return t.sighup()
}

func (t *XrayTransport) RemovePeer(id string) error {
	t.mu.Lock()
	delete(t.peers, id)
	if err := t.renderLocked(); err != nil {
		t.mu.Unlock()
		return err
	}
	t.mu.Unlock()
	return t.sighup()
}

// renderLocked — пересборка config.json под текущие inbound + peers. Ожидает t.mu.
func (t *XrayTransport) renderLocked() error {
	clients := make([]map[string]any, 0, len(t.peers))
	for _, c := range t.peers {
		clients = append(clients, c)
	}

	if t.currentCfg == nil {
		// fallback — пишем минимум только из peers без inbound-данных
		return t.writeMinimalStub()
	}

	// inject clients into existing config
	cfg := cloneMap(t.currentCfg)
	inbounds, _ := cfg["inbounds"].([]any)
	if len(inbounds) > 0 {
		first, _ := inbounds[0].(map[string]any)
		if settings, ok := first["settings"].(map[string]any); ok {
			settings["clients"] = clients
		}
	}
	return writeJSONFile(t.configPath, cfg)
}

func (t *XrayTransport) sighup() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cmd == nil || t.cmd.Process == nil {
		return nil
	}
	// Xray поддерживает SIGHUP для hot-reload (с v1.8+); до этой версии — full restart.
	if err := t.cmd.Process.Signal(syscall.SIGHUP); err != nil {
		// fallback: жёсткий перезапуск
		_ = t.cmd.Process.Kill()
		t.cmd = nil
	}
	return nil
}

func (t *XrayTransport) writeMinimalStub() error {
	stub := map[string]any{
		"log": map[string]any{"loglevel": "warning"},
		"inbounds": []any{
			map[string]any{
				"listen":   "0.0.0.0",
				"port":     443,
				"protocol": "vless",
				"settings": map[string]any{
					"clients":    []any{},
					"decryption": "none",
				},
				"streamSettings": map[string]any{
					"network":  "tcp",
					"security": "reality",
					"realitySettings": map[string]any{
						"show":         false,
						"dest":         "www.cloudflare.com:443",
						"serverNames":  []string{"www.cloudflare.com"},
						"privateKey":   "",
						"shortIds":     []string{""},
						"fingerprint":  "chrome",
					},
				},
			},
		},
		"outbounds": []any{
			map[string]any{"protocol": "freedom"},
			map[string]any{"protocol": "blackhole", "tag": "block"},
		},
	}
	return writeJSONFile(t.configPath, stub)
}

func writeJSONFile(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func coerceInbound(cfg any) (XrayInbound, error) {
	switch v := cfg.(type) {
	case XrayInbound:
		return v, nil
	case []byte:
		var x XrayInbound
		err := json.Unmarshal(v, &x)
		return x, err
	case string:
		var x XrayInbound
		err := json.Unmarshal([]byte(v), &x)
		return x, err
	case map[string]any:
		b, _ := json.Marshal(v)
		var x XrayInbound
		err := json.Unmarshal(b, &x)
		return x, err
	}
	return XrayInbound{}, fmt.Errorf("xray: unsupported config type %T", cfg)
}

func inboundToMap(in XrayInbound) map[string]any {
	port := in.Port
	if port == 0 {
		port = 443
	}
	dest := in.Dest
	if dest == "" {
		dest = "www.cloudflare.com:443"
	}
	sni := in.SNI
	if sni == "" {
		sni = "www.cloudflare.com"
	}
	fp := in.Fingerprint
	if fp == "" {
		fp = "chrome"
	}
	shortIDs := in.ShortIDs
	if len(shortIDs) == 0 {
		shortIDs = []string{""}
	}
	return map[string]any{
		"log": map[string]any{"loglevel": "warning"},
		"inbounds": []any{
			map[string]any{
				"listen":   "0.0.0.0",
				"port":     port,
				"protocol": "vless",
				"tag":      "vless-reality",
				"settings": map[string]any{
					"clients":    []any{},
					"decryption": "none",
				},
				"streamSettings": map[string]any{
					"network":  "tcp",
					"security": "reality",
					"realitySettings": map[string]any{
						"show":         false,
						"dest":         dest,
						"serverNames":  []string{sni},
						"privateKey":   in.PrivateKey,
						"shortIds":     shortIDs,
						"fingerprint":  fp,
					},
				},
				"sniffing": map[string]any{
					"enabled":      true,
					"destOverride": []string{"http", "tls", "quic"},
				},
			},
		},
		"outbounds": []any{
			map[string]any{"protocol": "freedom", "tag": "direct"},
			map[string]any{"protocol": "blackhole", "tag": "block"},
		},
	}
}

func cloneMap(m map[string]any) map[string]any {
	b, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}
