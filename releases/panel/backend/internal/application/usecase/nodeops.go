package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
)

// NodeOpsService — операции над runtime-агентом ноды.
//
// Phase 7: эндпоинты UI:
//   POST /api/v1/admin/servers/{id}/restart   → docker restart xray
//   GET  /api/v1/admin/servers/{id}/metrics   → JSON-метрики от агента
//
// Использует тот же AgentTransport, что ApplyPeer/DeployConfig.
type NodeOpsService struct {
	servers port.ServerRepository
	agents  port.AgentTransport
}

func NewNodeOpsService(s port.ServerRepository, a port.AgentTransport) *NodeOpsService {
	return &NodeOpsService{servers: s, agents: a}
}

func (n *NodeOpsService) Restart(ctx context.Context, serverID uuid.UUID) error {
	srv, err := n.servers.GetByID(ctx, serverID)
	if err != nil {
		return err
	}
	return n.agents.RestartService(ctx, srv)
}

func (n *NodeOpsService) Metrics(ctx context.Context, serverID uuid.UUID) ([]byte, error) {
	srv, err := n.servers.GetByID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	return n.agents.Metrics(ctx, srv)
}
