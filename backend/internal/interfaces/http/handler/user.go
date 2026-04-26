package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/usecase"
	"github.com/voidwg/control/internal/domain"
	"github.com/voidwg/control/internal/interfaces/http/dto"
)

type UserHandler struct{ svc *usecase.UserService }

func NewUser(s *usecase.UserService) *UserHandler { return &UserHandler{svc: s} }

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateUserRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	role := domain.Role(req.Role)
	if !role.Valid() {
		role = domain.RoleUser
	}
	u, err := h.svc.Create(r.Context(), usecase.CreateUserInput{Email: req.Email, Password: req.Password, Role: role})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.UserFromDomain(u))
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	users, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		writeErr(w, err)
		return
	}
	out := make([]dto.UserResponse, 0, len(users))
	for _, u := range users {
		out = append(out, dto.UserFromDomain(u))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
