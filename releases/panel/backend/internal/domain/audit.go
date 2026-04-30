package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuditEvent — запись журнала чувствительных действий.
type AuditEvent struct {
	ID         int64
	TS         time.Time
	ActorID    *uuid.UUID
	ActorEmail string
	Action     string // "auth.login", "auth.login_failed", "peer.create", "peer.delete", ...
	TargetType string // "peer", "user", "server", "invite", ""
	TargetID   string
	IP         string
	UserAgent  string
	Result     string // "ok" | "denied" | "error"
	Meta       map[string]any
}
