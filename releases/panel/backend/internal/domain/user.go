package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role — RBAC уровни доступа.
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleUser     Role = "user"
)

func (r Role) Valid() bool {
	switch r {
	case RoleAdmin, RoleOperator, RoleUser:
		return true
	}
	return false
}

// User — корневая сущность аккаунта.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         Role
	Disabled     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewUser — конструктор с валидацией.
func NewUser(email string, hash string, role Role) (*User, error) {
	if email == "" || hash == "" {
		return nil, ErrValidation
	}
	if !role.Valid() {
		return nil, ErrValidation
	}
	now := time.Now().UTC()
	return &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (u *User) IsAdmin() bool    { return u.Role == RoleAdmin }
func (u *User) IsOperator() bool { return u.Role == RoleOperator || u.Role == RoleAdmin }
