// Package client — клиент к control-plane: heartbeat, забор ожидающих команд.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"time"
)

type Control struct {
	baseURL string
	token   string
	hc      *http.Client
}

func New(baseURL, token string, insecure bool) *Control {
	return &Control{
		baseURL: baseURL,
		token:   token,
		hc: &http.Client{
			Timeout:   5 * time.Second,
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}},
		},
	}
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
