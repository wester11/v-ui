package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ConfigVersionStatus string

const (
	CVStatusActive     ConfigVersionStatus = "active"
	CVStatusFailed     ConfigVersionStatus = "failed"
	CVStatusRolledBack ConfigVersionStatus = "rolled_back"
)

// ConfigVersion — снимок config.json для одного server'а.
//
// Phase 8: каждый успешный Activate/Deploy создаёт запись. Используется для:
//   - rollback к предыдущей версии
//   - drift-detection (сравнение config_hash agent ↔ control-plane)
//   - audit trail (кто, когда, какой config задеплоил)
type ConfigVersion struct {
	ID         uuid.UUID
	ServerID   uuid.UUID
	Version    int
	ConfigJSON json.RawMessage
	ConfigHash string // sha256 hex(config_json)
	Status     ConfigVersionStatus
	Note       string
	ActorID    *uuid.UUID
	CreatedAt  time.Time
}
