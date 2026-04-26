package persistence

import (
	"context"
	"errors"
	"net/netip"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/voidwg/control/internal/domain"
)

type ServerRepo struct{ db *pgxpool.Pool }

func NewServerRepo(db *pgxpool.Pool) *ServerRepo { return &ServerRepo{db: db} }

func (r *ServerRepo) Create(ctx context.Context, s *domain.Server) error {
	dns := make([]string, 0, len(s.DNS))
	for _, a := range s.DNS {
		dns = append(dns, a.String())
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO servers (id,name,endpoint,public_key,listen_port,subnet,dns,obfs_enabled,agent_token,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		s.ID, s.Name, s.Endpoint, s.PublicKey, int(s.ListenPort), s.Subnet.String(),
		strings.Join(dns, ","), s.ObfsEnabled, s.AgentToken, s.CreatedAt, s.UpdatedAt)
	return err
}

func (r *ServerRepo) Update(ctx context.Context, s *domain.Server) error {
	dns := make([]string, 0, len(s.DNS))
	for _, a := range s.DNS {
		dns = append(dns, a.String())
	}
	_, err := r.db.Exec(ctx, `
		UPDATE servers SET name=$2,endpoint=$3,listen_port=$4,subnet=$5,dns=$6,obfs_enabled=$7,
		                  online=$8,last_heartbeat=$9,updated_at=NOW() WHERE id=$1`,
		s.ID, s.Name, s.Endpoint, int(s.ListenPort), s.Subnet.String(),
		strings.Join(dns, ","), s.ObfsEnabled, s.Online, s.LastHeartbeat)
	return err
}

func (r *ServerRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Server, error) {
	row := r.db.QueryRow(ctx, serverSelect+` WHERE id=$1`, id)
	return scanServer(row)
}

func (r *ServerRepo) GetByToken(ctx context.Context, token string) (*domain.Server, error) {
	row := r.db.QueryRow(ctx, serverSelect+` WHERE agent_token=$1`, token)
	return scanServer(row)
}

func (r *ServerRepo) List(ctx context.Context) ([]*domain.Server, error) {
	rows, err := r.db.Query(ctx, serverSelect+` ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Server
	for rows.Next() {
		s, err := scanServer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ServerRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM servers WHERE id=$1`, id)
	return err
}

func (r *ServerRepo) CountOnline(ctx context.Context) (int, int, error) {
	var total, online int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN online THEN 1 ELSE 0 END),0) FROM servers`).
		Scan(&total, &online)
	return total, online, err
}

const serverSelect = `SELECT id,name,endpoint,public_key,listen_port,subnet,dns,obfs_enabled,
                            agent_token,online,last_heartbeat,created_at,updated_at FROM servers`

func scanServer(s scanner) (*domain.Server, error) {
	srv := &domain.Server{}
	var subnet, dns string
	var lp int
	err := s.Scan(&srv.ID, &srv.Name, &srv.Endpoint, &srv.PublicKey, &lp, &subnet, &dns,
		&srv.ObfsEnabled, &srv.AgentToken, &srv.Online, &srv.LastHeartbeat, &srv.CreatedAt, &srv.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	srv.ListenPort = uint16(lp)
	pfx, err := netip.ParsePrefix(subnet)
	if err == nil {
		srv.Subnet = pfx
	}
	if dns != "" {
		for _, d := range strings.Split(dns, ",") {
			if a, err := netip.ParseAddr(strings.TrimSpace(d)); err == nil {
				srv.DNS = append(srv.DNS, a)
			}
		}
	}
	return srv, nil
}
