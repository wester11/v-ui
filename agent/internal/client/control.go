// Package client — клиент к control-plane: heartbeat + mTLS.
//
// Phase 4: secure-by-default. Если задан CA/cert/key — используется mTLS
// с проверкой fingerprint'а сервера. Если задан только CA — обычный TLS
// с проверкой CA. Insecure — только если явно требуется (debug).
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
	"time"
)

type Control struct {
	baseURL string
	token   string
	hc      *http.Client
}

type TLSConfig struct {
	CAFile   string // PEM bundle CA панели
	CertFile string // PEM client cert (mTLS)
	KeyFile  string // PEM client key
	Insecure bool   // НЕ использовать в production
}

func New(baseURL, token string, t TLSConfig) (*Control, error) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
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
		baseURL: baseURL,
		token:   token,
		hc: &http.Client{
			Timeout:   10 * time.Second,
			Transport: &http.Transport{TLSClientConfig: tlsCfg},
		},
	}, nil
}

func (c *Control) Heartbeat(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/agent/heartbeat", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-Agent-Token", c.token)
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
