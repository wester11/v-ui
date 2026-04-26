// Package bootstrap — однократная инициализация администратора при старте API.
package bootstrap

import (
	"context"
	"errors"
	"os"

	"github.com/rs/zerolog"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
)

// Admin создаёт админа из BOOTSTRAP_ADMIN_EMAIL / BOOTSTRAP_ADMIN_PASSWORD,
// если такого пользователя в БД ещё нет. Идемпотентно.
func Admin(
	ctx context.Context,
	repo port.UserRepository,
	hasher port.PasswordHasher,
	log *zerolog.Logger,
) error {
	email := os.Getenv("BOOTSTRAP_ADMIN_EMAIL")
	password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if email == "" || password == "" {
		log.Info().Msg("bootstrap: BOOTSTRAP_ADMIN_* env vars not set — skipping")
		return nil
	}

	existing, err := repo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		log.Info().Str("email", email).Msg("bootstrap: admin already exists")
		return nil
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}

	hash, err := hasher.Hash(password)
	if err != nil {
		return err
	}
	u, err := domain.NewUser(email, hash, domain.RoleAdmin)
	if err != nil {
		return err
	}
	if err := repo.Create(ctx, u); err != nil {
		return err
	}
	log.Info().Str("email", email).Msg("bootstrap: admin created")
	return nil
}
