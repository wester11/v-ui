package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
)

const (
	AccessTTL  = 15 * time.Minute
	RefreshTTL = 30 * 24 * time.Hour
)

// Auth — use-case аутентификации.
type Auth struct {
	users  port.UserRepository
	hasher port.PasswordHasher
	tokens port.TokenIssuer
}

func NewAuth(u port.UserRepository, h port.PasswordHasher, t port.TokenIssuer) *Auth {
	return &Auth{users: u, hasher: h, tokens: t}
}

type LoginInput struct {
	Email    string
	Password string
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

func (a *Auth) Login(ctx context.Context, in LoginInput) (*TokenPair, *domain.User, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	user, err := a.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, domain.ErrInvalidCredential
	}
	if user.Disabled {
		return nil, nil, domain.ErrForbidden
	}
	if !a.hasher.Verify(in.Password, user.PasswordHash) {
		return nil, nil, domain.ErrInvalidCredential
	}
	access, err := a.tokens.Issue(user.ID, user.Role, AccessTTL)
	if err != nil {
		return nil, nil, err
	}
	refresh, err := a.tokens.IssueRefresh(user.ID, RefreshTTL)
	if err != nil {
		return nil, nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(AccessTTL.Seconds()),
	}, user, nil
}

func (a *Auth) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := a.tokens.Verify(refreshToken)
	if err != nil {
		return nil, domain.ErrInvalidCredential
	}
	user, err := a.users.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, domain.ErrInvalidCredential
	}
	if user.Disabled {
		return nil, domain.ErrForbidden
	}
	access, err := a.tokens.Issue(user.ID, user.Role, AccessTTL)
	if err != nil {
		return nil, err
	}
	rfr, err := a.tokens.IssueRefresh(user.ID, RefreshTTL)
	if err != nil {
		return nil, err
	}
	return &TokenPair{AccessToken: access, RefreshToken: rfr, ExpiresIn: int64(AccessTTL.Seconds())}, nil
}

// ChangePassword меняет пароль пользователя после проверки текущего.
func (a *Auth) ChangePassword(ctx context.Context, userID uuid.UUID, oldPass, newPass string) error {
	if len(newPass) < 8 {
		return domain.ErrInvalidInput
	}
	user, err := a.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !a.hasher.Verify(oldPass, user.PasswordHash) {
		return domain.ErrInvalidCredential
	}
	hash, err := a.hasher.Hash(newPass)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	user.UpdatedAt = time.Now().UTC()
	return a.users.Update(ctx, user)
}
