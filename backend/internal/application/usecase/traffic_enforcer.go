package usecase

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
)

// TrafficEnforcer runs as a background goroutine (every 60 seconds).
// It finds all enabled peers that have exceeded their traffic_limit_bytes
// and disables them, recording the time in traffic_limited_at.
//
// Usage in main.go:
//
//	enforcer := usecase.NewTrafficEnforcer(peerRepo, auditSvc, &log)
//	go enforcer.Run(ctx)
type TrafficEnforcer struct {
	peers  port.PeerRepository
	audit  *AuditService
	log    *zerolog.Logger
	ticker time.Duration
}

func NewTrafficEnforcer(peers port.PeerRepository, audit *AuditService, log *zerolog.Logger) *TrafficEnforcer {
	return &TrafficEnforcer{
		peers:  peers,
		audit:  audit,
		log:    log,
		ticker: 60 * time.Second,
	}
}

// Run blocks until ctx is cancelled, ticking every 60 s.
func (e *TrafficEnforcer) Run(ctx context.Context) {
	t := time.NewTicker(e.ticker)
	defer t.Stop()
	// run once immediately on startup
	e.enforce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			e.enforce(ctx)
		}
	}
}

// enforce is a single enforcement pass.
func (e *TrafficEnforcer) enforce(ctx context.Context) {
	peers, err := e.peers.ListOverLimit(ctx)
	if err != nil {
		e.log.Error().Err(err).Msg("traffic_enforcer: list_over_limit")
		return
	}
	if len(peers) == 0 {
		return
	}

	now := time.Now().UTC()
	disabled := 0
	for _, p := range peers {
		p.Enabled = false
		p.TrafficLimitedAt = &now
		if err := e.peers.Update(ctx, p); err != nil {
			e.log.Error().Err(err).Str("peer_id", p.ID.String()).Msg("traffic_enforcer: update_peer")
			continue
		}
		disabled++
		used := p.BytesRx + p.BytesTx
		e.log.Info().
			Str("peer_id", p.ID.String()).
			Str("peer_name", p.Name).
			Uint64("used_bytes", used).
			Uint64("limit_bytes", p.TrafficLimitBytes).
			Msg("traffic_enforcer: peer disabled (limit reached)")

		e.audit.Log(ctx, domain.AuditEvent{
			Action:     "peer.traffic_limit_reached",
			Result:     "ok",
			TargetType: "peer",
			TargetID:   p.ID.String(),
			Meta: map[string]any{
				"used_bytes":  used,
				"limit_bytes": p.TrafficLimitBytes,
				"peer_name":   p.Name,
			},
		})
	}
	if disabled > 0 {
		e.log.Info().Int("count", disabled).Msg("traffic_enforcer: enforcement pass complete")
	}
}
