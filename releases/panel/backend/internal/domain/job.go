package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// JobType — типы задач, которые агент выполняет.
type JobType string

const (
	JobDeploy       JobType = "deploy"
	JobRestart      JobType = "restart"
	JobUpdate       JobType = "update"
	JobRotateSecret JobType = "rotate_secret"
	JobExec         JobType = "exec"
)

// JobStatus — состояние задачи в очереди.
type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobRunning   JobStatus = "running"
	JobSuccess   JobStatus = "success"
	JobFailed    JobStatus = "failed"
	JobCancelled JobStatus = "cancelled"
)

// Job — distributed task для агента.
//
// Цикл жизни:
//   pending → running (lease) → success | failed (retry если attempts<max)
//
// Идемпотентность: payload содержит deterministic-input (например config_hash),
// агент сравнивает с текущим состоянием и отдаёт success без apply, если
// уже на нужной версии.
type Job struct {
	ID          uuid.UUID
	ServerID    uuid.UUID
	Type        JobType
	Status      JobStatus
	Payload     json.RawMessage
	Result      json.RawMessage
	Error       string
	Attempts    int
	MaxAttempts int
	NextRunAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ServerStatus — расширенная state-model (Phase 8).
//
// Старый bool Online остаётся для backward-compat; реальное состояние теперь
// в поле Status: pending | online | offline | deploying | error | degraded | drifted.
const (
	ServerPending   = "pending"
	ServerOnline    = "online"
	ServerOffline   = "offline"
	ServerDeploying = "deploying"
	ServerError     = "error"
	ServerDegraded  = "degraded"
	ServerDrifted   = "drifted"
)
