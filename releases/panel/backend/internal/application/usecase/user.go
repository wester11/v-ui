package usecase

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
)

type UserService struct {
	repo   port.UserRepository
	hasher port.PasswordHasher
}

func NewUserService(r port.UserRepository, h port.PasswordHasher) *UserService {
	return &UserService{repo: r, hasher: h}
}

type CreateUserInput struct {
	Email    string
	Password string
	Role     domain.Role
}

func (s *UserService) Create(ctx context.Context, in CreateUserInput) (*domain.User, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if existing, _ := s.repo.GetByEmail(ctx, email); existing != nil {
		return nil, domain.ErrAlreadyExists
	}
	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}
	u, err := domain.NewUser(email, hash, in.Role)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) Get(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repo.List(ctx, limit, offset)
}

func (s *UserService) Disable(ctx context.Context, id uuid.UUID, disabled bool) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	u.Disabled = disabled
	return s.repo.Update(ctx, u)
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
