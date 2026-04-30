package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/voidwg/control/internal/domain"
)

type InviteRepo struct{ db *pgxpool.Pool }

func NewInviteRepo(db *pgxpool.Pool) *InviteRepo { return &InviteRepo{db: db} }

func (r *InviteRepo) Create(ctx context.Context, inv *domain.Invite) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO invites (id,token,server_id,user_id,suggested_name,expires_at,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		inv.ID, inv.Token, inv.ServerID, inv.UserID, inv.SuggestedName, inv.ExpiresAt, inv.CreatedAt)
	return err
}

func (r *InviteRepo) GetByToken(ctx context.Context, token string) (*domain.Invite, error) {
	row := r.db.QueryRow(ctx, inviteSelect+` WHERE token=$1`, token)
	return scanInvite(row)
}

func (r *InviteRepo) MarkUsed(ctx context.Context, id uuid.UUID, peerID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE invites SET used_at=NOW(), peer_id=$2 WHERE id=$1`, id, peerID)
	return err
}

func (r *InviteRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Invite, error) {
	rows, err := r.db.Query(ctx, inviteSelect+` WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Invite
	for rows.Next() {
		inv, err := scanInvite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (r *InviteRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM invites WHERE id=$1`, id)
	return err
}

func (r *InviteRepo) DeleteExpired(ctx context.Context) (int, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM invites WHERE expires_at < NOW() AND used_at IS NULL`)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

const inviteSelect = `SELECT id,token,server_id,user_id,suggested_name,
                            expires_at,used_at,peer_id,created_at FROM invites`

func scanInvite(s scanner) (*domain.Invite, error) {
	inv := &domain.Invite{}
	err := s.Scan(&inv.ID, &inv.Token, &inv.ServerID, &inv.UserID, &inv.SuggestedName,
		&inv.ExpiresAt, &inv.UsedAt, &inv.PeerID, &inv.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return inv, nil
}
