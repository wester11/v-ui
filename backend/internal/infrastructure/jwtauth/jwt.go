// Package jwtauth — JWT адаптер.
package jwtauth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
)

type Issuer struct {
	secret []byte
	issuer string
}

func New(secret []byte, issuer string) *Issuer {
	return &Issuer{secret: secret, issuer: issuer}
}

type customClaims struct {
	Role string `json:"role,omitempty"`
	Typ  string `json:"typ,omitempty"`
	jwt.RegisteredClaims
}

func (i *Issuer) Issue(uid uuid.UUID, role domain.Role, ttl time.Duration) (string, error) {
	now := time.Now()
	c := customClaims{
		Role: string(role),
		Typ:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   uid.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        uuid.NewString(),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return t.SignedString(i.secret)
}

func (i *Issuer) IssueRefresh(uid uuid.UUID, ttl time.Duration) (string, error) {
	now := time.Now()
	c := customClaims{
		Typ: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   uid.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        uuid.NewString(),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return t.SignedString(i.secret)
}

func (i *Issuer) Verify(token string) (port.Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &customClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return i.secret, nil
	})
	if err != nil {
		return port.Claims{}, err
	}
	c, ok := parsed.Claims.(*customClaims)
	if !ok || !parsed.Valid {
		return port.Claims{}, errors.New("invalid token")
	}
	uid, err := uuid.Parse(c.Subject)
	if err != nil {
		return port.Claims{}, err
	}
	return port.Claims{
		UserID: uid,
		Role:   domain.Role(c.Role),
		Exp:    c.ExpiresAt.Time,
	}, nil
}
