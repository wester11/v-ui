package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/usecase"
	"github.com/voidwg/control/internal/domain"
	"github.com/voidwg/control/internal/interfaces/http/dto"
	mw "github.com/voidwg/control/internal/interfaces/http/middleware"
)

type InviteHandler struct {
	svc     *usecase.InviteService
	baseURL string
	audit   *usecase.AuditService
}

func NewInvite(s *usecase.InviteService, baseURL string, a *usecase.AuditService) *InviteHandler {
	return &InviteHandler{svc: s, baseURL: baseURL, audit: a}
}

// Create — operator+ выдаёт invite-token (POST /api/v1/invites).
func (h *InviteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateInviteRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	uid := mw.UserIDFromCtx(r.Context())
	inv, err := h.svc.Create(r.Context(), usecase.CreateInviteInput{
		UserID:        uid,
		ServerID:      req.ServerID,
		SuggestedName: req.SuggestedName,
		TTL:           timeFromSec(req.TTLSeconds),
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	h.audit.Log(r.Context(), domain.AuditEvent{
		ActorID: ptrUUID(uid), Action: "invite.create", Result: "ok",
		TargetType: "invite", TargetID: inv.ID.String(),
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})
	writeJSON(w, http.StatusCreated, dto.InviteFromDomain(inv, h.baseURL))
}

// List — operator+ просматривает свои invite'ы.
func (h *InviteHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := mw.UserIDFromCtx(r.Context())
	list, err := h.svc.ListByUser(r.Context(), uid)
	if err != nil {
		writeErr(w, err)
		return
	}
	out := make([]dto.InviteResponse, 0, len(list))
	for _, i := range list {
		out = append(out, dto.InviteFromDomain(i, h.baseURL))
	}
	writeJSON(w, http.StatusOK, out)
}

// Delete — отзыв invite-токена.
func (h *InviteHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
		ActorID: ptrUUID(uid), Action: "invite.delete", Result: "ok",
		TargetType: "invite", TargetID: id.String(),
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})
	w.WriteHeader(http.StatusNoContent)
}

// Lookup — публичный (без auth) GET /api/v1/invites/<token>.
// Возвращает то, что нужно клиенту, чтобы локально сгенерить keypair и собрать конфиг.
func (h *InviteHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	view, err := h.svc.Lookup(r.Context(), token)
	if err != nil {
		writeErr(w, err)
		return
	}
	srv := view.Server
	dns := make([]string, 0, len(srv.DNS))
	for _, d := range srv.DNS {
		dns = append(dns, d.String())
	}
	resp := dto.InviteLookupResponse{
		Endpoint:    srv.Endpoint,
		PublicKey:   srv.PublicKey,
		ListenPort:  srv.ListenPort,
		TCPPort:     srv.TCPPort,
		TLSPort:     srv.TLSPort,
		DNS:         dns,
		ObfsEnabled: srv.ObfsEnabled,
		AWG: dto.AWGParamsDTO{
			Jc: srv.AWG.Jc, Jmin: srv.AWG.Jmin, Jmax: srv.AWG.Jmax,
			S1: srv.AWG.S1, S2: srv.AWG.S2,
			H1: srv.AWG.H1, H2: srv.AWG.H2, H3: srv.AWG.H3, H4: srv.AWG.H4,
		},
		ExpiresAt: view.ExpiresAt,
		Suggested: view.SuggestedName,
	}
	writeJSON(w, http.StatusOK, resp)
}

// Redeem — публичный (без auth) POST /api/v1/invites/<token>/redeem.
// Тело: { public_key, name? }. Создаёт peer'а и возвращает финальный конфиг-stub.
func (h *InviteHandler) Redeem(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	var req dto.RedeemInviteRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	peer, conf, err := h.svc.Redeem(r.Context(), usecase.RedeemInput{
		Token:     token,
		PublicKey: req.PublicKey,
		Name:      req.Name,
	})
	if err != nil {
		h.audit.Log(r.Context(), domain.AuditEvent{
			Action: "invite.redeem", Result: "error",
			TargetType: "invite", TargetID: token,
			IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
			Meta: map[string]any{"err": err.Error()},
		})
		writeErr(w, err)
		return
	}
	h.audit.Log(r.Context(), domain.AuditEvent{
		Action: "invite.redeem", Result: "ok",
		TargetType: "peer", TargetID: peer.ID.String(),
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})
	writeJSON(w, http.StatusOK, dto.CreatePeerResponse{
		Peer:       dto.PeerFromDomain(peer),
		ConfigStub: conf,
	})
}

func timeFromSec(s int64) time.Duration {
	if s <= 0 {
		return 0
	}
	return time.Duration(s) * time.Second
}
