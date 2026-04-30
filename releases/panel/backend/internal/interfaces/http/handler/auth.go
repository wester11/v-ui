package handler

import (
	"net/http"

	"github.com/voidwg/control/internal/application/usecase"
	"github.com/voidwg/control/internal/domain"
	"github.com/voidwg/control/internal/interfaces/http/dto"
	mw "github.com/voidwg/control/internal/interfaces/http/middleware"
)

type AuthHandler struct {
	auth  *usecase.Auth
	users *usecase.UserService
	audit *usecase.AuditService
}

func NewAuth(a *usecase.Auth, u *usecase.UserService, audit *usecase.AuditService) *AuthHandler {
	return &AuthHandler{auth: a, users: u, audit: audit}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	pair, user, err := h.auth.Login(r.Context(), usecase.LoginInput{Email: req.Email, Password: req.Password})
	if err != nil {
		h.audit.Log(r.Context(), domain.AuditEvent{
			Action: "auth.login", Result: "denied",
			ActorEmail: req.Email,
			IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
		})
		writeErr(w, err)
		return
	}
	h.audit.Log(r.Context(), domain.AuditEvent{
		Action: "auth.login", Result: "ok",
		ActorID: ptrUUID(user.ID), ActorEmail: user.Email,
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})
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
		h.audit.Log(r.Context(), domain.AuditEvent{
			ActorID: ptrUUID(uid), Action: "auth.change_password", Result: "denied",
			IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
		})
		writeErr(w, err)
		return
	}
	h.audit.Log(r.Context(), domain.AuditEvent{
		ActorID: ptrUUID(uid), Action: "auth.change_password", Result: "ok",
		IP: mw.ClientIP(r), UserAgent: r.UserAgent(),
	})
	w.WriteHeader(http.StatusNoContent)
}
