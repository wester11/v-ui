import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import type { Peer, Server } from '../types'
import { Badge, Button, CopyField, Empty, Skeleton, StatusDot, toast } from '../components/ui'
import { formatBytes, formatRelative } from '../lib/format'

export default function ServerDetail() {
  const { id } = useParams<{ id: string }>()
  const [server, setServer] = useState<Server | null>(null)
  const [peers, setPeers] = useState<Peer[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function load() {
    if (!id) return
    try {
      const list = await api.servers.list()
      const s = list.find((x) => x.id === id)
      if (!s) {
        setError('Server not found')
        return
      }
      setServer(s)
      // Fetch peers (admin sees all; user sees own — both fine for "peers on this server" view)
      const myPeers = await api.peers.list().catch(() => [] as Peer[])
      setPeers(myPeers.filter((p) => p.server_id === id))
    } catch (e) {
      setError(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'load failed')
    }
  }

  useEffect(() => { load(); /* eslint-disable-next-line react-hooks/exhaustive-deps */ }, [id])

  const summary = useMemo(() => {
    if (!peers) return null
    const rx = peers.reduce((a, p) => a + p.bytes_rx, 0)
    const tx = peers.reduce((a, p) => a + p.bytes_tx, 0)
    const enabled = peers.filter((p) => p.enabled).length
    return { rx, tx, enabled, total: peers.length }
  }, [peers])

  if (error) {
    return (
      <div className="page">
        <Empty title="Cannot load server" sub={error} action={<Link to="/servers"><Button>← Back to servers</Button></Link>} />
      </div>
    )
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="text-sm">
            <Link to="/servers" className="text-dim">← Servers</Link>
          </div>
          <div className="page-title">{server?.name ?? <Skeleton width={180} height={28} />}</div>
          <div className="page-sub">{server?.endpoint}</div>
        </div>
        <Button onClick={load}>↻ Refresh</Button>
      </div>

      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-label">Status</div>
          <div className="stat-value">
            {server ? <StatusDot online={server.online} /> : <Skeleton height={28} width={120} />}
          </div>
          <div className="stat-meta">{server ? `Last heartbeat: ${formatRelative(server.last_heartbeat)}` : ''}</div>
        </div>
        <div className="stat-card violet">
          <div className="stat-label">Peers</div>
          <div className="stat-value">{summary ? summary.total : <Skeleton height={28} width={60} />}</div>
          <div className="stat-meta">{summary ? `${summary.enabled} enabled` : ''}</div>
        </div>
        <div className="stat-card success">
          <div className="stat-label">Total RX</div>
          <div className="stat-value">{summary ? formatBytes(summary.rx) : '—'}</div>
        </div>
        <div className="stat-card warn">
          <div className="stat-label">Total TX</div>
          <div className="stat-value">{summary ? formatBytes(summary.tx) : '—'}</div>
        </div>
      </div>

      <div className="card mb-4">
        <div className="card-header">
          <div>
            <div className="card-title">Configuration</div>
            <div className="card-sub">Read-only details of this node.</div>
          </div>
        </div>
        {server ? (
          <div className="stack">
            <Field label="Endpoint"><code>{server.endpoint}</code></Field>
            <Field label="Listen port"><code>{server.listen_port}</code></Field>
            <Field label="Subnet"><code>{server.subnet}</code></Field>
            <Field label="Obfuscation">
              {server.obfs_enabled ? <Badge tone="info">on</Badge> : <Badge>off</Badge>}
            </Field>
            <Field label="Public key"><CopyField value={server.public_key} /></Field>
            <Field label="Server ID"><CopyField value={server.id} /></Field>
          </div>
        ) : (
          <div className="stack">
            {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} height={20} />)}
          </div>
        )}
      </div>

      <div className="card">
        <div className="card-header">
          <div>
            <div className="card-title">Peers on this server</div>
            <div className="card-sub">Visible peers (your own; admins see all).</div>
          </div>
          <button
            className="btn"
            onClick={() => {
              if (server) {
                const cmd = `docker run -d --name void-wg-agent --restart=unless-stopped \\\n  --cap-add NET_ADMIN --network host \\\n  -e CONTROL_URL=http://control.example.com:8080 \\\n  -e AGENT_TOKEN=<your-token> \\\n  -e WG_IFACE=wg0 \\\n  -e OBFS_LISTEN=:51821 \\\n  -e WG_ADDR=127.0.0.1:51820 \\\n  -e OBFS_PSK=<random-32-bytes> \\\n  ghcr.io/wester11/void-wg-agent:latest`
                navigator.clipboard.writeText(cmd).then(() => toast.success('Agent install command copied'))
              }
            }}
          >
            ⧉ Copy agent command
          </button>
        </div>
        {!peers ? (
          <div className="stack">
            {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} height={36} />)}
          </div>
        ) : peers.length === 0 ? (
          <Empty title="No peers on this server" sub="Provision a peer from the Peers page." />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>IP</th>
                  <th>Status</th>
                  <th>Last handshake</th>
                  <th>RX / TX</th>
                </tr>
              </thead>
              <tbody>
                {peers.map((p) => (
                  <tr key={p.id}>
                    <td><strong>{p.name}</strong></td>
                    <td><code>{p.assigned_ip}</code></td>
                    <td>{p.enabled ? <Badge tone="success">enabled</Badge> : <Badge tone="warn">disabled</Badge>}</td>
                    <td className="text-dim">{formatRelative(p.last_handshake)}</td>
                    <td className="text-mono text-dim">↓ {formatBytes(p.bytes_rx)} · ↑ {formatBytes(p.bytes_tx)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="row" style={{ alignItems: 'center' }}>
      <div className="text-mute text-sm" style={{ width: 140, flexShrink: 0 }}>{label}</div>
      <div style={{ flex: 1 }}>{children}</div>
    </div>
  )
}
