package usecase

import (
	"context"

	"github.com/voidwg/control/internal/application/port"
)

// Stats — агрегированная статистика для дашборда.
type Stats struct {
	Users          int    `json:"users"`
	Peers          int    `json:"peers"`
	Servers        int    `json:"servers"`
	ServersOnline  int    `json:"servers_online"`
	BytesRxTotal   uint64 `json:"bytes_rx_total"`
	BytesTxTotal   uint64 `json:"bytes_tx_total"`
}

// StatsService собирает счётчики для UI.
type StatsService struct {
	users   port.UserRepository
	peers   port.PeerRepository
	servers port.ServerRepository
}

func NewStatsService(u port.UserRepository, p port.PeerRepository, s port.ServerRepository) *StatsService {
	return &StatsService{users: u, peers: p, servers: s}
}

func (s *StatsService) Get(ctx context.Context) (*Stats, error) {
	out := &Stats{}

	if n, err := s.users.Count(ctx); err == nil {
		out.Users = n
	} else {
		return nil, err
	}

	if n, err := s.peers.Count(ctx); err == nil {
		out.Peers = n
	} else {
		return nil, err
	}

	total, online, err := s.servers.CountOnline(ctx)
	if err != nil {
		return nil, err
	}
	out.Servers = total
	out.ServersOnline = online

	rx, tx, err := s.peers.TotalTraffic(ctx)
	if err != nil {
		return nil, err
	}
	out.BytesRxTotal = rx
	out.BytesTxTotal = tx

	return out, nil
}
