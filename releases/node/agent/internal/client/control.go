package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
)

type Control struct {
	baseURL   string
	agentToken string
	nodeID    string
	secret    string
	hc        *http.Client
}

type TLSConfig struct {
	CAFile   string
	CertFile string
	KeyFile  string
	Insecure bool
}

func New(baseURL, agentToken, nodeID, secret string, t TLSConfig) (*Control, error) {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if t.Insecure {
		tlsCfg.InsecureSkipVerify = true
	} else if t.CAFile != "" {
		caPEM, err := os.ReadFile(t.CAFile)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, errors.New("ca pem: no certs found")
		}
		tlsCfg.RootCAs = pool
	}
	if t.CertFile != "" && t.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
		if err != nil {
			return nil, err
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	return &Control{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		agentToken: strings.TrimSpace(agentToken),
		nodeID:     strings.TrimSpace(nodeID),
		secret:     strings.TrimSpace(secret),
		hc: &http.Client{
			Timeout:   10 * time.Second,
			Transport: &http.Transport{TLSClientConfig: tlsCfg},
		},
	}, nil
}

func (c *Control) Register(ctx context.Context, hostname, ip, version string) error {
	if c.nodeID == "" || c.secret == "" {
		return nil
	}
	body, _ := json.Marshal(map[string]string{
		"node_id":       c.nodeID,
		"secret":        c.secret,
		"hostname":      hostname,
		"ip":            ip,
		"agent_version": version,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/agent/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return errFromResp(resp)
	}
	return nil
}

func (c *Control) Heartbeat(ctx context.Context) error {
	body := map[string]string{}
	if c.nodeID != "" && c.secret != "" {
		body["node_id"] = c.nodeID
		body["secret"] = c.secret
	}
	payload, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/agent/heartbeat", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if c.agentToken != "" {
		req.Header.Set("X-Agent-Token", c.agentToken)
	}
	if c.nodeID != "" && c.secret != "" {
		req.Header.Set("X-Node-ID", c.nodeID)
		req.Header.Set("X-Node-Secret", c.secret)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return errFromResp(resp)
	}
	return nil
}

func errFromResp(resp *http.Response) error {
	var e struct{ Error string }
	_ = json.NewDecoder(resp.Body).Decode(&e)
	return &httpError{status: resp.StatusCode, msg: e.Error}
}

type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string { return e.msg }

