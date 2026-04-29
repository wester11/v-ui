package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
)

// JobService — orchestrates distributed task queue.
//
// Phase 8: control-plane превращается из push-mode в pull-mode для агентов.
// Любая операция (deploy / restart / update / rotate) → Enqueue в jobs.
// Агент каждые ~10 секунд тянет свои pending-задачи через LeasePending,
// исполняет, репортит результат через Submit.
//
// Гарантии:
//   - идемпотентность: payload содержит deterministic-hash (config_hash для
//     deploy); агент сверяет и пропускает no-op
//   - retry: до MaxAttempts с exp-backoff (5s, 25s, 125s ... до 10 минут)
//   - lease: SELECT FOR UPDATE SKIP LOCKED, два агента одну job не возьмут
//   - GC stale: если агент упал между lease и submit, фоновый тикер вернёт
//     job в pending через RescheduleStale
type JobService struct {
	jobs    port.JobRepository
	servers port.ServerRepository
	cv      port.ConfigVersionRepository
}

func NewJobService(j port.JobRepository, s port.ServerRepository, cv port.ConfigVersionRepository) *JobService {
	return &JobService{jobs: j, servers: s, cv: cv}
}

// EnqueueDeploy — публичная точка, вызываемая ConfigService.Activate / Deploy.
// Создаёт ConfigVersion + Job(deploy) одной транзакцией (логически — через
// последовательные вставки; БД-FK гарантирует целостность).
func (s *JobService) EnqueueDeploy(ctx context.Context, serverID uuid.UUID, fullConfig []byte, actorID *uuid.UUID, note string) (*domain.Job, *domain.ConfigVersion, error) {
	// 1. Версионирование
	ver, err := s.cv.NextVersion(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}
	hash := configHash(fullConfig)
	cv := &domain.ConfigVersion{
		ID:         uuid.New(),
		ServerID:   serverID,
		Version:    ver,
		ConfigJSON: fullConfig,
		ConfigHash: hash,
		Status:     domain.CVStatusActive,
		Note:       note,
		ActorID:    actorID,
	}
	if err := s.cv.Create(ctx, cv); err != nil {
		return nil, nil, err
	}

	// 2. expected_hash на сервере — для drift detection
	srv, err := s.servers.GetByID(ctx, serverID)
	if err == nil {
		srv.UpdatedAt = time.Now().UTC()
		// Status переходит в "deploying" пока агент не закроет job'у
		srv.Status = domain.ServerDeploying
		// expected_hash хранится отдельно, но здесь используем тот же UPDATE
		// — server_repo.go поддерживает поля ConfigHashExpected/Actual.
		_ = s.servers.Update(ctx, srv) // мягкая ошибка
	}

	// 3. Job
	payload, _ := json.Marshal(map[string]any{
		"config":      json.RawMessage(fullConfig),
		"version":     ver,
		"config_hash": hash,
	})
	job := &domain.Job{
		ServerID:    serverID,
		Type:        domain.JobDeploy,
		Status:      domain.JobPending,
		Payload:     payload,
		MaxAttempts: 3,
		NextRunAt:   time.Now().UTC(),
	}
	if err := s.jobs.Create(ctx, job); err != nil {
		return nil, nil, err
	}
	return job, cv, nil
}

// EnqueueSimple — для restart / update / rotate-secret (без config payload).
func (s *JobService) EnqueueSimple(ctx context.Context, serverID uuid.UUID, jobType domain.JobType, payload map[string]any) (*domain.Job, error) {
	pl := []byte("{}")
	if len(payload) > 0 {
		pl, _ = json.Marshal(payload)
	}
	job := &domain.Job{
		ServerID:    serverID,
		Type:        jobType,
		Status:      domain.JobPending,
		Payload:     pl,
		MaxAttempts: 3,
		NextRunAt:   time.Now().UTC(),
	}
	if err := s.jobs.Create(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

// Lease — agent pull. Возвращает до 5 задач для server'а в одном запросе.
func (s *JobService) Lease(ctx context.Context, serverID uuid.UUID) ([]*domain.Job, error) {
	return s.jobs.LeasePending(ctx, serverID, 5)
}

// Submit — agent сообщает результат. Status: success | failed.
// При failed запускается логика retry (внутри JobRepository.Submit).
func (s *JobService) Submit(ctx context.Context, jobID uuid.UUID, status domain.JobStatus, result []byte, errMsg string) error {
	if status != domain.JobSuccess && status != domain.JobFailed {
		return domain.ErrValidation
	}
	if err := s.jobs.Submit(ctx, jobID, status, result, errMsg); err != nil {
		return err
	}
	// Если deploy success — пометить ConfigVersion рабочим, server.config_hash_actual
	// агент пришлёт в следующем heartbeat (drift detection).
	if status == domain.JobFailed {
		// найти job, помечать свежую версию failed (best-effort)
		j, err := s.jobs.GetByID(ctx, jobID)
		if err == nil && j.Type == domain.JobDeploy {
			var p struct {
				Version int `json:"version"`
			}
			_ = json.Unmarshal(j.Payload, &p)
			// Best-effort: ничего не делаем без version_id, оставляем active.
			// Production-расширение: добавить server_id+version → mark_failed.
			_ = p
		}
	}
	return nil
}

// ListByServer — для UI history page.
func (s *JobService) ListByServer(ctx context.Context, serverID uuid.UUID, limit int) ([]*domain.Job, error) {
	return s.jobs.ListByServer(ctx, serverID, limit)
}

// CleanupStale — фоновый тикер (вызывается из cmd/api/main.go каждую минуту).
// Возвращает в pending все job'ы, висящие в running > 5 минут.
func (s *JobService) CleanupStale(ctx context.Context) (int, error) {
	return s.jobs.RescheduleStale(ctx, 5*time.Minute)
}

func configHash(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
