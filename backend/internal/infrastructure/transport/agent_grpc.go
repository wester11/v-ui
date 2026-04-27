// Package transport реализует HTTP-клиент к агентам узлов.
// Для простоты используется JSON over HTTPS вместо gRPC; протокол скрыт
// за интерфейсом port.AgentTransport.
package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/domain"
)

type AgentClient struct {
	hc *http.Client
}

func NewAgentClient(insecure bool) *AgentClient {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	return &AgentClient{hc: &http.Client{Timeout: 10 * time.Second, Transport: tr}}
}

// peerPayload — runtime-mutation для WG/AWG. Для xray не используется.
type peerPayload struct {
	ID        string `json:"id"`
	Protocol  string `json:"protocol"`
	PublicKey string `json:"public_key,omitempty"`
	AllowedIP string `json:"allowed_ip,omitempty"`
}

func (a *AgentClient) ApplyPeer(ctx context.Context, srv *domain.Server, p *domain.Peer) error {
	// Phase 5.1: для xray runtime-mutation не используется — control-plane
	// сам пересобирает config и пушит DeployConfig'ом. ApplyPeer для xray = no-op.
	if srv.Protocol == domain.ProtoXray {
		return nil
	}
	body, _ := json.Marshal(peerPayload{
		ID:        p.ID.String(),
		Protocol:  string(p.Protocol),
		PublicKey: p.PublicKey,
		AllowedIP: p.AssignedIP.String() + "/32",
	})
	return a.do(ctx, srv, http.MethodPost, "/v1/peers", "application/json", body)
}

func (a *AgentClient) RevokePeer(ctx context.Context, srv *domain.Server, peerID uuid.UUID) error {
	if srv.Protocol == domain.ProtoXray {
		return nil
	}
	return a.do(ctx, srv, http.MethodDelete, "/v1/peers/"+peerID.String(), "", nil)
}

// DeployConfig — пушит ПОЛНЫЙ config.json на агента.
// Используется для xray; для WG/AWG — no-op (там runtime-mutation).
func (a *AgentClient) DeployConfig(ctx context.Context, srv *domain.Server, configJSON []byte) error {
	if srv.Protocol != domain.ProtoXray {
		return nil
	}
	return a.do(ctx, srv, http.MethodPost, "/v1/xray/deploy", "application/json", configJSON)
}

func (a *AgentClient) Health(ctx context.Context, srv *domain.Server) error {
	return a.do(ctx, srv, http.MethodGet, "/healthz", "", nil)
}

func (a *AgentClient) do(ctx context.Context, srv *domain.Server, method, path, contentType string, body []byte) error {
	url := fmt.Sprintf("https://%s%s", srv.Endpoint, path)
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-Agent-Token", srv.AgentToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := a.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("agent %s %s %s: status %d", srv.Name, method, path, resp.StatusCode)
	}
	return nil
}
