package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/voidwg/control/internal/domain"
)

type ConfigRepo struct{ db *pgxpool.Pool }

func NewConfigRepo(db *pgxpool.Pool) *ConfigRepo { return &ConfigRepo{db: db} }

func (r *ConfigRepo) Create(ctx context.Context, cfg *domain.VPNConfig) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO vpn_configs
		    (id,server_id,name,protocol,template,setup_mode,routing_mode,settings,is_active,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		cfg.ID, cfg.ServerID, cfg.Name, string(cfg.Protocol), string(cfg.Template), string(cfg.SetupMode),
		string(cfg.RoutingMode), cfg.Settings, cfg.IsActive, cfg.CreatedAt, cfg.UpdatedAt)
	return err
}

func (r *ConfigRepo) Update(ctx context.Context, cfg *domain.VPNConfig) error {
	_, err := r.db.Exec(ctx, `
		UPDATE vpn_configs
		SET name=$2,protocol=$3,template=$4,setup_mode=$5,routing_mode=$6,settings=$7,is_active=$8,updated_at=NOW()
		WHERE id=$1`,
		cfg.ID, cfg.Name, string(cfg.Protocol), string(cfg.Template), string(cfg.SetupMode),
		string(cfg.RoutingMode), cfg.Settings, cfg.IsActive)
	return err
}

func (r *ConfigRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.VPNConfig, error) {
	row := r.db.QueryRow(ctx, configSelect+` WHERE id=$1`, id)
	return scanConfig(row)
}

func (r *ConfigRepo) ListByServer(ctx context.Context, serverID uuid.UUID) ([]*domain.VPNConfig, error) {
	rows, err := r.db.Query(ctx, configSelect+` WHERE server_id=$1 ORDER BY created_at DESC`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*domain.VPNConfig, 0)
	for rows.Next() {
		cfg, err := scanConfig(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, cfg)
	}
	return out, rows.Err()
}

func (r *ConfigRepo) GetActiveByServer(ctx context.Context, serverID uuid.UUID) (*domain.VPNConfig, error) {
	row := r.db.QueryRow(ctx, configSelect+` WHERE server_id=$1 AND is_active=true LIMIT 1`, serverID)
	return scanConfig(row)
}

func (r *ConfigRepo) SetActive(ctx context.Context, serverID, configID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE vpn_configs SET is_active=false WHERE server_id=$1`, serverID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE vpn_configs SET is_active=true, updated_at=NOW() WHERE id=$1 AND server_id=$2`, configID, serverID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *ConfigRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM vpn_configs WHERE id=$1`, id)
	return err
}

const configSelect = `SELECT id,server_id,name,protocol,template,setup_mode,routing_mode,settings,is_active,created_at,updated_at FROM vpn_configs`

func scanConfig(s scanner) (*domain.VPNConfig, error) {
	cfg := &domain.VPNConfig{}
	var protocol, template, setupMode, routingMode string
	err := s.Scan(
		&cfg.ID, &cfg.ServerID, &cfg.Name, &protocol, &template, &setupMode, &routingMode,
		&cfg.Settings, &cfg.IsActive, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	cfg.Protocol = domain.Protocol(protocol)
	cfg.Template = domain.ConfigTemplate(template)
	cfg.SetupMode = domain.ConfigSetupMode(setupMode)
	cfg.RoutingMode = domain.ConfigRoutingMode(routingMode)
	return cfg, nil
}

