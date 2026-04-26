export interface User {
  id: string
  email: string
  role: 'admin' | 'operator' | 'user'
  disabled: boolean
  created_at: string
}

export type Protocol = 'wireguard' | 'amneziawg' | 'xray'

export interface Server {
  id: string
  name: string
  protocol: Protocol
  endpoint: string
  public_key?: string         // wireguard / amneziawg
  listen_port?: number
  tcp_port?: number
  tls_port?: number
  subnet?: string
  obfs_enabled: boolean
  xray_inbound_port?: number  // xray
  xray_sni?: string
  xray_public_key?: string
  online: boolean
  last_heartbeat?: string | null
}

export interface Peer {
  id: string
  user_id: string
  server_id: string
  protocol: Protocol
  name: string
  public_key?: string       // WG/AWG
  xray_uuid?: string        // Xray
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
