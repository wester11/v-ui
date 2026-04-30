export interface User {
  id: string
  email: string
  role: 'admin' | 'operator' | 'user'
  disabled: boolean
  created_at: string
}

export type Protocol = 'none' | 'wireguard' | 'amneziawg' | 'xray'
export type ServerMode = 'standalone' | 'cascade'

export interface CascadeRule {
  match: string
  outbound: 'direct' | 'proxy'
}

export interface Server {
  id: string
  name: string
  node_id: string
  endpoint: string
  hostname?: string
  ip?: string
  status: 'pending' | 'online' | 'offline' | 'error'
  agent_version?: string
  online: boolean
  last_heartbeat?: string | null
  protocol?: Protocol
  mode?: ServerMode
}

export interface NodeCheckResult {
  server_id: string
  status: 'online' | 'offline'
  last_heartbeat?: string | null
  ip?: string
  hostname?: string
  agent_version?: string
}

export interface Config {
  id: string
  server_id: string
  name: string
  protocol: Exclude<Protocol, 'none'>
  template: 'vless_reality' | 'grpc_reality' | 'cascade' | 'empty'
  setup_mode: 'simple' | 'advanced'
  routing_mode: 'simple' | 'advanced' | 'cascade'
  is_active: boolean
  settings?: unknown
  created_at: string
  updated_at: string
}

export interface CreateConfigRequest {
  server_id: string
  name: string
  protocol: 'xray'
  template: 'vless_reality' | 'grpc_reality' | 'cascade' | 'empty'
  setup_mode: 'simple' | 'advanced'
  routing_mode: 'simple' | 'advanced' | 'cascade'
  activate: boolean
  raw_json?: string
  inbound_port?: number
  sni?: string
  dest?: string
  fingerprint?: string
  flow?: string
  short_ids_count?: number
  cascade_upstream_id?: string
  cascade_strategy?: string
  cascade_rules?: CascadeRule[]
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
  traffic_limit_bytes: number   // 0 = no limit
  traffic_limited_at?: string | null
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
  node_id: string
  secret: string
  install_command: string
  compose_snippet: string
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

export interface SystemVersionInfo {
  commit: string
  branch: string
  built_at: string
  uptime_seconds: number
}

export interface SystemUpdateResult { success: boolean; message: string }