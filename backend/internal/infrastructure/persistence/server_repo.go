package persistence

import (
	"context"
	"encoding/json"
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
	dns := dnsJoin(s.DNS)
	subnet := ""
	if s.Subnet.IsValid() {
		subnet = s.Subnet.String()
	}
	awg, err := json.Marshal(s.AWG)
	if err != nil {
		return err
	}
	pcfg := s.ProtocolConfig
	if len(pcfg) == 0 {
		pcfg = []byte("{}")
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO servers
		    (id,name,node_id,node_secret,hostname,ip,status,agent_version,
		     protocol,protocol_config,endpoint,public_key,listen_port,tcp_port,tls_port,subnet,dns,
		     obfs_enabled,awg_params,agent_token,agent_cert_fingerprint,online,last_heartbeat,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,
		        $9,$10,$11,$12,$13,$14,$15,$16,$17,
		        $18,$19,$20,$21,$22,$23,$24,$25)`,
		s.ID, s.Name, s.NodeID, s.NodeSecret, s.Hostname, s.IP, s.Status, s.AgentVersion,
		string(s.Protocol), pcfg, s.Endpoint, s.PublicKey, int(s.ListenPort), int(s.TCPPort), int(s.TLSPort),
		subnet, dns, s.ObfsEnabled, awg, s.AgentToken, s.AgentCertFingerprint, s.Online, s.LastHeartbeat,
		s.CreatedAt, s.UpdatedAt)
	return err
}

func (r *ServerRepo) Update(ctx context.Context, s *domain.Server) error {
	dns := dnsJoin(s.DNS)
	subnet := ""
	if s.Subnet.IsValid() {
		subnet = s.Subnet.String()
	}
	awg, err := json.Marshal(s.AWG)
	if err != nil {
		return err
	}
	pcfg := s.ProtocolConfig
	if len(pcfg) == 0 {
		pcfg = []byte("{}")
	}
	_, err = r.db.Exec(ctx, `
		UPDATE servers SET
		    name=$2,node_id=$3,node_secret=$4,hostname=$5,ip=$6,status=$7,agent_version=$8,
		    protocol=$9,protocol_config=$10,endpoint=$11,public_key=$12,
		    listen_port=$13,tcp_port=$14,tls_port=$15,subnet=$16,dns=$17,
		    obfs_enabled=$18,awg_params=$19,agent_token=$20,agent_cert_fingerprint=$21,
		    online=$22,last_heartbeat=$23,updated_at=NOW()
		WHERE id=$1`,
		s.ID, s.Name, s.NodeID, s.NodeSecret, s.Hostname, s.IP, s.Status, s.AgentVersion,
		string(s.Protocol), pcfg, s.Endpoint, s.PublicKey,
		int(s.ListenPort), int(s.TCPPort), int(s.TLSPort), subnet, dns,
		s.ObfsEnabled, awg, s.AgentToken, s.AgentCertFingerprint, s.Online, s.LastHeartbeat)
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

func (r *ServerRepo) GetByNodeID(ctx context.Context, nodeID uuid.UUID) (*domain.Server, error) {
	row := r.db.QueryRow(ctx, serverSelect+` WHERE node_id=$1`, nodeID)
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
	err := r.db.QueryRow(ctx, `SELECT COUNT(*), COALESCE(SUM(CASE WHEN online THEN 1 ELSE 0 END),0) FROM servers`).Scan(&total, &online)
	return total, online, err
}

const serverSelect = `SELECT id,name,node_id,node_secret,hostname,ip,status,agent_version,
                             protocol,protocol_config,endpoint,public_key,listen_port,tcp_port,tls_port,
                             subnet,dns,obfs_enabled,awg_params,agent_token,agent_cert_fingerprint,
                             online,last_heartbeat,created_at,updated_at
                      FROM servers`

func scanServer(s scanner) (*domain.Server, error) {
	srv := &domain.Server{}
	var subnet, dns, proto string
	var lp, tcpp, tlsp int
	var awg, pcfg []byte
	err := s.Scan(
		&srv.ID, &srv.Name, &srv.NodeID, &srv.NodeSecret, &srv.Hostname, &srv.IP, &srv.Status, &srv.AgentVersion,
		&proto, &pcfg, &srv.Endpoint, &srv.PublicKey, &lp, &tcpp, &tlsp,
		&subnet, &dns, &srv.ObfsEnabled, &awg, &srv.AgentToken, &srv.AgentCertFingerprint,
		&srv.Online, &srv.LastHeartbeat, &srv.CreatedAt, &srv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	srv.Protocol = domain.Protocol(proto)
	srv.ProtocolConfig = pcfg
	srv.ListenPort = uint16(lp)
	srv.TCPPort = uint16(tcpp)
	srv.TLSPort = uint16(tlsp)
	if pfx, err := netip.ParsePrefix(subnet); err == nil {
		srv.Subnet = pfx
	}
	if dns != "" {
		for _, d := range strings.Split(dns, ",") {
			if a, err := netip.ParseAddr(strings.TrimSpace(d)); err == nil {
				srv.DNS = append(srv.DNS, a)
			}
		}
	}
	if len(awg) > 0 {
		_ = json.Unmarshal(awg, &srv.AWG)
	}
	return srv, nil
}

func dnsJoin(addrs []netip.Addr) string {
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, a.String())
	}
	return strings.Join(out, ",")
}
