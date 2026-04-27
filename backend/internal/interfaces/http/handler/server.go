package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/usecase"
	"github.com/voidwg/control/internal/domain"
	"github.com/voidwg/control/internal/interfaces/http/dto"
	mw "github.com/voidwg/control/internal/interfaces/http/middleware"
)

type ServerHandler struct {
	svc        *usecase.ServerService
	configSvc  *usecase.ConfigService
	audit      *usecase.AuditService
}

func NewServer(s *usecase.ServerService, cfg *usecase.ConfigService, a *usecase.AuditService) *ServerHandler {
	return &ServerHandler{svc: s, configSvc: cfg, audit: a}
}

func (h *ServerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateServerRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	res, err := h.svc.Register(r.Context(), usecase.RegisterServerInput{
		Name: req.Name, Endpoint: req.Endpoint,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	uid := mw.UserIDFromCtx(r.Context())
	h.audit.Log(r.Context(), domain.AuditEvent{
		ActorID: ptrUUID(uid), Action: "server.create", Result: "ok",
		TargetType: "server", TargetID: res.Server.ID.String(),
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})
	writeJSON(w, http.StatusCreated, dto.CreateServerResponse{
		ServerResponse: dto.ServerFromDomain(res.Server),
		NodeID:         res.NodeID,
		Secret:         res.Secret,
		InstallCommand: res.InstallCommand,
		ComposeSnippet: res.ComposeSnippet,
	})
}

func (h *ServerHandler) List(w http.ResponseWriter, r *http.Request) {
	servers, err := h.svc.List(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	out := make([]dto.ServerResponse, 0, len(servers))
	for _, s := range servers {
		out = append(out, dto.ServerFromDomain(s))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *ServerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	uid := mw.UserIDFromCtx(r.Context())
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	h.audit.Log(r.Context(), domain.AuditEvent{
		ActorID: ptrUUID(uid), Action: "server.delete", Result: "ok",
		TargetType: "server", TargetID: id.String(),
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *ServerHandler) RegisterAgent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID       string `json:"node_id"`
		Secret       string `json:"secret"`
		Hostname     string `json:"hostname"`
		IP           string `json:"ip"`
		AgentVersion string `json:"agent_version"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	srv, err := h.svc.RegisterAgent(r.Context(), usecase.RegisterAgentInput{
		NodeID: req.NodeID, Secret: req.Secret, Hostname: req.Hostname, IP: req.IP, AgentVersion: req.AgentVersion,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.ServerFromDomain(srv))
}

func (h *ServerHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.Header.Get("X-Agent-Token"))
	if token != "" {
		srv, err := h.svc.Heartbeat(r.Context(), token)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, dto.ServerFromDomain(srv))
		return
	}

	nodeID := strings.TrimSpace(r.Header.Get("X-Node-ID"))
	secret := strings.TrimSpace(r.Header.Get("X-Node-Secret"))
	if nodeID == "" || secret == "" {
		var req struct {
			NodeID string `json:"node_id"`
			Secret string `json:"secret"`
		}
		if err := decode(r, &req); err == nil {
			nodeID = strings.TrimSpace(req.NodeID)
			secret = strings.TrimSpace(req.Secret)
		}
	}
	if nodeID == "" || secret == "" {
		writeErr(w, domain.ErrInvalidCredential)
		return
	}

	srv, err := h.svc.HeartbeatByNode(r.Context(), nodeID, secret)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.ServerFromDomain(srv))
}

func (h *ServerHandler) Check(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	srv, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	status := "offline"
	if srv.Online {
		status = "online"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"server_id":      srv.ID,
		"status":         status,
		"last_heartbeat": srv.LastHeartbeat,
		"ip":             srv.IP,
		"hostname":       srv.Hostname,
		"agent_version":  srv.AgentVersion,
	})
}

func (h *ServerHandler) InstallNodeScript(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(nodeInstallScript))
}

const nodeInstallScript = `#!/usr/bin/env bash
set -Eeuo pipefail

CONTROL_URL=""
NODE_ID=""
SECRET=""
NODE_VERSION="${NODE_VERSION:-latest}"

for arg in "$@"; do
  case "$arg" in
    --control-url=*) CONTROL_URL="${arg#*=}" ;;
    --node-id=*) NODE_ID="${arg#*=}" ;;
    --secret=*) SECRET="${arg#*=}" ;;
  esac
done

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root" >&2
  exit 1
fi

[ -n "$CONTROL_URL" ] || { echo "--control-url is required" >&2; exit 1; }
[ -n "$NODE_ID" ] || { echo "--node-id is required" >&2; exit 1; }
[ -n "$SECRET" ] || { echo "--secret is required" >&2; exit 1; }

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq ca-certificates curl gnupg lsb-release

if ! command -v docker >/dev/null 2>&1; then
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL "https://download.docker.com/linux/$(. /etc/os-release; echo "$ID")/gpg" | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/$(. /etc/os-release; echo "$ID") $(lsb_release -cs) stable" > /etc/apt/sources.list.d/docker.list
  apt-get update -qq
  apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  systemctl enable --now docker
fi

mkdir -p /opt/void-node
cat >/opt/void-node/docker-compose.yml <<EOF
services:
  void-node:
    image: void/node:${NODE_VERSION}
    network_mode: host
    restart: always
    environment:
      - CONTROL_URL=${CONTROL_URL}
      - NODE_ID=${NODE_ID}
      - SECRET=${SECRET}
      - HTTP_LISTEN=:7443
EOF

cd /opt/void-node
docker compose pull || true
docker compose up -d --remove-orphans
echo "node agent started"
`
