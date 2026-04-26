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

type ServerHandler struct {
	svc   *usecase.ServerService
	audit *usecase.AuditService
}

func NewServer(s *usecase.ServerService, a *usecase.AuditService) *ServerHandler {
	return &ServerHandler{svc: s, audit: a}
}

func (h *ServerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateServerRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	res, err := h.svc.Register(r.Context(), usecase.RegisterServerInput{
		Name:            req.Name,
		Protocol:        domain.Protocol(req.Protocol),
		Endpoint:        req.Endpoint,
		ListenPort:      req.ListenPort,
		TCPPort:         req.TCPPort,
		TLSPort:         req.TLSPort,
		Subnet:          req.Subnet,
		DNS:             req.DNS,
		ObfsEnabled:     req.ObfsEnabled,
		XrayInboundPort: req.XrayInboundPort,
		XraySNI:         req.XraySNI,
		XrayDest:        req.XrayDest,
		XrayShortIDsN:   req.XrayShortIDsN,
		XrayFingerprint: req.XrayFingerprint,
		XrayFlow:        req.XrayFlow,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	uid := mw.UserIDFromCtx(r.Context())
	h.audit.Log(r.Context(), domain.AuditEvent{
		ActorID: ptrUUID(uid), Action: "server.register", Result: "ok",
		TargetType: "server", TargetID: res.Server.ID.String(),
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})
	writeJSON(w, http.StatusCreated, dto.CreateServerResponse{
		ServerResponse: dto.ServerFromDomain(res.Server),
		AgentToken:     res.Server.AgentToken,
		AgentCA:        string(res.AgentCA),
		AgentCert:      string(res.AgentCert),
		AgentKey:       string(res.AgentKey),
	})
}

func (h *ServerHandler) List(w http.ResponseWriter, r *http.Request) {
	srv, err := h.svc.List(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	out := make([]dto.ServerResponse, 0, len(srv))
	for _, s := range srv {
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

func (h *ServerHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Agent-Token")
	if token == "" {
		writeErr(w, domain.ErrInvalidCredential)
		return
	}
	srv, err := h.svc.Heartbeat(r.Context(), token)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.ServerFromDomain(srv))
}
