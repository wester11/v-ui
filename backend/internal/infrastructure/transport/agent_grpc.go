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
	return &AgentClient{hc: &http.Client{Timeout: 5 * time.Second, Transport: tr}}
}

type peerPayload struct {
	ID           string `json:"id"`
	PublicKey    string `json:"public_key"`
	PresharedKey string `json:"preshared_key"`
	AllowedIP    string `json:"allowed_ip"`
}

func (a *AgentClient) ApplyPeer(ctx context.Context, srv *domain.Server, p *domain.Peer) error {
	body, _ := json.Marshal(peerPayload{
		ID:           p.ID.String(),
		PublicKey:    p.PublicKey,
		PresharedKey: p.PresharedKey,
		AllowedIP:    p.AssignedIP.String() + "/32",
	})
	return a.do(ctx, srv, http.MethodPost, "/v1/peers", body)
}

func (a *AgentClient) RevokePeer(ctx context.Context, srv *domain.Server, peerID uuid.UUID) error {
	return a.do(ctx, srv, http.MethodDelete, "/v1/peers/"+peerID.String(), nil)
}

func (a *AgentClient) Health(ctx context.Context, srv *domain.Server) error {
	return a.do(ctx, srv, http.MethodGet, "/healthz", nil)
}

func (a *AgentClient) do(ctx context.Context, srv *domain.Server, method, path string, body []byte) error {
	url := fmt.Sprintf("https://%s%s", srv.Endpoint, path)
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-Agent-Token", srv.AgentToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("agent %s: status %d", srv.Name, resp.StatusCode)
	}
	return nil
}
