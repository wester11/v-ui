package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/usecase"
	"github.com/voidwg/control/internal/domain"
	"github.com/voidwg/control/internal/interfaces/http/dto"
)

type ServerHandler struct{ svc *usecase.ServerService }

func NewServer(s *usecase.ServerService) *ServerHandler { return &ServerHandler{svc: s} }

func (h *ServerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateServerRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	srv, err := h.svc.Register(r.Context(), usecase.RegisterServerInput{
		Name:        req.Name,
		Endpoint:    req.Endpoint,
		ListenPort:  req.ListenPort,
		Subnet:      req.Subnet,
		DNS:         req.DNS,
		ObfsEnabled: req.ObfsEnabled,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	resp := dto.ServerFromDomain(srv)
	// Возвращаем agent_token только при создании
	writeJSON(w, http.StatusCreated, struct {
		dto.ServerResponse
		AgentToken string `json:"agent_token"`
	}{resp, srv.AgentToken})
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
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
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
