export interface User {
  id: string
  email: string
  role: 'admin' | 'operator' | 'user'
  disabled: boolean
  created_at: string
}

export type Protocol = 'wireguard' | 'amneziawg' | 'xray'
export type ServerMode = 'standalone' | 'cascade'

export interface CascadeRule {
  match: string
  outbound: 'direct' | 'proxy'
}

export interface Server {
  id: string
  name: string
  protocol: Protocol
  mode?: ServerMode
  endpoint: string
  public_key?: string
  listen_port?: number
  tcp_port?: number
  tls_port?: number
  subnet?: string
  obfs_enabled: boolean
  xray_inbound_port?: number
  xray_sni?: string
  xray_public_key?: string
  cascade_upstream_id?: string
  cascade_rules?: CascadeRule[]
  online: boolean
  last_heartbeat?: string | null
}

export interface Peer {
  id: string
  user_id: string
  server_id: string
  protocol: Protocol
  name: string
  public_key?: string
  xray_uuid?: string
  xray_short_id?: string
  assigned_ip?: string
  enabled: boolean
  bytes_rx: number
  bytes_tx: number
  last_handshake?: string | null
  created_at: string
}

export interface CreatePeerResponse {
  peer: Peer
  config: string
}

export interface TokenResponse {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
}

export interface Stats {
  users: number
  peers: number
  servers: number
  servers_online: number
  bytes_rx_total: number
  bytes_tx_total: number
}

export interface CreateServerResponse extends Server {
  agent_token?: string
}

export interface AuditEntry {
  id: number
  ts: string
  actor_id?: string
  actor_email?: string
  action: string
  target_type?: string
  target_id?: string
  ip?: string
  user_agent?: string
  result: string
  meta?: Record<string, unknown>
}

export interface FleetRedeployResult {
  server_id: string
  name: string
  status: 'ok' | 'error'
  retries: number
  error?: string
}

export interface FleetHealthResult {
  server_id: string
  name: string
  protocol: string
  status: 'online' | 'offline' | 'degraded'
  last_heartbeat?: string | null
  error?: string
}
