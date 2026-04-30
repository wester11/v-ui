package usecase

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
)

type ServerService struct {
	repo          port.ServerRepository
	publicBaseURL string
}

func NewServerService(r port.ServerRepository, _ port.KeyGenerator, _ port.MTLSIssuer, publicBaseURL string) *ServerService {
	return &ServerService{repo: r, publicBaseURL: strings.TrimSuffix(strings.TrimSpace(publicBaseURL), "/")}
}

type RegisterServerInput struct {
	Name     string
	Endpoint string
}

type RegisterResult struct {
	Server         *domain.Server
	NodeID         string
	Secret         string
	InstallCommand string
	ComposeSnippet string
}

type RegisterAgentInput struct {
	NodeID       string
	Secret       string
	Hostname     string
	IP           string
	AgentVersion string
}

func (s *ServerService) Register(ctx context.Context, in RegisterServerInput) (*RegisterResult, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, domain.ErrValidation
	}

	endpoint := strings.TrimSpace(in.Endpoint)
	if endpoint == "" {
		return nil, domain.ErrValidation
	}

	nodeSecret, err := randomHex(24)
	if err != nil {
		return nil, err
	}
	agentToken, err := randomHex(32)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	server := &domain.Server{
		ID:        uuid.New(),
		Name:      name,
		NodeID:    uuid.New(),
		NodeSecret: nodeSecret,
		Status:    "pending",
		Protocol:  domain.ProtoNone,
		Endpoint:  endpoint,
		Online:    false,
		AgentToken: agentToken, // legacy compatibility
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, server); err != nil {
		return nil, err
	}

	controlURL := s.controlURL(endpoint)

	// Encode credentials into a single base64 token so the install command is
	// short and the script is fetched from GitHub (valid SSL — no curl issues).
	// Format: base64("CONTROL_URL NODE_ID SECRET")
	rawToken := controlURL + " " + server.NodeID.String() + " " + server.NodeSecret
	installToken := base64.StdEncoding.EncodeToString([]byte(rawToken))
	const scriptURL = "https://raw.githubusercontent.com/wester11/v-ui/main/scripts/install-node.sh"
	installCmd := fmt.Sprintf("bash <(curl -Ls %s) %s", scriptURL, installToken)

	snippet := fmt.Sprintf(`services:
  void-node:
    image: void/node:latest
    network_mode: host
    restart: always
    environment:
      - CONTROL_URL=%s
      - NODE_ID=%s
      - SECRET=%s
`, controlURL, server.NodeID.String(), server.NodeSecret)

	return &RegisterResult{
		Server:         server,
		NodeID:         server.NodeID.String(),
		Secret:         server.NodeSecret,
		InstallCommand: installCmd,
		ComposeSnippet: snippet,
	}, nil
}

func (s *ServerService) RegisterAgent(ctx context.Context, in RegisterAgentInput) (*domain.Server, error) {
	nodeID, err := uuid.Parse(strings.TrimSpace(in.NodeID))
	if err != nil {
		return nil, domain.ErrValidation
	}
	server, err := s.repo.GetByNodeID(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if subtle.ConstantTimeCompare([]byte(server.NodeSecret), []byte(in.Secret)) != 1 {
		return nil, domain.ErrInvalidCredential
	}

	now := time.Now().UTC()
	server.Hostname = strings.TrimSpace(in.Hostname)
	server.IP = strings.TrimSpace(in.IP)
	server.AgentVersion = strings.TrimSpace(in.AgentVersion)
	server.Status = "online"
	server.Online = true
	server.LastHeartbeat = &now
	server.UpdatedAt = now

	// keep endpoint synced to reported ip if endpoint did not include explicit port
	if server.Endpoint == "" || !strings.Contains(server.Endpoint, ":") {
		host := server.IP
		if host == "" {
			host = server.Hostname
		}
		if host != "" {
			server.Endpoint = net.JoinHostPort(host, "7443")
		}
	}

	if err := s.repo.Update(ctx, server); err != nil {
		return nil, err
	}
	return server, nil
}

func (s *ServerService) Heartbeat(ctx context.Context, token string) (*domain.Server, error) {
	server, err := s.repo.GetByToken(ctx, strings.TrimSpace(token))
	if err != nil {
		return nil, err
	}
	return s.markOnline(ctx, server)
}

func (s *ServerService) HeartbeatByNode(ctx context.Context, nodeID, secret string) (*domain.Server, error) {
	id, err := uuid.Parse(strings.TrimSpace(nodeID))
	if err != nil {
		return nil, domain.ErrValidation
	}
	server, err := s.repo.GetByNodeID(ctx, id)
	if err != nil {
		return nil, err
	}
	if subtle.ConstantTimeCompare([]byte(server.NodeSecret), []byte(strings.TrimSpace(secret))) != 1 {
		return nil, domain.ErrInvalidCredential
	}
	return s.markOnline(ctx, server)
}

func (s *ServerService) markOnline(ctx context.Context, server *domain.Server) (*domain.Server, error) {
	now := time.Now().UTC()
	server.Status = "online"
	server.Online = true
	server.LastHeartbeat = &now
	server.UpdatedAt = now
	if err := s.repo.Update(ctx, server); err != nil {
		return nil, err
	}
	return server, nil
}

func (s *ServerService) List(ctx context.Context) ([]*domain.Server, error) { return s.repo.List(ctx) }

func (s *ServerService) Get(ctx context.Context, id uuid.UUID) (*domain.Server, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ServerService) Delete(ctx context.Context, id uuid.UUID) error { return s.repo.Delete(ctx, id) }

// MarkStaleOffline — публичный wrapper. Вызывается из background-monitor'а
// в cmd/api/main.go каждые 30с с threshold=60s.
func (s *ServerService) MarkStaleOffline(ctx context.Context, threshold time.Duration) (int, error) {
	return s.repo.MarkStaleOffline(ctx, threshold)
}

// RotateSecret — генерирует новый node_secret для сервера и сохраняет.
// После этого старый агент больше не сможет heartbeat'ить — оператору надо
// вручную обновить /opt/void-node/docker-compose.yml на VPS или переустановить
// ноду новой install-командой.
//
// Возвращает новый secret (одноразово).
func (s *ServerService) RotateSecret(ctx context.Context, id uuid.UUID) (string, error) {
	srv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	newSecret, err := randomHex(24)
	if err != nil {
		return "", err
	}
	srv.NodeSecret = newSecret
	srv.Status = "pending"
	srv.Online = false
	srv.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, srv); err != nil {
		return "", err
	}
	return newSecret, nil
}

