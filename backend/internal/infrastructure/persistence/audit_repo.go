package persistence

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/voidwg/control/internal/domain"
)

type AuditRepo struct{ db *pgxpool.Pool }

func NewAuditRepo(db *pgxpool.Pool) *AuditRepo { return &AuditRepo{db: db} }

func (r *AuditRepo) Append(ctx context.Context, ev *domain.AuditEvent) error {
	meta := []byte("{}")
	if len(ev.Meta) > 0 {
		if b, err := json.Marshal(ev.Meta); err == nil {
			meta = b
		}
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO audit_log (actor_id,actor_email,action,target_type,target_id,ip,user_agent,result,meta)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		ev.ActorID, ev.ActorEmail, ev.Action, ev.TargetType, ev.TargetID,
		ev.IP, ev.UserAgent, ev.Result, meta)
	return err
}

func (r *AuditRepo) List(ctx context.Context, limit int, before int64) ([]*domain.AuditEvent, error) {
	args := []any{}
	q := `SELECT id,ts,actor_id,actor_email,action,target_type,target_id,ip,user_agent,result,meta
	      FROM audit_log`
	if before > 0 {
		args = append(args, before)
		q += ` WHERE id < $1`
	}
	args = append(args, limit)
	q += ` ORDER BY id DESC LIMIT $` + intToStr(len(args))

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.AuditEvent
	for rows.Next() {
		ev := &domain.AuditEvent{}
		var meta []byte
		if err := rows.Scan(&ev.ID, &ev.TS, &ev.ActorID, &ev.ActorEmail, &ev.Action,
			&ev.TargetType, &ev.TargetID, &ev.IP, &ev.UserAgent, &ev.Result, &meta); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &ev.Meta)
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
