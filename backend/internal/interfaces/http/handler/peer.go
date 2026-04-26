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

type PeerHandler struct{ svc *usecase.PeerService }

func NewPeer(s *usecase.PeerService) *PeerHandler { return &PeerHandler{svc: s} }

func (h *PeerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePeerRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	uid := mw.UserIDFromCtx(r.Context())
	peer, conf, err := h.svc.Provision(r.Context(), usecase.CreatePeerInput{
		UserID:   uid,
		ServerID: req.ServerID,
		Name:     req.Name,
	})
	if err != nil && peer == nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.CreatePeerResponse{
		Peer:   dto.PeerFromDomain(peer),
		Config: conf,
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
	if err := h.svc.Revoke(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
