package handler

import (
	"net/http"

	"github.com/voidwg/control/internal/application/usecase"
	"github.com/voidwg/control/internal/interfaces/http/dto"
	mw "github.com/voidwg/control/internal/interfaces/http/middleware"
)

type AuthHandler struct {
	auth  *usecase.Auth
	users *usecase.UserService
}

func NewAuth(a *usecase.Auth, u *usecase.UserService) *AuthHandler {
	return &AuthHandler{auth: a, users: u}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	pair, _, err := h.auth.Login(r.Context(), usecase.LoginInput{Email: req.Email, Password: req.Password})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.TokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    pair.ExpiresIn,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	pair, err := h.auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.TokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    pair.ExpiresIn,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	uid := mw.UserIDFromCtx(r.Context())
	u, err := h.users.Get(r.Context(), uid)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.UserFromDomain(u))
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	uid := mw.UserIDFromCtx(r.Context())
	var req dto.ChangePasswordRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	if err := h.auth.ChangePassword(r.Context(), uid, req.OldPassword, req.NewPassword); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
