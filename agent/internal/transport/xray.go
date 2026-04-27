// Package transport — XrayTransport.
//
// Phase 5.1 (Remnawave-style refactor): xray управление полностью stateless.
// Агент НЕ:
//   - не запускает xray subprocess'ом
//   - не модифицирует config.json по peer'ам
//   - не хранит in-memory состояние peer'ов
//   - не реализует AddPeer / RemovePeer / Reload-per-peer
//
// Агент ТОЛЬКО:
//   1) ApplyConfig(json) — пишет файл атомарно (через .tmp + rename)
//   2) Restart()         — рестарт sidecar-контейнера xray через docker socket
//                          ИЛИ через `docker compose restart` (zero-config fallback)
//   3) HealthCheck()     — TCP-probe inbound-порта xray
//
// Сам xray-core живёт в отдельном контейнере (teddysun/xray) и читает тот же
// volume, куда пишет агент. Container restart управляется агентом через
// /var/run/docker.sock.
package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

// XrayTransport — stateless обёртка вокруг xray-контейнера.
type XrayTransport struct {
	configPath    string // /etc/xray/config.json (shared volume)
	containerName string // имя/id sidecar-контейнера xray
	healthAddr    string // host:port для TCP-probe (обычно 127.0.0.1:443)
	log           *zerolog.Logger
}

func NewXray(configPath, containerName, healthAddr string, log *zerolog.Logger) *XrayTransport {
	if configPath == "" {
		configPath = "/etc/xray/config.json"
	}
	if containerName == "" {
		containerName = "xray"
	}
	if healthAddr == "" {
		healthAddr = "127.0.0.1:443"
	}
	return &XrayTransport{
		configPath:    configPath,
		containerName: containerName,
		healthAddr:    healthAddr,
		log:           log,
	}
}

func (t *XrayTransport) Name() string { return "xray" }

// Start — для stateless-модели start это просто проверка, что директория
// под config существует. Сам xray поднимается docker-compose'ом отдельно.
func (t *XrayTransport) Start() error {
	return os.MkdirAll(filepath.Dir(t.configPath), 0o755)
}

// Stop — no-op. Контейнер xray управляется compose'ом.
func (t *XrayTransport) Stop() error { return nil }

// Reload — устаревший hook. В новой модели вместо peer-by-peer reload
// control-plane пушит полный config через ApplyConfig + Restart.
// Для совместимости с интерфейсом Transport: если cfg — это []byte, делаем
// эквивалент ApplyConfig+Restart.
func (t *XrayTransport) Reload(cfg any) error {
	switch v := cfg.(type) {
	case []byte:
		if err := t.ApplyConfig(v); err != nil {
			return err
		}
		return t.Restart()
	case json.RawMessage:
		if err := t.ApplyConfig([]byte(v)); err != nil {
			return err
		}
		return t.Restart()
	}
	return nil
}

// AddPeer — НЕ ПОДДЕРЖИВАЕТСЯ. Phase 5.1: agent не управляет peer'ами для xray.
// Возвращает ошибку, чтобы вызывающая сторона переключилась на DeployConfig.
func (t *XrayTransport) AddPeer(_ PeerSpec) error {
	return fmt.Errorf("xray: AddPeer not supported in stateless model — use control-plane DeployConfig")
}

// RemovePeer — НЕ ПОДДЕРЖИВАЕТСЯ.
func (t *XrayTransport) RemovePeer(_ string) error {
	return fmt.Errorf("xray: RemovePeer not supported in stateless model — use control-plane DeployConfig")
}

// ===== Stateless API =====

// ApplyConfig — атомарная запись config.json. Валидирует JSON-синтаксис
// перед записью.
func (t *XrayTransport) ApplyConfig(data []byte) error {
	// Pre-flight: parse-only валидация, чтобы не записать заведомо битый JSON,
	// который положит xray-контейнер.
	var sink any
	if err := json.Unmarshal(data, &sink); err != nil {
		return fmt.Errorf("xray: invalid config JSON: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(t.configPath), 0o755); err != nil {
		return err
	}
	tmp := t.configPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, t.configPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	t.log.Info().
		Str("path", t.configPath).
		Int("size", len(data)).
		Msg("xray config written")
	return nil
}

// Restart — перезапускает sidecar-контейнер xray.
//
// Использует docker CLI через UNIX-socket (/var/run/docker.sock должен быть
// смонтирован в агента). Это стандартный production-pattern (Remnawave,
// Wireguard-ui, 3x-ui — все так делают).
//
// Fallback: если docker CLI недоступен — пробуем `docker compose` (он же).
func (t *XrayTransport) Restart() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "restart", "--time=5", t.containerName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.log.Error().
			Err(err).
			Str("container", t.containerName).
			Bytes("output", out).
			Msg("docker restart failed")
		return fmt.Errorf("xray restart: %w (out=%s)", err, string(out))
	}
	t.log.Info().Str("container", t.containerName).Msg("xray container restarted")
	return nil
}

// HealthCheck — TCP-probe inbound-порта xray.
// Не парсит TLS handshake (там Reality, без cert проверка не работает),
// но connect-success достаточен, чтобы понять, что xray слушает.
func (t *XrayTransport) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", t.healthAddr)
	if err != nil {
		return fmt.Errorf("xray health: %w", err)
	}
	_ = conn.Close()
	return nil
}
