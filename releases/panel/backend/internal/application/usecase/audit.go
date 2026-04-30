package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
)

// AuditService — лёгкая обёртка над репозиторием журнала.
//
// API: nil-friendly — если actor unknown, ActorID=nil.
type AuditService struct {
	repo port.AuditRepository
}

func NewAuditService(r port.AuditRepository) *AuditService { return &AuditService{repo: r} }

// Log — fire-and-forget. Никогда не возвращает ошибку наружу — журнал не должен
// блокировать основной поток. Ошибки записи попадают только в логгер.
func (s *AuditService) Log(ctx context.Context, ev domain.AuditEvent) {
	_ = s.repo.Append(ctx, &ev)
}

func (s *AuditService) List(ctx context.Context, limit int, before int64) ([]*domain.AuditEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return s.repo.List(ctx, limit, before)
}

// AuditUser — helper для построения события с именем пользователя.
func AuditUser(actorID uuid.UUID, email, action, targetType, targetID string) domain.AuditEvent {
	return domain.AuditEvent{
		ActorID:    &actorID,
		ActorEmail: email,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Result:     "ok",
	}
}
