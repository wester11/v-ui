package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/voidwg/control/internal/domain"
)

type JobRepo struct{ db *pgxpool.Pool }

func NewJobRepo(db *pgxpool.Pool) *JobRepo { return &JobRepo{db: db} }

func (r *JobRepo) Create(ctx context.Context, j *domain.Job) error {
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}
	if j.Status == "" {
		j.Status = domain.JobPending
	}
	if j.MaxAttempts == 0 {
		j.MaxAttempts = 3
	}
	if j.NextRunAt.IsZero() {
		j.NextRunAt = time.Now().UTC()
	}
	payload := j.Payload
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO jobs (id, server_id, type, status, payload,
		                  attempts, max_attempts, next_run_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW(),NOW())`,
		j.ID, j.ServerID, string(j.Type), string(j.Status), payload,
		j.Attempts, j.MaxAttempts, j.NextRunAt)
	return err
}

// LeasePending — атомарно достаёт до `max` pending-задач для server'а
// и переводит их в running. SELECT FOR UPDATE SKIP LOCKED — параллельные
// агенты не дублируют работу.
func (r *JobRepo) LeasePending(ctx context.Context, serverID uuid.UUID, max int) ([]*domain.Job, error) {
	if max <= 0 {
		max = 5
	}
	rows, err := r.db.Query(ctx, `
		WITH leased AS (
			SELECT id FROM jobs
			 WHERE server_id = $1
			   AND status = 'pending'
			   AND next_run_at <= NOW()
			 ORDER BY created_at
			 LIMIT $2
			 FOR UPDATE SKIP LOCKED
		)
		UPDATE jobs SET status='running', started_at=NOW(), attempts=attempts+1, updated_at=NOW()
		 WHERE id IN (SELECT id FROM leased)
		RETURNING id, server_id, type, status, payload, result, error,
		          attempts, max_attempts, next_run_at, started_at, finished_at, created_at, updated_at`,
		serverID, max)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// Submit — финализирует задачу: success или failed с retry, если attempts<max.
func (r *JobRepo) Submit(ctx context.Context, jobID uuid.UUID, status domain.JobStatus, result []byte, errMsg string) error {
	if len(result) == 0 {
		result = []byte("{}")
	}

	// Если failed и attempts<max — переводим обратно в pending с exp-backoff.
	if status == domain.JobFailed {
		var attempts, maxAtt int
		err := r.db.QueryRow(ctx, `SELECT attempts, max_attempts FROM jobs WHERE id=$1`, jobID).
			Scan(&attempts, &maxAtt)
		if err != nil {
			return err
		}
		if attempts < maxAtt {
			// exponential backoff: 5s, 25s, 125s, ...
			delay := time.Duration(5) * time.Second
			for i := 0; i < attempts; i++ {
				delay *= 5
				if delay > 10*time.Minute {
					delay = 10 * time.Minute
					break
				}
			}
			_, err = r.db.Exec(ctx, `
				UPDATE jobs SET status='pending', result=$2, error=$3,
				                next_run_at=NOW()+$4::interval, updated_at=NOW()
				 WHERE id=$1`,
				jobID, result, errMsg,
				fmt.Sprintf("%d milliseconds", delay.Milliseconds()))
			return err
		}
	}

	_, err := r.db.Exec(ctx, `
		UPDATE jobs SET status=$2, result=$3, error=$4,
		                finished_at=NOW(), updated_at=NOW()
		 WHERE id=$1`,
		jobID, string(status), result, errMsg)
	return err
}

func (r *JobRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	row := r.db.QueryRow(ctx, jobSelect+` WHERE id=$1`, id)
	return scanJob(row)
}

func (r *JobRepo) ListByServer(ctx context.Context, serverID uuid.UUID, limit int) ([]*domain.Job, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, jobSelect+` WHERE server_id=$1 ORDER BY created_at DESC LIMIT $2`, serverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (r *JobRepo) RescheduleStale(ctx context.Context, threshold time.Duration) (int, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE jobs SET status='pending', updated_at=NOW(),
		                next_run_at=NOW()
		 WHERE status='running'
		   AND started_at < NOW() - $1::interval`,
		fmt.Sprintf("%d milliseconds", threshold.Milliseconds()))
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

const jobSelect = `SELECT id,server_id,type,status,payload,result,error,
                          attempts,max_attempts,next_run_at,started_at,finished_at,
                          created_at,updated_at FROM jobs`

func scanJob(s scanner) (*domain.Job, error) {
	j := &domain.Job{}
	var typ, status string
	err := s.Scan(&j.ID, &j.ServerID, &typ, &status, &j.Payload, &j.Result, &j.Error,
		&j.Attempts, &j.MaxAttempts, &j.NextRunAt, &j.StartedAt, &j.FinishedAt,
		&j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	j.Type = domain.JobType(typ)
	j.Status = domain.JobStatus(status)
	return j, nil
}
