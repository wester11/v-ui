package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/usecase"
	"github.com/voidwg/control/internal/domain"
	mw "github.com/voidwg/control/internal/interfaces/http/middleware"
)

// NodeOpsHandler — runtime-операции над агентом ноды (restart / metrics /
// rotate-secret). Phase 7.
type NodeOpsHandler struct {
	ops      *usecase.NodeOpsService
	servers  *usecase.ServerService
	audit    *usecase.AuditService
}

func NewNodeOps(ops *usecase.NodeOpsService, servers *usecase.ServerService, audit *usecase.AuditService) *NodeOpsHandler {
	return &NodeOpsHandler{ops: ops, servers: servers, audit: audit}
}

// Restart — POST /api/v1/admin/servers/{id}/restart
func (h *NodeOpsHandler) Restart(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	uid := mw.UserIDFromCtx(r.Context())
	if err := h.ops.Restart(r.Context(), id); err != nil {
		h.audit.Log(r.Context(), domain.AuditEvent{
			ActorID: ptrUUID(uid), Action: "node.restart", Result: "error",
			TargetType: "server", TargetID: id.String(),
			IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
			Meta: map[string]any{"err": err.Error()},
		})
		writeErr(w, err)
		return
	}
	h.audit.Log(r.Context(), domain.AuditEvent{
		ActorID: ptrUUID(uid), Action: "node.restart", Result: "ok",
		TargetType: "server", TargetID: id.String(),
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})
	w.WriteHeader(http.StatusNoContent)
}

// Metrics — GET /api/v1/admin/servers/{id}/metrics
func (h *NodeOpsHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	body, err := h.ops.Metrics(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// RotateSecret — POST /api/v1/admin/servers/{id}/rotate-secret
// Возвращает новый secret в JSON ОДИН РАЗ.
func (h *NodeOpsHandler) RotateSecret(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	newSecret, err := h.servers.RotateSecret(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	uid := mw.UserIDFromCtx(r.Context())
	h.audit.Log(r.Context(), domain.AuditEvent{
		ActorID: ptrUUID(uid), Action: "node.rotate_secret", Result: "ok",
		TargetType: "server", TargetID: id.String(),
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"server_id": id,
		"secret":    newSecret,
		"warning":   "Old agent will stop authenticating. Update node compose env or re-run install-node.sh.",
	})
}