func (s *ServerService) controlURL(endpoint string) string {
	if s.publicBaseURL != "" {
		return s.publicBaseURL
	}
	host := endpoint
	if h, p, err := net.SplitHostPort(endpoint); err == nil {
		if p == "80" || p == "443" {
			host = h
		}
	}
	if host == "" {
		host = "panel.example.com"
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return "https://" + host
	}
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return host
	}
	return "https://" + host
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func shellQuote(v string) string {
	if v == "" {
		return "''"
	}
	if !strings.ContainsAny(v, " \t\n'\"\\$`") {
		return v
	}
	return "'" + strings.ReplaceAll(v, "'", "'\"'\"'") + "'"
}

func normalizeCascadeRules(in []domain.XrayCascadeRule) []domain.XrayCascadeRule {
	if len(in) == 0 {
		return []domain.XrayCascadeRule{
			{Match: "geoip:ru", Outbound: "direct"},
			{Match: "geoip:!ru", Outbound: "proxy"},
		}
	}
	out := make([]domain.XrayCascadeRule, 0, len(in))
	for _, rule := range in {
		m := strings.TrimSpace(rule.Match)
		o := strings.ToLower(strings.TrimSpace(rule.Outbound))
		if m == "" {
			continue
		}
		if o != "proxy" {
			o = "direct"
		}
		out = append(out, domain.XrayCascadeRule{Match: m, Outbound: o})
	}
	if len(out) == 0 {
		return []domain.XrayCascadeRule{
			{Match: "geoip:ru", Outbound: "direct"},
			{Match: "geoip:!ru", Outbound: "proxy"},
		}
	}
	return out
}

func splitEndpointHostPort(endpoint string) (string, uint16) {
	host, p, err := net.SplitHostPort(endpoint)
	if err == nil {
		port, convErr := strconv.Atoi(p)
		if convErr == nil && port > 0 && port <= 65535 {
			return host, uint16(port)
		}
		return host, 443
	}
	if strings.Contains(err.Error(), "missing port in address") {
		return endpoint, 443
	}
	return "", 0
}

func ifEmpty(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

// ensureSystemClient — добавляет (или обновляет) служебный VLESS-client в
// XrayConfig.SystemClients. Идемпотентен по uuid и по email-метке. Используется
// для cascade-interconnect: upstream-нода должна знать о downstream'е, чтобы
// VLESS-handshake между нодами проходил.
func ensureSystemClient(cfg *domain.XrayConfig, vlessUUID, email, flow string) {
	if cfg == nil {
		return
	}
	vlessUUID = strings.TrimSpace(vlessUUID)
	email = strings.TrimSpace(email)
	if vlessUUID == "" {
		return
	}
	if flow == "" {
		flow = "xtls-rprx-vision"
	}
	for i := range cfg.SystemClients {
		c := &cfg.SystemClients[i]
		if c.ID == vlessUUID || (email != "" && c.Email == email) {
			c.ID = vlessUUID
			c.Email = email
			c.Flow = flow
			return
		}
	}
	cfg.SystemClients = append(cfg.SystemClients, domain.XraySystemClient{
		ID:    vlessUUID,
		Email: email,
		Flow:  flow,
	})
}
