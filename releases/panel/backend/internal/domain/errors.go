package domain

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrAlreadyExists     = errors.New("already exists")
	ErrInvalidCredential = errors.New("invalid credentials")
	ErrForbidden         = errors.New("forbidden")
	ErrValidation        = errors.New("validation error")
	ErrInvalidInput      = errors.New("invalid input")
	ErrPoolExhausted     = errors.New("ip pool exhausted")
)
