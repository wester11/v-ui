package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/voidwg/control/internal/domain"
)

type ConfigVersionRepo struct{ db *pgxpool.Pool }

func NewConfigVersionRepo(db *pgxpool.Pool) *ConfigVersionRepo { return &ConfigVersionRepo{db: db} }

func (r *ConfigVersionRepo) Create(ctx context.Context, v *domain.ConfigVersion) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	if v.Status == "" {
		v.Status = domain.CVStatusActive
	}
	cfg := v.ConfigJSON
	if len(cfg) == 0 {
		cfg = []byte("{}")
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO config_versions (id, server_id, version, config_json, config_hash,
		                              status, note, actor_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())`,
		v.ID, v.ServerID, v.Version, cfg, v.ConfigHash,
		string(v.Status), v.Note, v.ActorID)
	return err
}

func (r *ConfigVersionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.ConfigVersion, error) {
	row := r.db.QueryRow(ctx, cvSelect+` WHERE id=$1`, id)
	return scanConfigVersion(row)
}

func (r *ConfigVersionRepo) ListByServer(ctx context.Context, serverID uuid.UUID, limit int) ([]*domain.ConfigVersion, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, cvSelect+` WHERE server_id=$1 ORDER BY version DESC LIMIT $2`, serverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.ConfigVersion
	for rows.Next() {
		v, err := scanConfigVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *ConfigVersionRepo) NextVersion(ctx context.Context, serverID uuid.UUID) (int, error) {
	var v int
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(MAX(version),0)+1 FROM config_versions WHERE server_id=$1`,
		serverID).Scan(&v)
	return v, err
}

func (r *ConfigVersionRepo) MarkStatus(ctx context.Context, id uuid.UUID, status domain.ConfigVersionStatus) error {
	_, err := r.db.Exec(ctx, `UPDATE config_versions SET status=$2 WHERE id=$1`, id, string(status))
	return err
}

const cvSelect = `SELECT id,server_id,version,config_json,config_hash,status,note,actor_id,created_at FROM config_versions`

func scanConfigVersion(s scanner) (*domain.ConfigVersion, error) {
	v := &domain.ConfigVersion{}
	var status string
	err := s.Scan(&v.ID, &v.ServerID, &v.Version, &v.ConfigJSON, &v.ConfigHash,
		&status, &v.Note, &v.ActorID, &v.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	v.Status = domain.ConfigVersionStatus(status)
	return v, nil
}
