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

// InstallNodeScript отдаёт скрипт установки агента на новый VPS.
//
// Скрипт:
//   1) ставит docker
//   2) git-clone'ит репозиторий (REPO_URL env override, default из install-time)
//   3) docker build -t void/node:latest agent/
//   4) docker compose up -d с CONTROL_URL/NODE_ID/SECRET переданными как ENV
//
// Идемпотентен: повторный run пересобирает образ.
func (h *ServerHandler) InstallNodeScript(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(nodeInstallScript))
}

const nodeInstallScript = `#!/usr/bin/env bash
# void-wg node installer.
#
# Запускается с --control-url=<url> --node-id=<uuid> --secret=<hex>.
# Этот скрипт ОТДАЁТСЯ control-plane'ом по адресу /install-node.sh
# и встроен в backend как const nodeInstallScript.
#
# Логика:
#   1) проверка root + установка docker / docker compose
#   2) скачивание исходников агента из <CONTROL_URL>/static/agent.tar.gz
#      (control-plane отдаёт публично; альтернатива — клонировать репозиторий)
#   3) локальный docker build → image void/node:latest
#   4) генерация /opt/void-node/docker-compose.yml
#   5) docker compose up -d
#
# Идемпотентен: повторный запуск пересобирает образ с актуальным кодом.
set -Eeuo pipefail

CONTROL_URL=""
NODE_ID=""
SECRET=""
NODE_VERSION="${NODE_VERSION:-latest}"
NODE_INSTALL_DIR="${NODE_INSTALL_DIR:-/opt/void-node}"
# REPO_URL — откуда тянуть исходники. По умолчанию резолвим из control-plane'а
# (репозиторий должен совпадать с тем, откуда установлена сама панель).
REPO_URL="${REPO_URL:-https://github.com/wester11/v-ui.git}"
REPO_BRANCH="${REPO_BRANCH:-main}"

for arg in "$@"; do
  case "$arg" in
    --control-url=*) CONTROL_URL="${arg#*=}" ;;
    --node-id=*)     NODE_ID="${arg#*=}" ;;
    --secret=*)      SECRET="${arg#*=}" ;;
    --repo=*)        REPO_URL="${arg#*=}" ;;
    --branch=*)      REPO_BRANCH="${arg#*=}" ;;
  esac
done

# Pretty-печать
G='\033[0;32m'; Y='\033[1;33m'; R='\033[0;31m'; N`
