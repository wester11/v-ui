package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/usecase"
	"github.com/voidwg/control/internal/domain"
	"github.com/voidwg/control/internal/interfaces/http/dto"
	mw "github.com/voidwg/control/internal/interfaces/http/middleware"
)

type PeerHandler struct {
	svc   *usecase.PeerService
	audit *usecase.AuditService
}

func NewPeer(s *usecase.PeerService, a *usecase.AuditService) *PeerHandler {
	return &PeerHandler{svc: s, audit: a}
}

// Create — операторский endpoint: оператор регистрирует peer'а от имени user'а,
// клиент уже сгенерил public_key. Приватник остаётся у клиента.
func (h *PeerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePeerRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	uid := mw.UserIDFromCtx(r.Context())
	peer, conf, err := h.svc.Create(r.Context(), usecase.CreatePeerInput{
		UserID:    uid,
		ServerID:  req.ServerID,
		Name:      req.Name,
		PublicKey: req.PublicKey,
	})
	if err != nil && peer == nil {
		writeErr(w, err)
		h.audit.Log(r.Context(), domain.AuditEvent{
			ActorID: ptrUUID(uid), Action: "peer.create", Result: "error",
			IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
			Meta: map[string]any{"err": err.Error()},
		})
		return
	}
	h.audit.Log(r.Context(), domain.AuditEvent{
		ActorID: ptrUUID(uid), Action: "peer.create", Result: "ok",
		TargetType: "peer", TargetID: peer.ID.String(),
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})
	writeJSON(w, http.StatusCreated, dto.CreatePeerResponse{
		Peer:       dto.PeerFromDomain(peer),
		ConfigStub: conf,
	})
}

func (h *PeerHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := mw.UserIDFromCtx(r.Context())
	peers, err := h.svc.ListByUser(r.Context(), uid)
	if err != nil {
		writeErr(w, err)
		return
	}
	out := make([]dto.PeerResponse, 0, len(peers))
	for _, p := range peers {
		out = append(out, dto.PeerFromDomain(p))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *PeerHandler) Config(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	conf, err := h.svc.Config(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="peer.conf"`)
	_, _ = w.Write([]byte(conf))
}

func (h *PeerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	uid := mw.UserIDFromCtx(r.Context())
	if err := h.svc.Revoke(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	h.audit.Log(r.Context(), domain.AuditEvent{
		ActorID: ptrUUID(uid), Action: "peer.delete", Result: "ok",
		TargetType: "peer", TargetID: id.String(),
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})
	w.WriteHeader(http.StatusNoContent)
}

// Redeploy — POST /api/v1/admin/servers/{id}/redeploy.
// Operator+ может вручную пересобрать xray config и запушить агенту.
// Используется при desync (агент перезапустился, drift конфигов).
func (h *PeerHandler) Redeploy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	uid := mw.UserIDFromCtx(r.Context())
	if err := h.svc.Redeploy(r.Context(), id); err != nil {
		h.audit.Log(r.Context(), domain.AuditEvent{
			ActorID: ptrUUID(uid), Action: "server.redeploy", Result: "error",
			TargetType: "server", TargetID: id.String(),
			IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
			Meta: map[string]any{"err": err.Error()},
		})
		writeErr(w, err)
		return
	}
	h.audit.Log(r.Context(), domain.AuditEvent{
		ActorID: ptrUUID(uid), Action: "server.redeploy", Result: "ok",
		TargetType: "server", TargetID: id.String(),
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})
	w.WriteHeader(http.StatusNoContent)
}

// RedeployAll вЂ” POST /api/v1/admin/servers/redeploy-all.
// Запускает массовый redeploy для всех Xray-узлов c retry логикой.
func (h *PeerHandler) RedeployAll(w http.ResponseWriter, r *http.Request) {
	uid := mw.UserIDFromCtx(r.Context())
	res, err := h.svc.RedeployAll(r.Context())
	if err != nil {
		h.audit.Log(r.Context(), domain.AuditEvent{
			ActorID: ptrUUID(uid), Action: "server.redeploy_all", Result: "error",
			IP: mw.ClientIP(r), UserAgent: r.UserAgent(), Meta: map[string]any{"err": err.Error()},
		})
		writeErr(w, err)
		return
	}
	h.audit.Log(r.Context(), domain.AuditEvent{
		ActorID: ptrUUID(uid), Action: "server.redeploy_all", Result: "ok",
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
		Meta: map[string]any{"count": len(res)},
	})
	writeJSON(w, http.StatusOK, res)
}

// Health вЂ” GET /api/v1/admin/servers/health.
// Возвращает health-report по всем нодам (online/offline/degraded).
func (h *PeerHandler) Health(w http.ResponseWriter, r *http.Request) {
	report, err := h.svc.HealthReport(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func ptrUUID(u uuid.UUID) *uuid.UUID { return &u }

// Patch — PATCH /api/v1/peers/{id}
// Принимает {"enabled": bool} — включает/выключает пир.
func (h *PeerHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	if body.Enabled == nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	p, err := h.svc.SetEnabled(r.Context(), id, *body.Enabled)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.PeerFromDomain(p))
}
