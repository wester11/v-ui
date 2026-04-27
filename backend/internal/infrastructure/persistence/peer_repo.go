package persistence

import (
	"context"
	"database/sql"
	"errors"
	"net/netip"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/voidwg/control/internal/domain"
)

type PeerRepo struct{ db *pgxpool.Pool }

func NewPeerRepo(db *pgxpool.Pool) *PeerRepo { return &PeerRepo{db: db} }

func (r *PeerRepo) Create(ctx context.Context, p *domain.Peer) error {
	var assigned any
	if p.AssignedIP.IsValid() {
		assigned = p.AssignedIP.String()
	}
	var xrayUUID any
	if p.XrayUUID != "" {
		xrayUUID = p.XrayUUID
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO peers
		    (id,user_id,server_id,protocol,name,public_key,
		     xray_uuid,xray_flow,xray_short_id,
		     assigned_ip,enabled,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		p.ID, p.UserID, p.ServerID, string(p.Protocol), p.Name, p.PublicKey,
		xrayUUID, p.XrayFlow, p.XrayShortID,
		assigned, p.Enabled, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r *PeerRepo) Update(ctx context.Context, p *domain.Peer) error {
	_, err := r.db.Exec(ctx, `
		UPDATE peers SET name=$2,enabled=$3,bytes_rx=$4,bytes_tx=$5,last_handshake=$6,updated_at=NOW()
		WHERE id=$1`,
		p.ID, p.Name, p.Enabled, p.BytesRx, p.BytesTx, p.LastHandshake)
	return err
}

func (r *PeerRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Peer, error) {
	row := r.db.QueryRow(ctx, peerSelect+` WHERE id=$1`, id)
	return scanPeer(row)
}

func (r *PeerRepo) GetByPublicKey(ctx context.Context, pubKey string) (*domain.Peer, error) {
	row := r.db.QueryRow(ctx, peerSelect+` WHERE public_key=$1 AND public_key<>''`, pubKey)
	return scanPeer(row)
}

func (r *PeerRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Peer, error) {
	rows, err := r.db.Query(ctx, peerSelect+` WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPeers(rows)
}

func (r *PeerRepo) ListByServer(ctx context.Context, serverID uuid.UUID) ([]*domain.Peer, error) {
	rows, err := r.db.Query(ctx, peerSelect+` WHERE server_id=$1`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPeers(rows)
}

func (r *PeerRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM peers WHERE id=$1`, id)
	return err
}

func (r *PeerRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM peers`).Scan(&n)
	return n, err
}

func (r *PeerRepo) TotalTraffic(ctx context.Context) (uint64, uint64, error) {
	var rx, tx uint64
	err := r.db.QueryRow(ctx, `SELECT COALESCE(SUM(bytes_rx),0), COALESCE(SUM(bytes_tx),0) FROM peers`).Scan(&rx, &tx)
	return rx, tx, err
}

func (r *PeerRepo) UsedIPs(ctx context.Context, serverID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT assigned_ip FROM peers WHERE server_id=$1 AND assigned_ip IS NOT NULL`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ips []string
	for rows.Next() {
		var ip sql.NullString
		if err := rows.Scan(&ip); err != nil {
			return nil, err
		}
		if ip.Valid && ip.String != "" {
			ips = append(ips, ip.String)
		}
	}
	return ips, rows.Err()
}

const peerSelect = `SELECT id,user_id,server_id,protocol,name,public_key,
                          xray_uuid,xray_flow,xray_short_id,
                          assigned_ip,enabled,bytes_rx,bytes_tx,last_handshake,created_at,updated_at FROM peers`

func scanPeers(rows pgx.Rows) ([]*domain.Peer, error) {
	var out []*domain.Peer
	for rows.Next() {
		p, err := scanPeer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func scanPeer(s scanner) (*domain.Peer, error) {
	p := &domain.Peer{}
	var proto string
	var ip sql.NullString
	var xrayUUID sql.NullString
	err := s.Scan(&p.ID, &p.UserID, &p.ServerID, &proto, &p.Name, &p.PublicKey,
		&xrayUUID, &p.XrayFlow, &p.XrayShortID,
		&ip, &p.Enabled, &p.BytesRx, &p.BytesTx, &p.LastHandshake, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	p.Protocol = domain.Protocol(proto)
	if xrayUUID.Valid {
		p.XrayUUID = xrayUUID.String
	}
	if ip.Valid && ip.String != "" {
		if addr, err := netip.ParseAddr(ip.String); err == nil {
			p.AssignedIP = addr
			p.AllowedIPs = []netip.Prefix{netip.PrefixFrom(addr, 32)}
		}
	}
	return p, nil
}
