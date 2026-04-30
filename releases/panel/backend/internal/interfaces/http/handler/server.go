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

// InstallNodeScript — редирект на актуальную версию скрипта на GitHub.
// Скрипт больше не встроен в бинарник: это упрощает обновление без rebuild.
func (h *ServerHandler) InstallNodeScript(w http.ResponseWriter, r *http.Request) {
	const githubScriptURL = "https://raw.githubusercontent.com/wester11/v-ui/main/scripts/install-node.sh"
	http.Redirect(w, r, githubScriptURL, http.StatusFound)
}
