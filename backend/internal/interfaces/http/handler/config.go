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

type ConfigHandler struct {
	svc   *usecase.ConfigService
	audit *usecase.AuditService
}

func NewConfig(s *usecase.ConfigService, a *usecase.AuditService) *ConfigHandler {
	return &ConfigHandler{svc: s, audit: a}
}

func (h *ConfigHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateConfigRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}

	rules := make([]domain.XrayCascadeRule, 0, len(req.CascadeRules))
	for _, rr := range req.CascadeRules {
		rules = append(rules, domain.XrayCascadeRule{Match: rr.Match, Outbound: rr.Outbound})
	}

	cfg, err := h.svc.Create(r.Context(), usecase.CreateConfigInput{
		ServerID:      req.ServerID,
		Name:          req.Name,
		Protocol:      domain.Protocol(req.Protocol),
		Template:      domain.ConfigTemplate(req.Template),
		SetupMode:     domain.ConfigSetupMode(req.SetupMode),
		RoutingMode:   domain.ConfigRoutingMode(req.RoutingMode),
		Activate:      req.Activate,
		RawJSON:       req.RawJSON,
		InboundPort:   req.InboundPort,
		SNI:           req.SNI,
		Dest:          req.Dest,
		Fingerprint:   req.Fingerprint,
		Flow:          req.Flow,
		ShortIDsCount: req.ShortIDsCount,
		CascadeUpstreamID: req.CascadeUpstreamID,
		CascadeStrategy:   req.CascadeStrategy,
		CascadeRules:      rules,
	})
	if err != nil {
		writeErr(w, err)
		return
	}

	uid := mw.UserIDFromCtx(r.Context())
	h.audit.Log(r.Context(), domain.AuditEvent{
		ActorID: ptrUUID(uid), Action: "config.create", Result: "ok",
		TargetType: "config", TargetID: cfg.ID.String(),
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})

	writeJSON(w, http.StatusCreated, dto.ConfigFromDomain(cfg, true))
}

func (h *ConfigHandler) ListByServer(w http.ResponseWriter, r *http.Request) {
	serverID, err := uuid.Parse(chi.URLParam(r, "serverID"))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	configs, err := h.svc.ListByServer(r.Context(), serverID)
	if err != nil {
		writeErr(w, err)
		return
	}
	out := make([]dto.ConfigResponse, 0, len(configs))
	for _, cfg := range configs {
		out = append(out, dto.ConfigFromDomain(cfg, false))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *ConfigHandler) Activate(w http.ResponseWriter, r *http.Request) {
	configID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	if err := h.svc.Activate(r.Context(), configID); err != nil {
		writeErr(w, err)
		return
	}
	uid := mw.UserIDFromCtx(r.Context())
	h.audit.Log(r.Context(), domain.AuditEvent{
		ActorID: ptrUUID(uid), Action: "config.activate", Result: "ok",
		TargetType: "config", TargetID: configID.String(),
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})
	w.WriteHeader(http.StatusNoContent)
}

// Deploy — POST /api/v1/admin/servers/{id}/deploy
// Пересобирает active config + пушит агенту. Идемпотентно.
func (h *ConfigHandler) Deploy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	if err := h.svc.Deploy(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	uid := mw.UserIDFromCtx(r.Context())
	h.audit.Log(r.Context(), domain.AuditEvent{
		ActorID: ptrUUID(uid), Action: "config.deploy", Result: "ok",
		TargetType: "server", TargetID: id.String(),
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})
	w.WriteHeader(http.StatusNoContent)
}

